package draw

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/render"
	"github.com/rivo/uniseg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/SourceCodePro-Regular.ttf
var sourceCodePro []byte

// MonoTTF is the embedded SourceCodePro font file, for other text
// renderers such as the ImGui layer.
func MonoTTF() []byte { return sourceCodePro }

// Emoji is a set of pictures for grapheme clusters, drawn inline instead
// of glyphs.
type Emoji interface {
	Picture(cluster string) (*render.Texture, bool)
}

// Face is a parsed font file, with the faces that draw what it lacks and
// the pictures that replace emoji. Faces are rasterized on demand per
// size.
type Face struct {
	name string
	font *opentype.Font
	// layout is the file's OpenType layout tables, which shape its text.
	layout    *shape.Font
	fallbacks []*Face
	emoji     Emoji
	// only are the characters the face draws of those it has (Only), all
	// without any; blank those drawn as nothing when no face has them
	// (Blank); hidden those drawn invisible in their room (Hidden).
	only, blank, hidden []Range
	// look is how the face is drawn other than as its file has it.
	look  Look
	sizes map[float32]*Font
	// pixelSizes are the face's pixelated fonts (PixelSize), apart, so
	// smoothed text never shares their atlas.
	pixelSizes map[float32]*Font
	// exactSizes are its fonts without stem darkening (ExactSize), apart
	// as well.
	exactSizes map[float32]*Font
	// strokes are its fonts drawing each glyph's outline stroked.
	strokes map[strokeKey]*Font
}

// strokeKey is a stroked font: its size, its stroke's width and its
// smoothing.
type strokeKey struct {
	px, width float32
	how       smoothing
}

// The engine's faces.
//
//nolint:gochecknoglobals // one value each, parsed once, that every text shares
var (
	Regular   = mustFace("go-regular", goregular.TTF)
	Bold      = mustFace("go-bold", gobold.TTF)
	Italic    = mustFace("go-italic", goitalic.TTF)
	SmallCaps = mustFace("go-smallcaps", gosmallcaps.TTF)
	Mono      = mustFace("source-code-pro", sourceCodePro)
)

// ParseFace parses ttf, the bytes of a TrueType or OpenType font file
// called name in errors.
func ParseFace(name string, ttf []byte) (*Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("twod: parse font %s: %w", name, err)
	}
	return &Face{name: name, font: f, layout: shape.Parse(f, ttf), sizes: map[float32]*Font{}}, nil
}

func mustFace(name string, ttf []byte) *Face {
	f, err := ParseFace(name, ttf)
	if err != nil {
		panic(err)
	}
	return f
}

// With is f drawing what it lacks with fallbacks, in order, and the
// clusters emoji has as pictures.
func (f *Face) With(fallbacks []*Face, emoji Emoji) *Face {
	return &Face{name: f.name, font: f.font, layout: f.layout,
		only: f.only, blank: f.blank, hidden: f.hidden, look: f.look,
		fallbacks: fallbacks, emoji: emoji, sizes: map[float32]*Font{}}
}

// Look is how a face is drawn other than as its file has it, as an
// @font-face and the font asked for have it: its glyphs and advances
// scaled by Scale (0 for 1), made bold or slanted where its family has
// no such face, shaped with Features (HarfBuzz's, comma separated)
// before the text's own, and a variable font's at the instance
// Variation places it.
type Look struct {
	Scale         float32
	Bold, Oblique bool
	Features      string
	Variation     *Variation
}

// Variation is an instance of a variable font: the face's file read by
// varfont, and the place on its axes.
type Variation struct {
	Font   *varfont.Font
	Coords varfont.Coords
}

// varied is the instance l draws, nil for the font's default.
func (l Look) varied() *Variation {
	if l.Variation == nil || l.Variation.Coords == nil {
		return nil
	}
	return l.Variation
}

func (l Look) scale() float32 {
	if l.Scale == 0 {
		return 1
	}
	return l.Scale
}

