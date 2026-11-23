package layout

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

type blockItem struct {
	node  *node
	order uint32
	// inSameBFC is a block laid out in its parent's formatting context: its margins
	// collapse with its neighbours', its lines go round the floats and its own floats
	// reach the blocks after it. The others stand beside the floats, whole.
	inSameBFC bool
	// legend is a fieldset's legend, set on its top border.
	legend                 bool
	float                  style.Float
	clear                  style.Clear
	sizeStyle              style.Size[style.Length]
	size, minSize, maxSize size
	overflow               style.Size[style.Overflow]
	position               style.Position
	alignSelf              style.Align
	justifySelf            style.Align
	padding, border        rect
	pbSum                  size
	staticPosition         point
	collapsesThrough       bool
	final                  *placed
	oof                    []oofCandidate
}

func blockEstablishesBFC(st *style.Computed) bool {
	return isScrollContainer(st) || st.AlignContent != style.AlignNormal || st.FlowRoot || st.Table || st.Multicol() ||
		st.ContainerType != style.ContainerNormal || st.LineClamp > 0 || st.ClampAuto
}

// standsBesideFloats reports a block-level box the floats never overlap though it is no
// block container: a replaced element, a control, a table.
func standsBesideFloats(n *node) bool {
	return n.replaced || n.control != noControl || n.role == roleTable
}

