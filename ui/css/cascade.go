package css

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
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
	// inherits is set when a declaration sets a property that isn't
	// inherited to inherit.
	inherits bool
	// above are the classes, ids and attributes some selector reads in a
	// compound above its subject, keyed as dom.Pipeline.Changed.
	above map[string]bool
}

type entry struct {
	rule, sel int
}

func (s *Sheet) indexed() *index {
	s.once.Do(func() {
		x := &index{pseudos: map[string]*buckets{}, above: map[string]bool{}}
		for ri, r := range s.Rules {
			for si, sel := range r.Selectors {
				e := entry{ri, si}
				sel.aboveKeys(func(k string) { x.above[k] = true })
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
			for _, d := range r.Decls {
				if p, ok := properties[d.Name]; (!ok || !p.inherited) &&
					slices.ContainsFunc(items(d.Value), func(c Component) bool { return isWord(c, "inherit") }) {
					x.inherits = true
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
	order int
}

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
	return a.order < b.order
}

// matcher keeps the buffers matching and cascading use from one element
// to the next, so a restyle allocates per changed style, not per step.
type matcher struct {
	best   map[int]Specificity
	rules  []int
	out    []applied
	long   []*Declaration
	custom []*Declaration
	// off are, sheet by sheet, the rules whose @media rules don't hold
	// (mediaOff), and ranks each sheet's layers' depths (layerRanks);
	// both nil when no sheet has any.
	off   [][]bool
	ranks [][]int32
	// hints are the declarations the element's own attributes stand for,
	// an img's width and height: the author's, under every rule of the
	// sheets.
	hints []Declaration
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
		m.best = map[int]Specificity{}
	}
	m.out = m.out[:0]
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
					spec: m.best[r], order: order})
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
// most specific matching selector; the rules off are left out.
func (m *matcher) try(s *Sheet, es []entry, e *dom.Node, off []bool) {
	for _, en := range es {
		sel := s.Rules[en.rule].Selectors[en.sel]
		if off != nil && off[en.rule] || !sel.Matches(e) {
			continue
		}
		if sp, ok := m.best[en.rule]; !ok || sp.Less(sel.Specificity) {
			m.best[en.rule] = sel.Specificity
		}
	}
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
}

// cascade puts each longhand's winner at its place in order. The result
// lives until the next call.
func (m *matcher) cascade(list []applied) cascaded { return m.cascadeFor(list, "") }

// cascadeFor is cascade for a pseudo-element, which leaves out the
// declarations of the properties it doesn't take.
func (m *matcher) cascadeFor(list []applied, pseudo string) cascaded {
	if m.long == nil {
		m.long = make([]*Declaration, len(order))
	}
	clear(m.long)
	m.custom = m.custom[:0]
	for _, a := range list {
		d := a.decl
		if strings.HasPrefix(d.Name, "--") {
			m.custom = append(m.custom, d)
			continue
		}
		if long, ok := shorthands[d.Name]; ok {
			for _, l := range long {
				if pseudo == "" || takesProperty(pseudo, l) {
					m.long[orderIndex[l]] = d
				}
			}
			continue
		}
		if pseudo == "" || takesProperty(pseudo, d.Name) {
			m.long[orderIndex[d.Name]] = d
		}
	}
	return cascaded{longhands: m.long, custom: m.custom}
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
