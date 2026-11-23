package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/internal/woff"
	"golang.org/x/image/font/sfnt"
)

// FontFetcher gets the font file an @font-face's src names, as it is
// written there or, for a sheet that says where it came from (From),
// in full, when no file system of the Document holds it. It is called
// off the frame, a few fonts at a time, so it may take as long as it
// needs and must be safe to call from several goroutines. sheet is the
// address of the sheet whose rule names the font, "" when it says
// none: what a request for it gives as its referrer.
type FontFetcher func(url, sheet string) ([]byte, error)

// FetchFonts has fetch get the fonts of the @font-face rules whose
// files the Document's file systems don't hold: those of a sheet given
// as text (CSS), and of a Stylesheet whose file system lacks them. A
// font is TrueType, OpenType, WOFF or WOFF2. It is asked for once, when
// text first needs a character its face draws: a face no text uses, or
// one whose unicode-range has none of the text's characters, is never
// fetched. Until it lands the text shows in the next family of its
// list as the rule's font-display has it, timed on the World's real
// time as Chrome times it: invisible in its room for 3 s (auto, block)
// or 100 ms (fallback, optional), then in that family; a font landing
// after 3 s (fallback) or 100 ms (optional) is not used, the text staying
// there. A character no family draws shows as nothing. Once fonts land
// the Document lays out and paints again, once however many landed since
// the last frame.
// The entries of src are tried in order, past those ui does not read
// (an .eot, an SVG font, colour glyphs): a local() only with
// SystemFonts. A fetch that fails, or bytes that are no font, are a
// warning naming the address, and the text stays as it is. ui/web has
// a fetcher over HTTP.
func FetchFonts(fetch FontFetcher) DocumentOption {
	return func(d *Document) { d.pageFonts.fetch = fetch }
}

// fontFetchesAtOnce is how many fonts a Document fetches at a time.
const fontFetchesAtOnce = 6

// pageFonts are a Document's fonts that come after it is made: the
// files of its @font-face rules on their way, and the faces twod draws
// the landed ones with.
type pageFonts struct {
	fetch FontFetcher
	// files are the fonts by their src list, so the faces that share
	// one share its file; sources are the entries to try for each.
	files   map[string]*fonts.File
	sources map[*fonts.File][]css.FontSource
	// asked are the weight and style each file's first face is of, for
	// its local() entries.
	asked map[*fonts.File]css.FontFace
	// rules are the faces of the @font-face rules read so far, by what
	// each says, nil for one that gives none.
	rules map[string]*fonts.Face

	sync.WaitGroup
	mu     sync.Mutex
	landed []landedFont
	slots  chan struct{}

	// drawn are the landed faces as twod draws them, chained each
	// font's faces in order, made for the set's generation gen.
	drawn   map[*fonts.Face]*draw.Face
	chained map[*fonts.Chain]*twod.Face
	gen     uint64
}

// landedFont is a font back from its fetch, read off the frame, or why
// there is none.
type landedFont struct {
	file  *fonts.File
	ttf   []byte
	font  *sfnt.Font
	warns []error
}

// addFontFaces gives the Document the @font-face rules of a sheet, and
// their faces, nil for a rule that gives none: each face's file is read
// from fsys, the sheet's file system, when it holds it; otherwise, and
// for a sheet with none (fsys nil), it is fetched when text needs it. A
// rule read before is the same face, its file read or fetched once.
func (d *Document) addFontFaces(faces []css.FontFace, fsys fs.FS) []*fonts.Face {
	p := &d.pageFonts
	out := make([]*fonts.Face, len(faces))
	for i, f := range faces {
		key := fmt.Sprintf("%v", f)
		if c, ok := p.rules[key]; ok {
			out[i] = d.fonts.AddFace(faceDescriptors(f), c.File())
			continue
		}
		out[i] = d.addFontFace(f, fsys)
		if p.rules == nil {
			p.rules = map[string]*fonts.Face{}
		}
		if out[i] != nil {
			p.rules[key] = out[i]
		}
	}
	return out
}