// Styled is f drawn with look.
func (f *Face) Styled(look Look) *Face {
	return &Face{name: f.name, font: f.font, layout: f.layout,
		only: f.only, blank: f.blank, hidden: f.hidden, look: look,
		fallbacks: f.fallbacks, emoji: f.emoji, sizes: map[float32]*Font{}}
}

// Range is the characters from Lo to Hi.
type Range struct{ Lo, Hi rune }

// Covers reports whether r is in ranges, or ranges is empty: a face
// with none draws every character it has.
func Covers(ranges []Range, r rune) bool {
	for _, u := range ranges {
		if r >= u.Lo && r <= u.Hi {
			return true
		}
	}
	return len(ranges) == 0
}

// Only is f drawing only the characters in ranges, as an @font-face's
// unicode-range has it: another is left to the faces after it.
func (f *Face) Only(ranges []Range) *Face {
	return &Face{name: f.name, font: f.font, layout: f.layout,
		only: ranges, blank: f.blank, hidden: f.hidden, look: f.look,
		fallbacks: f.fallbacks, emoji: f.emoji, sizes: map[float32]*Font{}}
}

// Blank is f drawing nothing, in no room, for a character in ranges
// that neither it nor its fallbacks have, where another shows as '?':
// what a face still on its way may draw.
func (f *Face) Blank(ranges []Range) *Face {
	return &Face{name: f.name, font: f.font, layout: f.layout,
		only: f.only, blank: ranges, hidden: f.hidden, look: f.look,
		fallbacks: f.fallbacks, emoji: f.emoji, sizes: map[float32]*Font{}}
}

// Hidden is f drawing the characters in ranges invisible, in the room
// they take: those of a face in its font-display block period.
func (f *Face) Hidden(ranges []Range) *Face {
	return &Face{name: f.name, font: f.font, layout: f.layout,
		only: f.only, blank: f.blank, hidden: ranges, look: f.look,
		fallbacks: f.fallbacks, emoji: f.emoji, sizes: map[float32]*Font{}}
}

// Wrap makes the twod.Face of f, for the engine packages that build
// faces twod's NewFace refuses; twod sets it.
var Wrap func(f *Face) any //nolint:gochecknoglobals // a hook twod sets

// Kept is where an engine package keeps what it made of face, a
// *twod.Face, so it goes when the face does; twod sets it.
var Kept func(face any) *any //nolint:gochecknoglobals // a hook twod sets

// Draws reports whether the face draws text as written: a picture for
// each cluster of several characters (a flag, a ZWJ family, a keycap),
// else a glyph in the face or a fallback for a single character.
func (f *Face) Draws(text string) bool {
	var buf sfnt.Buffer
	ok := true
	eachCluster(text, func(c string, first rune) {
		if !ok || first == '\n' {
			return
		}
		if f.emoji != nil && len(c) > 1 {
			if _, pic := f.emoji.Picture(c); pic {
				return
			}
		}
		if utf8.RuneCountInString(strings.ReplaceAll(c, "\ufe0f", "")) > 1 {
			ok = false
			return
		}
		for _, src := range append([]*Face{f}, f.fallbacks...) {
			if !Covers(src.only, first) {
				continue
			}
			if idx, err := src.font.GlyphIndex(&buf, first); err == nil && idx != 0 {
				return
			}
		}
		ok = false
	})
	return ok
}

// oversample rasterizes glyphs at twice the logical size so text stays
// crisp on high-density displays.
const oversample = 2

// MaxAtlasSide is the longest side of a glyph atlas: of a texture every
// WebGPU device takes. A font so large that sixteen of its glyphs don't
// fit in a row starts with an empty atlas of that width and rasterizes
// each glyph when it is first drawn; one whose atlas is full draws no new
// glyph, keeping its advance.
const MaxAtlasSide = 8192

