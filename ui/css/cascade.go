package css

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// buckets hold selectors by their rightmost compound's id, else a class,
// else its type, so an element only tries selectors that can match it.
type buckets struct {
	byID, byClass map[string][]entry
	byTag         map[dom.Tag][]entry
	any           []entry
}

func (b *buckets) add(sel Selector, e entry) {
	last := sel.Compounds[len(sel.Compounds)-1]
	switch {
	case len(last.IDs) > 0:
		if b.byID == nil {
			b.byID = map[string][]entry{}
		}
		b.byID[last.IDs[0]] = append(b.byID[last.IDs[0]], e)
	case len(last.Classes) > 0:
		if b.byClass == nil {
			b.byClass = map[string][]entry{}
		}
		b.byClass[last.Classes[0]] = append(b.byClass[last.Classes[0]], e)
	case last.Tag != dom.TextTag:
		if b.byTag == nil {
			b.byTag = make(map[dom.Tag][]entry)
		}
		b.byTag[last.Tag] = append(b.byTag[last.Tag], e)
	default:
		b.any = append(b.any, e)
	}
}

// may reports whether some selector of b can match e, by its bucket.
func (b *buckets) may(e *dom.Node) bool {
	if len(b.any) > 0 || len(b.byTag[e.Tag()]) > 0 {
		return true
	}
	if id := e.ID(); id != "" && len(b.byID[id]) > 0 {
		return true
	}
	if len(b.byClass) > 0 {
		for _, c := range e.Classes() {
			if len(b.byClass[c]) > 0 {
				return true
			}
		}
	}
	return false
}

// index is a sheet's selectors in buckets: the elements' own, and each
// pseudo-element's apart, so an element never tries a pseudo-element's.
type index struct {
	buckets
	// pseudos are the selectors of each pseudo-element its rules name.
	pseudos map[string]*buckets
	// inherited are the longhands that aren't inherited and that some
	// declaration sets to inherit; inherits is set when a shorthand or a
	// custom property is.
	inherited []string
	inherits  bool
	// above are the classes, ids and attributes some selector reads in a
	// compound above its subject, keyed as dom.Pipeline.Changed.
	above map[string]bool
	// beside are those some selector reads before a sibling combinator: a
	// change of one on an element can restyle the elements after it.
	// siblings is set once a selector has such a combinator.
	beside   map[string]bool
	siblings bool
	// reads are the states the selectors read away from their subjects,
	// aboveStates those read above one and besideStates before one.
	reads                     []stateRead
	aboveStates, besideStates dom.State
	// has are the selectors' :has() pseudo-classes, hasKeys the classes,
	// ids and attributes their selectors read, keyed as
	// dom.Pipeline.Changed, and hasStates the states: a change of one
	// below or after an element can turn that element's :has().
	has       []hasRead
	hasKeys   map[string]bool
	hasStates dom.State
	// hasAway and hasAwayStates are those of them read through an :is(),
	// a :not() or a :where() on an element other than that selector's
	// subject, which may stand above or beside the element the :has() is
	// on: a change of one may turn the :has() of any element round it.
	hasAway       map[string]bool
	hasAwayStates dom.State
	// empty is set once a selector reads :empty.
	empty bool
}

// hasRead is a :has() of a rule: an element having what on names may
// match it or stop matching after a change below it, or after it when
// beside.
type hasRead struct {
	on     []*Compound
	beside bool
	rule   int
}

// names reports whether e has what the compounds holding the :has() name.
func (r *hasRead) names(e *dom.Node) bool {
	return !slices.ContainsFunc(r.on, func(c *Compound) bool { return !c.names(e) })
}

// inheritedAttrs are the attributes whose value an element's descendants read
// as their own, keyed as dom.Pipeline.Changed: the pseudo-classes reading
// them read them above their subject.
//
//nolint:gochecknoglobals // lookup table
var inheritedAttrs = map[string]bool{"[lang": true, "[contenteditable": true, "[dir": true}

