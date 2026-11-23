package css

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// sizeAxes are the axes a container answers size queries on, or those a
// query's size features read.
type sizeAxes uint8

const (
	inlineAxis sizeAxes = 1 << iota
	blockAxis
	bothAxes = inlineAxis | blockAxis
)

// axesOf are the axes a container of the type answers on.
func axesOf(t style.ContainerType) sizeAxes {
	switch t {
	case style.ContainerInlineSize:
		return inlineAxis
	case style.ContainerSize:
		return bothAxes
	case style.ContainerNormal:
	}
	return 0
}

// containerSpecs are the size features a container answers.
//
//nolint:gochecknoglobals,goconst // the container features' table
var containerSpecs = map[string]*mediaSpec{
	"width":        {axes: inlineAxis, kind: mediaLength, of: func(m Media) float64 { return float64(m.Width) }},
	"height":       {axes: blockAxis, kind: mediaLength, of: func(m Media) float64 { return float64(m.Height) }},
	"inline-size":  {axes: inlineAxis, kind: mediaLength, of: func(m Media) float64 { return float64(m.Width) }},
	"block-size":   {axes: blockAxis, kind: mediaLength, of: func(m Media) float64 { return float64(m.Height) }},
	"aspect-ratio": {axes: bothAxes, kind: mediaRatio, of: viewportRatio},
	"orientation":  mediaSpecs["orientation"].on(bothAxes),
}

func (s *mediaSpec) on(axes sizeAxes) *mediaSpec {
	c := *s
	c.axes = axes
	return &c
}

// containerRule is an @container rule's prelude: conditions one of which
// must hold, as written.
type containerRule struct {
	text  string
	conds []containerCond
	// viewport is set when one of its lengths is in a unit the viewport
	// measures.
	viewport bool
}

// containerCond is one condition: the container it asks, the nearest
// above the element that has its name, if it gives one, and answers its
// size features; and its query, nil for the name alone.
type containerCond struct {
	name  string
	query *mediaCond
	axes  sizeAxes
	// scroll is set when the query asks scroll-state(), which only a
	// scroll-state container answers.
	scroll bool
}

// holds reports whether one of r's conditions holds for e, or for its
// pseudo-element when self, whose container may be e itself.
func (r *containerRule) holds(e *dom.Node, self bool, env *measures) bool {
	for i := range r.conds {
		if r.conds[i].eval(e, self, env) == yes {
			return true
		}
	}
	return false
}

func (c *containerCond) eval(e *dom.Node, self bool, env *measures) tri {
	a := e
	if !self && a != nil {
		a = a.Parent()
	}
	for ; a != nil; a = a.Parent() {
		st := dom.Data(a).Style
		if st == nil || !c.picks(st) {
			continue
		}
		if c.query == nil {
			return yes
		}
		return c.query.eval(env.subjectOf(a))
	}
	return unknown
}

func (c *containerCond) picks(st *style.Computed) bool {
	if c.name != "" && !slices.Contains(st.ContainerNames, c.name) {
		return false
	}
	return c.axes&^axesOf(st.ContainerType) == 0 && (!c.scroll || st.ContainerScrollState)
}

// axes are the axes c's size features read.
func (c *mediaCond) axes() sizeAxes {
	var a sizeAxes
	if c.spec != nil {
		a = c.spec.axes
	}
	for i := range c.kids {
		a |= c.kids[i].axes()
	}
	return a
}

// containerRule reads an @container rule's prelude: a list of
// conditions, each a name, a query or a name then a query. A malformed
// one drops the rule.
func (p *mediaParser) containerRule(prelude []Component) (*containerRule, bool) {
	r := &containerRule{text: strings.TrimSpace(join(prelude))}
	for _, part := range splitOn(prelude, func(c Component) bool { return c.Kind == Comma }) {
		toks := items(part)
		var c containerCond
		if len(toks) > 0 && toks[0].Kind == Ident && !isWord(toks[0], "not") {
			if isWord(toks[0], kwNone, "and", "or", "default") {
				return nil, false
			}
			c.name, toks = toks[0].Text, toks[1:]
		}
		if len(toks) == 0 && c.name == "" {
			return nil, false
		}
		if len(toks) > 0 {
			p.standardStyle = false
			cond, ok := p.condition(toks, true)
			if !ok {
				return nil, false
			}
			if p.standardStyle {
				cond = &mediaCond{op: mediaOr}
			}
			c.query, c.axes, c.scroll = cond, cond.axes(), cond.readsScrollState()
			r.viewport = r.viewport || cond.readsViewport()
		}
		r.conds = append(r.conds, c)
	}
	return r, true
}