// block lays out a block container whose children are blocks, Taffy's compute_block_layout.
func (t *tree) block(n *node, in input, ctx *blockContext) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := n.borders()
	pbSize := padding.add(border).sum()
	adj := boxSizingAdjustment(st, pbSize)
	minSize := stretched(resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MinSize, st, in.parent)
	maxSize := stretched(resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MaxSize, st, in.parent)
	clamped := noSize
	if in.sizing == inherentSize {
		clamped = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj).mClamp(minSize, maxSize)
	}
	minMax := size{definiteMinMax(minSize.w, maxSize.w), definiteMinMax(minSize.h, maxSize.h)}
	styled := in.known.or(minMax).or(clamped).mMax(pbSize)
	// A block that places floats in its parent's context is laid out even when its size
	// is known, for the content after it to find them.
	if in.run == computeSize && (ctx == nil || !n.holdsFloats) && in.clamp.lines == 0 {
		if styled.both() {
			return sizeOutput(styled)
		}
		if in.axis == axisHorizontal && some(styled.w) {
			return sizeOutput(size{styled.w, 0})
		}
	}
	sub := in
	sub.known = styled
	if ctx == nil || blockEstablishesBFC(st) {
		ctx = newRootContext()
	}
	return t.blockInner(n, sub, ctx)
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Taffy's block compute_inner, kept in its shape
func (t *tree) blockInner(n *node, in input, ctx *blockContext) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := n.borders()
	gutter := gutterEdges(st, n.ownGutter())
	pb := padding.add(border)
	pbSize := pb.sum()
	inset := pb.add(gutter)
	ctx.setContentInset(inset.l, inset.r)
	adj := boxSizingAdjustment(st, pbSize)
	sz := resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
	minSize := stretched(resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MinSize, st, in.parent)
	maxSize := stretched(resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MaxSize, st, in.parent)
	derived := in.known.withAspectRatio(ratio).mClamp(minSize, maxSize)
	known := in.known.or(derived)
	percentBasis := size{known.w, none}
	if in.knownIsDefinite.h {
		percentBasis.h = known.h
	}
	contentBoxSize := percentBasis.mSub(inset.sum())
	newBFC := blockEstablishesBFC(st)
	oof := outOfFlow(st.Position)
	adjoin := [2]bool{
		in.collapsible[0] && !newBFC && !oof && padding.t == 0 && border.t == 0,
		in.collapsible[1] && !newBFC && !oof && padding.b == 0 && border.b == 0,
	}
	ownCollapse := [2]bool{adjoin[0], adjoin[1] && !some(sz.h)}
	preventsCollapse := st.Display != style.Block || ctx.root || newBFC || oof ||
		padding.t > 0 || padding.b > 0 || border.t > 0 || border.b > 0

	items := t.blockItems(n, contentBoxSize)
	outerW := known.w
	if !some(outerW) {
		availW := in.avail.w.mSub(inset.horizontal())
		intrinsic := t.blockContentWidth(items, availW) + inset.horizontal()
		outerW = mMax(mClamp(intrinsic, minSize.w, maxSize.w), pbSize.w)
	}
	if in.run == computeSize && (ctx.root || !n.holdsFloats) && in.clamp.lines == 0 {
		if some(known.h) {
			return sizeOutput(size{outerW, known.h})
		}
		if in.axis == axisHorizontal {
			return sizeOutput(size{outerW, 0})
		}
	}
	percentH := percentBasis.h
	if !some(percentH) && in.sizing == inherentSize {
		percentH = mMax(sz.h, minSize.h)
	}
	percentW := or(in.parent.w, outerW)
	rPadding := edges(st.Padding, percentW)
	rBorder := border
	rInset := rPadding.add(rBorder).add(gutter)
	legend := renderedLegend(n, items)
	if legend != nil {
		legend.legend = true
		rInset.t = 0
	}

	clamp := clampFor(n, in.clamp)
	if clamp.lines == style.ClampByHeight {
		clamp.lines = linesHeld(or(known.h, maxSize.h), func(lines int) (float32, int) {
			trial := clamp
			trial.lines = lines
			c := newRootContext()
			c.setContentInset(inset.l, inset.r)
			r := t.blockChildren(computeSize, items, outerW, percentH, inset.l, rInset,
				rBorder, rPadding, st, ownCollapse, isScrollContainer(st), c, trial)
			return fmax(r.height, c.floatHeight+rInset.b), r.lines
		})
	}
	res := t.blockChildren(in.run, items, outerW, percentH, inset.l, rInset,
		rBorder, rPadding, st, ownCollapse, isScrollContainer(st), ctx, clamp)
	if ctx.root || newBFC {
		res.height = fmax(res.height, ctx.floatHeight+rInset.b)
	}
	if n.box.Node != nil && n.box.Node.Tag() == dom.Fieldset {
		n.captionRoom = [2]float32{}
		if legend != nil && legend.final != nil {
			n.captionRoom[0] = fmax(0, (legend.final.size.h-rBorder.t)/2)
		}
	}
	outerH := mMax(or(known.h, mClamp(sizedHeight(st, res.height, adj.h, in.parent.h), minSize.h, maxSize.h)),
		pbSize.h)
	final := size{outerW, outerH}
	constrainedByMin := some(minSize.h) && minSize.h > 0 && minSize.h >= outerH
	ownBottomCollapse := ownCollapse[1] && !constrainedByMin
	alignContent := contentAlign(n)
	if alignContent != style.AlignNormal {
		innerH := outerH - rInset.vertical()
		contentH := res.height - rInset.vertical()
		free := innerH - contentH
		k := applyAlignmentFallback(free, 1, align(alignContent))
		offset := computeAlignmentOffset(free, 1, 0, k, false, true) + n.lowered
		if some(res.firstBaseline) {
			res.firstBaseline += offset
		}
		if some(res.lastBaseline) {
			res.lastBaseline += offset
		}
		anyInFlow := false
		for _, it := range items {
			if it.final != nil {
				it.final.location.y += offset
				anyInFlow = true
			}
			if outOfFlow(it.position) {
				it.staticPosition.y += offset
			}
		}
		if anyInFlow {
			res.overflow = rect{}
			container := overflowContainer{
				scroll:   isScrollContainer(st),
				size:     final.sub(rBorder.sum()).sub(gutter.sum()),
				padding:  rPadding,
				reversed: fromEnd(st),
			}
			for _, it := range items {
				if it.final == nil {
					continue
				}
				l := it.final
				loc := point{l.location.x - rBorder.l - gutter.l, l.location.y - rBorder.t}
				res.overflow = res.overflow.union(overflowContribution(loc, l.size,
					l.overflow.rect, l.border, it.overflow, it.node.st, container, negativeEdges(l.margin)))
			}
		}
	}
	allCollapse := true
	for _, it := range items {
		if !outOfFlow(it.position) && it.float == style.FloatNone && !it.collapsesThrough {
			allCollapse = false
		}
	}
	out := output{
		size:             final,
		firstBaseline:    res.firstBaseline,
		lastBaseline:     res.lastBaseline,
		collapsesThrough: !preventsCollapse && outerH == 0 && allCollapse,
		lines:            res.lines,
		clamped:          res.clamped,
	}
	if ownCollapse[0] {
		out.topMargin = res.firstTop
	} else {
		out.topMargin = marginFrom(resolveOrZero(st.Margin.Top, in.parent.w))
	}
	if ownBottomCollapse {
		out.bottomMargin = res.lastBottom
	} else {
		out.bottomMargin = marginFrom(resolveOrZero(st.Margin.Bottom, in.parent.w))
	}
	if in.run == computeSize {
		return out
	}
	for _, it := range items {
		if it.final != nil {
			it.node.unrounded = *it.final
		}
	}
	var candidates []oofCandidate
	for _, it := range items {
		if outOfFlow(it.position) {
			inlineArea := span{rInset.l, outerW - rInset.r}
			blockArea := span{it.staticPosition.y, it.staticPosition.y}
			justify := it.justifySelf
			if justify == style.AlignAuto {
				justify = st.JustifyItems
			}
			self := it.alignSelf
			if self == style.AlignAuto {
				self = style.AlignNormal
			}
			candidates = append(candidates, oofCandidate{
				node:     it.node,
				order:    it.order,
				position: it.position,
				x:        startOf(staticFromAlignment(align(justify), inlineArea), st),
				y:        staticFromAlignment(align(self), blockArea),
			})
		} else if len(it.oof) > 0 && it.final != nil {
			candidates = append(candidates, translateCandidates(it.oof, it.final.location)...)
		}
	}
	areaInset := rBorder.add(gutter)
	out.oof = candidates
	out.area = &oofArea{size: final.sub(areaInset.sum()), offset: point{areaInset.l, areaInset.t}}
	if isScrollContainer(st) {
		res.overflow = res.overflow.union(rect{r: rPadding.r, b: rPadding.b})
	}
	out.overflow = finalizeOverflow(res.overflow,
		final.sub(rBorder.sum()).sub(gutter.sum()), fromEnd(st), isScrollContainer(st))
	for i, k := range n.kids {
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: uint32(i)}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, inherentSize)
		}
	}
	return out
}

