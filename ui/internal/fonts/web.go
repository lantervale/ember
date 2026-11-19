package fonts

import (
	"fmt"
	"hash/maphash"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/internal/draw"
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
	web []*Face
	// wanted are the files text needed since Wanted was last asked.
	wanted []*File
	// loading counts the files wanted and not yet settled.
	loading int
	chains  map[chainKey]*Chain
	// gen counts the changes that make every chain stale.
	gen uint64
}

// AddWeb adds a face of family whose file comes later: for the weights
// from weight to weightMax, drawing the characters of ranges, all
// without any. It is measured and drawn with once Loaded gives its
// file; until then, and for the characters it lacks, text falls to the
// next family of its list.
func (s *Set) AddWeb(family string, weight, weightMax int, italic bool, ranges []Range, file *File) *Face {
	for _, c := range s.web {
		if c.web == file && c.Family == family && c.Weight == weight && c.WeightMax == weightMax &&
			c.Italic == italic && slices.Equal(c.Ranges, ranges) {
			return c
		}
	}
	c := &Face{Family: family, Weight: weight, WeightMax: weightMax, Italic: italic, Ranges: ranges, web: file}
	if file.state == fileLoaded {
		c.TTF, c.font, c.upem = file.ttf, file.font, float32(file.font.UnitsPerEm())
	}
	s.web = append(s.web, c)
	s.stale()
	return c
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
	s.loading++
	s.wanted = append(s.wanted, file)
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
		if _, err = f.LoadGlyph(&buf, idx, ppem, nil); err != nil {
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
	for _, c := range s.web {
		if c.web == file {
			c.TTF, c.font, c.upem = ttf, f, float32(f.UnitsPerEm())
		}
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

// covers reports whether c draws r when it has it.
func (c *Face) covers(r rune) bool { return draw.Covers(c.Ranges, r) }

// Descriptors are the face's weight and characters as an @font-face
// writes them: "font-weight: 400;", and a unicode-range when it has
// one.
func (c *Face) Descriptors() string {
	out := fmt.Sprintf("font-weight: %d;", c.Weight)
	if c.WeightMax > c.Weight {
		out = fmt.Sprintf("font-weight: %d %d;", c.Weight, c.WeightMax)
	}
	if len(c.Ranges) == 0 {
		return out
	}
	ranges := make([]string, len(c.Ranges))
	for i, r := range c.Ranges {
		ranges[i] = fmt.Sprintf("U+%X-%X", r.Lo, r.Hi)
	}
	return out + " unicode-range: " + strings.Join(ranges, ", ") + ";"
}

// distance is how far the face's weights are from weight, and further
// than any weight when its style is not the one asked.
func (c *Face) distance(weight int, italic bool) int {
	d := 0
	switch hi := max(c.Weight, c.WeightMax); {
	case weight < c.Weight:
		d = c.Weight - weight
	case weight > hi:
		d = weight - hi
	}
	if c.Italic != italic {
		d += 10000
	}
	return d
}

// Chain is the faces a font draws with, in the order a character is
// looked for in them: the faces of each family of its list that an
// @font-face gives, the closest in weight and style, then the face the
// set would draw with had it no such family, then the Fallback.
type Chain struct {
	family  []string
	entries []*Face
	// base is the face that is no @font-face's, drawing what nothing
	// else does.
	base *Face
	// web is whether one of the families is an @font-face's.
	web bool
}

// chainKey is a list of families by what it says, not by the slice that
// holds it: a style computed again names them in a new one.
type chainKey struct {
	families uint64
	n        int
	weight   int
	italic   bool
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
	return chainKey{families: h.Sum64(), n: len(f.Family), weight: f.Weight, italic: f.Italic}
}

// Chain is the faces f draws with.
func (s *Set) Chain(f style.Font) *Chain {
	key := keyOf(f)
	if ch, ok := s.chains[key]; ok && slices.Equal(ch.family, f.Family) {
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
	ch := &Chain{family: slices.Clone(f.Family)}
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
			ch.entries = append(ch.entries, s.webFaces(name, f.Weight, f.Italic)...)
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

// webFaces are family's faces closest to weight and style, one for each
// set of characters, the last declared first, as it wins a character
// two of them draw.
func (s *Set) webFaces(family string, weight int, italic bool) []*Face {
	best := -1
	for _, c := range s.web {
		if d := c.distance(weight, italic); strings.EqualFold(c.Family, family) && (best < 0 || d < best) {
			best = d
		}
	}
	var out []*Face
	for i := len(s.web) - 1; i >= 0; i-- {
		if c := s.web[i]; strings.EqualFold(c.Family, family) && c.distance(weight, italic) == best {
			out = append(out, c)
		}
	}
	return out
}

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
		if c.font != nil || c.web == nil || c.web.state == fileFailed {
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
			waiting = waiting || c.web.state != fileFailed
			continue
		}
		if g, err := c.font.GlyphIndex(&s.buf, r); err == nil && g != 0 {
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

// chainAdvance is text's width at size through ch: each character in
// the first face that draws it, kerned with the one before when both
// are of one face.
func (s *Set) chainAdvance(ch *Chain, size float32, text string) float32 {
	var w float32
	var run fixed.Int26_6
	var of *Face
	prev := sfnt.GlyphIndex(0)
	flush := func() {
		if of != nil {
			w += float32(run) / 64 * size / of.upem
		}
		run, prev = 0, 0
	}
	for _, r := range text {
		c, g, waiting := ch.glyph(s, r)
		if c == nil {
			if waiting || ch.base == nil {
				flush()
				of = nil
				continue
			}
			c = ch.base
		}
		if c != of {
			flush()
			of = c
		}
		ppem := fixed.I(int(c.upem))
		if prev != 0 && g != 0 {
			if k, err := c.font.Kern(&s.buf, prev, g, ppem, font.HintingNone); err == nil {
				run += k
			}
		}
		if a, err := c.font.GlyphAdvance(&s.buf, g, ppem, font.HintingNone); err == nil {
			run += a
		}
		prev = g
	}
	flush()
	return w
}

// chainFallbackMetrics are the largest metrics of ch's faces, besides
// its first, that draw some of text.
func (s *Set) chainFallbackMetrics(ch *Chain, size float32, text string) (box.Metrics, bool) {
	first := ch.first()
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
				if c.web.state == fileFailed {
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
