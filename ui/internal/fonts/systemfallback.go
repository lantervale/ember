package fonts

import (
	"slices"
	"strings"
	"unicode"

	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/style"
)

// systemFallback is a Set's drawing of the characters no face of a font
// has with the machine's fonts, as Chrome asks the platform for a font
// that has them: off until SetSystemFallback.
type systemFallback struct {
	on bool
	// chosen is the face drawing each character in each context, nil
	// where the machine has none.
	chosen map[fallbackKey]*Face
	// read are the installed fonts parsed for it, nil for one that
	// can't be drawn with.
	read map[*installedFace]*Face
	// drawn are the faces chosen in each context, in the order chosen,
	// with the characters each draws there.
	drawn map[fallbackContext]*fallbackFaces
	// gen counts the characters chosen so far.
	gen uint64
	// waiting is whether a character waits for the coverage of every
	// installed font, read off the frame.
	waiting bool
}

type fallbackFaces struct {
	faces []*Face
	runes map[*Face][]rune
}

// fallbackContext is what Chrome asks the platform for a font by,
// besides the character: the language of the text, the generic family
// its font list stands on, and its weight and style.
type fallbackContext struct {
	locale       locale
	class        fallbackClass
	bold, italic bool
}

// weight is the weight a face of a family the platform names for no
// weight of its own is picked closest to.
func (cx fallbackContext) weight() int {
	if cx.bold {
		return 700
	}
	return 400
}

type fallbackKey struct {
	r  rune
	cx fallbackContext
}

// locale is what of a language picks the faces of Han characters and
// the punctuation Chinese, Japanese and Korean share.
type locale uint8

const (
	localeNone locale = iota
	localeJa
	localeKo
	localeZhHans
	localeZhHant
)

// localeOf is lang's locale, as Chrome reads a BCP 47 tag: Chinese
// written in Taiwan, Hong Kong or Macao, or said to be Traditional, is
// Traditional, other Chinese Simplified.
func localeOf(lang string) locale {
	if lang == "" {
		return localeNone
	}
	tags := strings.Split(strings.ToLower(strings.ReplaceAll(lang, "_", "-")), "-")
	switch tags[0] {
	case "ja":
		return localeJa
	case "ko":
		return localeKo
	case "yue":
		return localeZhHant
	case "zh":
		for _, t := range tags[1:] {
			switch t {
			case "hant", "tw", "hk", "mo":
				return localeZhHant
			case "hans", "cn", "sg":
				return localeZhHans
			}
		}
		return localeZhHans
	}
	return localeNone
}

// fallbackClass is the kind of face the font list stands on, which Chrome's
// fallback follows: a serif font falls back to serif faces.
type fallbackClass uint8

const (
	classSans fallbackClass = iota
	classSerif
	classMono
)

// SetSystemFallback has the set draw a character that no face of a font
// has, nor the Fallback, with an installed face that has it, chosen as
// Chrome on the machine's platform chooses it: once SetInstalled or Land
// gives it the machine's fonts.
func (s *Set) SetSystemFallback(on bool) { s.fallback.on = on }

// SystemFallback reports whether SetSystemFallback turned it on.
func (s *Set) SystemFallback() bool { return s.fallback.on }

// FallbackContext is what picks the machine's faces drawing what a font
// lacks, besides the character: equal for two fonts that fall back to the
// same faces, zero without the system fallback.
type FallbackContext = fallbackContext

// FallbackContext is f's.
func (s *Set) FallbackContext(f style.Font) FallbackContext {
	if !s.fallback.on || s.installed == nil {
		return FallbackContext{}
	}
	return s.contextOf(f)
}

// SystemGen counts the characters the machine's faces were chosen for
// so far: what draws text changes with it.
func (s *Set) SystemGen() uint64 { return s.fallback.gen }

// FallbackFace is an installed face drawing what a font lacks, and the
// characters it draws for it.
type FallbackFace struct {
	Face   *Face
	Ranges []Range
}

// SystemFaces are the installed faces drawing what f lacks, in the
// order they were first chosen, each with the characters it draws in
// f's context so far: what layout measured them in.
func (s *Set) SystemFaces(f style.Font) []FallbackFace {
	if !s.fallback.on || s.installed == nil {
		return nil
	}
	d := s.fallback.drawn[s.contextOf(f)]
	if d == nil {
		return nil
	}
	out := make([]FallbackFace, 0, len(d.faces))
	for _, c := range d.faces {
		out = append(out, FallbackFace{Face: c, Ranges: rangesOf(d.runes[c])})
	}
	return out
}

func rangesOf(runes []rune) []Range {
	sorted := slices.Clone(runes)
	slices.Sort(sorted)
	var out []Range
	for _, r := range sorted {
		if k := len(out) - 1; k >= 0 && r <= out[k].Hi+1 {
			out[k].Hi = max(out[k].Hi, r)
			continue
		}
		out = append(out, Range{Lo: r, Hi: r})
	}
	return out
}

// contextOf is f's fallback context: its class is the generic family of
// its list that comes before any family the set or the machine has,
// sans-serif when none does.
func (s *Set) contextOf(f style.Font) fallbackContext {
	cx := fallbackContext{locale: localeOf(f.Lang), bold: f.Weight >= 600, italic: f.Italic}
	for _, name := range f.Family {
		switch strings.ToLower(name) {
		case "serif":
			cx.class = classSerif
			return cx
		case "monospace":
			cx.class = classMono
			return cx
		case "sans-serif", "system-ui":
			return cx
		}
		if s.has(name) || s.hasWeb(name) || s.installed != nil && s.installedFamily(name) != "" {
			return cx
		}
	}
	return cx
}