// renderedLegend is the legend a fieldset sets on its top border: its first box, when
// that is a legend in the flow.
// sizedHeight is the height of a block whose content takes h, its padding and
// border included: a calc-size() height worked on that size, in the box its
// box-sizing names (adj being the padding and border outside it), and h itself
// for any other.
func sizedHeight(st *style.Computed, h, adj, parent float32) float32 {
	l := st.Size.Height
	if !l.Sized {
		return h
	}
	return l.Scale*(h-adj) + l.Px + l.Percent*or(parent, 0)/100 + adj
}

func renderedLegend(n *node, items []*blockItem) *blockItem {
	if e := n.box.Node; e == nil || e.Tag() != dom.Fieldset || len(items) == 0 {
		return nil
	}
	it := items[0]
	if e := it.node.box.Node; e == nil || e.Tag() != dom.Legend || outOfFlow(it.position) || it.float != style.FloatNone {
		return nil
	}
	return it
}

func negativeEdges(r rect) rect {
	return rect{l: fmin(r.l, 0), r: fmin(r.r, 0), t: fmin(r.t, 0), b: fmin(r.b, 0)}
}

func staticFromAlignment(a alignment, area span) staticPosition {
	edge := func(k style.Align) staticEdge {
		switch k {
		case style.AlignCenter, style.AlignAnchorCenter:
			return edgeCenter
		case style.AlignEnd, style.AlignFlexEnd:
			return edgeEnd
		// they sit at the start
		case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignFlexStart,
			style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
		}
		return edgeStart
	}
	fallback := a.keyword
	if a.safe {
		fallback = style.AlignStart
	}
	return staticPosition{area: area,
		align: staticAlign{keyword: edge(a.keyword), safe: a.safe, unsafe: a.unsafe, fallback: edge(fallback)}}
}

func (t *tree) blockItems(n *node, inner size) []*blockItem {
	var items []*blockItem
	order := uint32(0)
	for _, k := range n.kids {
		st := k.st
		if st.Display == style.DisplayNone {
			continue
		}
		ratio := aspectRatio(st)
		padding := edges(st.Padding, inner.w)
		border := k.borders()
		pbSum := padding.add(border).sum()
		adj := boxSizingAdjustment(st, pbSum)
		same := st.Display == style.Block && !outOfFlow(st.Position) && st.Float == style.FloatNone &&
			!blockEstablishesBFC(st) && !standsBesideFloats(k) && k.inner == nil
		items = append(items, &blockItem{
			node:        k,
			order:       order,
			inSameBFC:   same,
			float:       st.Float,
			clear:       st.Clear,
			sizeStyle:   st.Size,
			size:        resolveSize(st.Size, inner).withAspectRatio(ratio).mAdd(adj),
			minSize:     stretched(resolveSize(st.MinSize, inner).withAspectRatio(ratio).mAdd(adj), st.MinSize, st, inner),
			maxSize:     stretched(resolveMax(st.MaxSize, inner).withAspectRatio(ratio).mAdd(adj), st.MaxSize, st, inner),
			overflow:    st.Overflow,
			position:    st.Position,
			alignSelf:   st.AlignSelf,
			justifySelf: st.JustifySelf,
			padding:     padding,
			border:      border,
			pbSum:       pbSum,
		})
		order++
	}
	return items
}

func relativeOffset(it *blockItem, parent size) point {
	if it.position != style.Relative {
		return point{}
	}
	return insetOffset(it.node.st, parent)
}

// insetOffset is how far position: relative moves a box of style st in a containing block
// of size parent.
func insetOffset(st *style.Computed, parent size) point {
	l, r := resolve(st.Inset.Left, parent.w), resolve(st.Inset.Right, parent.w)
	tp, b := resolve(st.Inset.Top, parent.h), resolve(st.Inset.Bottom, parent.h)
	return point{or(l, or(-r, 0)), or(tp, or(-b, 0))}
}

