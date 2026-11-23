package css

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// FontFace is an @font-face rule: a family's face for a weight, or the
// weights from Weight to WeightMax a variable font covers, and a style,
// drawing the characters of Ranges, all of them when it has none. Src
// is the first of Sources ui reads that names a file: a path in the
// sheet's file system, or an address when the sheet has one (From).
type FontFace struct {
	Family  string
	Src     string
	Sources []FontSource
	Weight  int
	// WeightMax is Weight unless font-weight gives two.
	WeightMax int
	// AutoWeight is a face whose font-weight is auto, as one that gives
	// none: normal, or a variable font's own weights.
	AutoWeight bool
	Italic     bool
	// Display is font-display's word, auto without one.
	Display string
	Ranges  []UnicodeRange
	// Stretch is the width the face is of, as a percentage of the
	// normal one, or the widths from Stretch to StretchMax a variable
	// font covers.
	Stretch, StretchMax float32
	// AutoStretch is a face whose font-stretch is auto, as one that gives
	// none: normal, or a variable font's own widths.
	AutoStretch bool
	// SizeAdjust scales the face's glyphs and metrics, 1 without
	// size-adjust.
	SizeAdjust float32
	// Ascent, Descent and LineGap are the face's metrics as fractions of
	// the font size that override its own, negative for its own.
	Ascent, Descent, LineGap float32
	// Features are font-feature-settings' features, in order.
	Features []FontFeature
	// From is the address of the sheet the rule is in, "" for a sheet
	// that says none.
	From string
}

// FontSource is one entry of an @font-face's src list: a file at URL,
// resolved from the sheet as its url()s are, or the installed font
// Local names, with the format() and tech() the entry gives.
type FontSource struct {
	URL    string
	Local  string
	Format string
	Tech   []string
}

// FontFeature is an OpenType feature an @font-face turns on, off (0)
// or to one of its alternates.
type FontFeature struct {
	Tag   string
	Value int
}

// UnicodeRange is the characters from Lo to Hi.
type UnicodeRange struct{ Lo, Hi rune }

// FontFaces lists the sheet's @font-face rules in order.
func (s *Sheet) FontFaces() []FontFace { return s.faces }

// Covers reports whether the face draws r: every character without a
// unicode-range, else those of its ranges.
func (f FontFace) Covers(r rune) bool {
	for _, u := range f.Ranges {
		if r >= u.Lo && r <= u.Hi {
			return true
		}
	}
	return f.Ranges == nil
}

// Readable are the sources ui may read, in the order written.
func (f FontFace) Readable() []FontSource {
	var out []FontSource
	for _, s := range f.Sources {
		if s.Reads() {
			out = append(out, s)
		}
	}
	return out
}

// Reads reports whether ui reads the source: by its format() when it
// gives one (TrueType, OpenType, WOFF and WOFF2, a variable font drawn
// as its default instance), else by its file's extension, one that says
// nothing being read to see. A tech() of colour glyphs is not read, and
// a local() is.
func (s FontSource) Reads() bool {
	for _, tech := range s.Tech {
		if strings.HasPrefix(tech, "color-") || tech == "incremental" {
			return false
		}
	}
	if s.Local != "" {
		return true
	}
	if s.Format != "" {
		switch strings.TrimSuffix(s.Format, "-variations") {
		case "truetype", "opentype", "woff", "woff2":
			return true
		}
		return false
	}
	if strings.HasPrefix(s.URL, "data:") {
		return true
	}
	file := s.URL
	if i := strings.IndexAny(file, "?#"); i >= 0 {
		file = file[:i]
	}
	switch strings.ToLower(path.Ext(file)) {
	case ".eot", ".svg", ".svgz":
		return false
	}
	return true
}

// ignoredFaceDescriptors are the @font-face descriptors ui reads past:
// a browser's sheet may give them without a warning.
//
//nolint:gochecknoglobals // a lookup table
var ignoredFaceDescriptors = map[string]bool{
	"font-width": true, "font-variant": true,
	"font-variation-settings": true, "font-named-instance": true, "font-language-override": true,
	"subscript-position-override": true, "subscript-size-override": true,
	"superscript-position-override": true, "superscript-size-override": true,
}

