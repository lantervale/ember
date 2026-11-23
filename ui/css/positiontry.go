package css

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// PositionTry is an @position-try rule: the insets, margins, sizes,
// self-alignment, position-anchor and position-area a fallback of
// position-try-fallbacks naming it gives the element.
type PositionTry struct {
	Name  string
	Decls []Declaration
	Pos   Pos
}

//nolint:gochecknoglobals // the longhands an @position-try rule may set
var tryLonghands = map[string]bool{
	"top": true, "right": true, "bottom": true, "left": true,
	"margin-top": true, "margin-right": true, "margin-bottom": true, "margin-left": true,
	"width": true, "height": true, "min-width": true, "min-height": true, "max-width": true, "max-height": true,
	"align-self": true, "justify-self": true, "position-anchor": true, "position-area": true,
}

// positionTry reads an @position-try rule: a dashed name, then a block
// of the declarations it may set, without !important, as Chrome reads it.
func (p *parser) positionTry(t Token) {
	prelude, block := p.prelude()
	if !block {
		p.warnf(t.Pos, "@position-try has no { block }")
		return
	}
	name := items(prelude)
	var body []Component
	for kind := p.peek().Kind; kind != EOF && kind != RBrace; kind = p.peek().Kind {
		body = append(body, p.component())
	}
	if p.peek().Kind == RBrace {
		p.next()
	}
	if len(name) != 1 || !dashed(name[0]) {
		p.warnf(t.Pos, "@position-try: %q is no --name; the rule is dropped", strings.TrimSpace(join(prelude)))
		return
	}
	pt := PositionTry{Name: name[0].Text, Pos: t.Pos}
	for _, d := range p.declarations(body) {
		switch {
		case d.Important:
			p.warnf(d.Pos, "!important in @position-try drops %s, as in Chrome", d.Name)
		case !slices.ContainsFunc(longhandsOf(d.Name), func(l string) bool { return tryLonghands[l] }):
			p.warnf(d.Pos, "@position-try can't set %s", d.Name)
		default:
			pt.Decls = append(pt.Decls, d)
		}
	}
	p.sheet.PositionTries = append(p.sheet.PositionTries, pt)
}

// positionTryNamed is the last @position-try rule named name.
func (r *restyler) positionTryNamed(name string) *PositionTry {
	var found *PositionTry
	for _, s := range r.sheets {
		for i := range s.PositionTries {
			if s.PositionTries[i].Name == name {
				found = &s.PositionTries[i]
			}
		}
	}
	return found
}

// tries works out the style each of c's position-try-fallbacks gives the
// element: casc with the @position-try rule's declarations over those
// no !important one beats, then flipped as its tactics say, or c with
// the position-area it names. A fallback naming no rule is left out.
func (r *restyler) tries(x *ctx, c *style.Computed, casc cascaded, root bool) {
	if len(c.Anchoring.Fallbacks) == 0 || !outOfFlow(c.Position) {
		return
	}
	var out []*style.Computed
	for _, f := range c.Anchoring.Fallbacks {
		if f.Area.Set {
			t := *c
			t.Anchoring.Area, t.Anchoring.Tries = f.Area, nil
			out = append(out, &t)
			continue
		}
		long := slices.Clone(casc.longhands)
		if f.Name != "" {
			rule := r.positionTryNamed(f.Name)
			if rule == nil {
				continue
			}
			for i := range rule.Decls {
				d := &rule.Decls[i]
				for _, name := range longhandsOf(d.Name) {
					if casc.rtl && d.inline {
						name = mirrored(name)
					}
					if w := long[orderIndex[name]]; w == nil || !w.Important {
						long[orderIndex[name]] = d
					}
				}
			}
		}
		y := *x
		t := initial
		y.c = &t
		y.transit, y.over = nil, nil
		y.customs(casc.custom)
		for _, flip := range f.Flips {
			long = y.flipped(long, flip)
		}
		y.cascaded(cascaded{longhands: long, custom: casc.custom, rtl: casc.rtl}, root)
		t.Anchoring.Tries = nil
		out = append(out, &t)
	}
	c.Anchoring.Tries = out
}

func outOfFlow(p style.Position) bool { return p == style.Absolute || p == style.FixedPosition }