func (d *Document) addFontFace(f css.FontFace, fsys fs.FS) *fonts.Face {
	if fsys != nil {
		if c, held := d.readFontFace(f, fsys); held {
			return c
		}
	}
	sources := f.Readable()
	if !d.fetchable(sources) {
		if fsys != nil {
			d.warn(fmt.Errorf("css: @font-face %q: no file %s", f.Family, f.Src))
		} else {
			d.warn(fmt.Errorf(
				"css: @font-face %q in CSS has no file to read; load it with Stylesheet or fetch it with FetchFonts",
				f.Family))
		}
		return nil
	}
	return d.fonts.AddFace(faceDescriptors(f), d.pageFonts.file(f, sources))
}

// faceDescriptors are what f says of its face, as the font set reads it.
func faceDescriptors(f css.FontFace) fonts.Descriptors {
	look := fonts.Look{SizeAdjust: f.SizeAdjust}
	if look.SizeAdjust == 1 {
		look.SizeAdjust = 0
	}
	for _, o := range []struct {
		at *fonts.Override
		of float32
	}{{&look.Ascent, f.Ascent}, {&look.Descent, f.Descent}, {&look.LineGap, f.LineGap}} {
		if o.of >= 0 {
			*o.at = fonts.Override{Set: true, Of: o.of}
		}
	}
	feats := make([]string, len(f.Features))
	for i, ft := range f.Features {
		feats[i] = fmt.Sprintf("%s=%d", ft.Tag, ft.Value)
	}
	look.Features = strings.Join(feats, ",")
	return fonts.Descriptors{Family: f.Family, Weight: f.Weight, WeightMax: f.WeightMax, Italic: f.Italic,
		Stretch: f.Stretch, StretchMax: f.StretchMax, AutoWeight: f.AutoWeight, AutoStretch: f.AutoStretch,
		Ranges: fontRanges(f.Ranges), Look: look, Display: displays[f.Display]}
}

// displays are font-display's words as the font set reads them.
//
//nolint:gochecknoglobals // a lookup table
var displays = map[string]fonts.Display{"auto": fonts.DisplayAuto, "block": fonts.DisplayBlock,
	"swap": fonts.DisplaySwap, "fallback": fonts.DisplayFallback, "optional": fonts.DisplayOptional}

// readFontFace adds f from the first of its files fsys holds, and
// reports whether it held one; a file that is no font is a warning, and
// gives no face.
func (d *Document) readFontFace(f css.FontFace, fsys fs.FS) (*fonts.Face, bool) {
	for _, s := range f.Readable() {
		if s.URL == "" {
			continue
		}
		data, err := fs.ReadFile(fsys, s.URL)
		if err != nil {
			continue
		}
		var c *fonts.Face
		ttf, err := woff.Sfnt(data)
		if err == nil {
			var parsed *sfnt.Font
			if parsed, err = fonts.Parse(ttf); err == nil {
				file := &fonts.File{}
				c = d.fonts.AddFace(faceDescriptors(f), file)
				d.fonts.Loaded(file, ttf, parsed)
			}
		}
		if err != nil {
			d.warn(fmt.Errorf("ui: font %s: %w", s.URL, err))
		}
		return c, true
	}
	return nil, false
}

// takeFontFaces has the Document's text drawn with the faces of the
// @font-face rules of its sheets as they now stand, in their order: a
// rule inserted alone gives its face, and a sheet removed or disabled
// takes its faces with it.
func (d *Document) takeFontFaces() {
	var faces []*fonts.Face
	for _, s := range d.authored {
		if s.disabled {
			continue
		}
		for _, c := range d.addFontFaces(s.sheet().FontFaces(), s.fsys) {
			if c != nil && !slices.Contains(faces, c) {
				faces = append(faces, c)
			}
		}
	}
	if d.fonts.SetWeb(faces) {
		d.relayout = true
		d.remeasureFields()
	}
}

// fetchable reports whether one of sources may give a font: a data
// URL, an address with a fetcher, an installed font with SystemFonts.
func (d *Document) fetchable(sources []css.FontSource) bool {
	for _, s := range sources {
		_, inline := dataURL(s.URL)
		if inline || (s.URL != "" && d.pageFonts.fetch != nil) || (s.Local != "" && d.systemFonts) {
			return true
		}
	}
	return false
}

func fontRanges(ranges []css.UnicodeRange) []fonts.Range {
	if len(ranges) == 0 {
		return nil
	}
	out := make([]fonts.Range, len(ranges))
	for i, r := range ranges {
		out[i] = fonts.Range{Lo: r.Lo, Hi: r.Hi}
	}
	return out
}

