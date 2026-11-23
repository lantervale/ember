package css

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

// ParseRules reads a stylesheet as Parse does, each of its top-level
// rules a sheet of its own, in order: a style rule, or an at-rule with
// what it holds. Text gives each its own text back, and Join makes one
// sheet of them again. It is for a sheet whose rules are added and
// removed one at a time, as a page's scripts do.
func ParseRules(name, src string, opts ...ParseOption) ([]*Sheet, []Warning) {
	p := parser{file: name, toks: Tokenize(src)}
	for _, o := range opts {
		o(&p)
	}
	from := ""
	if name != "" {
		p.origin = &origin{dir: path.Dir(name)}
	}
	if p.from != "" {
		if at, err := url.Parse(p.from); err == nil {
			p.origin, from = &origin{from: at}, p.from
		} else {
			p.warnf(Pos{1, 1}, "the sheet's address %q is no address: %v", p.from, err)
		}
	}
	at := cursor{src: preprocess(src), pos: Pos{1, 1}}
	var parts []*Sheet
	for {
		for k := p.peek().Kind; k == Whitespace || k == CDO || k == CDC; k = p.peek().Kind {
			p.next()
		}
		if p.peek().Kind == EOF {
			return parts, p.warns
		}
		start := at.offset(p.peek().Pos)
		p.sheet = &Sheet{Name: name, browser: p.browser, from: from, origin: p.origin}
		p.item(scope{})
		p.sheet.text = strings.TrimSpace(at.src[start:at.offset(p.peek().Pos)])
		parts = append(parts, p.sheet)
	}
}

// cursor finds where the tokens of src stand in it, asked in order.
type cursor struct {
	src string
	i   int
	pos Pos
}

// offset is where pos is in the text, the text's end for one past it.
func (c *cursor) offset(pos Pos) int {
	for c.i < len(c.src) && (c.pos.Line < pos.Line || c.pos.Line == pos.Line && c.pos.Col < pos.Col) {
		r, w := utf8.DecodeRuneInString(c.src[c.i:])
		c.i += w
		if r == '\n' {
			c.pos = Pos{c.pos.Line + 1, 1}
		} else {
			c.pos.Col++
		}
	}
	return c.i
}

// Text is the text of a sheet ParseRules read: its one rule, as it was
// written.
func (s *Sheet) Text() string { return s.text }

// nestedFrom is r in a sheet that holds its sheet's nests base places
// further on.
func (r Rule) nestedFrom(base int) Rule {
	if r.nest > 0 {
		r.nest += base
	}
	return r
}

// keyframesWithin is ks in a sheet that holds their media query lists
// base places further on, each inside the lists outer too.
func keyframesWithin(ks []Keyframes, outer []int, base int) []Keyframes {
	out := make([]Keyframes, len(ks))
	for i, k := range ks {
		k.media = append(slices.Clone(outer), k.media...)
		for j := len(outer); j < len(k.media); j++ {
			k.media[j] += base
		}
		out[i] = k
	}
	return out
}

// Join is sheets as one sheet, the rules of each after those of the one
// before, as if their texts followed each other in one file: a layer two
// of them name is one layer. media, when it says something, is a media
// query list every rule stands in, as the media attribute of a style
// element; what it gets wrong is the warnings. The sheet is read as a
// browser's, and from the address, of the first.
func Join(name, media string, sheets []*Sheet) (*Sheet, []Warning) {
	out := &Sheet{Name: name, joined: sheets}
	var warns []Warning
	var within []int
	if strings.TrimSpace(media) != "" {
		opts := []ParseOption{}
		if len(sheets) > 0 && sheets[0].browser {
			opts = append(opts, Browser())
		}
		var asked *Sheet
		asked, warns = Parse(name, "@media "+media+" {}", opts...)
		if len(asked.media) > 0 {
			out.media, within = append(out.media, asked.media[0]), []int{0}
			out.within = asked.media[0].text
		}
	}
	for at, s := range sheets {
		if at == 0 {
			out.browser, out.from, out.origin = s.browser, s.from, s.origin
		}
		named := len(out.layers)
		layers := make([]int, len(s.layers))
		for i, full := range s.layers {
			full = strings.ReplaceAll(full, anonymous, fmt.Sprint(anonymous, at, "-"))
			layers[i] = slices.Index(out.layers, full) + 1
			if layers[i] == 0 {
				out.layers = append(out.layers, full)
				layers[i] = len(out.layers)
			}
		}
		base, nests := len(out.media), len(out.nests)
		out.media, out.nests = append(out.media, s.media...), append(out.nests, s.nests...)
		for _, imp := range s.imports {
			media := slices.Clone(within)
			for _, m := range imp.media {
				media = append(media, base+m)
			}
			imp.media = media
			if imp.layer > 0 {
				imp.layer = layers[imp.layer-1]
			}
			if imp.layers > 0 {
				imp.layers = slices.Max(layers[:imp.layers])
			} else {
				imp.layers = named
			}
			imp.rules += len(out.Rules)
			out.imports = append(out.imports, imp)
		}
		for _, r := range s.Rules {
			if len(r.media) > 0 || within != nil {
				moved := make([]int, 0, len(within)+len(r.media))
				moved = append(moved, within...)
				for _, m := range r.media {
					moved = append(moved, base+m)
				}
				r.media = moved
			}
			if r.layer > 0 {
				r.layer = layers[r.layer-1]
			}
			out.Rules = append(out.Rules, r.nestedFrom(nests))
		}
		out.Keyframes = append(out.Keyframes, keyframesWithin(s.Keyframes, within, base)...)
		out.faces, out.palettes = append(out.faces, s.faces...), append(out.palettes, s.palettes...)
		out.CounterStyles, out.PositionTries = append(out.CounterStyles,
			counterStylesWithin(s.CounterStyles, within, base)...), append(out.PositionTries, s.PositionTries...)
	}
	return out, warns
}
