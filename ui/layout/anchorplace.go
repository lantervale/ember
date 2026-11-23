package layout

import (
	"slices"

	"github.com/lantervale/ember/ui/internal/style"
)

// placedStyle is one position option of an element its anchors placed:
// its style with each anchor function worked out, and the containing
// block a position-area cut, with the inset-modified containing block in
// it, which the box must fit in for the option to be taken.
type placedStyle struct {
	st     *style.Computed
	area   size
	offset point
	imcb   rect
}

// lengthKind is the property a length is of, for an anchor function in
// it: which inset, or the axis of a size or a margin.
type lengthKind uint8

const (
	insetTop lengthKind = iota
	insetRight
	insetBottom
	insetLeft
	alongX
	alongY
)

// layoutOOFBox sizes and places one absolutely positioned box in its
// containing block; one that reads anchors with the first of its
// position options whose margin box fits its inset-modified containing
// block, in the order position-try-order gives, else its own.
func (t *tree) layoutOOFBox(c oofCandidate, area size, offset point,
	containerOverflow style.Size[style.Overflow], gridOwner *node) oofBoxLayout {
	n := c.node
	e := n.box.Node
	if t.anchors == nil || e == nil || !anchorUser(n.st) {
		lb, _ := t.placeOOFBox(c, area, offset, containerOverflow, gridOwner, nil)
		return lb
	}
	env, seen := t.anchors.of[e]
	if !seen {
		t.anchors.of[e] = nil
	}
	base := n.st
	defer func() { n.st = base }()
	options := append([]*style.Computed{base}, base.Anchoring.Tries...)
	placed := make([]placedStyle, len(options))
	for i, o := range options {
		placed[i] = env.place(o, area, offset)
	}
	order := make([]int, len(options))
	for i := range order {
		order[i] = i
	}
	switch base.Anchoring.Order {
	case style.TryMostWidth:
		slices.SortStableFunc(order, func(a, b int) int { return cmpDesc(placed[a].imcb.horizontalSpan(), placed[b].imcb.horizontalSpan()) })
	case style.TryMostHeight:
		slices.SortStableFunc(order, func(a, b int) int { return cmpDesc(placed[a].imcb.verticalSpan(), placed[b].imcb.verticalSpan()) })
	case style.TryNormal:
	}
	try := func(i int) (oofBoxLayout, bool) {
		p := placed[i]
		n.st = p.st
		n.cache.clear()
		return t.placeOOFBox(c, p.area, p.offset, containerOverflow, gridOwner, &p.imcb)
	}
	var lb oofBoxLayout
	fits := false
	for _, i := range order {
		if lb, fits = try(i); fits || len(order) == 1 {
			break
		}
	}
	if !fits && len(order) > 1 {
		lb, _ = try(order[0])
	}
	vis := base.Anchoring.Visibility
	n.discarded = vis&style.NoOverflow != 0 && !fits || vis&style.AnchorsVisible != 0 && env != nil && env.hidden
	return lb
}

func cmpDesc(a, b float32) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	}
	return 0
}

func (r rect) horizontalSpan() float32 { return r.r - r.l }
func (r rect) verticalSpan() float32   { return r.b - r.t }

// defaultFor is the default anchor st names, where a's layout placed it.
func (a *anchorsOf) defaultFor(st *style.Computed) (rect, bool) {
	if a == nil {
		return rect{}, false
	}
	switch st.Anchoring.Anchor {
	case style.AnchorNamed:
		r, ok := a.named[st.Anchoring.AnchorName]
		return r, ok
	case style.AnchorAuto, style.AnchorNormal:
		return a.implicit, a.hasImplicit
	case style.AnchorNone:
	}
	return rect{}, false
}