// indexHas notes in x what the :has() pseudo-classes of sel, a selector of
// the rule ri, read, and what its other pseudo-classes read away from the
// element they are on.
func (x *index) indexHas(sel Selector, ri int) {
	sel.hasReads(func(on []*Compound, rels []Selector, away, beside bool) {
		switch {
		case away && beside:
			x.beside[":has"], x.siblings = true, true
		case away:
			x.above[":has"] = true
		}
		for _, r := range rels {
			x.has = append(x.has, hasRead{on: on, beside: r.Combinators[0].sibling(), rule: ri})
			r.walk(func(c *Compound) {
				c.keys(func(k string) { x.hasKeys[k] = true })
				for i := range c.pseudos {
					x.hasStates |= c.pseudos[i].reads()
				}
			})
			for i := range r.Compounds {
				for _, p := range r.Compounds[i].pseudos {
					for _, n := range p.nested() {
						n.awayReads(func(k string) { x.hasAway[k] = true }, func(s dom.State) { x.hasAwayStates |= s })
					}
				}
			}
		}
	})
	sel.walk(func(c *Compound) {
		for i := range c.pseudos {
			p := &c.pseudos[i]
			x.empty = x.empty || p.kind == pseudoState && p.state&dom.Empty != 0
			for _, k := range p.attrs() {
				if inheritedAttrs[k] {
					x.above[k] = true
				}
			}
		}
	})
}

// indexScope notes in x what the @scope rules r, the rule ri, stands in
// read: their roots and ends stand above the elements r styles, so a
// change of what they read restyles those under it.
func (x *index) indexScope(r Rule, ri int) {
	for _, s := range r.scoped.chain() {
		for _, sel := range slices.Concat(s.start, s.end) {
			sel.walk(func(c *Compound) {
				c.keys(func(k string) { x.above[k] = true })
				var states dom.State
				for i := range c.pseudos {
					states |= c.pseudos[i].reads()
				}
				if states == 0 {
					return
				}
				x.aboveStates |= states
				for i := range r.Selectors {
					subject := &r.Selectors[i].Compounds[len(r.Selectors[i].Compounds)-1]
					x.reads = append(x.reads, stateRead{states: states, on: []*Compound{c}, subject: subject, rule: ri})
				}
			})
		}
	}
}

type entry struct {
	rule, sel int
}

func (s *Sheet) indexed() *index {
	s.once.Do(func() {
		x := &index{pseudos: map[string]*buckets{}, above: map[string]bool{}, beside: map[string]bool{},
			hasKeys: map[string]bool{}, hasAway: map[string]bool{}}
		for ri, r := range s.Rules {
			for si, sel := range r.Selectors {
				e := entry{ri, si}
				sel.aboveKeys(func(k string) { x.above[k] = true })
				x.indexHas(sel, ri)
				if sel.siblings() {
					x.siblings = true
					sel.besideKeys(func(k string) { x.beside[k] = true })
				}
				subject := &sel.Compounds[len(sel.Compounds)-1]
				sel.stateReads(func(on []*Compound, states dom.State, beside bool) {
					x.reads = append(x.reads, stateRead{states: states, on: on, subject: subject, beside: beside, rule: ri})
					if beside {
						x.besideStates |= states
					} else {
						x.aboveStates |= states
					}
				})
				into := &x.buckets
				if sel.PseudoElement != "" {
					into = x.pseudos[sel.PseudoElement]
					if into == nil {
						into = &buckets{}
						x.pseudos[sel.PseudoElement] = into
					}
				}
				into.add(sel, e)
			}
			x.indexScope(r, ri)
			for _, d := range r.Decls {
				p, ok := properties[d.Name]
				if ok && p.inherited ||
					!slices.ContainsFunc(items(d.Value), func(c Component) bool { return isWord(c, "inherit") }) {
					continue
				}
				if _, long := props[d.Name]; !long {
					x.inherits = true
				} else if !slices.Contains(x.inherited, d.Name) {
					x.inherited = append(x.inherited, d.Name)
				}
			}
		}
		s.index = x
	})
	return s.index
}

// level is a declaration's place in the cascade by origin and importance.
type level uint8

const (
	agentNormal level = iota
	authorNormal
	authorImportant
	agentImportant
)