// Size returns the font rasterized at px logical pixels.
func (f *Face) Size(px float32) *Font {
	px = float32(math.Round(float64(px)))
	if fnt, ok := f.sizes[px]; ok {
		return fnt
	}
	fnt, err := newFont(f, px, darkened, 0)
	if err != nil {
		panic(err)
	}
	f.sizes[px] = fnt
	return fnt
}

// ExactSize returns the font at px logical pixels rasterized at each
// glyph's exact coverage, without stem darkening: a square is its pixels
// and nothing beside them, as Chrome draws
// -webkit-font-smoothing: antialiased.
func (f *Face) ExactSize(px float32) *Font {
	px = float32(math.Round(float64(px)))
	if fnt, ok := f.exactSizes[px]; ok {
		return fnt
	}
	fnt, err := newFont(f, px, exact, 0)
	if err != nil {
		panic(err)
	}
	if f.exactSizes == nil {
		f.exactSizes = map[float32]*Font{}
	}
	f.exactSizes[px] = fnt
	return fnt
}

// PixelSize returns the font at px logical pixels rasterized without
// smoothing: each glyph pixel covered or not, at half its coverage, as
// Chrome draws -webkit-font-smoothing: none, for pixel fonts.
func (f *Face) PixelSize(px float32) *Font {
	px = float32(math.Round(float64(px)))
	if fnt, ok := f.pixelSizes[px]; ok {
		return fnt
	}
	fnt, err := newFont(f, px, unsmoothed, 0)
	if err != nil {
		panic(err)
	}
	if f.pixelSizes == nil {
		f.pixelSizes = map[float32]*Font{}
	}
	f.pixelSizes[px] = fnt
	return fnt
}

// stroked returns the font at px logical pixels, exact or unsmoothed, whose
// glyphs are their outlines stroked width pixels wide, half outside and
// half inside, with mitred joins: -webkit-text-stroke. Its advances are
// the plain font's.
func (f *Face) stroked(px, width float32, how smoothing) *Font {
	key := strokeKey{float32(math.Round(float64(px))), width, how}
	if fnt, ok := f.strokes[key]; ok {
		return fnt
	}
	fnt, err := newFont(f, key.px, how, width)
	if err != nil {
		panic(err)
	}
	if f.strokes == nil {
		f.strokes = map[strokeKey]*Font{}
	}
	f.strokes[key] = fnt
	return fnt
}

type glyph struct {
	advance float32
	offset  geom.Vec2
	size    geom.Vec2
	// rect is where the glyph is in the atlas, in pixels.
	rect image.Rectangle
	// face is which of the font's faces draws it and idx its index
	// there, for kerning; ready marks an ASCII slot filled.
	face  int
	idx   sfnt.GlyphIndex
	r     rune
	ready bool
}

// phases is how many horizontal positions within an atlas pixel a glyph
// is rasterized at, so text keeps its exact advances yet each glyph is
// drawn 1:1 on whole pixels.
const phases = 4

// glyphKey is a rune rasterized at a phase, in 1/phases of a pixel.
type glyphKey struct {
	r     rune
	phase int
}

// Font is a face at one size, backed by a glyph atlas that grows as new
// characters are drawn.
type Font struct {
	Atlas   *image.Alpha
	version uint64
	glyphs  map[glyphKey]glyph
	// ascii holds the ASCII glyphs at every phase, looked up without a
	// map, as most text is ASCII.
	ascii [128 * phases]glyph
	// indexed are the glyphs shaping put in place of a character's,
	// by face, index and phase.
	indexed map[indexKey]glyph
	faces   []font.Face
	sources []*opentype.Font
	// layouts are each source's layout tables, memos the plan each
	// shaped with last, and shapedRuns the runs shaped so far, by plan
	// and glyphs, shapedCount of them; pending, run and ids are scratch
	// for a text being shaped.
	layouts     []*shape.Font
	memos       []planMemo
	shapedRuns  map[*shape.Plan]map[string][]shape.Glyph
	shapedCount int
	pending     []pending
	run         []shape.Glyph
	ids         []byte
	// only are the characters each source draws, and blank those drawn
	// as nothing when no source has them (Face.Only, Face.Blank).
	only          [][]Range
	blank, hidden []Range
	// looks are how each source is drawn (Face.Styled).
	looks      []Look
	emoji      Emoji
	buf        sfnt.Buffer
	cell       int
	penX, penY int
	ascent     float32
	lineHeight float32
	size       float32
	// smoothing is how glyphs are rasterized.
	smoothing smoothing
	// stroke is how wide the glyphs' outlines are stroked, in pixels of
	// size, 0 for the glyphs filled.
	stroke float32
}

