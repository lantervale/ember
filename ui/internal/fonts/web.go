package fonts

import (
	"errors"
	"fmt"
	"hash/maphash"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Range is the characters from Lo to Hi a face draws, as an
// @font-face's unicode-range gives them.
type Range = draw.Range

// File is a font file a page names and the set does not hold yet: the
// faces whose @font-face give the same src share one, asked for when
// text first needs a character one of them draws.
type File struct {
	state fileState
	ttf   []byte
	font  *sfnt.Font
	// vf is the font's axes and variations, nil for a font that has
	// none, or no TrueType outlines.
	vf *varfont.Font
	// asked is the set's clock when text first needed the file, -1
	// before the set had one.
	asked float32
}

type fileState uint8

const (
	// fileIdle is a file no text has needed yet.
	fileIdle fileState = iota
	// fileWanted is one text needs, on its way.
	fileWanted
	fileLoaded
	fileFailed
)

// Loaded reports whether the file has arrived and draws.
func (f *File) Loaded() bool { return f.state == fileLoaded }

// Settled reports whether the file has arrived or failed.
func (f *File) Settled() bool { return f.state == fileLoaded || f.state == fileFailed }

// webFonts is a Set's fonts of the web: the faces of its @font-face
// rules whose files are fetched, and the chains text is measured and
// drawn through once it has any.
type webFonts struct {
	// web are the faces text is drawn with, of those known.
	web, known []*Face
	// wanted are the files text needed since Wanted was last asked.
	wanted []*File
	// loading counts the files wanted and not yet settled.
	loading int
	chains  map[chainKey]*Chain
	// gen counts the changes that make every chain stale.
	gen uint64
	// now is the clock the faces' periods are timed on, once clocked.
	now     float32
	clocked bool
}

// AddWeb adds a face of family whose file comes later: for the weights
// from weight to weightMax, of the normal width, drawing the characters
// of ranges, all without any. It is AddFace with those descriptors.
func (s *Set) AddWeb(family string, weight, weightMax int, italic bool, ranges []Range, file *File) *Face {
	return s.AddFace(Descriptors{Family: family, Weight: weight, WeightMax: weightMax, Italic: italic,
		Stretch: 100, StretchMax: 100, Ranges: ranges}, file)
}

// Descriptors are what an @font-face says of its face.
type Descriptors struct {
	Family            string
	Weight, WeightMax int
	Italic            bool
	// Stretch is the width the face is of, a percentage, to StretchMax.
	Stretch, StretchMax float32
	// AutoWeight and AutoStretch are a face that gives no weight or no
	// width: normal, or a variable font's own.
	AutoWeight, AutoStretch bool
	Ranges                  []Range
	Look                    Look
	Display                 Display
}

// Look is how an @font-face has its face measured and drawn: scaled by
// SizeAdjust (0 for 1), with metrics in place of its own, shaped with
// the OpenType features its font-feature-settings turns on or off,
// HarfBuzz's way (shape.ParseFeatures), before the text's own.
type Look struct {
	SizeAdjust               float32
	Ascent, Descent, LineGap Override
	Features                 string
}

// Override is a metric an @font-face gives in place of its file's, as a
// fraction of the font size, when Set.
type Override struct {
	Set bool
	Of  float32
}

func (o Override) or(own, size float32) float32 {
	if o.Set {
		return o.Of * size
	}
	return own
}

func (l Look) adjust() float32 {
	if l.SizeAdjust == 0 {
		return 1
	}
	return l.SizeAdjust
}

// AddFace adds the face an @font-face describes, whose file comes later:
// it is measured and drawn with once Loaded gives its file; until then,
// and for the characters it lacks, text falls to the next family of its
// list. A face the set knew is the same face, and is among those text
// is drawn with again.
func (s *Set) AddFace(d Descriptors, file *File) *Face {
	for _, c := range s.known {
		if c.web == file && c.Family == d.Family && c.Weight == d.Weight && c.WeightMax == d.WeightMax &&
			c.Italic == d.Italic && c.Stretch == d.Stretch && c.StretchMax == d.StretchMax && c.Look == d.Look &&
			c.AutoWeight == d.AutoWeight && c.AutoStretch == d.AutoStretch && c.Display == d.Display &&
			slices.Equal(c.Ranges, d.Ranges) {
			if !slices.Contains(s.web, c) {
				s.web = append(s.web, c)
				s.stale()
			}
			return c
		}
	}
	c := &Face{Family: d.Family, Weight: d.Weight, WeightMax: d.WeightMax, Italic: d.Italic, Stretch: d.Stretch,
		StretchMax: d.StretchMax, AutoWeight: d.AutoWeight, AutoStretch: d.AutoStretch, Ranges: d.Ranges, Look: d.Look,
		Display: d.Display, web: file}
	if file.state == fileLoaded {
		c.TTF, c.font, c.upem = file.ttf, file.font, float32(file.font.UnitsPerEm())
	}
	s.known = append(s.known, c)
	s.web = append(s.web, c)
	s.stale()
	return c
}

// SetWeb has text drawn with faces alone of those the set knows, in
// their order: the @font-face rules of the sheets a Document has now.
// A face left out is kept, as is its file, for a sheet that gives it
// again. It reports whether the faces changed.
func (s *Set) SetWeb(faces []*Face) bool {
	if slices.Equal(s.web, faces) {
		return false
	}
	s.web = slices.Clone(faces)
	s.stale()
	return true
}

// stale drops the chains: the faces they were made of have changed.
func (s *Set) stale() {
	s.gen++
	clear(s.chains)
}

// Gen changes whenever a face arrives, fails or is added: what was
// measured or drawn through a Chain before is to be done again.
func (s *Set) Gen() uint64 { return s.gen }

// Wanted takes the files text has needed since it was last asked: each
// is given once, and is on its way until Loaded or Failed.
func (s *Set) Wanted() []*File {
	w := s.wanted
	s.wanted = nil
	return w
}

// Loading is how many files are on their way.
func (s *Set) Loading() int { return s.loading }

// Want asks for file as text needing it would.
func (s *Set) Want(file *File) {
	if file.state != fileIdle {
		return
	}
	file.state = fileWanted
	file.asked = -1
	if s.clocked {
		file.asked = s.now
	}
	s.loading++
	s.wanted = append(s.wanted, file)
	s.stale()
}

// Parse reads a font file and checks it draws, off the frame: what
// Loaded then takes.
func Parse(ttf []byte) (*sfnt.Font, error) {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return nil, err
	}
	var buf sfnt.Buffer
	ppem := fixed.I(int(f.UnitsPerEm()))
	if _, err = f.Metrics(&buf, ppem, font.HintingNone); err != nil {
		return nil, err
	}
	n := f.NumGlyphs()
	for _, g := range []int{0, 1, n / 2, n - 1} {
		if g < 0 || g >= n {
			continue
		}
		idx := sfnt.GlyphIndex(g) //nolint:gosec // under the font's 65535 glyphs
		if _, err = f.LoadGlyph(&buf, idx, ppem, nil); err != nil && !errors.Is(err, sfnt.ErrColoredGlyph) {
			return nil, err
		}
		if _, err = f.GlyphAdvance(&buf, idx, ppem, font.HintingNone); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Loaded gives file its font, read by Parse: its faces measure and draw
// from now on.
func (s *Set) Loaded(file *File, ttf []byte, f *sfnt.Font) {
	s.settle(file, fileLoaded)
	file.ttf, file.font = ttf, f
	if vf, err := varfont.Parse(ttf); err == nil && vf.Variable() {
		file.vf = vf
	}
	for _, c := range s.known {
		if c.web != file || c.expired {
			continue
		}
		if s.late(c) {
			c.expired = true
			continue
		}
		c.TTF, c.font, c.upem = ttf, f, float32(f.UnitsPerEm())
	}
}

// Failed says file never comes: its faces' text stays in the next
// family.
func (s *Set) Failed(file *File) { s.settle(file, fileFailed) }

func (s *Set) settle(file *File, state fileState) {
	if file.state == fileWanted {
		s.loading--
	}
	file.state = state
	s.stale()
}

// Web reports whether the face is one of an @font-face whose file is
// fetched.
func (c *Face) Web() bool { return c.web != nil }

// File is the file a face of the web draws from, nil for another.
func (c *Face) File() *File { return c.web }

// gone reports whether c draws nothing, ever: its file failed, or
// landed or would land past its swap period.
func (c *Face) gone() bool { return c.web != nil && (c.web.state == fileFailed || c.expired) }

// covers reports whether c draws r when it has it.
func (c *Face) covers(r rune) bool { return draw.Covers(c.Ranges, r) }

// Descriptors are the face's descriptors as an @font-face writes them:
// "font-weight: 400;", and its width, unicode-range, size-adjust,
// metrics and kerning when it has them; a variable font's weight or
// width left auto when it gives none.
func (c *Face) Descriptors() string {
	var out string
	_, _, weighs := c.axis("wght")
	_, _, widens := c.axis("wdth")
	switch {
	case c.AutoWeight && weighs:
	case c.WeightMax > c.Weight:
		out = fmt.Sprintf("font-weight: %d %d;", c.Weight, c.WeightMax)
	default:
		out = fmt.Sprintf("font-weight: %d;", c.Weight)
	}
	if lo, hi := c.stretches(); (lo != 100 || hi != 100) && (!c.AutoStretch || !widens) {
		out += fmt.Sprintf(" font-stretch: %g%% %g%%;", lo, hi)
	}
	if c.Look.SizeAdjust != 0 {
		out += fmt.Sprintf(" size-adjust: %g%%;", c.Look.SizeAdjust*100)
	}
	for _, o := range []struct {
		name string
		o    Override
	}{{"ascent-override", c.Look.Ascent}, {"descent-override", c.Look.Descent}, {"line-gap-override", c.Look.LineGap}} {
		if o.o.Set {
			out += fmt.Sprintf(" %s: %g%%;", o.name, o.o.Of*100)
		}
	}
	if c.Look.Features != "" {
		out += " font-feature-settings: " + featureSettings(c.Look.Features) + ";"
	}
	if len(c.Ranges) == 0 {
		return strings.TrimSpace(out)
	}
	ranges := make([]string, len(c.Ranges))
	for i, r := range c.Ranges {
		ranges[i] = fmt.Sprintf("U+%X-%X", r.Lo, r.Hi)
	}
	return strings.TrimSpace(out + " unicode-range: " + strings.Join(ranges, ", ") + ";")
}

// stretches are the widths the face covers, 100 for one that says none
// and a variable font's own for one whose width is auto.
func (c *Face) stretches() (float32, float32) {
	if c.AutoStretch {
		if lo, hi, ok := c.axis("wdth"); ok {
			return lo, hi
		}
	}
	lo, hi := c.Stretch, c.StretchMax
	if lo == 0 {
		lo = 100
	}
	return lo, max(lo, hi)
}

// weights are the weights the face covers, a variable font's own for
// one whose weight is auto.
func (c *Face) weights() (int, int) {
	if c.AutoWeight {
		if lo, hi, ok := c.axis("wght"); ok {
			return int(lo), int(hi)
		}
	}
	return c.Weight, max(c.Weight, c.WeightMax)
}

// axis is the range of the face's axis tag, when its file has landed
// and is a variable font with one.
func (c *Face) axis(tag string) (float32, float32, bool) {
	if c.web == nil || c.web.vf == nil {
		return 0, 0, false
	}
	vf := c.web.vf
	if i := vf.Tag(tag); i >= 0 {
		return vf.Axes[i].Min, vf.Axes[i].Max, true
	}
	return 0, 0, false
}

// Variable is the face's file as a variable font, nil for one that is
// not or has not landed.
func (c *Face) Variable() *varfont.Font {
	if c.web == nil {
		return nil
	}
	return c.web.vf
}

// Instance is the place on the face's axes text in f is drawn at: its
// weight and width, kept within the face's, on wght and wdth, then
// font-variation-settings' axes; nil for the default instance or a
// face that is no variable font.
func (c *Face) Instance(f style.Font) varfont.Coords {
	vf := c.Variable()
	if vf == nil {
		return nil
	}
	user := make([]float32, len(vf.Axes))
	for i, a := range vf.Axes {
		user[i] = a.Default
	}
	if i := vf.Tag("wght"); i >= 0 {
		lo, hi := c.weights()
		user[i] = min(max(f.Weight, float32(lo)), float32(hi))
	}
	if i := vf.Tag("wdth"); i >= 0 {
		stretch := f.Stretch
		if stretch == 0 {
			stretch = 100
		}
		lo, hi := c.stretches()
		user[i] = min(max(stretch, lo), hi)
	}
	for _, v := range f.Variations {
		if i := vf.Tag(v.Tag); i >= 0 {
			user[i] = v.Value
		}
	}
	return vf.Normalize(user)
}

// stretchThreshold puts a width on the side of normal not asked for
// past every one on the side asked, as Chrome's font matching does.
const stretchThreshold = 1000

// distance is how far the face is from the width, style and weight asked,
// in that order: the width first, as CSS's font matching narrows the
// faces by it before their style and weight, the narrower first for a
// width at most normal and the wider for one above; then the style, then
// the weight.
func (c *Face) distance(stretch float32, weight int, italic bool) float64 {
	if stretch == 0 {
		stretch = 100
	}
	var ds float32
	switch lo, hi := c.stretches(); {
	case stretch >= lo && stretch <= hi:
	case stretch > 100 && lo > stretch:
		ds = lo - stretch
	case stretch > 100:
		ds = stretch - hi + stretchThreshold
	case hi < stretch:
		ds = stretch - hi
	default:
		ds = lo - stretch + stretchThreshold
	}
	d := 0
	switch lo, hi := c.weights(); {
	case weight < lo:
		d = lo - weight
	case weight > hi:
		d = weight - hi
	}
	if c.Italic != italic {
		d += 10000
	}
	return float64(ds)*1e6 + float64(d)
}

// boldThreshold is the weight from which a bold is made up from a face
// lighter than it, as Chrome's.
const boldThreshold = 600

// Synthetic reports whether text in f is drawn with c made bold, or
// slanted, as the family has no such face: a bold one where f is at
// least 600 and c lighter than that, an italic one where f is italic
// and c is not, unless font-synthesis forbids it.
func (c *Face) Synthetic(f style.Font) (bool, bool) {
	_, heaviest := c.weights()
	bold := f.Weight >= boldThreshold && heaviest < boldThreshold &&
		f.Synthesis&style.NoSyntheticBold == 0
	italic := f.Italic && !c.Italic && f.Synthesis&style.NoSyntheticItalic == 0
	return bold, italic
}

// Chain is the faces a font draws with, in the order a character is
// looked for in them: the faces of each family of its list that an
// @font-face gives, the closest in weight and style, then the face the
// set would draw with had it no such family, then the Fallback.
type Chain struct {
	family []string
	// font is the font the chain is of.
	font    style.Font
	entries []*Face
	// base is the face that is no @font-face's, drawing what nothing
	// else does.
	base *Face
	// web is whether one of the families is an @font-face's.
	web bool
	// instances are where each variable face of the chain draws it.
	instances map[*Face]varfont.Coords
}

// Instance is where on c's axes the chain draws, nil for c's default
// instance or a face that is no variable font.
func (ch *Chain) Instance(c *Face) varfont.Coords {
	if c.Variable() == nil {
		return nil
	}
	if v, ok := ch.instances[c]; ok {
		return v
	}
	v := c.Instance(ch.font)
	if ch.instances == nil {
		ch.instances = map[*Face]varfont.Coords{}
	}
	ch.instances[c] = v
	return v
}

// chainKey is a list of families by what it says, not by the slice that
// holds it: a style computed again names them in a new one.
type chainKey struct {
	families  uint64
	n         int
	weight    int
	italic    bool
	stretch   float32
	synthesis style.FontSynthesis
	palette   string
	// locale is the language's, which picks the machine's faces.
	locale locale
}

//nolint:gochecknoglobals // the seed every chain key is hashed with
var chainSeed = maphash.MakeSeed()

func keyOf(f style.Font) chainKey {
	var h maphash.Hash
	h.SetSeed(chainSeed)
	for _, name := range f.Family {
		h.WriteString(name)
		h.WriteByte(0)
	}
	for _, v := range f.Variations {
		h.WriteString(v.Tag)
		h.WriteString(strconv.FormatFloat(float64(v.Value), 'g', -1, 32))
		h.WriteByte(1)
	}
	return chainKey{families: h.Sum64(), n: len(f.Family), weight: int(f.Weight), italic: f.Italic,
		stretch: f.Stretch, synthesis: f.Synthesis, palette: f.Palette, locale: localeOf(f.Lang)}
}

// Chain is the faces f draws with.
func (s *Set) Chain(f style.Font) *Chain {
	key := keyOf(f)
	if ch, ok := s.chains[key]; ok && slices.Equal(ch.family, f.Family) && slices.Equal(ch.font.Variations, f.Variations) {
		return ch
	}
	ch := s.chain(f)
	if s.chains == nil {
		s.chains = map[chainKey]*Chain{}
	}
	s.chains[key] = ch
	return ch
}

func (s *Set) chain(f style.Font) *Chain {
	ch := &Chain{family: slices.Clone(f.Family), font: f}
	var plain []string
	for _, name := range f.Family {
		if !s.hasWeb(name) {
			plain = append(plain, name)
		}
	}
	ch.web = len(plain) < len(f.Family)
	if !ch.web {
		return ch
	}
	rest := f
	rest.Family = plain
	ch.base = s.plainFace(rest)
	based := false
	for _, name := range f.Family {
		if s.hasWeb(name) {
			ch.entries = append(ch.entries, s.webFaces(name, f)...)
			continue
		}
		if !based && ch.base != nil && s.draws(name, ch.base) {
			ch.entries, based = append(ch.entries, ch.base), true
		}
	}
	if !based && ch.base != nil {
		ch.entries = append(ch.entries, ch.base)
	}
	if fb := s.Fallback(); fb != nil && fb != ch.base {
		ch.entries = append(ch.entries, fb)
	}
	return ch
}

// draws reports whether name is the family c is of: its own name, a
// generic one drawn with it, or an installed one in another case.
func (s *Set) draws(name string, c *Face) bool {
	if g, ok := genericFace(name); ok {
		name = g
	}
	return name == c.Family || (s.installed != nil && s.named[name] == c.Family)
}

func (s *Set) hasWeb(family string) bool {
	for _, c := range s.web {
		if strings.EqualFold(c.Family, family) {
			return true
		}
	}
	return false
}

// webFaces are family's faces closest to f's width, style and weight,
// one for each set of characters, the last declared first, as it wins a
// character two of them draw.
func (s *Set) webFaces(family string, f style.Font) []*Face {
	weight := int(f.Weight)
	best := -1.0
	for _, c := range s.web {
		if d := c.distance(f.Stretch, weight, f.Italic); strings.EqualFold(c.Family, family) && (best < 0 || d < best) {
			best = d
		}
	}
	var out []*Face
	for i := len(s.web) - 1; i >= 0; i-- {
		if c := s.web[i]; strings.EqualFold(c.Family, family) && c.distance(f.Stretch, weight, f.Italic) == best {
			out = append(out, c)
		}
	}
	return out
}

// Font is the font the chain draws.
func (ch *Chain) Font() style.Font { return ch.font }

// Faces are the chain's faces that have their file, in order: what
// draws its text now.
func (ch *Chain) Faces() []*Face {
	var out []*Face
	for _, c := range ch.entries {
		if c.font != nil {
			out = append(out, c)
		}
	}
	return out
}

// Loading are the characters whose face is on its way: one of them no
// face of the chain has yet shows as nothing, not as a missing glyph.
func (ch *Chain) Loading() []Range {
	var out []Range
	for _, c := range ch.entries {
		if c.font != nil || c.web == nil || c.gone() {
			continue
		}
		if len(c.Ranges) == 0 {
			return []Range{{Lo: 0, Hi: utf8.MaxRune}}
		}
		out = append(out, c.Ranges...)
	}
	return out
}

// glyph is the face of the chain drawing r and r's glyph there: the
// first that covers r and has it. A face covering r whose file has not
// come is asked for, the first of them only, and the rest of the chain
// draws r meanwhile; waiting reports that one is on its way when no
// face draws r.
func (ch *Chain) glyph(s *Set, r rune) (*Face, sfnt.GlyphIndex, bool) {
	waiting := false
	for _, c := range ch.entries {
		if !c.covers(r) {
			continue
		}
		if c.font == nil {
			if c.web.state == fileIdle && !waiting {
				s.Want(c.web)
			}
			waiting = waiting || !c.gone()
			continue
		}
		if g, err := c.font.GlyphIndex(&s.buf, r); err == nil && g != 0 {
			return c, g, false
		}
	}
	if !waiting {
		if c, g := s.systemGlyph(ch.font, r); c != nil {
			return c, g, false
		}
	}
	return nil, 0, waiting
}

// first is the face the chain's metrics are: the first that has its
// file and draws a space, as CSS's first available font.
func (ch *Chain) first() *Face {
	for _, c := range ch.entries {
		if c.font != nil && c.covers(' ') {
			return c
		}
	}
	return ch.base
}

// chainAdvance is text's width in f through ch: each character in the
// first face that draws it, shaped with those beside it of one face.
func (s *Set) chainAdvance(ch *Chain, f style.Font, text, before, after string, own part) float32 {
	var w float32
	var of *Face
	script := shape.Script(text)
	run := s.run[:0]
	flush := func() {
		if of != nil && len(run) > 0 {
			w += s.shaped(of, script, f.Features, run, ch.Instance(of), around(text, run, before, after), own) *
				f.Size * of.Look.adjust() / of.upem
		}
		run = run[:0]
	}
	for i, r := range text {
		if sc, ok := shape.ScriptOf(r); ok && sc != script {
			flush()
			script = sc
		}
		c, g, waiting := ch.glyph(s, r)
		if c == nil && of != nil && shape.VariationSelector(r) {
			// Shaped with its character, whose face may draw the two as one.
			c = of
		}
		if c == nil {
			if waiting || ch.base == nil {
				flush()
				of = nil
				continue
			}
			if a, ok := s.spaceAdvance(ch.first(), r, f.Size); ok {
				flush()
				of = nil
				if own.has(i) {
					w += a
					s.counted++
				}
				continue
			}
			c = ch.base
		}
		if c != of {
			flush()
			of = c
		}
		//nolint:gosec // a glyph index and a text offset
		run = append(run, shape.Glyph{ID: uint16(g), Cluster: int32(i), Char: r})
	}
	flush()
	s.run = run
	return w
}

// chainFallbackMetrics are the largest metrics of ch's faces, besides
// its first, that draw some of text.
func (s *Set) chainFallbackMetrics(ch *Chain, size float32, text string) (box.Metrics, bool) {
	first := ch.first()
	if isASCII(text) && s.drawsASCII(first) {
		return box.Metrics{}, false
	}
	var out box.Metrics
	var seen *Face
	found := false
	for _, r := range text {
		c, _, _ := ch.glyph(s, r)
		if c == nil || c == first || c == seen {
			continue
		}
		seen = c
		m := s.metricsOf(c, size)
		out.Ascent, out.Descent = max(out.Ascent, m.Ascent), max(out.Descent, m.Descent)
		out.LineGap = max(out.LineGap, m.LineGap)
		found = true
	}
	return out, found
}

// chainDraws is Draws through ch.
func (s *Set) chainDraws(ch *Chain, text string) bool {
	rest, state := text, -1
	var g string
	for len(rest) > 0 {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if s.emoji != nil && s.emoji(g) {
			continue
		}
		bare := strings.ReplaceAll(g, "️", "")
		if utf8.RuneCountInString(bare) != 1 {
			return false
		}
		r, _ := utf8.DecodeRuneInString(bare)
		if c, _, _ := ch.glyph(s, r); c == nil {
			return false
		}
	}
	return true
}

// chained is f's Chain when one of its families is an @font-face's
// whose file is fetched: such a font is measured and drawn through it.
// Nil for any other font, which the set draws as it does without such
// faces.
func (s *Set) chained(f style.Font) *Chain {
	if len(s.web) == 0 {
		return nil
	}
	if ch := s.Chain(f); ch.web {
		return ch
	}
	return nil
}

// Chained is f's Chain when it is drawn through one, nil otherwise.
func (s *Set) Chained(f style.Font) *Chain { return s.chained(f) }

// styleWords are the words a font's full name ends with, and the
// weight and style each says.
//
//nolint:gochecknoglobals // a lookup table
var styleWords = []struct {
	word   string
	weight int
	italic bool
}{
	{"bold italic", 700, true}, {"bolditalic", 700, true}, {"bold oblique", 700, true},
	{"regular", 400, false}, {"italic", 400, true}, {"oblique", 400, true}, {"bold", 700, false},
	{"thin", 100, false}, {"extralight", 200, false}, {"light", 300, false}, {"medium", 500, false},
	{"semibold", 600, false}, {"extrabold", 800, false}, {"black", 900, false},
}

// Local is the reader of the installed font a src's local(name) names,
// for a face of weight and style: a family the machine has, in the
// closest face, or a family followed by its style ("Georgia Bold",
// "Georgia-Bold"), in that face. Nil when the machine has none, or its
// fonts are not looked at or not scanned yet. The reader is safe off
// the frame.
func (s *Set) Local(name string, weight int, italic bool) func() ([]byte, error) {
	if s.installed == nil {
		return nil
	}
	fam, exact := s.installed.family(name), false
	if fam == nil {
		lower := strings.ToLower(name)
		for _, w := range styleWords {
			if rest, ok := strings.CutSuffix(lower, w.word); ok && len(rest) > 1 {
				if fam = s.installed.family(strings.TrimRight(rest, " -")); fam != nil {
					weight, italic, exact = w.weight, w.italic, true
					break
				}
			}
		}
	}
	if fam == nil {
		return nil
	}
	var best *installedFace
	score := func(f *installedFace) int {
		d := f.weight - weight
		if d < 0 {
			d = -d
		}
		if f.italic != italic {
			d += 10000
		}
		return d
	}
	for _, f := range fam.faces {
		if best == nil || score(f) < score(best) {
			best = f
		}
	}
	if best == nil || (exact && score(best) != 0) {
		return nil
	}
	return func() ([]byte, error) { return best.read() }
}

// Ready reports whether text in f waits for no file: each face of the
// web that would draw one of its characters has its file, or never
// will. With ask the files it waits for are asked for, as measuring
// the text does.
func (s *Set) Ready(f style.Font, text string, ask bool) bool {
	ch := s.chained(f)
	if ch == nil {
		return true
	}
	ready := true
	for _, r := range text {
		for _, c := range ch.entries {
			if !c.covers(r) {
				continue
			}
			if c.font == nil {
				if c.gone() {
					continue
				}
				ready = false
				if ask {
					s.Want(c.web)
				}
				break
			}
			if g, err := c.font.GlyphIndex(&s.buf, r); err == nil && g != 0 {
				break
			}
		}
	}
	return ready
}
