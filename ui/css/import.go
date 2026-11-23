package css

import (
	"fmt"
	"slices"
	"strings"
)

// Import is an @import rule a browser follows: URL is the sheet it
// names, resolved as the sheet's url()s are (Sheet.URL).
type Import struct {
	URL string
	Pos Pos
	// media are the media query lists it stands in, as places in its
	// sheet's list: its own last; layer is its layer, as a place from 1
	// in its sheet's list, 0 for none.
	media []int
	layer int
	// layers and rules count its sheet's layers named and rules read
	// before it: where what it imports stands.
	layers, rules int
}

// Imports are the sheet's @import rules a browser follows, in order:
// those written before any rule but @charset and @layer statements,
// whose supports() holds. Imported puts the sheets they name in their
// places.
func (s *Sheet) Imports() []Import { return s.imports }

// Imported is the sheet with the sheet each of its imports names in
// its place, sheets[i] for Imports()[i]: its rules as if they stood
// where the @import is, under its media query list and in its layer,
// its layers named there, inside the import's own. A nil sheet, one not
// there (on its way, failed, a cycle), imports nothing. Each of sheets
// is one whose own imports are already in it.
func (s *Sheet) Imported(sheets []*Sheet) *Sheet {
	out := &Sheet{Name: s.Name, browser: s.browser, src: s.src, from: s.from, origin: s.origin, text: s.text,
		joined: s.joined, within: s.within, media: slices.Clone(s.media), base: s, nests: slices.Clone(s.nests)}
	own := make([]int, len(s.layers))
	named := 0
	nameOwn := func(upto int) {
		for ; named < upto; named++ {
			own[named] = out.declared(s.layers[named])
		}
	}
	type placed struct {
		imp   Import
		sheet *Sheet
		media int
		nests int
		as    []int
	}
	var put []placed
	for i, imp := range s.imports {
		nameOwn(imp.layers)
		if i >= len(sheets) || sheets[i] == nil { //nolint:gosec // i is checked against sheets first
			continue
		}
		in := sheets[i] //nolint:gosec // i is checked against sheets above
		prefix := ""
		if imp.layer > 0 {
			prefix = s.layers[imp.layer-1] + "."
		}
		as := make([]int, len(in.layers))
		for k, full := range in.layers {
			as[k] = out.declared(prefix + strings.ReplaceAll(full, anonymous, fmt.Sprint(anonymous, "import", i, "-")))
		}
		put = append(put, placed{imp: imp, sheet: in, media: len(out.media), nests: len(out.nests), as: as})
		out.faces, out.Keyframes, out.nests = append(out.faces, in.faces...),
			append(out.Keyframes, keyframesWithin(in.Keyframes, imp.media, len(out.media))...), append(out.nests, in.nests...)
		out.CounterStyles, out.media = append(out.CounterStyles,
			counterStylesWithin(in.CounterStyles, imp.media, len(out.media))...), append(out.media, in.media...)
	}
	nameOwn(len(s.layers))
	ownRule := func(r Rule) Rule {
		if r.layer > 0 {
			r.layer = own[r.layer-1]
		}
		return r
	}
	read := 0
	for _, p := range put {
		for ; read < p.imp.rules; read++ {
			out.Rules = append(out.Rules, ownRule(s.Rules[read]))
		}
		for _, r := range p.sheet.Rules {
			media := make([]int, 0, len(p.imp.media)+len(r.media))
			media = append(media, p.imp.media...)
			for _, m := range r.media {
				media = append(media, p.media+m)
			}
			r.media = media
			if r.in == nil {
				r.in = p.sheet
			}
			switch {
			case r.layer > 0:
				r.layer = p.as[r.layer-1]
			case p.imp.layer > 0:
				r.layer = own[p.imp.layer-1]
			}
			out.Rules = append(out.Rules, r.nestedFrom(p.nests))
		}
		out.spliced = append(out.spliced, splice{imp: p.imp, sheet: p.sheet})
	}
	for ; read < len(s.Rules); read++ {
		out.Rules = append(out.Rules, ownRule(s.Rules[read]))
	}
	out.faces, out.Keyframes, out.CounterStyles = append(out.faces, s.faces...), append(out.Keyframes, s.Keyframes...),
		append(out.CounterStyles, s.CounterStyles...)
	return out
}

// splice is a sheet Imported put where its import stands.
type splice struct {
	imp   Import
	sheet *Sheet
}

