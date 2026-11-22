package css

import (
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// Origin is where the declarations of a Match come from.
type Origin uint8

const (
	// Author is a rule of one of the Document's sheets.
	Author Origin = iota
	// Agent is a rule of ember's own sheet.
	Agent
	// Inline is the element's style attribute.
	Inline
	// Hint is what the element's presentational attributes stand for, a
	// picture's width and height.
	Hint
)

// Match is one source of declarations that apply to an element: a rule
// one of whose selectors matches it, its style attribute, or its
// presentational attributes.
type Match struct {
	Origin Origin
	// Sheet is the rule's sheet, the one an @import brought it from for
	// an imported rule, nil for what is no rule.
	Sheet *Sheet
	// Selector is the rule's selectors as CSS, "" for what is no rule.
	Selector string
	// Pos is where the rule opens in its sheet.
	Pos Pos
	// Media are the @media rules the rule stands in, the outermost
	// first, and Layer its cascade layer's full name, "" in none or in
	// one without a name.
	Media []string
	Layer string
	// Containers are the @container rules it stands in, the outermost
	// first, as written.
	Containers []string
	// Declarations are what it declares, in the order written.
	Declarations []MatchedDeclaration
}

// MatchedDeclaration is one declaration of a Match.
type MatchedDeclaration struct {
	Name string
	// Value is the value as written, a var() not replaced.
	Value     string
	Important bool
	Pos       Pos
	// Overridden is set when a stronger declaration sets every property
	// this one does.
	Overridden bool
}

// Property is one property of an element after the cascade.
type Property struct {
	Name string
	// Value is the winning declaration's value for the property, a
	// shorthand's part of it, with every var() replaced and its lengths
	// in px; the parent's where it inherits, the initial value otherwise.
	// Keywords, colours, percentages and functions stay as written.
	Value string
	// Declared is set when a declaration gave the value, on the element
	// or on the ancestor it inherits it from.
	Declared bool
}

// found is one declaration that applies, with the rule it is from.
type found struct {
	applied
	sheet *Sheet
	rule  int
	hint  bool
}

// inspection is the cascade of one element, worked out again off the
// frame's own.
type inspection struct {
	t      *Transitions
	m      matcher
	inline []Declaration
}

// ready reports whether n is an element the last Compute styled.
func (t *Transitions) ready(n *dom.Node) bool {
	return n != nil && !n.IsText() && !n.Generated() && t.all != nil && dom.Data(n).Style != nil
}

func (t *Transitions) inspect(n *dom.Node) *inspection {
	in := &inspection{t: t, m: matcher{off: t.off, ranks: t.ranks, env: &t.env, best: map[int]Specificity{}}}
	if src := n.Style(); src != "" {
		p := parser{toks: Tokenize(src), browser: t.browser}
		in.inline, _ = p.inline()
	}
	in.m.hints = hints(n)
	return in
}

// found lists the declarations that apply to n as matcher.matched does,
// least important first, each with its rule.
func (in *inspection) found(n *dom.Node) []found {
	var out []found
	order := 0
	add := func(f found) {
		f.order = order
		out = append(out, f)
		order++
	}
	m := &in.m
	for i := range m.hints {
		add(found{applied: applied{decl: &m.hints[i], level: authorNormal, layer: hintLayer}, hint: true, rule: -1})
	}
	for si, s := range in.t.all {
		x := &s.indexed().buckets
		clear(m.best)
		var off []bool
		if m.off != nil {
			off = m.off[si]
		}
		if id := n.ID(); id != "" {
			m.try(s, x.byID[id], n, off)
		}
		for _, c := range n.Classes() {
			m.try(s, x.byClass[c], n, off)
		}
		m.try(s, x.byTag[n.Tag()], n, off)
		m.try(s, x.any, n, off)
		for _, r := range slices.Sorted(mapKeys(m.best)) {
			var layer int32
			if l := s.Rules[r].layer; l > 0 && m.ranks != nil {
				layer = m.ranks[si][l-1]
			}
			for di := range s.Rules[r].Decls {
				d := &s.Rules[r].Decls[di]
				add(found{applied: applied{decl: d, level: levelOf(si == 0, d.Important), layer: layer, spec: m.best[r]},
					sheet: s, rule: r})
			}
		}
	}
	for i := range in.inline {
		d := &in.inline[i]
		add(found{applied: applied{decl: d, level: levelOf(false, d.Important), inline: true}, rule: -1})
	}
	slices.SortStableFunc(out, func(a, b found) int {
		switch {
		case a.less(b.applied):
			return -1
		case b.less(a.applied):
			return 1
		}
		return 0
	})
	return out
}

// hintLayer is the layer presentational hints stand in: under every
// other.
const hintLayer = int32(1<<31 - 1)

func mapKeys(m map[int]Specificity) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// sets are the properties d sets: its longhands, or itself.
func sets(d *Declaration) []string {
	if long, ok := shorthands[d.Name]; ok {
		return long
	}
	return []string{d.Name}
}

// Matched is what applies to the element n as the last Compute found
// it, the strongest first: each rule of the sheets one of whose
// selectors matches it and whose @media rules hold, its style attribute
// and its presentational attributes, with the declarations a stronger
// one overrides marked. A rule's place is its strongest declaration's.
// It is empty for a text node and for an element Compute has not styled.
// Rules of pseudo-elements are left out.
func (t *Transitions) Matched(n *dom.Node) []Match {
	if !t.ready(n) {
		return nil
	}
	list := t.inspect(n).found(n)
	winner := map[string]*Declaration{}
	for _, f := range list {
		for _, name := range sets(f.decl) {
			winner[name] = f.decl
		}
	}
	type key struct {
		sheet        *Sheet
		rule         int
		inline, hint bool
	}
	at := map[key]int{}
	var out []Match
	for _, f := range slices.Backward(list) {
		k := key{f.sheet, f.rule, f.inline, f.hint}
		if _, seen := at[k]; seen {
			continue
		}
		at[k] = len(out)
		out = append(out, t.match(f))
	}
	for _, f := range list {
		m := &out[at[key{f.sheet, f.rule, f.inline, f.hint}]]
		d := f.decl
		m.Declarations = append(m.Declarations, MatchedDeclaration{
			Name: d.Name, Value: join(d.Value), Important: d.Important, Pos: d.Pos,
			Overridden: !slices.ContainsFunc(sets(d), func(name string) bool { return winner[name] == d }),
		})
	}
	for i := range out {
		slices.SortStableFunc(out[i].Declarations, func(a, b MatchedDeclaration) int {
			if a.Pos.Line != b.Pos.Line {
				return a.Pos.Line - b.Pos.Line
			}
			return a.Pos.Col - b.Pos.Col
		})
	}
	return out
}

// match is the Match f's declaration is of, without its declarations.
func (t *Transitions) match(f found) Match {
	switch {
	case f.inline:
		return Match{Origin: Inline}
	case f.hint:
		return Match{Origin: Hint}
	}
	s, r := f.sheet, &f.sheet.Rules[f.rule]
	m := Match{Sheet: s, Pos: r.Pos}
	if r.in != nil {
		m.Sheet = r.in
	}
	if s == t.all[0] {
		m.Origin = Agent
	}
	selectors := make([]string, len(r.Selectors))
	for i, sel := range r.Selectors {
		selectors[i] = sel.String()
	}
	m.Selector = strings.Join(selectors, ", ")
	for _, i := range r.media {
		m.Media = append(m.Media, s.media[i].text)
	}
	for _, c := range r.containers {
		m.Containers = append(m.Containers, c.text)
	}
	if r.layer > 0 && !strings.Contains(s.layers[r.layer-1], anonymous) {
		m.Layer = s.layers[r.layer-1]
	}
	return m
}

// longhandNames are the longhands by name.
//
//nolint:gochecknoglobals // derived from the properties
var longhandNames = slices.Sorted(func(yield func(string) bool) {
	for name := range properties {
		if !yield(name) {
			return
		}
	}
})

// Computed is every longhand of the element n after the cascade, by
// name, then its custom properties: see Property for what a value is.
// It is empty for a text node and for an element Compute has not styled.
func (t *Transitions) Computed(n *dom.Node) []Property {
	if !t.ready(n) {
		return nil
	}
	values := t.cascaded(n, longhandNames)
	out := make([]Property, 0, len(longhandNames))
	for _, name := range longhandNames {
		out = append(out, Property{Name: name, Value: values[name].text, Declared: values[name].declared})
	}
	vars := dom.Data(n).Style.Vars
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for name := range vars {
			if !yield(name) {
				return
			}
		}
	}) {
		out = append(out, Property{Name: name, Value: vars[name], Declared: true})
	}
	return out
}