// place is the position option st of an element in a containing block of
// padding box area at offset in its border box, with a's anchors: its
// position-area's cell of the grid the default anchor cuts the
// containing block into, its anchor functions worked out against that,
// a length whose anchor is missing and has no fallback its property's
// initial value, and anchor-center the largest span centred on the
// anchor.
func (a *anchorsOf) place(st *style.Computed, area size, offset point) placedStyle {
	s := *st
	def, hasDef := a.defaultFor(st)
	if s.Anchoring.Area.Set && hasDef {
		at := rect{l: def.l - offset.x, r: def.r - offset.x, t: def.t - offset.y, b: def.b - offset.y}
		xs := [4]float32{min(0, at.l), at.l, at.r, max(area.w, at.r)}
		ys := [4]float32{min(0, at.t), at.t, at.b, max(area.h, at.b)}
		ar := s.Anchoring.Area
		x0, x1 := xs[ar.X.From], xs[ar.X.To+1]
		y0, y1 := ys[ar.Y.From], ys[ar.Y.To+1]
		offset = point{offset.x + x0, offset.y + y0}
		area = size{max(x1-x0, 0), max(y1-y0, 0)}
		for _, l := range []*style.Length{&s.Inset.Top, &s.Inset.Right, &s.Inset.Bottom, &s.Inset.Left} {
			if l.Kind == style.Auto {
				*l = style.Px(0)
			}
		}
		if s.JustifySelf == style.AlignNormal || s.JustifySelf == style.AlignAuto {
			s.JustifySelf = areaAlign(ar.X)
		}
		if s.AlignSelf == style.AlignNormal || s.AlignSelf == style.AlignAuto {
			s.AlignSelf = areaAlign(ar.Y)
		}
	}
	rel := rect{l: def.l - offset.x, r: def.r - offset.x, t: def.t - offset.y, b: def.b - offset.y}
	lookup := func(fn style.AnchorFn) (rect, bool) {
		var r rect
		ok := false
		if fn.Name == "" {
			r, ok = def, hasDef
		} else if a != nil {
			r, ok = a.named[fn.Name]
		}
		return rect{l: r.l - offset.x, r: r.r - offset.x, t: r.t - offset.y, b: r.b - offset.y}, ok
	}
	rtl := st.Dir == style.RTL
	work := func(l *style.Length, kind lengthKind, basis float32, initial style.Length) {
		if !l.Anchored() {
			return
		}
		v, ok := l.AtAnchors(basis, func(fn style.AnchorFn) (float32, bool) {
			r, found := lookup(fn)
			if !found {
				return 0, false
			}
			return anchorValue(fn, r, kind, area, rtl)
		})
		if ok {
			*l = style.Px(v)
		} else {
			*l = initial
		}
	}
	auto, noMax := style.Length{}, style.Length{Kind: style.None}
	work(&s.Inset.Top, insetTop, area.h, auto)
	work(&s.Inset.Bottom, insetBottom, area.h, auto)
	work(&s.Inset.Left, insetLeft, area.w, auto)
	work(&s.Inset.Right, insetRight, area.w, auto)
	work(&s.Size.Width, alongX, area.w, auto)
	work(&s.Size.Height, alongY, area.h, auto)
	work(&s.MinSize.Width, alongX, area.w, auto)
	work(&s.MinSize.Height, alongY, area.h, auto)
	work(&s.MaxSize.Width, alongX, area.w, noMax)
	work(&s.MaxSize.Height, alongY, area.h, noMax)
	work(&s.Margin.Left, alongX, area.w, style.Px(0))
	work(&s.Margin.Right, alongX, area.w, style.Px(0))
	work(&s.Margin.Top, alongY, area.w, style.Px(0))
	work(&s.Margin.Bottom, alongY, area.w, style.Px(0))
	imcb := rect{l: 0, r: area.w, t: 0, b: area.h}
	if v := resolve(s.Inset.Left, area.w); some(v) {
		imcb.l = v
	}
	if v := resolve(s.Inset.Right, area.w); some(v) {
		imcb.r = area.w - v
	}
	if v := resolve(s.Inset.Top, area.h); some(v) {
		imcb.t = v
	}
	if v := resolve(s.Inset.Bottom, area.h); some(v) {
		imcb.b = area.h - v
	}
	if hasDef {
		if s.JustifySelf == style.AlignAnchorCenter {
			s.Inset.Left, s.Inset.Right, s.JustifySelf = anchorCenter(imcb.l, imcb.r, (rel.l+rel.r)/2, area.w)
		}
		if s.AlignSelf == style.AlignAnchorCenter {
			s.Inset.Top, s.Inset.Bottom, s.AlignSelf = anchorCenter(imcb.t, imcb.b, (rel.t+rel.b)/2, area.h)
		}
	}
	if s.JustifySelf == style.AlignAnchorCenter {
		s.JustifySelf = style.AlignCenter
	}
	if s.AlignSelf == style.AlignAnchorCenter {
		s.AlignSelf = style.AlignCenter
	}
	return placedStyle{st: &s, area: area, offset: offset, imcb: imcb}
}