// file is the file of the faces whose src is sources, f the first of
// them.
func (p *pageFonts) file(f css.FontFace, sources []css.FontSource) *fonts.File {
	var key strings.Builder
	for _, s := range sources {
		key.WriteString(s.URL + "\x00" + s.Local + "\x00")
	}
	if file, ok := p.files[key.String()]; ok {
		return file
	}
	if p.files == nil {
		p.files = map[string]*fonts.File{}
		p.sources, p.asked = map[*fonts.File][]css.FontSource{}, map[*fonts.File]css.FontFace{}
	}
	file := &fonts.File{}
	p.files[key.String()], p.sources[file], p.asked[file] = file, sources, f
	return file
}

// fontSource is one entry of a src list as its fetch reads it: get
// gives its bytes.
type fontSource struct {
	name string
	get  func() ([]byte, error)
}

// fetchFonts starts the fetch of each file text has needed since it
// was last called.
func (d *Document) fetchFonts() {
	p := &d.pageFonts
	for _, file := range d.fonts.Wanted() {
		if p.slots == nil {
			p.slots = make(chan struct{}, fontFetchesAtOnce)
		}
		sources := d.fontSources(file)
		p.Add(1)
		go func() {
			defer p.Done()
			p.slots <- struct{}{}
			defer func() { <-p.slots }()
			got := fetchedFont(file, sources)
			p.mu.Lock()
			defer p.mu.Unlock()
			p.landed = append(p.landed, got)
		}()
	}
}

// fontSources are file's entries as its fetch tries them: a local()
// the machine has not, or that is not looked for, is left out.
func (d *Document) fontSources(file *fonts.File) []fontSource {
	p := &d.pageFonts
	var out []fontSource
	for _, s := range p.sources[file] {
		switch data, inline := dataURL(s.URL); {
		case s.Local != "":
			f := p.asked[file]
			if read := d.fonts.Local(s.Local, f.Weight, f.Italic); read != nil && d.systemFonts {
				out = append(out, fontSource{"local(" + s.Local + ")", read})
			}
		case inline:
			out = append(out, fontSource{"data:…", func() ([]byte, error) { return data, nil }})
		case p.fetch != nil:
			fetch, url, sheet := p.fetch, s.URL, p.asked[file].From
			out = append(out, fontSource{url, func() ([]byte, error) { return fetch(url, sheet) }})
		}
	}
	return out
}

var errNoFontSource = errors.New("none of its sources can be read")

// fetchedFont is file's font from the first of sources that gives one,
// read and checked off the frame; each that fails is a warning.
func fetchedFont(file *fonts.File, sources []fontSource) landedFont {
	got := landedFont{file: file}
	for _, s := range sources {
		ttf, parsed, err := readFont(s)
		if err == nil {
			got.ttf, got.font = ttf, parsed
			return got
		}
		got.warns = append(got.warns, fmt.Errorf("ui: font %s: %w", s.name, err))
	}
	if len(sources) == 0 {
		got.warns = append(got.warns, fmt.Errorf("ui: font: %w", errNoFontSource))
	}
	return got
}

// readFont is the font s gives: the sfnt file its bytes are or pack,
// parsed. The bytes are a stranger's, so a panic of a reader is an
// error too.
//
//nolint:nonamedreturns // the deferred recover sets them
func readFont(s fontSource) (ttf []byte, parsed *sfnt.Font, err error) {
	defer func() {
		if r := recover(); r != nil {
			ttf, parsed, err = nil, nil, fmt.Errorf("the font can't be read: %v", r)
		}
	}()
	data, err := s.get()
	if err != nil {
		return nil, nil, err
	}
	if ttf, err = woff.Sfnt(data); err != nil {
		return nil, nil, err
	}
	if parsed, err = fonts.Parse(ttf); err != nil {
		return nil, nil, err
	}
	return ttf, parsed, nil
}

// landPageFonts takes the fonts fetched since the last layout, which
// lays the Document out again with them, once for them all, and starts
// the fetches of those the last layout needed.
func (d *Document) landPageFonts() {
	p := &d.pageFonts
	p.mu.Lock()
	landed := p.landed
	p.landed = nil
	p.mu.Unlock()
	for _, got := range landed {
		for _, w := range got.warns {
			d.warn(w)
		}
		if got.font == nil {
			d.fonts.Failed(got.file)
		} else {
			d.fonts.Loaded(got.file, got.ttf, got.font)
		}
	}
	ticked := d.clock >= 0 && d.fonts.Tick(d.clock)
	if len(landed) > 0 || ticked {
		d.relayout = true
		d.remeasureFields()
	}
	d.fetchFonts()
}