// ComputedValue is the one property name of Computed, a longhand or a
// custom property; false for another name and where Computed is empty.
// It costs the cascade of n alone unless the property is inherited.
func (t *Transitions) ComputedValue(n *dom.Node, name string) (string, bool) {
	if !t.ready(n) {
		return "", false
	}
	if strings.HasPrefix(name, "--") {
		v, ok := dom.Data(n).Style.Vars[name]
		return v, ok
	}
	if _, ok := properties[name]; !ok {
		return "", false
	}
	return t.cascaded(n, []string{name})[name].text, true
}

type cascadedValue struct {
	text     string
	declared bool
}

// cascaded is the value of each of names on n: its own where a
// declaration of n gives one, its parent's where it inherits.
func (t *Transitions) cascaded(n *dom.Node, names []string) map[string]cascadedValue {
	in := t.inspect(n)
	casc := in.m.cascade(in.m.matched(n, t.all, in.inline))
	parent := &initial
	above := n.Parent()
	if above != nil && dom.Data(above).Style != nil {
		parent = dom.Data(above).Style
	} else {
		above = nil
	}
	root := n
	for root.Parent() != nil && dom.Data(root.Parent()).Style != nil {
		root = root.Parent()
	}
	c := *dom.Data(n).Style
	x := &ctx{c: &c, parent: parent, em: parent.Font.Size, rem: dom.Data(root).Style.Font.Size, env: &t.env,
		root: root == n, node: n}
	x.customs(casc.custom)
	x.em, x.sized, x.lined = c.Font.Size, true, true

	out := make(map[string]cascadedValue, len(names))
	var inherited []string
	for _, name := range names {
		d := casc.longhands[orderIndex[name]]
		v, kw := x.declaredValue(name, d)
		switch {
		case kw == kwInherit, kw == kwUnset && properties[name].inherited:
			inherited = append(inherited, name)
		case kw != "":
			out[name] = cascadedValue{text: properties[name].initial}
		case name == propFontSize:
			out[name] = cascadedValue{text: pixels(float64(c.Font.Size)), declared: true}
		default:
			out[name] = cascadedValue{text: x.written(v), declared: true}
		}
	}
	switch {
	case len(inherited) == 0:
	case above == nil:
		for _, name := range inherited {
			out[name] = cascadedValue{text: properties[name].initial}
		}
	default:
		from := t.cascaded(above, inherited)
		for _, name := range inherited {
			out[name] = from[name]
		}
	}
	return out
}