func (p *parser) fontFace(pos Pos) (FontFace, bool) {
	block, ok := p.fontFaceBlock(pos)
	if !ok {
		return FontFace{}, false
	}
	f := FontFace{Weight: 400, WeightMax: 400, AutoWeight: true, Display: kwAuto, Stretch: 100, StretchMax: 100,
		AutoStretch: true, SizeAdjust: 1, Ascent: -1, Descent: -1, LineGap: -1}
	if p.origin != nil && p.origin.from != nil {
		f.From = p.origin.from.String()
	}
	for _, part := range splitOn(block, func(c Component) bool { return c.Kind == Semicolon }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		name := part[0]
		v := trim(part[1:])
		if name.Kind != Ident || len(v) < 2 || v[0].Kind != Colon {
			p.warnf(name.Pos, "expected a descriptor in @font-face")
			continue
		}
		raw := trim(v[1:])
		v = items(raw)
		if len(v) == 0 {
			p.warnf(name.Pos, "expected a descriptor in @font-face")
			continue
		}
		p.fontFaceDescriptor(&f, name, raw, v)
	}
	if f.Family == "" || len(f.Sources) == 0 {
		p.warnf(pos, "@font-face needs font-family and src; dropped")
		return FontFace{}, false
	}
	readable := f.Readable()
	if len(readable) == 0 {
		p.warnf(pos, "@font-face %q: no source ui reads (TrueType, OpenType, WOFF, WOFF2); dropped", f.Family)
		return FontFace{}, false
	}
	for _, s := range readable {
		if s.URL != "" {
			f.Src = s.URL
			break
		}
	}
	return f, true
}

// fontFaceBlock reads up to an @font-face's { block } and returns what
// it holds.
func (p *parser) fontFaceBlock(pos Pos) ([]Component, bool) {
	for {
		t := p.peek()
		if t.Kind == EOF {
			p.warnf(pos, "@font-face has no { block }")
			return nil, false
		}
		c := p.component()
		if c.Kind == LBrace {
			return c.Args, true
		}
		if c.Kind != Whitespace {
			p.warnf(c.Pos, "@font-face takes no prelude")
		}
	}
}

// fontFaceDescriptor reads one descriptor of an @font-face into f: raw
// is its value as written, v the value's items.
func (p *parser) fontFaceDescriptor(f *FontFace, name Component, raw, v []Component) {
	switch descriptor := strings.ToLower(name.Text); descriptor {
	case propFontFamily:
		if fam := familyName(v); fam != "" {
			f.Family = fam
		} else {
			p.warnf(v[0].Pos, "font-family: expected one family name")
		}
	case "src":
		if sources := p.fontSources(raw); len(sources) > 0 {
			f.Sources = sources
		} else {
			p.warnf(v[0].Pos, "src: expected url(file.ttf) or local(name)")
		}
	case propFontWeight:
		if lo, hi, ok := fontFaceWeight(v); ok {
			f.Weight, f.WeightMax, f.AutoWeight = lo, hi, len(v) == 1 && isWord(v[0], kwAuto)
		} else {
			p.warnf(v[0].Pos, "font-weight: expected normal, bold or 1 to 1000, once or twice")
		}
	case propFontStyle:
		if isWord(v[0], "normal", "italic", "oblique") && (len(v) == 1 || isWord(v[0], "oblique")) {
			f.Italic = !isWord(v[0], "normal")
		} else {
			p.warnf(v[0].Pos, "font-style: expected normal, italic or oblique")
		}
	case "font-display":
		if len(v) == 1 && isWord(v[0], "auto", "block", "swap", "fallback", "optional") {
			f.Display = strings.ToLower(v[0].Text)
		} else {
			p.warnf(v[0].Pos, "font-display: expected auto, block, swap, fallback or optional")
		}
	case "unicode-range":
		if ranges, ok := unicodeRanges(raw); ok {
			f.Ranges = ranges
		} else {
			p.warnf(v[0].Pos, "unicode-range: expected U+26, U+0-7F or U+4??, with commas between")
		}
	default:
		if !p.fontFaceLook(f, descriptor, raw, v) && (!p.browser || !ignoredFaceDescriptors[descriptor]) {
			p.warnf(name.Pos, "@font-face: %q is not supported", name.Text)
		}
	}
}

// fontFaceLook reads a descriptor of how the face is drawn into f, and
// reports whether descriptor is one.
func (p *parser) fontFaceLook(f *FontFace, descriptor string, raw, v []Component) bool {
	switch descriptor {
	case propFontStretch:
		if lo, hi, ok := fontFaceStretch(v); ok {
			f.Stretch, f.StretchMax, f.AutoStretch = lo, hi, len(v) == 1 && isWord(v[0], kwAuto)
		} else {
			p.warnf(v[0].Pos, "font-stretch: expected auto, normal, condensed… or a percentage, once or twice")
		}
	case "size-adjust":
		if len(v) == 1 && v[0].Kind == Percentage && v[0].Num >= 0 {
			f.SizeAdjust = float32(v[0].Num / 100)
		} else {
			p.warnf(v[0].Pos, "size-adjust: expected a percentage")
		}
	case "ascent-override", "descent-override", "line-gap-override":
		at := map[string]*float32{"ascent-override": &f.Ascent, "descent-override": &f.Descent,
			"line-gap-override": &f.LineGap}[descriptor]
		switch {
		case len(v) == 1 && isWord(v[0], kwNormal):
			*at = -1
		case len(v) == 1 && v[0].Kind == Percentage && v[0].Num >= 0:
			*at = float32(v[0].Num / 100)
		default:
			p.warnf(v[0].Pos, "%s: expected normal or a percentage", descriptor)
		}
	case "font-feature-settings":
		if features, ok := fontFeatures(raw); ok {
			f.Features = features
		} else {
			p.warnf(v[0].Pos, `font-feature-settings: expected normal, or "tag" with on, off or a number, `+
				"with commas between")
		}
	default:
		return false
	}
	return true
}