// flipPairs are the longhands a try tactic swaps.
//
//nolint:gochecknoglobals // the tactics' swaps
var flipPairs = map[style.Flip][][2]string{
	style.FlipBlock:  {{"top", "bottom"}, {"margin-top", "margin-bottom"}, {"align-self", "align-self"}},
	style.FlipInline: {{"left", "right"}, {"margin-left", "margin-right"}, {"justify-self", "justify-self"}},
	style.FlipStart: {{"top", "left"}, {"bottom", "right"}, {"margin-top", "margin-left"},
		{"margin-bottom", "margin-right"}, {"width", "height"}, {"min-width", "min-height"},
		{"max-width", "max-height"}, {"align-self", "justify-self"}},
}

// flipWords are the keywords a try tactic turns in the values it moves:
// anchor() sides, anchor-size() sizes, self-alignment and position-area.
//
//nolint:gochecknoglobals // the tactics' keywords
var flipWords = map[style.Flip]map[string]string{
	style.FlipBlock: swaps("top", "bottom", "start", "end", "self-start", "self-end", "flex-start", "flex-end",
		"span-top", "span-bottom", "y-start", "y-end", "span-y-start", "span-y-end", "block-start", "block-end",
		"span-block-start", "span-block-end", "self-block-start", "self-block-end", "span-self-block-start",
		"span-self-block-end", "self-y-start", "self-y-end", "span-self-y-start", "span-self-y-end"),
	style.FlipInline: swaps("left", "right", "start", "end", "self-start", "self-end", "flex-start", "flex-end",
		"span-left", "span-right", "x-start", "x-end", "span-x-start", "span-x-end", "inline-start", "inline-end",
		"span-inline-start", "span-inline-end", "self-inline-start", "self-inline-end", "span-self-inline-start",
		"span-self-inline-end", "self-x-start", "self-x-end", "span-self-x-start", "span-self-x-end"),
	style.FlipStart: swaps("top", "left", "bottom", "right", "width", "height", "block", "inline",
		"self-block", "self-inline", "span-top", "span-left", "span-bottom", "span-right", "y-start", "x-start",
		"y-end", "x-end", "span-y-start", "span-x-start", "span-y-end", "span-x-end", "block-start",
		"inline-start", "block-end", "inline-end", "span-block-start", "span-inline-start", "span-block-end",
		"span-inline-end"),
}

func swaps(ws ...string) map[string]string {
	m := make(map[string]string, len(ws))
	for i := 0; i+1 < len(ws); i += 2 {
		m[ws[i]], m[ws[i+1]] = ws[i+1], ws[i]
	}
	return m
}

// flipped is long with the declarations flip swaps moved to their
// pair's place, their values turned, as a try tactic flips them.
func (x *ctx) flipped(long []*Declaration, flip style.Flip) []*Declaration {
	out := slices.Clone(long)
	area := orderIndex["position-area"]
	if d := long[area]; d != nil {
		out[area] = x.turned("position-area", "position-area", d, flip)
	}
	for _, pair := range flipPairs[flip] {
		a, b := orderIndex[pair[0]], orderIndex[pair[1]]
		out[a], out[b] = x.turned(pair[1], pair[0], long[b], flip), x.turned(pair[0], pair[1], long[a], flip)
	}
	return out
}

// turned is d's value for the longhand from as a declaration of to, its
// keywords turned by flip; nil for nil.
func (x *ctx) turned(from, to string, d *Declaration, flip style.Flip) *Declaration {
	if d == nil {
		return nil
	}
	v, ok := x.valueOf(from, d)
	if !ok {
		return d
	}
	words := flipWords[flip]
	return &Declaration{Name: to, Value: v, vals: turnWords(items(v), words, flip), Important: d.Important,
		Pos: d.Pos, base: d.base}
}

// turnWords is v with each keyword words pairs swapped for its pair, in
// functions too, and anchor() percentages taken from the other end.
func turnWords(v []Component, words map[string]string, flip style.Flip) []Component {
	out := make([]Component, len(v))
	for i, c := range v {
		switch {
		case c.Kind == Ident:
			if w, ok := words[strings.ToLower(c.Text)]; ok {
				c.Text = w
			}
		case c.Kind == Function:
			anchor := strings.EqualFold(c.Text, "anchor")
			c.Args = turnWords(c.Args, words, flip)
			if anchor && flip != style.FlipStart {
				for j, a := range c.Args {
					if a.Kind == Comma {
						break
					}
					if a.Kind == Percentage {
						c.Args[j].Num = 100 - a.Num
					}
				}
			}
		}
		out[i] = c
	}
	return out
}