// smoothing is what a font makes of a glyph's coverage: darkened, its
// stems heavier as Chrome's text on macOS (Size); exact, as covered
// (ExactSize); unsmoothed, each pixel covered or not (PixelSize).
type smoothing uint8

const (
	darkened smoothing = iota
	exact
	unsmoothed
)

const atlasRunes = "ÀÂÄÇÉÈÊËÎÏÔÖÙÛÜàâäçéèêëîïôöùûüœ·•×÷°±…—–’“”«»€£¥©®™←↑→↓★♥"

func newFont(f *Face, px float32, how smoothing, stroke float32) (*Font, error) {
	out := &Font{glyphs: map[glyphKey]glyph{}, indexed: map[indexKey]glyph{}, emoji: f.emoji, size: px,
		smoothing: how, blank: f.blank, hidden: f.hidden, stroke: stroke}
	for _, src := range append([]*Face{f}, f.fallbacks...) {
		face, err := opentype.NewFace(
			src.font,
			&opentype.FaceOptions{Size: float64(px * oversample * src.look.scale()), DPI: 72, Hinting: font.HintingNone},
		)
		if err != nil {
			return nil, err
		}
		out.faces, out.sources = append(out.faces, face), append(out.sources, src.font)
		out.only, out.looks = append(out.only, src.only), append(out.looks, src.look)
		out.layouts = append(out.layouts, src.layout)
	}
	m := out.faces[0].Metrics()
	out.ascent = float32(m.Ascent.Ceil()) / oversample
	out.lineHeight = float32(m.Height.Ceil()) / oversample
	out.cell = (m.Ascent + m.Descent).Ceil() + 4
	if w := nextPow2(16 * (out.cell + 2)); w > MaxAtlasSide {
		out.Atlas = image.NewAlpha(image.Rect(0, 0, MaxAtlasSide, min(nextPow2(out.cell+2), MaxAtlasSide)))
		return out, nil
	}
	out.Atlas = image.NewAlpha(image.Rect(0, 0, nextPow2(16*(out.cell+2)), nextPow2(8*(out.cell+2))))
	for r := rune(32); r < 127; r++ {
		out.glyph(r)
	}
	for _, r := range atlasRunes {
		out.glyph(r)
	}
	return out, nil
}

func nextPow2(v int) int {
	p := 64
	for p < v {
		p *= 2
	}
	return p
}

// Version changes whenever a glyph is added to the atlas, so the GPU knows
// to upload it again.
func (f *Font) Version() uint64 { return f.version }

func (f *Font) LineHeight() float32 { return f.lineHeight }

func (f *Font) Ascent() float32 { return f.ascent }

// Measure returns the size of a single- or multi-line string.
func (f *Font) Measure(s string) geom.Vec2 { return f.measure(s, "", 0) }

// measure is the size of s shaped with features, with tracking after
// each cluster.
func (f *Font) measure(s, features string, tracking float32) geom.Vec2 {
	var w, line float32
	lines := 1
	f.items(s, features, func(it item) {
		if it.newline {
			w = max(w, line)
			line = 0
			lines++
			return
		}
		line += it.advance + tracking*float32(it.clusters)
	})
	return geom.V2(max(w, line), float32(lines)*f.lineHeight)
}

