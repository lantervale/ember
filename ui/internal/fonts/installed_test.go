package fonts //nolint:testpackage // tests unexported internals

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
)

// collection packs fonts into one .ttc file.
func collection(fonts ...[]byte) []byte {
	out := make([]byte, 12+4*len(fonts))
	binary.BigEndian.PutUint32(out, tagTTCF)
	binary.BigEndian.PutUint32(out[4:], 0x00010000)
	binary.BigEndian.PutUint32(out[8:], uint32(len(fonts)))
	for i, f := range fonts {
		base := uint32(len(out))
		binary.BigEndian.PutUint32(out[12+4*i:], base)
		out = append(out, f...)
		font := out[base:]
		for t := range int(binary.BigEndian.Uint16(font[4:])) {
			rec := font[12+16*t:]
			binary.BigEndian.PutUint32(rec[8:], binary.BigEndian.Uint32(rec[8:])+base)
		}
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	return out
}

// withoutCmap is font with its cmap table zeroed: named as before, but
// no font to draw with.
func withoutCmap(t *testing.T, font []byte) []byte {
	t.Helper()
	out := append([]byte(nil), font...)
	for i := range int(binary.BigEndian.Uint16(out[4:])) {
		rec := out[12+16*i:]
		if string(rec[:4]) == "cmap" {
			off, n := binary.BigEndian.Uint32(rec[8:]), binary.BigEndian.Uint32(rec[12:])
			clear(out[off : off+n])
			return out
		}
	}
	t.Fatal("no cmap table")
	return nil
}

// fontDir is a folder of font files as a machine keeps them: the Go
// family a file a face, Go Medium, Go Mono as a collection in a folder
// below, and files that are no fonts.
func fontDir(t *testing.T, more map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string][]byte{
		"Go-Regular.ttf":     goregular.TTF,
		"Go-Bold.TTF":        gobold.TTF,
		"Go-Italic.ttf":      goitalic.TTF,
		"Go-Bold-Italic.ttf": gobolditalic.TTF,
		"Go-Medium.ttf":      gomedium.TTF,
		"mono/GoMono.ttc":    collection(gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF),
		"junk.ttf":           []byte("this is no font, whatever its name says"),
		"empty.otf":          nil,
		"README.txt":         []byte("Go fonts"),
	}
	for name, data := range more {
		files[name] = data
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}
	return dir
}

// installedOnly is a set with no face of its own, drawing with dir's.
func installedOnly(dir string) *Set {
	s := &Set{}
	s.SetInstalled(Scan(dir))
	return s
}

func TestAScanFindsFamiliesWeightsAndStylesByTheNameTable(t *testing.T) {
	idx := Scan(fontDir(t, nil), filepath.Join(t.TempDir(), "missing"), "")
	assert.Equal(t, 3, idx.Families())
	for _, c := range []struct {
		family, name string
		faces        int
	}{{"go", "Go", 4}, {"GO MEDIUM", "Go Medium", 1}, {"Go Mono", "Go Mono", 4}} {
		f := idx.family(c.family)
		require.NotNil(t, f, c.family)
		assert.Equal(t, c.name, f.name)
		assert.Len(t, f.faces, c.faces, c.family)
	}
	assert.Nil(t, idx.family("Georgia"))
}

func TestAnInstalledFamilyIsFoundInAnyCaseByWeightAndStyle(t *testing.T) {
	s := installedOnly(fontDir(t, nil))
	for _, c := range []struct {
		family string
		weight int
		italic bool
		ttf    []byte
	}{
		{"go", 400, false, goregular.TTF},
		{"GO", 700, false, gobold.TTF},
		{"Go", 400, true, goitalic.TTF},
		{"gO", 700, true, gobolditalic.TTF},
		{"go", 600, false, gobold.TTF},
	} {
		f := s.Face(style.Font{Family: []string{c.family}, Weight: c.weight, Italic: c.italic})
		require.NotNil(t, f)
		assert.Equal(t, "Go", f.Family)
		assert.Equal(t, c.italic, f.Italic)
		assert.Equal(t, c.ttf, f.TTF, "%s %d %v", c.family, c.weight, c.italic)
	}
}