// inlineOffset is how far position: relative moves an inline element of style st, nothing
// for another position.
func inlineOffset(st *style.Computed, parent size) geom.Vec2 {
	if st.Position != style.Relative {
		return geom.Vec2{}
	}
	o := insetOffset(st, parent)
	return geom.Vec2{X: o.x, Y: o.y}
}

func collapsibleIf(same bool) [2]bool { return [2]bool{same, same} }

func (t *tree) blockContentWidth(items []*blockItem, availW space) float32 {
	maxW := float32(0)
	floats := floatWidths{avail: availW}
	for _, it := range items {
		if outOfFlow(it.position) {
			continue
		}
		known := it.size.mClamp(it.minSize, it.maxSize)
		marginX := edges(it.node.st.Margin, availW.value()).horizontal()
		w := known.w
		if !some(w) {
			itemAvail := availW.mSub(marginX)
			if k, s := resolveSizingKeyword(it.sizeStyle.Width, none, none); k == keywordMeasure {
				itemAvail = s
			}
			w = t.measureChildSize(it.node, known, noSize, spaces{itemAvail, minContentSpace},
				inherentSize, horizontal, collapsibleIf(it.inSameBFC))
		}
		if st := it.node.st; it.node.replaced && (st.Size.Width.IsPercent || st.MaxSize.Width.IsPercent) {
			// A picture as wide as a share of the width being found gives way to it:
			// none of it at the box's narrowest, as Chrome's.
			switch availW.kind {
			case minContent:
				w = 0
			case definite:
				w = fmin(w, availW.v-marginX)
			case maxContent:
			}
		}
		w = fmax(w, it.pbSum.w) + marginX
		if it.float != style.FloatNone {
			floats.add(w, sideOf(it.float), it.clear)
			continue
		}
		if run := floats.leading(); run > 0 && availW.kind != minContent && it.node.inline != nil &&
			it.node.box.Kind == box.AnonymousBlock {
			// The floats before text are on its lines, as Chrome's inline formatting
			// context holds them: side by side with it.
			wide := w + run
			if availW.definite() {
				wide = fmax(fmin(wide, availW.v), w)
			}
			w = wide
		}
		floats.run = [2]float32{}
		maxW = fmax(maxW, w)
	}
	return fmax(maxW, floats.result())
}

type blockResult struct {
	overflow      rect
	height        float32
	firstTop      marginSet
	lastBottom    marginSet
	firstBaseline float32
	lastBaseline  float32
	lines         int
	clamped       bool
}

// slotBesideFloats is the first room at or below minY wide enough for the border box of
// it, which no float may overlap; an auto width is never less than the margins take
// back.
func (t *tree) slotBesideFloats(ctx *blockContext, it *blockItem, parent size, minY float32, margin rect) bfcSlot {
	minAuto := -margin.horizontal()
	after := -1
	for {
		slot := ctx.findBFCSlot(minY, [2]float32{margin.l, margin.r}, it.clear, after)
		if slot.segment < 0 {
			return slot
		}
		stretch := fmax(fmax(slot.stretchWidth, minAuto), 0)
		w := it.size.w
		switch {
		case some(w):
		case standsBesideFloats(it.node) || it.node.st.Table:
			w = t.measureChildSize(it.node, noSize, parent, spaces{def(stretch), maxContentSpace},
				inherentSize, horizontal, collapsibleIf(false))
		default:
			w = stretch
		}
		if mClamp(w, it.minSize.w, it.maxSize.w) <= slot.borderWidth+fitTolerance {
			return slot
		}
		after = slot.segment
	}
}