// fontSources reads a src descriptor's list. An entry that is no url()
// or local() is left out, as a browser leaves out what it can't read.
func (p *parser) fontSources(v []Component) []FontSource {
	var out []FontSource
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		part = items(part)
		if len(part) == 0 {
			continue
		}
		var s FontSource
		switch first := part[0]; {
		case first.Kind == Function && strings.EqualFold(first.Text, "local"):
			s.Local = familyName(items(first.Args))
		case urlTerm(first) == nil:
			s.URL = p.origin.resolve(imageSource(nil, part[:1]))
		}
		if s.URL == "" && s.Local == "" {
			continue
		}
		for _, c := range part[1:] {
			if c.Kind != Function {
				continue
			}
			args := items(c.Args)
			switch strings.ToLower(c.Text) {
			case "format":
				if len(args) > 0 && (args[0].Kind == String || args[0].Kind == Ident) {
					s.Format = strings.ToLower(args[0].Text)
				}
			case "tech":
				for _, a := range args {
					if a.Kind == Ident {
						s.Tech = append(s.Tech, strings.ToLower(a.Text))
					}
				}
			}
		}
		out = append(out, s)
	}
	return out
}

// fontFaceWeight is an @font-face's font-weight: one weight, or the two
// ends of a variable font's, the lighter first; auto is normal.
func fontFaceWeight(v []Component) (int, int, bool) {
	if len(v) == 1 && isWord(v[0], kwAuto) {
		return 400, 400, true
	}
	if len(v) < 1 || len(v) > 2 {
		return 0, 0, false
	}
	var w [2]int
	for i, c := range v {
		switch {
		case isWord(c, "normal"):
			w[i] = 400
		case isWord(c, "bold"):
			w[i] = 700
		case fontWeightNumber(c) == nil:
			w[i] = int(c.Num)
		default:
			return 0, 0, false
		}
	}
	if len(v) == 1 {
		return w[0], w[0], true
	}
	return min(w[0], w[1]), max(w[0], w[1]), true
}

// unicodeRanges reads a unicode-range descriptor. The tokens of CSS cut
// a range where a number or a name would end (U+0-7F is a name, a
// number and a dimension), so each range is read from its tokens'
// text, put back together.
func unicodeRanges(v []Component) ([]UnicodeRange, bool) {
	var out []UnicodeRange
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		var text strings.Builder
		for _, c := range trim(part) {
			switch c.Kind {
			case Ident, Delim, Number:
				text.WriteString(c.Text)
			case Dimension:
				text.WriteString(c.Text + c.Unit)
			case EOF, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Percentage, Whitespace, CDO, CDC, Colon,
				Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
				return nil, false
			}
		}
		u, ok := unicodeRange(text.String())
		if !ok {
			return nil, false
		}
		out = append(out, u)
	}
	return out, len(out) > 0
}

func unicodeRange(s string) (UnicodeRange, bool) {
	const last = 0x10ffff
	if len(s) < 3 || (s[0] != 'u' && s[0] != 'U') || s[1] != '+' {
		return UnicodeRange{}, false
	}
	from, to, ranged := strings.Cut(s[2:], "-")
	hex := func(s string) (rune, bool) {
		if len(s) < 1 || len(s) > 6 {
			return 0, false
		}
		v, err := strconv.ParseUint(s, 16, 32)
		return rune(v), err == nil && v <= last //nolint:gosec // at most six hex digits
	}
	if ranged {
		lo, ok := hex(from)
		hi, ok2 := hex(to)
		return UnicodeRange{lo, hi}, ok && ok2 && lo <= hi
	}
	digits := strings.TrimRight(from, "?")
	wild := len(from) - len(digits)
	if len(from) > 6 || strings.Contains(digits, "?") {
		return UnicodeRange{}, false
	}
	if wild == 0 {
		v, ok := hex(from)
		return UnicodeRange{v, v}, ok
	}
	lo, ok := hex(digits + strings.Repeat("0", wild))
	hi, ok2 := hex(digits + strings.Repeat("F", wild))
	if ok && !ok2 && lo <= last {
		hi, ok2 = last, true
	}
	return UnicodeRange{lo, hi}, ok && ok2
}

