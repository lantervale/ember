package fonts

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf16"
)

// Installed is the font files under some directories, by family: what
// each file's name table calls it, in any case.
type Installed struct {
	families map[string]*installedFamily
}

type installedFamily struct {
	name  string
	faces []*installedFace
}

// installedFace is one font of a file: the file itself, or the font of a
// collection whose table directory is at offset.
type installedFace struct {
	path       string
	offset     uint32
	collection bool
	weight     int
	width      int
	italic     bool
}

// SystemDirs are the folders the machine keeps its fonts in; none where
// there is no file system to read.
func SystemDirs() []string {
	home, _ := os.UserHomeDir()
	under := func(base string, elem ...string) string {
		if base == "" {
			return ""
		}
		return filepath.Join(append([]string{base}, elem...)...)
	}
	var dirs []string
	switch runtime.GOOS {
	case "darwin", "ios":
		dirs = []string{"/System/Library/Fonts", "/Library/Fonts", under(home, "Library", "Fonts")}
	case "windows":
		dirs = []string{
			under(os.Getenv("WINDIR"), "Fonts"),
			under(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts"),
		}
	case "js", "wasip1":
	default:
		dirs = []string{
			"/usr/share/fonts", "/usr/local/share/fonts",
			under(home, ".fonts"), under(home, ".local", "share", "fonts"),
		}
	}
	return dirs
}

// Scan indexes the .ttf, .otf and .ttc files under dirs, reading no more
// of each than says what it is. Of two fonts a family has for one weight
// and style, the one of the more normal width is kept, then the first
// found. A file that can't be read is left out.
func Scan(dirs ...string) *Installed {
	var found []scanned
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil //nolint:nilerr // a folder that can't be read holds no fonts
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc", ".otc":
				found = append(found, scanFile(path)...)
			}
			return nil
		})
	}
	idx := &Installed{families: map[string]*installedFamily{}}
	for i := range found {
		idx.add(found[i].family, &found[i].installedFace)
	}
	for i := range found {
		l := found[i].legacy
		found[i].aliased = l != "" && !strings.EqualFold(l, found[i].family) && idx.family(l) == nil
	}
	for i := range found {
		if found[i].aliased {
			idx.add(found[i].legacy, &found[i].installedFace)
		}
	}
	return idx
}

// scanned is a font as its file names it: family the typographic family
// when it has one, legacy the family of the four-style model.
type scanned struct {
	installedFace
	family, legacy string
	aliased        bool
}

func (idx *Installed) add(family string, f *installedFace) {
	key := strings.ToLower(family)
	fam := idx.families[key]
	if fam == nil {
		fam = &installedFamily{name: family}
		idx.families[key] = fam
	}
	for i, o := range fam.faces {
		if o.weight == f.weight && o.italic == f.italic {
			if abs(f.width-normalWidth) < abs(o.width-normalWidth) {
				fam.faces[i] = f
			}
			return
		}
	}
	fam.faces = append(fam.faces, f)
}

// normalWidth is OS/2's usWidthClass for a face neither condensed nor
// expanded.
const normalWidth = 5

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// family is the family named name, in any case, nil when no file has it.
func (idx *Installed) family(name string) *installedFamily {
	if idx == nil {
		return nil
	}
	return idx.families[strings.ToLower(name)]
}

// Families is how many families the index holds.
func (idx *Installed) Families() int { return len(idx.families) }

var errNotAFont = errors.New("fonts: not a TrueType or OpenType font")

const (
	tagTTCF = 0x74746366
	tagOTTO = 0x4f54544f
	tagTrue = 0x74727565
	tagName = 0x6e616d65
	tagOS2  = 0x4f532f32
	tagHead = 0x68656164
	tagKern = 0x6b65726e
)

// scanFile is the fonts of the file at path, none when it is no font.
func scanFile(path string) []scanned {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	offsets, collection, err := fontOffsets(f)
	if err != nil {
		return nil
	}
	var out []scanned
	for _, off := range offsets {
		s, err := scanFont(f, off)
		if err != nil {
			continue
		}
		s.path, s.offset, s.collection = path, off, collection
		out = append(out, s)
	}
	return out
}