// applied is one declaration that applies to an element, with what ranks
// it against the others.
type applied struct {
	decl   *Declaration
	level  level
	inline bool
	// layer is how far below the rules in no layer its cascade layer
	// stands: 0 in none, 1 the last layer named.
	layer int32
	spec  Specificity
	// near ranks an @scope rule's declaration by how close to its root the
	// element is, higher the closer; 0 out of any @scope.
	near  int32
	order int
}

// ranked is how a rule's most specific selector matching an element
// weighs: its specificity, then how near its @scope's root it matched.
type ranked struct {
	spec Specificity
	near int32
}

// rankOf is the near of an element n generations under its @scope's
// root.
func rankOf(n int) int32 { return math.MaxInt32 - int32(min(n, math.MaxInt32-1)) } //nolint:gosec // bounded above

func (a applied) less(b applied) bool {
	if a.level != b.level {
		return a.level < b.level
	}
	if a.inline != b.inline {
		return b.inline
	}
	if a.layer != b.layer {
		// A later layer wins, and no layer over any; !important turns
		// the layers round.
		return (a.layer > b.layer) != (a.level == authorImportant)
	}
	if a.spec != b.spec {
		return a.spec.Less(b.spec)
	}
	if a.near != b.near {
		return a.near < b.near
	}
	return a.order < b.order
}

// matcher keeps the buffers matching and cascading use from one element
// to the next, so a restyle allocates per changed style, not per step.
type matcher struct {
	best   map[int]ranked
	rules  []int
	out    []applied
	long   []*Declaration
	custom []*Declaration
	// off are, sheet by sheet, the rules whose @media rules don't hold
	// (mediaOff), and ranks each sheet's layers' depths (layerRanks);
	// both nil when no sheet has any.
	off   [][]bool
	ranks [][]int32
	// at is, for each longhand of the last cascade, where its winner
	// stands in the matched list.
	at []int
	// env holds the containers' sizes @container rules are answered
	// for, and self is set while a pseudo-element is matched, whose
	// container may be its element.
	env  *measures
	self bool
	// hints are the declarations the element's own attributes stand for,
	// an img's width and height: the author's, under every rule of the
	// sheets.
	hints []Declaration
	// part is the state of the scrollbar part being matched: a selector
	// asking for states it is not in holds nothing.
	part partState
	// bare leaves out the rules of a control's own look, for one whose
	// appearance is none; native is set once such a rule matched.
	bare, native bool
}

// matched lists the declarations that apply to e, least important first.
// sheets[0] is the user agent's. The list lives until the next call.
func (m *matcher) matched(e *dom.Node, sheets []*Sheet, inline []Declaration) []applied {
	return m.matchedFor(e, sheets, inline, "")
}

// matchedFor is matched for the element's pseudo-element pseudo ("" for the element
// itself); a pseudo-element takes no inline declarations.
func (m *matcher) matchedFor(e *dom.Node, sheets []*Sheet, inline []Declaration, pseudo string) []applied {
	if pseudo != "" {
		inline = nil
	}
	if m.best == nil {
		m.best = map[int]ranked{}
	}
	m.out, m.self, m.native = m.out[:0], pseudo != "", false
	order := 0
	if pseudo == "" {
		for i := range m.hints {
			m.out = append(m.out, applied{decl: &m.hints[i], level: authorNormal, layer: math.MaxInt32, order: order})
			order++
		}
	}
	for si, s := range sheets {
		x := &s.indexed().buckets
		if pseudo != "" {
			if x = s.indexed().pseudos[pseudo]; x == nil {
				continue
			}
		}
		clear(m.best)
		var off []bool
		if m.off != nil {
			off = m.off[si]
		}
		if id := e.ID(); id != "" {
			m.try(s, x.byID[id], e, off)
		}
		if len(x.byClass) > 0 {
			for _, c := range e.Classes() {
				m.try(s, x.byClass[c], e, off)
			}
		}
		m.try(s, x.byTag[e.Tag()], e, off)
		m.try(s, x.any, e, off)
		m.rules = m.rules[:0]
		for r := range m.best {
			m.rules = append(m.rules, r)
		}
		slices.Sort(m.rules)
		for _, r := range m.rules {
			var layer int32
			if l := s.Rules[r].layer; l > 0 && m.ranks != nil {
				layer = m.ranks[si][l-1]
			}
			for di := range s.Rules[r].Decls {
				d := &s.Rules[r].Decls[di]
				m.out = append(m.out, applied{decl: d, level: levelOf(si == 0, d.Important), layer: layer,
					spec: m.best[r].spec, near: m.best[r].near, order: order})
				order++
			}
		}
	}
	for i := range inline {
		d := &inline[i]
		m.out = append(m.out, applied{decl: d, level: levelOf(false, d.Important), inline: true, order: order})
		order++
	}
	slices.SortStableFunc(m.out, func(a, b applied) int {
		switch {
		case a.less(b):
			return -1
		case b.less(a):
			return 1
		}
		return 0
	})
	return m.out
}