func familyName(v []Component) string {
	if len(v) == 1 && v[0].Kind == String {
		return v[0].Text
	}
	if len(v) == 0 || !allIdents(v) {
		return ""
	}
	var names []string
	for _, c := range v {
		names = append(names, c.Text)
	}
	return strings.Join(names, " ")
}

// stretches are font-stretch's words and the widths they stand for,
// as percentages of the normal width.
//
//nolint:gochecknoglobals // a lookup table
var stretches = map[string]float32{
	"ultra-condensed": 50, "extra-condensed": 62.5, "condensed": 75, "semi-condensed": 87.5, kwNormal: 100,
	"semi-expanded": 112.5, "expanded": 125, "extra-expanded": 150, "ultra-expanded": 200,
}

// fontStretch is a width font-stretch gives, as a percentage: a word or
// a percentage, not negative.
func fontStretch(c Component) (float32, bool) {
	if c.Kind == Ident {
		w, ok := stretches[strings.ToLower(c.Text)]
		return w, ok
	}
	if c.Kind == Percentage && c.Num >= 0 {
		return float32(c.Num), true
	}
	return 0, false
}

// fontSynthesisWords are the faces font-synthesis may let be made up.
//
//nolint:gochecknoglobals // a lookup table
var fontSynthesisWords = strings.Fields("weight style small-caps position")

func checkFontSynthesis(v []Component) error {
	if len(v) == 1 && isWord(v[0], "none") {
		return nil
	}
	seen := map[string]bool{}
	for _, c := range v {
		w := strings.ToLower(c.Text)
		if c.Kind != Ident || !slices.Contains(fontSynthesisWords, w) || seen[w] {
			return fmt.Errorf("font-synthesis: expected none, or weight, style, small-caps and position, each once")
		}
		seen[w] = true
	}
	return nil
}

// fontFaceStretch is an @font-face's font-stretch: one width, or the
// two ends of a variable font's, the narrower first; auto is normal.
func fontFaceStretch(v []Component) (float32, float32, bool) {
	if len(v) == 1 && isWord(v[0], kwAuto) {
		return 100, 100, true
	}
	if len(v) < 1 || len(v) > 2 {
		return 0, 0, false
	}
	var w [2]float32
	for i, c := range v {
		s, ok := fontStretch(c)
		if !ok {
			return 0, 0, false
		}
		w[i] = s
	}
	if len(v) == 1 {
		return w[0], w[0], true
	}
	return min(w[0], w[1]), max(w[0], w[1]), true
}

// fontFeatures reads font-feature-settings: normal, none at all, or a
// list of four-letter tags, each on (1) unless off, 0 or another number
// follows.
func fontFeatures(v []Component) ([]FontFeature, bool) {
	if all := items(v); len(all) == 1 && isWord(all[0], "normal") {
		return nil, true
	}
	var out []FontFeature
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		part = items(part)
		if len(part) == 0 || len(part) > 2 || part[0].Kind != String || len(part[0].Text) != 4 {
			return nil, false
		}
		f := FontFeature{Tag: part[0].Text, Value: 1}
		if len(part) == 2 {
			switch c := part[1]; {
			case isWord(c, "on"):
			case isWord(c, "off"):
				f.Value = 0
			case c.Kind == Number && c.Num >= 0 && c.Num == float64(int(c.Num)):
				f.Value = int(c.Num)
			default:
				return nil, false
			}
		}
		out = append(out, f)
	}
	return out, len(out) > 0
}

// fontVariations reads font-variation-settings: normal, none at all,
// or a list of four-letter tags each followed by a number, kept as
// Chrome computes it: one a tag, the last written, in the tags' order.
func fontVariations(v []Component) ([]style.FontVariation, bool) {
	if all := items(v); len(all) == 1 && isWord(all[0], kwNormal) {
		return nil, true
	}
	var out []style.FontVariation
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		part = items(part)
		if len(part) != 2 || part[0].Kind != String || len(part[0].Text) != 4 || part[1].Kind != Number {
			return nil, false
		}
		out = slices.DeleteFunc(out, func(f style.FontVariation) bool { return f.Tag == part[0].Text })
		out = append(out, style.FontVariation{Tag: part[0].Text, Value: float32(part[1].Num)})
	}
	slices.SortFunc(out, func(a, b style.FontVariation) int { return strings.Compare(a.Tag, b.Tag) })
	return out, len(out) > 0
}

// Feature is the value the face gives feature tag, and whether it
// gives one.
func (f FontFace) Feature(tag string) (int, bool) {
	value, found := 0, false
	for _, ft := range f.Features {
		if ft.Tag == tag {
			value, found = ft.Value, true
		}
	}
	return value, found
}