// fontOffsets is where each font's table directory starts in r: 0 for a
// font file, the collection's own list for a .ttc.
func fontOffsets(r io.ReaderAt) (offsets []uint32, collection bool, err error) {
	var head [12]byte
	if _, err := r.ReadAt(head[:], 0); err != nil {
		return nil, false, err
	}
	switch binary.BigEndian.Uint32(head[:]) {
	case 0x00010000, tagOTTO, tagTrue:
		return []uint32{0}, false, nil
	case tagTTCF:
	default:
		return nil, false, errNotAFont
	}
	n := binary.BigEndian.Uint32(head[8:])
	if n == 0 || n > 1024 {
		return nil, false, errNotAFont
	}
	buf := make([]byte, 4*n)
	if _, err := r.ReadAt(buf, 12); err != nil {
		return nil, false, err
	}
	for i := range n {
		offsets = append(offsets, binary.BigEndian.Uint32(buf[4*i:]))
	}
	return offsets, true, nil
}

type tableRecord struct{ tag, offset, length uint32 }

// tableDirectory is the tables of the font whose directory is at off.
func tableDirectory(r io.ReaderAt, off uint32) (version uint32, tables []tableRecord, err error) {
	var head [12]byte
	if _, err := r.ReadAt(head[:], int64(off)); err != nil {
		return 0, nil, err
	}
	version = binary.BigEndian.Uint32(head[:])
	if version != 0x00010000 && version != tagOTTO && version != tagTrue {
		return 0, nil, errNotAFont
	}
	n := int(binary.BigEndian.Uint16(head[4:]))
	buf := make([]byte, 16*n)
	if _, err := r.ReadAt(buf, int64(off)+12); err != nil {
		return 0, nil, err
	}
	for i := range n {
		rec := buf[16*i:]
		tables = append(tables, tableRecord{
			tag:    binary.BigEndian.Uint32(rec),
			offset: binary.BigEndian.Uint32(rec[8:]),
			length: binary.BigEndian.Uint32(rec[12:]),
		})
	}
	return version, tables, nil
}

// maxNameTable is the longest name table read: longer is no real font's.
const maxNameTable = 1 << 20

// scanFont reads the names, weight and style of the font whose table
// directory is at off: its name table, and the few bytes of OS/2 and
// head that say how bold, wide and slanted it is.
func scanFont(r io.ReaderAt, off uint32) (scanned, error) {
	_, tables, err := tableDirectory(r, off)
	if err != nil {
		return scanned{}, err
	}
	var s scanned
	s.weight, s.width = 400, normalWidth
	var bold, hasOS2 bool
	for _, t := range tables {
		switch t.tag {
		case tagName:
			if t.length > maxNameTable {
				return scanned{}, errNotAFont
			}
			buf := make([]byte, t.length)
			if _, err := r.ReadAt(buf, int64(t.offset)); err != nil {
				return scanned{}, err
			}
			s.family, s.legacy = familyNames(buf)
		case tagOS2:
			var buf [64]byte
			if t.length < 64 {
				continue
			}
			if _, err := r.ReadAt(buf[:], int64(t.offset)); err != nil {
				continue
			}
			hasOS2 = true
			s.weight = int(binary.BigEndian.Uint16(buf[4:]))
			s.width = int(binary.BigEndian.Uint16(buf[6:]))
			const italic, oblique = 1 << 0, 1 << 9
			s.italic = s.italic || binary.BigEndian.Uint16(buf[62:])&(italic|oblique) != 0
		case tagHead:
			var buf [2]byte
			if _, err := r.ReadAt(buf[:], int64(t.offset)+44); err != nil {
				continue
			}
			style := binary.BigEndian.Uint16(buf[:])
			bold = style&1 != 0
			s.italic = s.italic || style&2 != 0
		}
	}
	if s.family == "" || strings.HasPrefix(s.family, ".") {
		return scanned{}, errNotAFont
	}
	switch {
	case !hasOS2 && bold:
		s.weight = 700
	case s.weight >= 1 && s.weight <= 9:
		s.weight *= 100
	case s.weight < 1 || s.weight > 1000:
		s.weight = 400
		if bold {
			s.weight = 700
		}
	}
	return s, nil
}