// picture is the emoji picture for cluster c, when the face has one.
func (f *Font) picture(c string) (*render.Texture, bool) {
	if f.emoji == nil || len(c) == 1 {
		return nil, false
	}
	return f.emoji.Picture(c)
}

// advance is glyph idx's advance in face, in points, measured in font
// units as layout measures it, so the two agree to the last fraction.
func (f *Font) advance(face int, idx sfnt.GlyphIndex) float32 {
	src := f.sources[face]
	upem := src.UnitsPerEm()
	if v := f.looks[face].varied(); v != nil {
		return v.Font.Advance(uint16(idx), v.Coords) * f.size * f.looks[face].scale() / float32(upem)
	}
	a, err := src.GlyphAdvance(&f.buf, idx, fixed.I(int(upem)), font.HintingNone)
	if err != nil {
		return 0
	}
	return float32(a) / 64 * f.size * f.looks[face].scale() / float32(upem)
}

// glyph is r's glyph at phase 0: its advance and face.
func (f *Font) glyph(r rune) glyph { return f.glyphAt(r, 0) }

// glyphAt is r's glyph rasterized phase/phases of a pixel to the right,
// from the first face that has it, the first time it is asked for; '?'
// when none has it. Its advance is the font's, unrounded.
func (f *Font) glyphAt(r rune, phase int) glyph {
	if r >= 0 && r < 128 {
		if g := f.ascii[int(r)*phases+phase]; g.ready {
			return g
		}
	} else if g, ok := f.glyphs[glyphKey{r, phase}]; ok {
		return g
	}
	g := f.raster(r, phase)
	g.ready = true
	if r >= 0 && r < 128 {
		f.ascii[int(r)*phases+phase] = g
	} else {
		f.glyphs[glyphKey{r, phase}] = g
	}
	return g
}

// raster rasterizes r at phase into the atlas, from the first face that
// has it; '?' when none has it.
func (f *Font) raster(r rune, phase int) glyph {
	face, idx := -1, sfnt.GlyphIndex(0)
	for i, src := range f.sources {
		if !Covers(f.only[i], r) {
			continue
		}
		if x, err := src.GlyphIndex(&f.buf, r); err == nil && x != 0 {
			face, idx = i, x
			break
		}
	}
	if face < 0 {
		if r == '?' || (len(f.blank) > 0 && Covers(f.blank, r)) {
			return glyph{}
		}
		return f.glyphAt('?', phase)
	}
	return f.rasterIndex(face, idx, r, phase)
}

// rasterIndex rasterizes glyph idx of face at phase into the atlas: r's
// glyph, or for r < 0 one shaping put in a character's place.
func (f *Font) rasterIndex(face int, idx sfnt.GlyphIndex, r rune, phase int) glyph {
	g := glyph{advance: f.advance(face, idx), face: face, idx: idx, r: r}
	if len(f.hidden) > 0 && Covers(f.hidden, r) {
		return g
	}
	x := fixed.Int26_6(phase * 64 / phases) //nolint:gosec // phase < phases, so at most 64
	var dr image.Rectangle
	var mask image.Image
	var mp image.Point
	var ok bool
	if l := f.looks[face]; l.Bold || l.Oblique || l.varied() != nil || r < 0 || f.stroke > 0 {
		dr, mask, ok = f.synthetic(face, idx, x, l)
	} else {
		dr, mask, mp, _, ok = f.faces[face].Glyph(fixed.Point26_6{X: x}, r)
	}
	if ok && !dr.Empty() {
		dst, room := f.place(dr.Dx()+2, dr.Dy()+2)
		if !room {
			return g
		}
		draw.Draw(f.Atlas, dst.Inset(1), mask, mp, draw.Src)
		switch f.smoothing {
		case unsmoothed:
			threshold(f.Atlas, dst)
		case darkened:
			embolden(f.Atlas, dst, darkening(f.size*oversample))
		case exact:
		}
		f.version++
		g.offset = geom.V2(float32(dr.Min.X-1), float32(dr.Min.Y-1)).Scale(1.0 / oversample)
		g.size = geom.V2(float32(dst.Dx()), float32(dst.Dy())).Scale(1.0 / oversample)
		g.rect = dst
	}
	return g
}