func TestAFaceOfACollectionMeasuresAsItsFontAlone(t *testing.T) {
	s := installedOnly(fontDir(t, nil))
	alone := &Set{}
	require.NoError(t, alone.Add("Go Mono", 700, true, gomonobolditalic.TTF))
	require.NoError(t, alone.Add("Go Mono", 400, false, gomono.TTF))
	const text = "A face of a collection, AVA."
	for _, f := range []style.Font{
		{Family: []string{"go mono"}, Size: 20, Weight: 700, Italic: true},
		{Family: []string{"go mono"}, Size: 20, Weight: 400},
	} {
		want := f
		want.Family = []string{"Go Mono"}
		assert.Equal(t, f.Weight >= 600, s.Face(f).Weight >= 600, "Go Mono's bold is a 600")
		assert.Equal(t, f.Italic, s.Face(f).Italic)
		assert.Equal(t, alone.Advance(want, text), s.Advance(f, text))
		assert.Equal(t, alone.Metrics(want), s.Metrics(f))
	}
	assert.NotEqual(t, gomono.TTF, gomonobolditalic.TTF)
}

func TestAFamilyTheMachineLacksFallsToTheNextName(t *testing.T) {
	s := Builtin()
	s.SetInstalled(Scan(fontDir(t, nil)))
	assert.Equal(t, "Go Medium", s.Face(style.Font{Family: []string{"Nope", "go medium", "serif"}, Weight: 400}).Family)
	assert.Equal(t, "Go", s.Face(style.Font{Family: []string{"Nope", "sans-serif"}, Weight: 400}).Family)
	assert.Equal(t, goMono, s.Face(style.Font{Family: []string{"Nope", "monospace"}, Weight: 400}).Family)
	assert.Empty(t, s.Warnings())
}

func TestTheSetsOwnFamiliesWinOverInstalledOnes(t *testing.T) {
	s := Builtin()
	require.NoError(t, s.Add("Go Medium", 400, false, gosmallcaps.TTF))
	s.SetInstalled(Scan(fontDir(t, nil)))
	assert.Equal(t, gosmallcaps.TTF, s.Face(style.Font{Family: []string{"Go Medium"}, Weight: 400}).TTF)
	assert.Equal(t, gosmallcaps.TTF, s.Face(style.Font{Family: []string{"go medium"}, Weight: 400}).TTF,
		"another case names the set's family, not the machine's")
	assert.Len(t, s.Faces(), len(Builtin().Faces())+1)
}

func TestNothingIsLookedUpWithoutAnIndex(t *testing.T) {
	s := Builtin()
	before := len(s.Faces())
	assert.Equal(t, "Go", s.Face(style.Font{Family: []string{"Go Medium"}, Weight: 400}).Family)
	s.Await()
	assert.False(t, s.Land())
	assert.Len(t, s.Faces(), before)
}

func TestAFontFileIsReadWhenItsFaceIsFirstDrawnWith(t *testing.T) {
	s := installedOnly(fontDir(t, nil))
	assert.Empty(t, s.Faces())
	regular := style.Font{Family: []string{"Go"}, Size: 16, Weight: 400}
	assert.Positive(t, s.Advance(regular, "Hello"))
	require.Len(t, s.Faces(), 1)
	assert.Equal(t, goregular.TTF, s.Faces()[0].TTF)
	s.Face(style.Font{Family: []string{"Go"}, Weight: 700})
	assert.Len(t, s.Faces(), 2)
}