func (c *mediaCond) readsViewport() bool {
	for _, b := range c.bounds {
		if b.value.dim != nil && b.value.dim.rel&viewportUnits != 0 {
			return true
		}
	}
	return slices.ContainsFunc(c.kids, func(k mediaCond) bool { return k.readsViewport() })
}

// containerLengthOf reads a container query's length: a zero, or a
// length in any unit, calc() of them included.
func containerLengthOf(v []Component) (mediaValue, bool) {
	if len(v) != 1 {
		return mediaValue{}, false
	}
	if v[0].Kind == Number && v[0].Num == 0 {
		return mediaValue{}, true
	}
	d, err := parseDim(v[0])
	if err != nil || d.typ != typeLength || d.pct {
		return mediaValue{}, false
	}
	if d.Em == 0 && d.Rem == 0 && d.rel == 0 && len(d.terms) == 0 {
		return mediaValue{num: d.Px}, true
	}
	return mediaValue{dim: &d}, true
}

// styleQuery reads style(): one custom property, or conditions on
// several in parentheses.
func (p *mediaParser) styleQuery(fn Component) mediaCond {
	inner := items(fn.Args)
	if len(inner) > 0 && (inner[0].Kind == LParen || isWord(inner[0], "not")) {
		p.inStyle = true
		cond, ok := p.condition(inner, true)
		p.inStyle = false
		if !ok {
			p.warn(fn.Pos, "@container: %s is no style query; its query never holds", fn.String())
			return mediaCond{op: mediaUnknown}
		}
		return *cond
	}
	return p.styleFeature(fn, fn.Args)
}

// styleFeature reads --name or --name: value.
func (p *mediaParser) styleFeature(at Component, args []Component) mediaCond {
	toks := trim(args)
	if len(toks) == 0 || toks[0].Kind != Ident || !strings.HasPrefix(toks[0].Text, "--") {
		p.standardStyle = true
		p.warn(at.Pos, "@container: style() reads custom properties alone, not %s; the condition never holds",
			strings.TrimSpace(join(args)))
		return mediaCond{op: mediaUnknown}
	}
	c := mediaCond{op: mediaStyle, name: toks[0].Text}
	rest := trim(toks[1:])
	if len(rest) == 0 {
		return c
	}
	if rest[0].Kind != Colon {
		p.warn(at.Pos, "@container: %s is no style query; its query never holds", at.String())
		return mediaCond{op: mediaUnknown}
	}
	c.word, c.valued = join(trim(rest[1:])), true
	return c
}

// subjectOf is what a query asks of the container a: its content box,
// on the axes it answers once laid out, and its custom properties.
func (m *measures) subjectOf(a *dom.Node) *subject {
	st := dom.Data(a).Style
	s := &subject{vars: st.Vars, of: a, env: m, rtl: st.Dir == style.RTL}
	if st.ContainerScrollState {
		scroll := m.scrollOf(a)
		s.scroll = &scroll
	}
	if size, ok := m.size(a); ok {
		s.m, s.sized = Media{Width: size[0], Height: size[1]}, axesOf(st.ContainerType)
	}
	return s
}

func (m *measures) size(a *dom.Node) ([2]float32, bool) {
	if m == nil {
		return [2]float32{}, false
	}
	s, ok := m.sizes[a]
	return s, ok
}

// unitsAt are what lengths stand for on the styled element a: its font's
// units, the root's, the viewport's and its own containers'.
func (m *measures) unitsAt(a *dom.Node) *Units {
	st := dom.Data(a).Style
	u := &Units{Em: float64(st.Font.Size), Rem: float64(st.Font.Size)}
	root := st
	if r := m.rootStyle(); r != nil {
		root = r
		u.Rem = float64(r.Font.Size)
	}
	for i, s := range []*style.Computed{st, root} {
		in := m.fontUnits(s.Font)
		line := in.Line
		switch lh := s.LineHeight; {
		case lh.Normal:
		case lh.Factor != 0:
			line = lh.Factor * s.Font.Size
		default:
			line = lh.Px
		}
		at := relUnit(i) * unitRex
		u.Rel[at+unitEx], u.Rel[at+unitCh], u.Rel[at+unitCap] = float64(in.Ex), float64(in.Ch), float64(in.Cap)
		u.Rel[at+unitIc], u.Rel[at+unitLh] = float64(in.Ic), float64(line)
	}
	w, h := m.viewport()
	u.Rel[unitVw], u.Rel[unitVh] = float64(w)/100, float64(h)/100
	u.Rel[unitVmin], u.Rel[unitVmax] = float64(min(w, h))/100, float64(max(w, h))/100
	for unit := unitCqw; unit < relUnits; unit++ {
		u.Rel[unit] = float64(m.containerUnit(a, false, unit)) / 100
	}
	return u
}