// fallsBack reports whether r is drawn by a glyph some face may have:
// not a space, a control, a format character or a variation selector,
// which the face of the text round them takes.
func fallsBack(r rune) bool {
	return unicode.IsGraphic(r) && !unicode.IsSpace(r) && !unicode.Is(unicode.Variation_Selector, r)
}

// systemGlyph is the installed face drawing r in f where no face of f
// has it, and its glyph there: nil when the set has no such fallback,
// the machine has no face for r, or the face is still being looked for.
func (s *Set) systemGlyph(f style.Font, r rune) (*Face, sfnt.GlyphIndex) {
	if !s.fallback.on || s.installed == nil || !fallsBack(r) {
		return nil, 0
	}
	cx := s.contextOf(f)
	key := fallbackKey{r, cx}
	c, ok := s.fallback.chosen[key]
	if !ok {
		var settled bool
		if c, settled = s.choose(r, cx); !settled {
			return nil, 0
		}
		if s.fallback.chosen == nil {
			s.fallback.chosen = map[fallbackKey]*Face{}
		}
		s.fallback.chosen[key] = c
		if c != nil {
			s.assign(cx, c, r)
		}
	}
	if c == nil {
		return nil, 0
	}
	g, err := c.font.GlyphIndex(&s.buf, r)
	if err != nil || g == 0 {
		return nil, 0
	}
	return c, g
}

func (s *Set) assign(cx fallbackContext, c *Face, r rune) {
	if s.fallback.drawn == nil {
		s.fallback.drawn = map[fallbackContext]*fallbackFaces{}
	}
	d := s.fallback.drawn[cx]
	if d == nil {
		d = &fallbackFaces{runes: map[*Face][]rune{}}
		s.fallback.drawn[cx] = d
	}
	if _, ok := d.runes[c]; !ok {
		d.faces = append(d.faces, c)
	}
	d.runes[c] = append(d.runes[c], r)
	s.fallback.gen++
}

// choose is the face drawing r in cx, settled false while the coverage
// of the machine's fonts it needs is read: the first face of the
// platform's families for r's script and cx that has r; for a character
// of no script, one already chosen in cx; then the first installed
// family, by name, that has it.
func (s *Set) choose(r rune, cx fallbackContext) (*Face, bool) {
	script, ok := scriptOf(r)
	cjk := cjkCommon(r)
	if ok || cjk {
		for _, p := range preferred(script, cjk, cx.locale, cx.class) {
			fam := s.installed.family(p.name)
			if fam == nil {
				continue
			}
			weight := p.regular
			if cx.bold {
				weight = p.bold
			}
			if weight == 0 {
				weight = cx.weight()
			}
			if c := s.fallbackFace(fam, closestFace(fam.faces, weight, cx.italic), r); c != nil {
				return c, true
			}
		}
	}
	if !ok && !cjk {
		if d := s.fallback.drawn[cx]; d != nil {
			for _, c := range d.faces {
				if s.hasGlyph(c, r) {
					return c, true
				}
			}
		}
	}
	families, done := s.installed.covered()
	if !done {
		s.fallback.waiting = true
		return nil, false
	}
	for _, fam := range families {
		if c := s.fallbackFace(fam, closestFace(fam.faces, cx.weight(), cx.italic), r); c != nil {
			return c, true
		}
	}
	return nil, true
}

// fallbackFace is f, of fam, parsed once, when its cmap has r and ember
// draws it; nil otherwise. A font that can't be read is passed over
// without a warning: the page never named it.
func (s *Set) fallbackFace(fam *installedFamily, f *installedFace, r rune) *Face {
	if f == nil || skipped(fam.name) {
		return nil
	}
	if cov := f.coverage(); !cov.drawable || !cov.has(r) {
		return nil
	}
	c, ok := s.fallback.read[f]
	if !ok {
		c = &Face{Family: fam.name, Weight: f.weight, Italic: f.italic, file: f}
		if c.parse() != nil {
			c = nil
		}
		if s.fallback.read == nil {
			s.fallback.read = map[*installedFace]*Face{}
		}
		s.fallback.read[f] = c
	}
	if c == nil || !s.hasGlyph(c, r) {
		return nil
	}
	return c
}

// skipped reports whether family is one no fallback draws with: the
// font of placeholders that covers every character.
func skipped(family string) bool {
	return strings.EqualFold(family, "LastResort") || strings.EqualFold(family, "Last Resort")
}

// closestFace is the face of faces nearest weight, in the asked style
// where there is one.
func closestFace(faces []*installedFace, weight int, italic bool) *installedFace {
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
	for _, f := range faces {
		if best == nil || score(f) < score(best) {
			best = f
		}
	}
	return best
}

// scriptOf is r's script as the platforms' fallback tells them apart:
// shape's, Han and kana besides, false for a character of none.
func scriptOf(r rune) (shape.Tag, bool) {
	switch {
	case unicode.Is(unicode.Han, r):
		return tagHani, true
	case unicode.In(r, unicode.Hiragana, unicode.Katakana):
		return tagKana, true
	}
	return shape.ScriptOf(r)
}

// cjkCommon reports whether r is a character of no script that Chinese,
// Japanese and Korean share: their punctuation, symbols and full-width
// forms, and the kana's prolonged sound mark.
func cjkCommon(r rune) bool {
	switch {
	case r >= 0x3000 && r <= 0x303F, r >= 0x3200 && r <= 0x33FF, r >= 0xFE30 && r <= 0xFE4F,
		r >= 0xFF00 && r <= 0xFFEF, r == 0x30FB, r == 0x30FC:
		return true
	}
	return false
}
