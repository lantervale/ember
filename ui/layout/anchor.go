package layout

import (
	"maps"
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// ImplicitAnchors gives layout each element's implicit anchor, the
// element whose popover it is shown by, nil for none: position-anchor's
// auto, and its normal for a popover, read it.
func ImplicitAnchors(f func(*dom.Node) *dom.Node) Option { return func(t *tree) { t.implicit = f } }

// maxAnchorPasses bounds the layouts one Layout runs for anchors placed
// by anchors placed by anchors.
const maxAnchorPasses = 4

// anchoring is what the elements positioned by anchors read of them,
// between layouts: each one's anchors where the last layout placed them.
type anchoring struct {
	of map[*dom.Node]*anchorsOf
}

// anchorsOf is one positioned element's anchors, their border boxes in
// its containing block's border box, as its containing block lays it
// out. A name it reads that no acceptable anchor has is missing.
type anchorsOf struct {
	named map[string]rect
	// def is the default anchor's, and implicit the implicit anchor's.
	def, implicit       rect
	hasDef, hasImplicit bool
	// hidden is set when the default anchor is clipped out of sight by
	// what scrolls or clips it below the containing block.
	hidden bool
}

func (a *anchorsOf) equal(o *anchorsOf) bool {
	if a == nil || o == nil {
		return a == o
	}
	return maps.Equal(a.named, o.named) && a.def == o.def && a.hasDef == o.hasDef && a.hidden == o.hidden &&
		a.implicit == o.implicit && a.hasImplicit == o.hasImplicit
}

// anchorUser reports whether a positioned element of style st reads
// anchors: by anchor() or anchor-size(), position-area, anchor-center, a
// fallback, or a default anchor it hides with.
func anchorUser(st *style.Computed) bool {
	a := &st.Anchoring
	if a.Area.Set || len(a.Tries) > 0 || a.Anchor == style.AnchorNamed || a.Anchor == style.AnchorAuto ||
		st.AlignSelf == style.AlignAnchorCenter || st.JustifySelf == style.AlignAnchorCenter {
		return true
	}
	return slices.ContainsFunc(anchoredLengths(st), style.Length.Anchored)
}

func anchoredLengths(st *style.Computed) []style.Length {
	return []style.Length{st.Inset.Top, st.Inset.Right, st.Inset.Bottom, st.Inset.Left,
		st.Size.Width, st.Size.Height, st.MinSize.Width, st.MinSize.Height, st.MaxSize.Width, st.MaxSize.Height,
		st.Margin.Top, st.Margin.Right, st.Margin.Bottom, st.Margin.Left}
}

// names are the anchor names st and its fallbacks read.
func names(st *style.Computed) []string {
	var out []string
	add := func(s *style.Computed) {
		if s.Anchoring.Anchor == style.AnchorNamed {
			out = append(out, s.Anchoring.AnchorName)
		}
		for _, l := range anchoredLengths(s) {
			if l.Anchored() {
				c := l.Calc.Value()
				c.Anchors(func(fn style.AnchorFn) {
					if fn.Name != "" {
						out = append(out, fn.Name)
					}
				})
			}
		}
	}
	add(st)
	for _, t := range st.Anchoring.Tries {
		add(t)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// anchorsFor works out, from the laid-out tree out, the anchors of the
// positioned element whose box is b: for each name it reads, the last
// acceptable anchor of that name in tree order.
func (t *tree) anchorsFor(out *box.Tree, b *box.Box, idx *anchorIndex) *anchorsOf {
	st := b.Style
	cb := containingBox(b)
	origin := cbOrigin(cb, st.Position)
	at := func(a *box.Box) rect {
		r := a.Rect()
		return rect{l: r.X - origin.X, r: r.X + r.W - origin.X, t: r.Y - origin.Y, b: r.Y + r.H - origin.Y}
	}
	res := &anchorsOf{named: map[string]rect{}}
	for _, name := range names(st) {
		if a := idx.last(b, cb, name); a != nil {
			res.named[name] = at(a)
		}
	}
	if t.implicit != nil {
		if e := t.implicit(b.Node); e != nil {
			if a := out.Of[e]; a != nil && acceptable(b, cb, a, idx.order) {
				res.implicit, res.hasImplicit = at(a), true
			}
		}
	}
	var def *box.Box
	switch st.Anchoring.Anchor {
	case style.AnchorNamed:
		def = idx.last(b, cb, st.Anchoring.AnchorName)
	case style.AnchorAuto, style.AnchorNormal:
		if res.hasImplicit && (st.Anchoring.Anchor == style.AnchorAuto || st.Anchoring.Area.Set) {
			def = out.Of[t.implicit(b.Node)]
		}
	case style.AnchorNone:
	}
	if def != nil {
		res.def, res.hasDef = at(def), true
		res.hidden = clippedAway(def, cb)
	}
	return res
}

// anchorIndex is a laid-out tree's boxes in tree order, and those of
// each anchor name.
type anchorIndex struct {
	order map[*box.Box]int
	named map[string][]*box.Box
}

func indexAnchors(root *box.Box) *anchorIndex {
	idx := &anchorIndex{order: map[*box.Box]int{}, named: map[string][]*box.Box{}}
	var walk func(b *box.Box)
	walk = func(b *box.Box) {
		idx.order[b] = len(idx.order)
		if b.Kind == box.Element && b.Style != nil {
			for _, name := range b.Style.Anchoring.Names {
				idx.named[name] = append(idx.named[name], b)
			}
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	if root != nil {
		walk(root)
	}
	return idx
}

// last is the last box in tree order named name that b, placed in cb,
// may be anchored to.
func (idx *anchorIndex) last(b, cb *box.Box, name string) *box.Box {
	var found *box.Box
	for _, a := range idx.named[name] {
		if acceptable(b, cb, a, idx.order) && inScope(a, b, name) &&
			(found == nil || idx.order[a] > idx.order[found]) {
			found = a
		}
	}
	return found
}

// acceptable reports whether b, positioned in cb, may be anchored to a,
// as CSS Anchor Positioning's acceptable anchor element: neither b nor
// under it, under cb, and laid out before b: no box between a and cb is
// positioned in cb after b in tree order.
func acceptable(b, cb, a *box.Box, order map[*box.Box]int) bool {
	if a == b || a.Style == nil || a.Style.Display == style.DisplayNone {
		return false
	}
	for x := a; x != nil; x = x.Parent {
		if x == b {
			return false
		}
		if x == cb {
			return true
		}
		if x.Kind == box.Element && x.Style != nil && outOfFlow(x.Style.Position) && containingBox(x) == cb &&
			order[x] > order[b] {
			return false
		}
	}
	return cb == nil
}

// inScope reports whether a's name is in scope for b: no anchor-scope
// above a keeps the name to a subtree b is out of.
func inScope(a, b *box.Box, name string) bool {
	for s := a; s != nil; s = s.Parent {
		if s.Style == nil || s.Kind != box.Element {
			continue
		}
		sc := &s.Style.Anchoring
		if sc.Scope == style.ScopeAll || sc.Scope == style.ScopeNames && slices.Contains(sc.ScopeNames, name) {
			for x := b; x != nil; x = x.Parent {
				if x == s {
					return true
				}
			}
			return false
		}
	}
	return true
}

// containingBox is the box b, absolutely or fixed positioned, is laid
// out in: the nearest that claims it, the Viewport, or nil for the
// initial containing block.
func containingBox(b *box.Box) *box.Box {
	for a := b.Parent; a != nil; a = a.Parent {
		if a.Kind == box.Viewport {
			return a
		}
		if a.Kind == box.Element && a.Style != nil && containingBlockClaims(a.Style).take(b.Style.Position) {
			return a
		}
	}
	return nil
}

// cbOrigin is where, in the root's space as drawn, the origin of the
// space cb lays a box positioned p out in stands: its border box's
// top-left, before its own scrolling; a scrolled viewport holds a fixed
// box in place.
func cbOrigin(cb *box.Box, p style.Position) geom.Vec2 {
	if cb == nil {
		return geom.Vec2{}
	}
	r := cb.Rect()
	at := geom.V2(r.X, r.Y)
	if cb.Kind == box.Viewport && p == style.FixedPosition {
		return at
	}
	return at.Sub(cb.Scroll)
}

// clippedAway reports whether a is out of sight in a box between it and
// cb that clips: what it scrolls or hides leaves none of a showing.
func clippedAway(a, cb *box.Box) bool {
	r := a.Rect()
	for x := a.Parent; x != nil && x != cb; x = x.Parent {
		if x.Style == nil || x.Kind != box.Element && x.Kind != box.Viewport {
			continue
		}
		if x.Style.Overflow.Width == style.Visible && x.Style.Overflow.Height == style.Visible {
			continue
		}
		port := scrollport(x)
		at := x.Rect()
		port.X, port.Y = port.X+at.X, port.Y+at.Y
		if r.X >= port.X+port.W || r.X+r.W <= port.X || r.Y >= port.Y+port.H || r.Y+r.H <= port.Y {
			return true
		}
	}
	return false
}

// refreshAnchors works out again, from the tree out laid out with them,
// the anchors of each element that reads them, and lists the elements
// whose anchors moved.
func (t *tree) refreshAnchors(out *box.Tree, a *anchoring) []*dom.Node {
	var moved []*dom.Node
	idx := indexAnchors(out.Root)
	for e := range a.of {
		b := out.Of[e]
		if b == nil || b.Style == nil || !outOfFlow(b.Style.Position) || !anchorUser(b.Style) {
			delete(a.of, e)
			continue
		}
		now := t.anchorsFor(out, b, idx)
		if !now.equal(a.of[e]) {
			a.of[e] = now
			moved = append(moved, e)
		}
	}
	return moved
}