// remeasureFields drops what the text fields measured of their text:
// it was measured with other faces.
func (d *Document) remeasureFields() {
	for _, m := range d.machines {
		if t := textOf(m); t != nil {
			t.Layout = nil
		}
	}
}

// FontsLoading is how many of the Document's fonts are on their way:
// asked for by its text and neither landed nor failed.
func (d *Document) FontsLoading() int { return d.fonts.Loading() }

// chainFace is the face twod draws ch with: its landed faces in order,
// each drawing its own characters, a character whose face is still on
// its way as nothing.
func (d *Document) chainFace(ch *fonts.Chain) *twod.Face {
	p := &d.pageFonts
	if p.gen != d.fonts.Gen() {
		p.chained, p.gen = nil, d.fonts.Gen()
	}
	if tf, ok := p.chained[ch]; ok {
		return tf
	}
	var faces []*twod.Face
	for i, c := range ch.Faces() {
		df := p.drawFace(c)
		if df == nil {
			continue
		}
		if look := drawLook(c, ch); look != (draw.Look{}) {
			df = df.Styled(look)
		}
		if i == 0 {
			df = df.Blank(ch.Loading()).Hidden(ch.Hidden(d.fonts))
		}
		if tf, ok := draw.Wrap(df).(*twod.Face); ok {
			faces = append(faces, tf)
		}
	}
	tf := twod.Regular()
	if len(faces) > 0 {
		var opts []twod.FaceOption
		if len(faces) > 1 {
			opts = append(opts, twod.Fallback(faces[1:]...))
		}
		if d.emoji != nil {
			opts = append(opts, twod.Emoji(d.emoji))
		}
		tf = faces[0]
		if len(opts) > 0 {
			tf = tf.With(opts...)
		}
	}
	if p.chained == nil {
		p.chained = map[*fonts.Chain]*twod.Face{}
	}
	p.chained[ch] = tf
	return tf
}

// drawLook is how c is drawn in ch: scaled by its size-adjust, shaped
// with the features its @font-face sets, made bold or slanted where the
// family has no such face, and at the chain's instance of a variable font.
func drawLook(c *fonts.Face, ch *fonts.Chain) draw.Look {
	if !c.Web() {
		return draw.Look{}
	}
	bold, italic := c.Synthetic(ch.Font())
	look := draw.Look{Scale: c.Look.SizeAdjust, Bold: bold, Oblique: italic, Features: c.Look.Features}
	if at := ch.Instance(c); at != nil {
		look.Variation = &draw.Variation{Font: c.Variable(), Coords: at}
	}
	return look
}

// drawFace is c as twod draws it, parsed once: TrueType or CFF
// outlines, only the characters of its unicode-range.
func (p *pageFonts) drawFace(c *fonts.Face) *draw.Face {
	if df, ok := p.drawn[c]; ok {
		return df
	}
	df, err := draw.ParseFace(c.Family, c.TTF)
	if err != nil {
		df = nil
	} else if len(c.Ranges) > 0 {
		df = df.Only(c.Ranges)
	}
	if p.drawn == nil {
		p.drawn = map[*fonts.Face]*draw.Face{}
	}
	p.drawn[c] = df
	return df
}

// FontsReady reports whether text set in the first of families the
// Document draws, at weight and in italic or not, waits for no font:
// the faces that draw its characters have landed or failed, or are
// the Document's own. It fetches nothing.
func (d *Document) FontsReady(families []string, weight int, italic bool, text string) bool {
	return d.fonts.Ready(style.Font{Family: families, Size: 16, Weight: float32(weight), Italic: italic}, text, false)
}

// LoadFonts asks for the fonts text set in families needs, as laying
// it out would, and reports whether none is on its way: FontsLoading
// counts those that are, and the Document lays out again as they land.
func (d *Document) LoadFonts(families []string, weight int, italic bool, text string) bool {
	ready := d.fonts.Ready(style.Font{Family: families, Size: 16, Weight: float32(weight), Italic: italic}, text, true)
	d.fetchFonts()
	return ready
}