func TestAFontThatCannotBeDrawnWithIsAWarningAndItsFamilyFallsThrough(t *testing.T) {
	dir := fontDir(t, map[string][]byte{"Go-Medium.ttf": withoutCmap(t, gomedium.TTF)})
	s := Builtin()
	s.SetInstalled(Scan(dir))
	f := style.Font{Family: []string{"Go Medium", "monospace"}, Size: 16, Weight: 400}
	assert.Equal(t, goMono, s.Face(f).Family)
	warns := s.Warnings()
	require.Len(t, warns, 1)
	assert.Contains(t, warns[0].Error(), "Go Medium")
	assert.Contains(t, warns[0].Error(), "Go-Medium.ttf")
	assert.Equal(t, goMono, s.Face(f).Family)
	assert.Empty(t, s.Warnings(), "said once")
}

func TestOfTwoFontsOfOneWeightTheNormalWidthIsKept(t *testing.T) {
	idx := &Installed{families: map[string]*installedFamily{}}
	condensed := &installedFace{path: "condensed", weight: 700, width: 3}
	normal := &installedFace{path: "normal", weight: 700, width: normalWidth}
	idx.add("Sans", condensed)
	idx.add("Sans", normal)
	idx.add("sans", &installedFace{path: "expanded", weight: 700, width: 7})
	idx.add("Sans", &installedFace{path: "italic", weight: 700, width: 3, italic: true})
	f := idx.family("SANS")
	require.Len(t, f.faces, 2)
	assert.Same(t, normal, f.faces[0])
}

func TestAnInstalledFaceIsFoundWithoutAllocating(t *testing.T) {
	s := Builtin()
	s.SetInstalled(Scan(fontDir(t, nil)))
	f := style.Font{Family: []string{"Nope", "go medium", "sans-serif"}, Size: 16, Weight: 400}
	s.Face(f)
	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Face(f) }))
}

func TestAScanOffTheFrameLandsOnceItIsDone(t *testing.T) {
	s := Builtin()
	s.Scan(ScanDirs(fontDir(t, nil)))
	f := style.Font{Family: []string{"Go Medium"}, Weight: 400}
	if !s.Land() {
		assert.Equal(t, "Go", s.Face(f).Family)
		s.Await()
		assert.True(t, s.Land())
	}
	assert.Equal(t, "Go Medium", s.Face(f).Family)
	assert.False(t, s.Land(), "taken once")
}

func TestNameTablesPreferTheTypographicFamilyInEnglish(t *testing.T) {
	type rec struct {
		platform, encoding, language, id uint16
		text                             string
	}
	table := func(recs ...rec) []byte {
		head := make([]byte, 6+12*len(recs))
		binary.BigEndian.PutUint16(head[2:], uint16(len(recs)))
		binary.BigEndian.PutUint16(head[4:], uint16(len(head)))
		var strs []byte
		for i, r := range recs {
			raw := []byte(r.text)
			if r.platform != 1 {
				raw = nil
				for _, c := range r.text {
					raw = binary.BigEndian.AppendUint16(raw, uint16(c))
				}
			}
			at := head[6+12*i:]
			for j, v := range []uint16{r.platform, r.encoding, r.language, r.id, uint16(len(raw)), uint16(len(strs))} {
				binary.BigEndian.PutUint16(at[2*j:], v)
			}
			strs = append(strs, raw...)
		}
		return append(head, strs...)
	}
	family, legacy := familyNames(table(
		rec{3, 1, 0x407, 1, "Schrift Halbfett"},
		rec{1, 0, 0, 1, "Face Mac"},
		rec{3, 1, 0x409, 1, "Face Semibold"},
		rec{3, 1, 0x409, 2, "Regular"},
		rec{3, 1, 0x409, 16, "Face"},
	))
	assert.Equal(t, "Face", family)
	assert.Equal(t, "Face Semibold", legacy)
	family, legacy = familyNames(table(rec{1, 0, 0, 1, "Old Mac"}))
	assert.Equal(t, "Old Mac", family)
	assert.Equal(t, "Old Mac", legacy)
	family, _ = familyNames([]byte{0, 0, 0, 9})
	assert.Empty(t, family)
}