//nolint:cyclop,funlen,gocognit,gocyclo,maintidx // Taffy's perform_final_layout_on_in_flow_children, kept in its shape
func (t *tree) blockChildren(run runMode, items []*blockItem, outerW, percentH, insetL float32,
	rInset, rBorder, rPadding rect, st *style.Computed, ownCollapse [2]bool, scroll bool,
	ctx *blockContext, clamp clampIn) blockResult {
	innerW := fmax(outerW-rInset.horizontal(), 0)
	container := overflowContainer{scroll: scroll, size: size{innerW + rPadding.horizontal(), 0}, padding: rPadding,
		reversed: fromEnd(st)}
	percentH = mSub(percentH, rInset.vertical())
	parent := size{innerW, percentH}
	if ctx.root {
		ctx.bfc.floats.width = outerW
		ctx.setContentInset(rInset.l, rInset.r)
	}
	if !ownCollapse[0] {
		ctx.commitStrut()
	}
	var overflow rect
	committed := rInset.t
	yForAbsolute := rInset.t
	var firstTop, active marginSet
	collapsingWithFirst := true
	firstBaseline, lastBaseline := none, none
	// activeHasClearance is set while the margins waiting to collapse are those of an
	// empty element with clearance: they collapse with what follows, never out of the
	// block's bottom.
	activeHasClearance := false
	hasActive := ctx.active(committed)
	// left is the lines the clamp leaves the blocks to come, and clampEnd where the block
	// holding its last line ends: this block's content ends there. A line-clamp that hides
	// what follows ends it below that block's bottom margin, as Chrome: clampHeld is what
	// stood committed and active after it.
	left, clampEnd := clamp.lines, none
	var clampHeld struct {
		committed, floats float32
		active            marginSet
	}
	lastInFlow := -1
	for i, it := range items {
		if !outOfFlow(it.position) && it.float == style.FloatNone {
			lastInFlow = i
		}
	}
	// floats are the floats since the last block in the flow: those an anonymous block's
	// lines go round stand on its first line in Chrome.
	floats := 0
	// inLines are the floats since the last block in the flow that its lines place.
	var inLines []lineFloatItem
	for i, it := range items {
		if outOfFlow(it.position) {
			it.staticPosition = point{rInset.l, yForAbsolute}
			continue
		}
		cst := it.node.st
		margin := optionalEdges(cst.Margin, innerW)
		nonAuto := rect{or(margin.l, 0), or(margin.r, 0), or(margin.t, 0), or(margin.b, 0)}
		g := scrollbarGutter(cst)
		gutter := size{g.x, g.y}

		if it.float != style.FloatNone {
			if run == performLayout {
				it.node.discarded = clamp.hides && some(clampEnd)
			}
			hasActive = true
			// A float with an auto width shrinks to fit: the room left, between its
			// min-content and max-content widths.
			availW := fmax(innerW-nonAuto.horizontal(), 0)
			itemAvail, floatKnown := def(availW), noSize
			switch k, s := resolveSizingKeyword(it.sizeStyle.Width, availW, innerW); k {
			case keywordMeasure:
				itemAvail = s
			case keywordExact:
				floatKnown.w = mClamp(s.v, it.minSize.w, it.maxSize.w)
			case keywordNone:
			}
			lo := t.computeChild(it.node, input{
				run:             run,
				sizing:          inherentSize,
				axis:            axisBoth,
				known:           floatKnown,
				knownIsDefinite: bools{true, true},
				parent:          parent,
				avail:           spaces{itemAvail, maxContentSpace},
			})
			it.oof = lo.oof
			off := relativeOffset(it, parent)
			if lf := it.node.inLines; lf != nil {
				// placed by the lines it stands among, once they are broken
				lf.pending, lf.placed = true, false
				lf.box, lf.side, lf.clearing = lo.size.add(nonAuto.sum()), sideOf(it.float), it.clear
				it.final = &placed{order: it.order, size: lo.size, overflow: lo.overflow, scrollbar: gutter,
					padding: it.padding, border: it.border, margin: nonAuto}
				inLines = append(inLines, lineFloatItem{it: it, lf: lf, margin: nonAuto, off: off, out: lo})
				floats++
				continue
			}
			// Between collapsing margins a float sits as an empty block would, below the
			// margins before it, unless those leave through the block's top: then its
			// place still moves with the margins after it.
			adjoins := collapsingWithFirst && ownCollapse[0]
			y := committed
			if !adjoins {
				y += active.resolve()
			}
			loc := ctx.place(lo.size.add(nonAuto.sum()), y, sideOf(it.float), it.clear, adjoins)
			loc = point{loc.x + nonAuto.l + off.x, loc.y + nonAuto.t + off.y}
			it.final = &placed{
				order:     it.order,
				size:      lo.size,
				overflow:  lo.overflow,
				scrollbar: gutter,
				location:  loc,
				padding:   it.padding,
				border:    it.border,
				margin:    nonAuto,
			}
			contributionLoc := point{loc.x - rInset.l + rPadding.l, loc.y - rBorder.t}
			overflow = overflow.union(overflowContribution(contributionLoc, lo.size,
				lo.overflow.rect, it.border, it.overflow, it.node.st, container, negativeEdges(nonAuto)))
			floats++
			continue
		}

		yMarginOffset := float32(0)
		stretchW := fmax(innerW-nonAuto.horizontal(), 0)
		avoid, avoidW := point{}, float32(0)
		avoids, pushedBelow := false, false
		if !it.inSameBFC {
			if !collapsingWithFirst || !ownCollapse[0] {
				yMarginOffset = active.collapseWithMargin(nonAuto.t).resolve()
			}
			minY := committed + yMarginOffset
			avoid, avoidW = point{rInset.l, minY}, innerW
			if hasActive || ctx.active(minY) {
				minAuto := -nonAuto.horizontal()
				slot := t.slotBesideFloats(ctx, it, parent, minY, nonAuto)
				// Moved down past a float, its top margin no longer collapses with the
				// block's.
				pushedBelow = slot.y > minY
				hasActive = slot.segment >= 0
				avoids = true
				stretchW = fmax(fmax(slot.stretchWidth, minAuto), 0)
				avoid, avoidW = point{slot.x, slot.y}, slot.borderWidth
			}
		}
		justify := it.justifySelf
		if justify == style.AlignAuto {
			justify = st.JustifyItems
		}
		nonStretch := justify != style.AlignNormal && justify != style.AlignStretch
		widthStyle := it.sizeStyle.Width
		shrinks := nonStretch || it.node.role == roleTable || it.node.replaced || it.node.st.Table || it.node.across
		if shrinks && widthStyle.Kind == style.Auto {
			widthStyle = style.Length{Kind: style.FitContent}
		}
		keywordW := none
		availH := maxContentSpace
		if it.node.across && some(percentH) {
			// an orthogonal flow's lines are as long as the height its container gives
			availH = def(fmax(percentH-nonAuto.vertical(), 0))
		}
		switch k, s := resolveSizingKeyword(widthStyle, stretchW, innerW); k {
		case keywordMeasure:
			keywordW = t.measureChildSize(it.node, noSize, parent, spaces{s, availH},
				inherentSize, horizontal, collapsibleIf(it.inSameBFC))
		case keywordExact:
			keywordW = s.v
		case keywordNone:
		}
		known := it.size
		if it.sizeStyle.Height.Kind == style.Stretch && some(percentH) {
			known.h = fmax(percentH-nonAuto.vertical(), 0)
		}
		known.w = mClamp(or(or(known.w, keywordW), stretchW), it.minSize.w, it.maxSize.w)
		known = known.mClamp(it.minSize, it.maxSize)
		in := input{
			run:             run,
			sizing:          inherentSize,
			axis:            axisBoth,
			known:           known,
			knownIsDefinite: bools{true, true},
			parent:          parent,
			avail:           spaces{def(stretchW), availH},
			collapsible:     collapsibleIf(it.inSameBFC),
		}
		if left > 0 && it.inSameBFC {
			in.clamp = clampIn{lines: left, more: clamp.more || i < lastInFlow, hides: clamp.hides, bare: clamp.bare}
		}
		if run == performLayout {
			it.node.discarded = clamp.hides && some(clampEnd)
		}
		clearThreshold := ctx.clearedThreshold(it.clear)
		clearPos := or(clearThreshold, negInf)
		// childFloats is how far down the floats under a block of this context reach,
		// from its border top.
		childFloats := negInf
		var lo output
		if it.inSameBFC {
			// Where the block's border box stands in the context, its auto margins
			// sharing the room left; its top is estimated, the margins it collapses
			// through not known yet.
			width := fmax(known.w, it.pbSum.w)
			left := estimatedLeft(st, margin, nonAuto.l, fmax(0, stretchW-width)) + insetL
			estimated := committed + active.collapseWithMargin(nonAuto.t).resolve()
			childY := fmax(estimated, clearPos)
			child := ctx.sub(childY, [2]float32{left, outerW - width - left})
			c := it.node.inline
			if c.pendingFloats() {
				c.floatsFrom = &floatsFrom{floats: ctx.bfc.floats.clone(), floatHeight: child.floatHeight}
			}
			lo = t.computeChildIn(it.node, in, child)
			if c != nil {
				c.floatsFrom = nil
			}
			overflow = t.placeLineFloats(ctx, inLines, point{left, childY}, rInset, rBorder, rPadding, container,
				overflow)
			inLines = nil
			childFloats = child.floatHeight
			top := child.topAdjoiningFloats()
			ctx.adjoining = [2]bool{ctx.adjoining[0] || top[0], ctx.adjoining[1] || top[1]}
		} else {
			lo = t.computeChild(it.node, in)
		}
		overflow = t.placeLineFloats(ctx, inLines, point{rInset.l, committed}, rInset, rBorder, rPadding, container,
			overflow)
		inLines = nil
		it.oof = lo.oof
		finalSize := lo.size
		topSet := lo.topMargin.collapseWithMargin(or(margin.t, 0))
		bottomSet := lo.bottomMargin.collapseWithMargin(or(margin.b, 0))
		freeX := fmax(0, stretchW-finalSize.w)
		autoCount := 0
		if !some(margin.l) {
			autoCount++
		}
		if !some(margin.r) {
			autoCount++
		}
		autoSize := float32(0)
		if autoCount > 0 {
			autoSize = freeX / float32(autoCount)
		}
		resolved := rect{l: or(margin.l, autoSize), r: or(margin.r, autoSize), t: or(margin.t, 0), b: or(margin.b, 0)}
		if autoCount == 0 {
			resolved.l += blockAlignOffset(st, freeX)
		}
		if over := stretchW - finalSize.w; over < 0 && st.Dir == style.RTL {
			// over-constrained, the left margin gives way: the box overflows its left
			resolved.l += over
		}
		insetOffset := relativeOffset(it, parent)
		if it.inSameBFC && (!collapsingWithFirst || !ownCollapse[0]) {
			yMarginOffset = active.collapseWithSet(topSet).resolve()
		}
		// Clearance comes in when the border top, placed by its margins alone, is not
		// below the floats cleared: the border top then goes to their bottom, or stays
		// where it was when lower. A float whose place still moves with these margins
		// forces it.
		hasClearance := false
		if it.inSameBFC && some(clearThreshold) {
			hypothetical := committed + active.collapseWithSet(topSet).resolve()
			if ctx.hasAdjoining(it.clear) || hypothetical < clearThreshold {
				hasClearance = true
				escaped := float32(0)
				if collapsingWithFirst && ownCollapse[0] {
					escaped = active.resolve()
				}
				yMarginOffset = clearThreshold - committed - escaped
			}
		}
		it.collapsesThrough = lo.collapsesThrough && !hasClearance
		if it.inSameBFC {
			it.staticPosition = point{rInset.l, fmax(committed+active.resolve(), clearPos)}
		} else {
			it.staticPosition = avoid
		}
		var loc point
		if it.inSameBFC {
			loc = point{rInset.l + insetOffset.x + resolved.l, committed + yMarginOffset + insetOffset.y}
		} else {
			// Beside floats the margins are in the room found already, but for what
			// auto margins add.
			extra := resolved.l
			if avoids {
				extra -= nonAuto.l
			}
			loc = point{avoid.x + extra + insetOffset.x, avoid.y + insetOffset.y}
		}
		if it.inSameBFC {
			ctx.floatHeight = fmax(ctx.floatHeight, loc.y-insetOffset.y+childFloats)
		}
		hasAutoX := !some(margin.l) || !some(margin.r)
		outerItemW := finalSize.w + resolved.horizontal()
		if nonStretch && !hasAutoX {
			start, free := rInset.l, innerW-outerItemW
			if avoids {
				start, free = avoid.x-resolved.l, avoidW-finalSize.w
			}
			off := float32(0)
			switch justify {
			case style.AlignCenter, style.AlignAnchorCenter:
				off = free / 2
			case style.AlignEnd, style.AlignFlexEnd:
				off = free
			// they stay at the start
			case style.AlignNormal, style.AlignAuto, style.AlignStretch, style.AlignStart, style.AlignFlexStart,
				style.AlignBaseline, style.AlignSpaceBetween, style.AlignSpaceAround, style.AlignSpaceEvenly:
			}
			if st.Dir == style.RTL && justify != style.AlignCenter {
				off = free - off
			}
			loc.x = start + nonAuto.l + insetOffset.x + off
		}
		if !some(firstBaseline) {
			if b := childFirstBaseline(lo.firstBaseline, finalSize.h, it.overflow.Height); some(b) {
				firstBaseline = loc.y - insetOffset.y + b
			}
		}
		if b := childLastBaseline(lo, finalSize.h, it.overflow.Height); some(b) {
			lastBaseline = loc.y - insetOffset.y + b
		}
		if avoids {
			if !some(margin.l) {
				resolved.l += avoid.x - rInset.l
			}
			if !some(margin.r) {
				resolved.r += outerW - rInset.r - avoid.x - avoidW
			}
		}
		it.final = &placed{
			order:     it.order,
			size:      finalSize,
			overflow:  lo.overflow,
			scrollbar: gutter,
			location:  loc,
			padding:   it.padding,
			border:    it.border,
			margin:    resolved,
		}
		contributionLoc := point{loc.x - rInset.l + rPadding.l, loc.y - rBorder.t}
		overflow = overflow.union(overflowContribution(contributionLoc, finalSize,
			lo.overflow.rect, it.border, it.overflow, it.node.st, container, negativeEdges(resolved)))
		if run == performLayout && floats > 0 {
			t.raiseFloats(items[:i], floats, it.node, finalSize.w)
		}
		floats = 0
		if in.clamp.lines > 0 {
			left -= lo.lines
			if lo.clamped {
				left, clampEnd = 0, loc.y-insetOffset.y+finalSize.h
				clampHeld.floats = ctx.floatHeight
			}
		}
		if collapsingWithFirst && (pushedBelow || hasClearance) {
			collapsingWithFirst = false
		} else if collapsingWithFirst {
			if it.collapsesThrough {
				firstTop = firstTop.collapseWithSet(topSet).collapseWithSet(bottomSet)
			} else {
				firstTop = firstTop.collapseWithSet(topSet)
				collapsingWithFirst = false
			}
		}
		if it.collapsesThrough {
			active = active.collapseWithSet(topSet).collapseWithSet(bottomSet)
			yForAbsolute = committed + finalSize.h + yMarginOffset
			continue
		}
		committed = loc.y - insetOffset.y + finalSize.h
		if hasClearance && lo.collapsesThrough {
			// An empty element with clearance keeps its border top where it cleared,
			// its collapsed margins reaching below it.
			committed -= topSet.resolve()
			active = topSet.collapseWithSet(bottomSet)
			activeHasClearance = true
		} else {
			active = bottomSet
			activeHasClearance = false
		}
		if it.legend {
			// What follows a fieldset's legend starts below it, or below the border where
			// that is thicker, then the padding: no margin collapses across them.
			committed = fmax(committed+active.resolve(), rBorder.t) + rPadding.t
			active = marginSet{}
		}
		if lo.clamped && in.clamp.lines > 0 {
			clampHeld.committed, clampHeld.active = committed, active
		}
		yForAbsolute = committed + active.resolve()
		ctx.commitStrut()
	}
	if some(clampEnd) {
		// the floats after the clamp make the block no taller, as Chrome's
		ctx.floatHeight = clampHeld.floats
	}
	if some(clampEnd) && clamp.hides {
		committed, active, activeHasClearance = clampHeld.committed, clampHeld.active, false
	}
	lastBottom := active
	bottomOffset := lastBottom.resolve()
	switch {
	case activeHasClearance:
		lastBottom = marginSet{}
	case ownCollapse[1]:
		bottomOffset = 0
	}
	overflow = t.placeLineFloats(ctx, inLines, point{rInset.l, committed}, rInset, rBorder, rPadding, container,
		overflow)
	committed += rInset.b + bottomOffset
	if some(clampEnd) && !clamp.hides {
		committed, lastBottom = clampEnd+rInset.b, marginSet{}
	}
	return blockResult{
		overflow:      overflow,
		height:        fmax(0, committed),
		firstTop:      firstTop,
		lastBottom:    lastBottom,
		firstBaseline: firstBaseline,
		lastBaseline:  lastBaseline,
		lines:         clamp.lines - max(left, 0),
		clamped:       some(clampEnd),
	}
}