// try keeps, for each rule of s with a selector among es matching e, its
// most specific matching selector; the rules off are left out, and those
// whose @container rules don't all hold for e.
func (m *matcher) try(s *Sheet, es []entry, e *dom.Node, off []bool) {
	for _, en := range es {
		r := &s.Rules[en.rule]
		sel := r.Selectors[en.sel]
		if off != nil && off[en.rule] || !m.active(sel) {
			continue
		}
		rank := ranked{spec: sel.Specificity}
		if r.scoped != nil {
			n, ok := r.scoped.matchesScoped(sel, e)
			if !ok {
				continue
			}
			rank.near = rankOf(n)
		} else if !sel.Matches(e) {
			continue
		}
		if len(r.containers) > 0 && !m.contained(r, e) {
			continue
		}
		if was, ok := m.best[en.rule]; !ok || was.spec.Less(rank.spec) || was.spec == rank.spec && was.near < rank.near {
			m.best[en.rule] = rank
		}
		m.native = m.native || sel.native
	}
}

// active reports whether sel holds as the window stands: one with
// :window-inactive only while it is inactive, which the cascade then reads.
func (m *matcher) active(sel Selector) bool {
	if sel.part&^m.part != 0 || sel.native && m.bare {
		return false
	}
	if !sel.inactive {
		return true
	}
	if m.env == nil {
		return false
	}
	m.env.inactiveRead = true
	return m.env.inactive
}

func (m *matcher) contained(r *Rule, e *dom.Node) bool {
	for _, c := range r.containers {
		if !c.holds(e, m.self, m.env) {
			return false
		}
	}
	return true
}

func levelOf(agent, important bool) level {
	switch {
	case agent && important:
		return agentImportant
	case agent:
		return agentNormal
	case important:
		return authorImportant
	}
	return authorNormal
}

// cascaded is each longhand's winning declaration, and the custom
// properties set on the element, in cascade order.
type cascaded struct {
	longhands []*Declaration
	custom    []*Declaration
	// rtl is set when the element's direction is rtl: its declarations
	// of an inline side stand for the other physical one.
	rtl bool
}

// cascade puts each longhand's winner at its place in order, under a
// parent of direction dir. The result lives until the next call.
func (m *matcher) cascade(list []applied, dir style.Direction) cascaded {
	return m.cascadeFor(list, "", dir)
}

// cascadeFor is cascade for a pseudo-element, which leaves out the
// declarations of the properties it doesn't take.
func (m *matcher) cascadeFor(list []applied, pseudo string, dir style.Direction) cascaded {
	if m.long == nil {
		m.long = make([]*Declaration, len(order))
	}
	clear(m.long)
	if m.at == nil {
		m.at = make([]int, len(order))
	}
	m.custom = m.custom[:0]
	rtl := directionIn(list, dir) == style.RTL
	reverts := false
	for k, a := range list {
		d := a.decl
		if strings.HasPrefix(d.Name, "--") {
			m.custom = append(m.custom, d)
			continue
		}
		reverts = reverts || revertOf(d) != ""
		mirror := rtl && d.inline
		if long, ok := shorthands[d.Name]; ok {
			for _, l := range long {
				if mirror {
					l = mirrored(l)
				}
				if pseudo == "" || takesProperty(pseudo, l) {
					m.long[orderIndex[l]], m.at[orderIndex[l]] = d, k
				}
			}
			continue
		}
		name := d.Name
		if mirror {
			name = mirrored(name)
		}
		if pseudo == "" || takesProperty(pseudo, name) {
			m.long[orderIndex[name]], m.at[orderIndex[name]] = d, k
		}
	}
	if reverts {
		for i, d := range m.long {
			if d != nil && revertOf(d) != "" {
				m.long[i] = rolledBack(list, m.at[i], order[i])
			}
		}
	}
	return cascaded{longhands: m.long, custom: m.custom, rtl: rtl}
}