// areaAlign is how a box aligns in a position-area's span of tracks when
// its self-alignment is normal: towards the anchor, centred in the
// anchor's track, and on the anchor across all three.
func areaAlign(sp style.AreaSpan) style.Align {
	switch {
	case sp.From == 0 && sp.To == 2:
		return style.AlignAnchorCenter
	case sp.From == 1 && sp.To == 1:
		return style.AlignCenter
	case sp.To <= 1:
		return style.AlignEnd
	}
	return style.AlignStart
}

// anchorCenter is the insets of the largest span within [start, end]
// centred on c, the anchor's centre, in a containing block length long,
// and the alignment that centres the box in it.
func anchorCenter(start, end, c, length float32) (style.Length, style.Length, style.Align) {
	half := max(min(c-start, end-c), 0)
	return style.Px(c - half), style.Px(length - c - half), style.AlignCenter
}

// anchorValue is what the anchor function fn makes of an anchor at r, in
// the containing block's padding box of size cb, for a length of kind.
func anchorValue(fn style.AnchorFn, r rect, kind lengthKind, cb size, rtl bool) (float32, bool) {
	alongXAxis := kind == insetLeft || kind == insetRight || kind == alongX
	if fn.Size {
		switch fn.Side {
		case style.AnchorWidth, style.AnchorInline, style.AnchorSelfInline:
			return r.r - r.l, true
		case style.AnchorHeight, style.AnchorBlock, style.AnchorSelfBlock:
			return r.b - r.t, true
		case style.AnchorAxis:
			if alongXAxis {
				return r.r - r.l, true
			}
			return r.b - r.t, true
		}
		return 0, false
	}
	if kind == alongX || kind == alongY {
		return 0, false
	}
	lo, hi := r.t, r.b
	if alongXAxis {
		lo, hi = r.l, r.r
	}
	flipped := alongXAxis && rtl
	startSide := kind == insetTop || kind == insetLeft
	var at float32
	switch fn.Side {
	case style.AnchorTop, style.AnchorBottom:
		if alongXAxis {
			return 0, false
		}
		at = r.t
		if fn.Side == style.AnchorBottom {
			at = r.b
		}
	case style.AnchorLeft, style.AnchorRight:
		if !alongXAxis {
			return 0, false
		}
		at = r.l
		if fn.Side == style.AnchorRight {
			at = r.r
		}
	case style.AnchorStart, style.AnchorSelfStart:
		at = pick(flipped, hi, lo)
	case style.AnchorEnd, style.AnchorSelfEnd:
		at = pick(flipped, lo, hi)
	case style.AnchorInside:
		at = pick(startSide, lo, hi)
	case style.AnchorOutside:
		at = pick(startSide, hi, lo)
	case style.AnchorCenter:
		at = (lo + hi) / 2
	case style.AnchorPercent:
		if flipped {
			at = hi - fn.Percent/100*(hi-lo)
		} else {
			at = lo + fn.Percent/100*(hi-lo)
		}
	default:
		return 0, false
	}
	switch kind {
	case insetBottom:
		return cb.h - at, true
	case insetRight:
		return cb.w - at, true
	case insetTop, insetLeft, alongX, alongY:
	}
	return at, true
}

func pick(first bool, a, b float32) float32 {
	if first {
		return a
	}
	return b
}