// declaredValue is the value d gives the longhand name, as ctx.apply
// reads it, or the global keyword that stands in its place: unset for
// no declaration and for a value that cannot be read.
func (x *ctx) declaredValue(name string, d *Declaration) ([]Component, string) {
	if d == nil {
		return nil, kwUnset
	}
	var v []Component
	ok := true
	switch {
	case d.longhands != nil:
		v = d.longhands[name]
	case d.Name != name:
		var long map[string][]Component
		if sub, subOK := x.substitute(d.Value); subOK {
			long, _ = expand(d.Name, items(sub))
		}
		v, ok = long[name], long != nil
	case d.vals != nil:
		v = d.vals
	default:
		v, ok = x.substitute(d.Value)
	}
	if !ok || len(items(v)) == 0 {
		return nil, kwUnset
	}
	v = items(v)
	if kw := globalKeyword(v); kw != "" {
		return nil, kw
	}
	if hasVar(d.Value) {
		if chk := checks[name]; chk != nil && chk(v) != nil {
			return nil, kwUnset
		}
	}
	return v, ""
}

// written is v as CSS, its lengths in px.
func (x *ctx) written(v []Component) string {
	var b strings.Builder
	for i, c := range v {
		if i > 0 && c.Kind != Comma {
			b.WriteByte(' ')
		}
		if c.Kind == Dimension {
			if d, err := parseDim(c); err == nil && !d.IsFlex() {
				b.WriteString(pixels(d.Absolute(x.units(&d)).Px))
				continue
			}
		}
		b.WriteString(c.String())
	}
	return b.String()
}

func pixels(px float64) string {
	return strconv.FormatFloat(float64(float32(px*1000))/1000, 'f', -1, 32) + "px"
}