// darkening is how far stem darkening moves a glyph's edges out, in
// pixels of a raster px pixels tall: 0.005 of it, fitted to Chrome's text
// on macOS, which reads heavier than exact coverage.
func darkening(px float32) float32 { return 0.005 * px }

// embolden moves the edges of the shapes in r out by d pixels: every
// pixel partly covered, or beside one more than half covered, gains d of
// coverage, as an outline offset by d would give it.
func embolden(a *image.Alpha, r image.Rectangle, d float32) {
	add := int(math.Round(float64(d) * 255))
	if add <= 0 {
		return
	}
	src := image.NewAlpha(r)
	draw.Draw(src, r, a, r.Min, draw.Src)
	covered := func(x, y int) bool { return image.Pt(x, y).In(r) && src.AlphaAt(x, y).A > 127 }
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := int(src.AlphaAt(x, y).A)
			if v > 0 || covered(x-1, y) || covered(x+1, y) || covered(x, y-1) || covered(x, y+1) {
				a.SetAlpha(x, y, color.Alpha{A: uint8(min(255, v+add))}) //nolint:gosec // capped at 255, add ≥ 0
			}
		}
	}
}

// threshold makes each pixel in r covered or not, at half coverage: a
// glyph unsmoothed.
func threshold(a *image.Alpha, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			v := uint8(0)
			if a.AlphaAt(x, y).A >= 128 {
				v = 255
			}
			a.SetAlpha(x, y, color.Alpha{A: v})
		}
	}
}

// place finds room for a w×h glyph in the atlas, row by row, doubling the
// atlas's height when it is full; there is none once the atlas would
// pass MaxAtlasSide.
func (f *Font) place(w, h int) (image.Rectangle, bool) {
	row := max(f.cell, h) + 2
	penX, penY := f.penX, f.penY
	if penX+w+2 > f.Atlas.Rect.Dx() {
		penX, penY = 0, penY+row
	}
	if w+2 > f.Atlas.Rect.Dx() || penY+row > MaxAtlasSide {
		return image.Rectangle{}, false
	}
	f.penX, f.penY = penX, penY
	for f.penY+row > f.Atlas.Rect.Dy() {
		grown := image.NewAlpha(image.Rect(0, 0, f.Atlas.Rect.Dx(), min(f.Atlas.Rect.Dy()*2, MaxAtlasSide)))
		draw.Draw(grown, f.Atlas.Rect, f.Atlas, image.Point{}, draw.Src)
		f.Atlas = grown
	}
	at := image.Rect(f.penX+1, f.penY+1, f.penX+1+w, f.penY+1+h)
	f.penX += w + 2
	return at, true
}

// uv is g's rectangle in the atlas, in texels: the atlas only grows,
// downwards, so it stays right however much it grows, in the same frame
// or later; the shader divides by the atlas's size as it samples.
func (f *Font) uv(g glyph) [4]float32 {
	return [4]float32{float32(g.rect.Min.X), float32(g.rect.Min.Y), float32(g.rect.Max.X), float32(g.rect.Max.Y)}
}

// eachCluster calls fn with each grapheme cluster of s and its first rune:
// a ZWJ family, a flag or a letter with its accents is one.
func eachCluster(s string, fn func(cluster string, first rune)) {
	state := -1
	for len(s) > 0 {
		var c string
		c, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		r := []rune(c)[0]
		if strings.ContainsRune(c, '\n') {
			r = '\n'
		}
		fn(c, r)
	}
}
