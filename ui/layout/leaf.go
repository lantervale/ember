package layout

import "github.com/lantervale/ember/ui/internal/style"

// flowAmong has the lines of n take the room the floats of ctx leave them, at the width
// the block has, and reports whether there are such floats.
func (c *inlineContent) flowAmong(ctx *blockContext, n *node, width float32, inset rect) bool {
	flows := ctx != nil && n.control == noControl && some(width) && (ctx.hasFloats() || c.pendingFloats())
	if flows || c.flowed {
		c.broken = nil
	}
	c.flowed = flows
	if flows {
		ctx.setContentInset(inset.l, inset.r)
		c.flow, c.flowTop, c.flowLeft = ctx, inset.t, inset.l
	}
	return flows
}

// leaf lays out a node without child boxes: empty, or holding inline content, Taffy's
// compute_leaf_layout with text as its measure function.
//
//nolint:cyclop,funlen,gocyclo,gocognit,nestif // Taffy's compute_leaf_layout, kept in its shape
func (t *tree) leaf(n *node, in input, ctx *blockContext) output {
	st := n.st
	padding := edges(st.Padding, in.parent.w)
	border := n.borders()
	pb := padding.add(border)
	pbSum := pb.sum()
	adj := boxSizingAdjustment(st, pbSum)

	room := n.room(in)
	var nodeSize, minSize, maxSize size
	if in.sizing == contentSize {
		nodeSize, minSize, maxSize = in.known, noSize, noSize
		if n.framed() && !some(nodeSize.w) && !some(nodeSize.h) {
			nodeSize = n.naturalIn(room, pbSum).add(pbSum)
		}
	} else {
		ratio := aspectRatio(st)
		styleSize := resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
		minSize = stretched(resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MinSize, st, in.parent)
		maxSize = stretched(resolveMax(st.MaxSize, in.parent).mAdd(adj), st.MaxSize, st, in.parent)
		if !some(ratio) {
			// A natural ratio holds between content boxes, whatever box-sizing.
			ratio = n.naturalRatio()
			n.intrinsic = n.intrinsic || n.replaced && (styleSize.oneSided() || minSize.oneSided())
			styleSize = styleSize.clampedOneSide(ratio, minSize, maxSize).mSub(pbSum).withAspectRatio(ratio).mAdd(pbSum)
			minSize = minSize.mSub(pbSum).withAspectRatio(ratio).mAdd(pbSum)
		}
		nodeSize = in.known.or(styleSize)
		if n.framed() {
			nodeSize = nodeSize.mSub(adj).withAspectRatio(ratio).mAdd(adj)
		}
		if n.replaced && !some(nodeSize.w) && !some(nodeSize.h) {
			n.intrinsic = true
			nodeSize = replacedSize(n.preferredIn(room, pbSum, st), ratio, minSize, maxSize)
		}
	}

	gutter := n.ownGutter()
	inset := pb
	inset.r += gutter.x
	inset.b += gutter.y

	block := isBlockDisplay(st.Display)
	preventsCollapse := !block || blockEstablishesBFC(st) || outOfFlow(st.Position) ||
		padding.t > 0 || padding.b > 0 || border.t > 0 || border.b > 0 ||
		(some(nodeSize.h) && nodeSize.h > 0) || (some(minSize.h) && minSize.h > 0)

	if in.run == computeSize && preventsCollapse && nodeSize.both() && in.clamp.lines == 0 {
		s := nodeSize.mClamp(minSize, maxSize).mMax(pbSum)
		return sizeOutput(s)
	}

	avail := spaces{w: spaceOr(in.known.w, in.avail.w), h: spaceOr(in.known.h, in.avail.h)}
	if some(nodeSize.w) {
		avail.w = def(nodeSize.w)
	}
	if some(nodeSize.h) {
		avail.h = def(nodeSize.h)
	}
	avail.w = avail.w.mapDefinite(func(v float32) float32 { return mClamp(v, minSize.w, maxSize.w) - inset.horizontal() })
	avail.h = avail.h.mapDefinite(func(v float32) float32 { return mClamp(v, minSize.h, maxSize.h) - inset.vertical() })

	known := in.known.mSub(inset.sum())
	if in.run == performLayout {
		known = noSize
	}
	if n.replaced {
		known = in.known.or(nodeSize).mSub(inset.sum())
		n.intrinsic = n.intrinsic || !known.both()
	}
	if c := n.inline; c != nil && some(nodeSize.w) && n.control == noControl && c.fills() {
		// A picture that fills is as wide as the lines' room, which is the box's, not
		// what the content would take.
		known.w = mClamp(nodeSize.w, minSize.w, maxSize.w) - inset.horizontal()
	}
	if c := n.inline; c != nil && c.flowAmong(ctx, n, in.known.w, inset) {
		defer func() { c.flow = nil }()
		known.w = in.known.w - inset.horizontal()
	}
	if c := n.inline; c != nil {
		c.height = none
		if some(nodeSize.h) {
			c.height = max(mClamp(nodeSize.h, minSize.h, maxSize.h)-inset.vertical(), 0)
		}
	}
	// A box that takes its content's width gives its content's percentages that width.
	basis := avail.w.value()
	if !some(in.known.w) && !some(nodeSize.w) {
		basis = none
	}
	clamp := textClamp(n, in.clamp)
	if clamp.lines == style.ClampByHeight {
		clamp.lines = linesHeld(or(nodeSize.h, maxSize.h), func(lines int) (float32, int) {
			s, _, _, all := t.measureContent(n, size{known.w, none}, avail, basis, lines)
			return s.h + inset.vertical(), all
		})
	}
	measured, first, last, lines := t.measureContent(n, known, avail, basis, clamp.lines)
	if !n.replaced {
		measured = size{max(measured.w, n.natural.w), max(measured.h, n.natural.h)}
	}
	content := measured.add(inset.sum())
	if in.sizing != contentSize {
		content.h = sizedHeight(st, content.h, adj.h, in.parent.h)
	}
	clamped := in.known.or(nodeSize).or(content).mClamp(minSize, maxSize)
	s := clamped
	if r := aspectRatio(st); in.sizing == inherentSize && some(r) {
		s.h = fmax(clamped.h, ratioHeight(clamped.w, r))
	}
	s = s.mMax(pbSum)
	if in.run == performLayout && n.control == textareaControl {
		measured, gutter = t.textareaText(n, s, pb, gutter)
		styled := scrollbarGutter(st)
		n.gutter = size{gutter.x - styled.x, gutter.y - styled.y}
	}

	if in.run == performLayout && n.control == inputControl {
		measured.w = max(measured.w, t.inputText(n))
	}
	scroll := isScrollContainer(st)
	end := padding.l + measured.w
	bottom := padding.t + measured.h
	if scroll {
		end += padding.r
		bottom += padding.b
	}
	port := s.sub(border.sum()).sub(size{gutter.x, gutter.y})
	reach := reachLeft(n, rect{r: end, b: bottom}, port.w, measured.w, padding, scroll)
	overflow := finalizeOverflow(reach, port, fromEnd(st), scroll)

	out := output{
		size:             s,
		overflow:         overflow,
		firstBaseline:    none,
		lastBaseline:     none,
		collapsesThrough: !preventsCollapse && s.h == 0 && measured.h == 0,
		lines:            lines,
		clamped:          clamp.ends(lines),
	}
	if in.run == performLayout {
		n.clamp = clamp.out(lines)
	}
	if some(first) {
		shift := fmax(s.h-inset.vertical()-measured.h, 0)*contentShare(n) + n.lowered
		out.firstBaseline = pb.t + shift + first
		out.lastBaseline = pb.t + shift + last
	}
	if in.run == performLayout && n.inline != nil {
		ge := gutterEdges(st, gutter)
		text := pb
		text.l, text.r, text.b = text.l+ge.l, text.r+ge.r, text.b+ge.b
		t.placeText(n, s, text)
	}
	if in.run == performLayout && len(n.kids) > 0 {
		t.leafChildren(n, &out, pb, border, gutter)
	}
	return out
}

