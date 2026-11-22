package layout

import "github.com/lantervale/ember/ui/internal/style"

// containedAxes are the axes n's size containment frees from its content:
// a size container's both, an inline-size container's width. Containment
// skips non-atomic inline boxes, tables and their parts.
func (n *node) containedAxes() bools {
	st := n.st
	switch {
	case st.ContainerType == style.ContainerNormal:
		return bools{}
	case st.Display == style.Inline && !n.replaced, st.Table, st.TablePart != style.NoTablePart:
		return bools{}
	case st.ContainerType == style.ContainerInlineSize:
		return bools{w: true}
	}
	return bools{w: true, h: true}
}

// contain is in with the size n would have without content on each axis
// its containment frees and in leaves open: its own size, else its
// padding, border and gutter, within its min and max. Its content is then
// laid out in that size, overflowing it as it may.
func (n *node) contain(in input) input {
	axes := n.containedAxes()
	if !axes.w && !axes.h || (!axes.w || some(in.known.w)) && (!axes.h || some(in.known.h)) {
		return in
	}
	st := n.st
	pb := edges(st.Padding, in.parent.w).add(n.borders())
	gutter := n.ownGutter()
	inset := pb.sum().add(size{gutter.x, gutter.y})
	styled, minSize, maxSize := noSize, noSize, noSize
	if in.sizing == inherentSize {
		ratio := aspectRatio(st)
		adj := boxSizingAdjustment(st, pb.sum())
		styled = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
		minSize = resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
		maxSize = resolveMax(st.MaxSize, in.parent).mAdd(adj)
	}
	empty := in.known.or(styled).or(inset).mClamp(minSize, maxSize).mMax(inset)
	if axes.w && !some(in.known.w) {
		in.known.w = empty.w
	}
	if axes.h && !some(in.known.h) {
		in.known.h = empty.h
	}
	return in
}