// familyNames are a name table's typographic family (name 16, else the
// family) and its family (name 1), in English where it says so.
func familyNames(table []byte) (family, legacy string) {
	if len(table) < 6 {
		return "", ""
	}
	count := int(binary.BigEndian.Uint16(table[2:]))
	strs := int(binary.BigEndian.Uint16(table[4:]))
	const nameFamily, nameTypographic = 1, 16
	var names [nameTypographic + 1]struct {
		s    string
		rank int
	}
	for i := range count {
		at := 6 + 12*i
		if at+12 > len(table) {
			break
		}
		rec := table[at:]
		platform, encoding := binary.BigEndian.Uint16(rec), binary.BigEndian.Uint16(rec[2:])
		language, id := binary.BigEndian.Uint16(rec[4:]), binary.BigEndian.Uint16(rec[6:])
		length, offset := int(binary.BigEndian.Uint16(rec[8:])), int(binary.BigEndian.Uint16(rec[10:]))
		if id != nameFamily && id != nameTypographic || strs+offset+length > len(table) {
			continue
		}
		raw := table[strs+offset : strs+offset+length]
		const unicode, mac, windows, english = 0, 1, 3, 0x409
		var s string
		rank := 0
		switch {
		case platform == windows && language == english:
			s, rank = utf16BE(raw), 4
		case platform == unicode:
			s, rank = utf16BE(raw), 3
		case platform == mac && encoding == 0 && language == 0:
			s, rank = ascii(raw), 2
		case platform == windows:
			s, rank = utf16BE(raw), 1
		}
		if s = strings.TrimSpace(s); s != "" && rank > names[id].rank {
			names[id].s, names[id].rank = s, rank
		}
	}
	legacy = names[nameFamily].s
	if family = names[nameTypographic].s; family == "" {
		family = legacy
	}
	return family, legacy
}

func utf16BE(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.BigEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

// ascii is b as text when it is ASCII, which Mac Roman shares.
func ascii(b []byte) string {
	for _, c := range b {
		if c >= 0x80 || c < 0x20 {
			return ""
		}
	}
	return string(b)
}

// read is the face's font as a file of its own: the file, or the tables
// of its font in a collection, put together as one font. without leaves
// the tables of those tags out.
func (f *installedFace) read(without ...uint32) ([]byte, error) {
	if !f.collection && len(without) == 0 {
		return os.ReadFile(f.path)
	}
	file, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	version, all, err := tableDirectory(file, f.offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", f.path, err)
	}
	var tables []tableRecord
	size := uint32(0)
	for _, t := range all {
		if !contains(without, t.tag) {
			tables = append(tables, t)
			size += (t.length + 3) &^ 3
		}
	}
	head := uint32(12 + 16*len(tables))
	out := make([]byte, head+size)
	binary.BigEndian.PutUint32(out, version)
	binary.BigEndian.PutUint16(out[4:], uint16(len(tables)))
	at := head
	for i, t := range tables {
		if _, err := file.ReadAt(out[at:at+t.length], int64(t.offset)); err != nil {
			return nil, fmt.Errorf("%s: %w", f.path, err)
		}
		rec := out[12+16*i:]
		binary.BigEndian.PutUint32(rec, t.tag)
		binary.BigEndian.PutUint32(rec[8:], at)
		binary.BigEndian.PutUint32(rec[12:], t.length)
		at += (t.length + 3) &^ 3
	}
	return out, nil
}

func contains(tags []uint32, tag uint32) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Scanning is an index on its way, scanned off the frame.
type Scanning struct {
	done chan struct{}
	idx  *Installed
}

// ScanDirs scans dirs on a goroutine of its own.
func ScanDirs(dirs ...string) *Scanning {
	s := &Scanning{done: make(chan struct{})}
	go func() {
		defer close(s.done)
		s.idx = Scan(dirs...)
	}()
	return s
}

//nolint:gochecknoglobals // the machine's fonts, scanned once for every Document
var (
	systemOnce sync.Once
	system     *Scanning
)

// ScanSystem is the scan of SystemDirs, begun the first time it is asked
// for and shared from then on.
func ScanSystem() *Scanning {
	systemOnce.Do(func() { system = ScanDirs(SystemDirs()...) })
	return system
}

// Wait is the index, once the scan is done.
func (s *Scanning) Wait() *Installed {
	<-s.done
	return s.idx
}

// Ready is the index when the scan is done, without waiting for it.
func (s *Scanning) Ready() (*Installed, bool) {
	select {
	case <-s.done:
		return s.idx, true
	default:
		return nil, false
	}
}