// declared is the place from 1 of the layer full in the sheet's list,
// added last when the list has it not.
func (s *Sheet) declared(full string) int {
	if at := slices.Index(s.layers, full); at >= 0 {
		return at + 1
	}
	s.layers = append(s.layers, full)
	return len(s.layers)
}

// importedText is a sheet Imported made as a browser's sheet written
// out: its named layers in their order, each imported sheet's text in
// its import's layer and media, then its own text.
func (s *Sheet) importedText() string {
	var b strings.Builder
	var names []string
	for _, full := range s.layers {
		if !strings.Contains(full, anonymous) {
			names = append(names, full)
		}
	}
	if len(names) > 0 {
		b.WriteString("@layer " + strings.Join(names, ", ") + ";\n")
	}
	for _, sp := range s.spliced {
		var open, closing string
		if sp.imp.layer > 0 {
			name := s.base.layers[sp.imp.layer-1]
			if strings.Contains(name, anonymous) {
				name = ""
			} else {
				name += " "
			}
			open, closing = "@layer "+name+"{\n", "}\n"
		}
		for _, m := range sp.imp.media {
			open += "@media " + s.media[m].text + " {\n"
			closing += "}\n"
		}
		b.WriteString(open + strings.TrimSpace(sp.sheet.String()) + "\n" + closing)
	}
	b.WriteString(s.base.String())
	return b.String()
}

// importRule reads an @import rule: @import url [layer | layer(name)]
// [supports(condition)] [media query list].
func (p *parser) importRule(t Token) {
	prelude, block := p.prelude()
	switch {
	case block:
		p.warnf(t.Pos, "@import takes no { block }")
		p.skipBlock()
		return
	case p.depth > 0:
		p.warnf(t.Pos, "@import inside a block is ignored, as browsers ignore it")
		return
	case p.begun:
		p.warnf(t.Pos, "@import after other rules is ignored, as browsers ignore it")
		return
	}
	cs := trim(prelude)
	ref, ok := importedSheet(cs)
	if !ok {
		p.warnf(t.Pos, "@import names no sheet")
		return
	}
	rest := trim(cs[1:])
	layer, layered := "", len(rest) > 0 && isFunctionOrWord(rest[0], "layer")
	if layered {
		if !p.browser {
			p.warnf(t.Pos, "@import: layer() is read only as a browser reads a sheet")
			return
		}
		if rest[0].Kind == Function {
			if layer, ok = layerName(trim(rest[0].Args)); !ok {
				p.warnf(t.Pos, "@import: %q is no layer name", strings.TrimSpace(join(rest[0].Args)))
				return
			}
		}
		rest = trim(rest[1:])
	}
	if len(rest) > 0 && rest[0].Kind == Function && strings.EqualFold(rest[0].Text, atSupports) {
		if !p.browser {
			p.warnf(t.Pos, "@import: supports() is read only as a browser reads a sheet")
			return
		}
		if !supportedIn(Component{Token: Token{Kind: LParen}, Args: rest[0].Args}) {
			return
		}
		rest = trim(rest[1:])
	}
	imp := Import{URL: p.origin.resolve(ref), Pos: t.Pos}
	if layered {
		if layer == "" {
			layer = fmt.Sprint(anonymous, len(p.sheet.layers))
		}
		imp.layer = p.declare(0, layer)
	}
	if len(rest) > 0 {
		queries := mediaParser{browser: p.browser, warn: p.warnf}
		p.sheet.media = append(p.sheet.media, queries.list(rest))
		imp.media = []int{len(p.sheet.media) - 1}
	}
	imp.layers, imp.rules = len(p.sheet.layers), len(p.sheet.Rules)
	p.sheet.imports = append(p.sheet.imports, imp)
}

// importedSheet is the address an @import's prelude begins with: a url
// or a string.
func importedSheet(cs []Component) (string, bool) {
	if len(cs) == 0 {
		return "", false
	}
	switch c := cs[0]; {
	case c.Kind == URL || c.Kind == String:
		return c.Text, true
	case c.Kind == Function && strings.EqualFold(c.Text, "url"):
		if args := items(c.Args); len(args) == 1 && args[0].Kind == String {
			return args[0].Text, true
		}
	}
	return "", false
}

// isFunctionOrWord reports whether c is the word name, or name( … ).
func isFunctionOrWord(c Component, name string) bool {
	return isWord(c, name) || c.Kind == Function && strings.EqualFold(c.Text, name)
}