// estimatedLeft is how far right of its room a block with margins margin (left the
// resolved left one) stands in a container of style st, free being the room it leaves.
func estimatedLeft(st *style.Computed, margin rect, left, free float32) float32 {
	switch {
	case !some(margin.l) && !some(margin.r):
		return free / 2
	case !some(margin.l):
		return free
	case some(margin.r):
		return left + blockAlignOffset(st, free)
	}
	return left
}

// blockAlignOffset is how far right of its place a block with free room beside it and
// no auto margin stands in the container of style st: at its end under direction rtl,
// where the left margin gives way, unless the container's text-align places blocks
// (-webkit-left, -webkit-center, -webkit-right).
func blockAlignOffset(st *style.Computed, free float32) float32 {
	switch st.TextAlign {
	case style.BlockCenter:
		return free / 2
	case style.BlockRight:
		return free
	case style.BlockLeft:
		return 0
	case style.Start, style.End, style.Left, style.Right, style.Center, style.Justify:
	}
	if st.Dir == style.RTL {
		return free
	}
	return 0
}

// raiseFloats moves the last count floats of items, which stand before the anonymous
// block anon of width w, up by what text-box-trim takes off its first line, but for
// those its lines placed below it: Chrome lays them out on that line, which the trim
// raises with what stands on it.
func (t *tree) raiseFloats(items []*blockItem, count int, anon *node, w float32) {
	c := anon.inline
	if c == nil || anon.box.Kind != box.AnonymousBlock {
		return
	}
	top, _ := t.trims(c, c.lines(w))
	if top == 0 {
		return
	}
	for i := len(items) - 1; i >= 0 && count > 0; i-- {
		if it := items[i]; it.float != style.FloatNone && !outOfFlow(it.position) && it.final != nil {
			if lf := it.node.inLines; lf == nil || lf.first {
				it.final.location.y -= top
			}
			count--
		}
	}
}