// directionIn is the direction the last declaration of list sets, dir
// where none does or it inherits; a var() in it reads as inherit.
func directionIn(list []applied, dir style.Direction) style.Direction {
	for i := len(list) - 1; i >= 0; i-- {
		d := list[i].decl
		if d.Name != propDirection {
			continue
		}
		if len(d.vals) == 1 && isWord(d.vals[0], "initial") {
			return style.LTR
		}
		if len(d.vals) == 1 {
			if v, ok := directions[strings.ToLower(d.vals[0].Text)]; ok {
				return v
			}
		}
		return dir
	}
	return dir
}

// revertOf is revert or revert-layer when d's value is that keyword
// alone, "" otherwise.
func revertOf(d *Declaration) string {
	if v := items(d.Value); len(v) == 1 && isWord(v[0], kwRevert, kwRevertLayer) {
		return strings.ToLower(v[0].Text)
	}
	return ""
}

// rolledBack is the declaration of the longhand name that wins once
// list[at], a revert or a revert-layer, is rolled back: the cascade of the
// user agent's origin under a revert of the author's, of the layers
// under its own for a revert-layer, nil when nothing is left, which
// leaves the longhand unset.
func rolledBack(list []applied, at int, name string) *Declaration {
	a := list[at]
	for k := at - 1; k >= 0; k-- {
		b := list[k]
		if !setsLonghand(b.decl, name) || !survives(a, b) {
			continue
		}
		if revertOf(b.decl) != "" {
			a = b
			continue
		}
		return b.decl
	}
	return nil
}

// survives reports whether b, ranked under a, is left by a's rollback.
func survives(a, b applied) bool {
	if revertOf(a.decl) == kwRevertLayer && b.level == a.level && (b.layer != a.layer || b.inline != a.inline) {
		return true
	}
	return (a.level == authorNormal || a.level == authorImportant) && b.level == agentNormal
}

func setsLonghand(d *Declaration, name string) bool {
	return d.Name == name || slices.Contains(shorthands[d.Name], name)
}

// declared reports whether the last cascade has a declaration of the
// longhand name.
func (m *matcher) declared(name string) bool { return m.long[orderIndex[name]] != nil }

// layerRanks gives the cascade layers of sheets their depths, sheet by
// sheet in the order of Sheet.layers: 1 for the layer that wins over the
// others, more for each it wins over. Layers are ordered as they are
// first named across the sheets, a layer inside another under that
// one's own rules; nil when no sheet has a layer.
func layerRanks(sheets []*Sheet) [][]int32 {
	type node struct {
		kids []string
		rank int32
	}
	var tree map[string]*node
	name := func(si int, full string) string {
		return strings.ReplaceAll(full, anonymous, fmt.Sprint(anonymous, si, ":"))
	}
	for si, s := range sheets {
		for _, full := range s.layers {
			if tree == nil {
				tree = map[string]*node{"": {}}
			}
			full = name(si, full)
			if tree[full] != nil {
				continue
			}
			tree[full] = &node{}
			parent := ""
			if dot := strings.LastIndexByte(full, '.'); dot >= 0 {
				parent = full[:dot]
			}
			tree[parent].kids = append(tree[parent].kids, full)
		}
	}
	if tree == nil {
		return nil
	}
	rank := int32(min(len(tree), math.MaxInt32))
	var walk func(full string)
	walk = func(full string) {
		for _, kid := range tree[full].kids {
			walk(kid)
		}
		rank--
		tree[full].rank = rank
	}
	walk("")
	ranks := make([][]int32, len(sheets))
	for si, s := range sheets {
		for _, full := range s.layers {
			ranks[si] = append(ranks[si], tree[name(si, full)].rank)
		}
	}
	return ranks
}