// leafChildren places what an inline formatting context holds besides its text: hidden
// boxes, and positioned ones: from their spot in the line when a span holds them, else
// from the content's start, waiting for their containing block unless a positioned span
// is it.
func (t *tree) leafChildren(n *node, out *output, pb, border rect, gutter point) {
	for i, k := range n.kids {
		if k.atomic {
			out.oof = append(out.oof, translateCandidates(k.pendingOOF, k.unrounded.location)...)
			continue
		}
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: uint32(i)}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, inherentSize)
			continue
		}
		c := oofCandidate{
			node:     k,
			order:    uint32(i),
			position: k.st.Position,
			x:        staticFromStart(pb.l),
			y:        staticFromStart(pb.t),
			inline:   true,
		}
		if o := n.inline.oofOf(k); o != nil {
			c.x, c.y = startOf(staticFromStart(o.static.x), n.st), staticFromStart(o.static.y)
			if o.positioned && k.st.Position == style.Absolute {
				c.area = &oofArea{size: o.areaSize, offset: o.areaAt}
			}
		}
		out.oof = append(out.oof, c)
	}
	inset := border.add(rect{r: gutter.x, b: gutter.y})
	out.area = &oofArea{size: out.size.sub(inset.sum()), offset: point{inset.l, inset.t}}
}

// reachLeft is reach for n's text under direction rtl, measured wide and laid out from
// the right of a padding box port wide: lines too long overflow its left, past its left
// padding where it scrolls. A control's is reach.
func reachLeft(n *node, reach rect, port, measured float32, padding rect, scroll bool) rect {
	if n.st.Dir != style.RTL || n.control != noControl {
		return reach
	}
	reach.r = port - padding.r
	reach.l = reach.r - measured
	if scroll {
		reach.l -= padding.l
	}
	return reach
}

// room is the width a replaced box that fills takes: the room it is given, or with no
// natural width its containing block's at any content size.
func (n *node) room(in input) space {
	if in.avail.w.kind != definite && n.framed() && some(in.parent.w) {
		return def(in.parent.w)
	}
	return in.avail.w
}

func ratioHeight(w, ratio float32) float32 {
	if some(ratio) {
		return w / ratio
	}
	return 0
}

// measureContent is the size of a leaf's content: its text, or nothing. basis is the
// width its percentages are of, none for the width found here.
func (t *tree) measureContent(n *node, known size, avail spaces, basis float32, clamp int) (size, float32, float32,
	int) {
	if n.replaced {
		return measureReplaced(n, known, avail.w), none, none, 0
	}
	if n.inline == nil {
		return known.or(size{}), none, none, 0
	}
	if n.control != noControl {
		s, first, last := t.measureControl(n, known)
		return s, first, last, 0
	}
	s, first, last, lines := t.measure(n.inline, known, avail, size{basis, none}, clamp)
	return size{or(known.w, s.w), or(known.h, s.h)}, first, last, lines
}
