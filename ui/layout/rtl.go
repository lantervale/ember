package layout

import "github.com/lantervale/ember/ui/internal/style"

// mirrored lays n out by lay, a left-to-right algorithm (flex, grid), and, when n is
// right to left, has it lay out n's mirror image instead: n's own left and right padding
// and border swapped, and its children's margins and insets, then each child placed back
// as in a mirror. Its start, a row's main start or a column's cross start, then stands on
// the right, as CSS has it under direction: rtl. Its scrollable overflow starts from the
// right, and its scrollbar gutter stands on its left, as Chrome's.
func (t *tree) mirrored(n *node, in input, lay func() output) output {
	if n.st.Dir != style.RTL {
		return lay()
	}
	own := n.st
	flipped := *own
	flipped.Padding.Left, flipped.Padding.Right = own.Padding.Right, own.Padding.Left
	flipped.Border.Left, flipped.Border.Right = own.Border.Right, own.Border.Left
	n.st = &flipped
	n.flipCollapsed()
	kids := make([]*style.Computed, len(n.kids))
	for i, k := range n.kids {
		kids[i] = k.st
		f := *k.st
		f.Margin.Left, f.Margin.Right = f.Margin.Right, f.Margin.Left
		f.Inset.Left, f.Inset.Right = f.Inset.Right, f.Inset.Left
		k.st = &f
	}
	out := lay()
	n.st = own
	n.flipCollapsed()
	for i, k := range n.kids {
		k.st = kids[i]
	}
	w := out.size.w
	if in.run == performLayout {
		for _, k := range n.kids {
			if outOfFlow(k.st.Position) {
				continue
			}
			p := &k.unrounded
			p.location.x = w - p.location.x - p.size.w
			p.margin.l, p.margin.r = p.margin.r, p.margin.l
		}
	}
	for i := range out.oof {
		x := &out.oof[i].x
		x.area = span{w - x.area.end, w - x.area.start}
		x.align.keyword, x.align.fallback = flipEdge(x.align.keyword), flipEdge(x.align.fallback)
	}
	if a := out.area; a != nil {
		a.offset.x = w - a.offset.x - a.size.w
	}
	if o := &out.overflow; o.rect != (rect{}) || o.reversed[0] {
		pw := w - n.borders().horizontal()
		o.rect.l, o.rect.r = pw-o.rect.r, pw-o.rect.l
		o.reversed[0] = !o.reversed[0]
	}
	return out
}

// mirroredTable is mirrored for a table, whose columns run from the right under
// direction rtl: inside the table mirrored places its row groups, rows and columns, and
// each of those places its own rows, cells and columns back as in a mirror too.
func (t *tree) mirroredTable(n *node, in input) output {
	out := t.mirrored(n, in, func() output { return t.table(n, in) })
	if n.st.Dir != style.RTL || in.run != performLayout {
		return out
	}
	var flip func(p *node)
	flip = func(p *node) {
		for _, k := range p.kids {
			if k.role != roleRow && k.role != roleGroup && k.role != roleColumnGroup {
				continue
			}
			w := k.unrounded.size.w
			for _, c := range k.kids {
				c.unrounded.location.x = w - c.unrounded.location.x - c.unrounded.size.w
			}
			if k.role == roleGroup {
				flip(k)
			}
		}
	}
	flip(n)
	for i := range n.box.Collapsed {
		r := &n.box.Collapsed[i].Rect
		r.X = out.size.w - r.X - r.W
	}
	return out
}

// flipCollapsed swaps the left and right of the borders collapsing left n: a table's are
// its mirror image's while it lays out, collapse resolving them there.
func (n *node) flipCollapsed() {
	if n.collapsed != nil {
		c := *n.collapsed
		c.l, c.r = c.r, c.l
		n.collapsed = &c
	}
}

func flipEdge(e staticEdge) staticEdge {
	switch e {
	case edgeStart:
		return edgeEnd
	case edgeEnd:
		return edgeStart
	case edgeCenter:
	}
	return e
}

// fromEnd is whether a block container of style st scrolls from the end of each axis:
// its right under direction rtl.
func fromEnd(st *style.Computed) [2]bool { return [2]bool{st.Dir == style.RTL, false} }

// startOf is p under a container of style st: its start and end on the right and the
// left under direction rtl.
func startOf(p staticPosition, st *style.Computed) staticPosition {
	if st.Dir == style.RTL {
		p.align.keyword, p.align.fallback = flipEdge(p.align.keyword), flipEdge(p.align.fallback)
	}
	return p
}