// lineFloatItem is a float the lines after it place, with what its place adds to where
// they put its margin box.
type lineFloatItem struct {
	it     *blockItem
	lf     *lineFloat
	margin rect
	off    point
	out    output
}

// placeLineFloats gives the floats the lines of the block at corner placed their places
// in its parent, and places at its top those they did not, and returns the parent's
// overflow with theirs.
func (t *tree) placeLineFloats(ctx *blockContext, floats []lineFloatItem, corner point,
	rInset, rBorder, rPadding rect, container overflowContainer, overflow rect) rect {
	for _, f := range floats {
		at := point{corner.x + f.lf.at.x, corner.y + f.lf.at.y}
		if !f.lf.placed {
			at = ctx.place(f.lf.box, corner.y, f.lf.side, f.lf.clearing, false)
		}
		f.lf.pending = false
		loc := point{at.x + f.margin.l + f.off.x, at.y + f.margin.t + f.off.y}
		f.it.final.location = loc
		contributionLoc := point{loc.x - rInset.l + rPadding.l, loc.y - rBorder.t}
		overflow = overflow.union(overflowContribution(contributionLoc, f.out.size,
			f.out.overflow.rect, f.it.border, f.it.overflow, f.it.node.st, container, negativeEdges(f.margin)))
	}
	return overflow
}