// containerUnit is what a hundred of the container unit u stand for on
// the element e, or its pseudo-element when self: the content box of the
// nearest container above it that answers on u's axis, or the viewport's
// with none laid out. Each container passed is noted as read.
func (m *measures) containerUnit(e *dom.Node, self bool, u relUnit) float32 {
	if u == unitCqmin || u == unitCqmax {
		w, h := m.containerUnit(e, self, unitCqw), m.containerUnit(e, self, unitCqh)
		if u == unitCqmin {
			return min(w, h)
		}
		return max(w, h)
	}
	axis, at := inlineAxis, 0
	if u == unitCqh {
		axis, at = blockAxis, 1
	}
	a := e
	if !self && a != nil {
		a = a.Parent()
	}
	for ; a != nil && m != nil; a = a.Parent() {
		st := dom.Data(a).Style
		if st == nil || axesOf(st.ContainerType)&axis == 0 {
			continue
		}
		if m.read == nil {
			m.read = map[*dom.Node]bool{}
		}
		m.read[a] = true
		if s, ok := m.sizes[a]; ok {
			return s[at]
		}
	}
	if m != nil {
		m.used |= 1 << unitVw
	}
	w, h := m.viewport()
	return [2]float32{w, h}[at]
}

// contain notes n, a styled element, as a container whose size Requery
// asks for.
func (m *measures) contain(n *dom.Node) {
	if m.containers == nil {
		m.containers = map[*dom.Node]bool{}
	}
	m.containers[n] = true
}

// queries are the @container rules of sheets, each once.
func containerQueries(sheets []*Sheet) []*containerRule {
	var out []*containerRule
	for _, s := range sheets {
		for i := range s.Rules {
			for _, c := range s.Rules[i].containers {
				if !slices.Contains(out, c) {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// Requery tells the cascade the content box of each container, as size
// reports it after a layout (false for one laid out with no box of a
// size: an inline box, a table, no box at all), and reports whether that
// changes the answer of some @container rule's condition or what a
// container unit stands for under one: the caller then lays out again
// after the next Compute, which restyles what those containers hold. A
// container whose size changed nothing costs only its conditions.
func (t *Transitions) Requery(size func(n *dom.Node) (w, h float32, ok bool)) bool {
	env := &t.env
	viewport := [2]float32{env.width, env.height}
	moved := viewport != t.queried && slices.ContainsFunc(t.queries, func(r *containerRule) bool { return r.viewport })
	t.queried = viewport
	changed := false
	for n := range env.containers {
		st := dom.Data(n).Style
		if st == nil || st.ContainerType == style.ContainerNormal && !st.ContainerScrollState || !t.attached(n) {
			delete(env.containers, n)
			delete(env.sizes, n)
			delete(env.read, n)
			delete(env.scroll, n)
			continue
		}
		if st.ContainerType == style.ContainerNormal {
			continue
		}
		w, h, ok := size(n)
		old, had := env.sizes[n]
		now := [2]float32{w, h}
		if ok == had && (!ok || old == now) && !moved {
			continue
		}
		before := env.subjectOf(n)
		if ok {
			if env.sizes == nil {
				env.sizes = map[*dom.Node][2]float32{}
			}
			env.sizes[n] = now
		} else {
			delete(env.sizes, n)
		}
		if moved || env.read[n] || t.answersMoved(before, env.subjectOf(n)) {
			restyle(n)
			if env.requery == nil {
				env.requery = map[*dom.Node]bool{}
			}
			env.requery[n] = true
			changed = true
		}
	}
	return changed
}

// answersMoved reports whether some condition of the sheets' @container
// rules answers otherwise for a than for b.
func (t *Transitions) answersMoved(a, b *subject) bool {
	for _, r := range t.queries {
		for _, c := range r.conds {
			if c.query != nil && c.query.eval(a) != c.query.eval(b) {
				return true
			}
		}
	}
	return false
}
