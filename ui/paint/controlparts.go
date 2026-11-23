package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// nativeThumb is the side of Chrome's own range thumb, kept by a range of
// appearance none whose ::-webkit-slider-thumb isn't.
const nativeThumb = 16

// partContent is r less st's borders and padding: the content box of a part
// whose border box is r.
func partContent(st *style.Computed, r geom.Rect) geom.Rect {
	if st == nil {
		return r
	}
	pad := func(l style.Length) float32 { v, _ := l.Resolve(r.W); return v }
	l, t := st.Border.Left+pad(st.Padding.Left), st.Border.Top+pad(st.Padding.Top)
	return geom.Rect{X: r.X + l, Y: r.Y + t, W: max(r.W-l-st.Border.Right-pad(st.Padding.Right), 0),
		H: max(r.H-t-st.Border.Bottom-pad(st.Padding.Bottom), 0)}
}

// barRects are a progress's or a meter's parts of appearance none in its
// content box c, as Chrome lays them out: the inner element filling c,
// the bar filling the inner's content box (a meter's half as tall,
// centred), and the value along the bar's content box f of its width
// from its start, its end in rtl, never narrower than its own padding,
// borders and min-width; an indeterminate progress's is as at 0.
func barRects(c geom.Rect, inner, bar, value *style.Computed, f float32, rtl, meter bool) (geom.Rect, geom.Rect,
	geom.Rect) {
	in := partContent(inner, c)
	if meter {
		in = meterBar(in, bar)
	}
	v := partContent(bar, in)
	w := v.W * min(max(f, 0), 1)
	if value != nil {
		least, _ := value.MinSize.Width.Resolve(v.W)
		pad := func(l style.Length) float32 { x, _ := l.Resolve(v.W); return x }
		w = max(w, least, value.Border.Left+value.Border.Right+pad(value.Padding.Left)+pad(value.Padding.Right))
	}
	if rtl {
		v.X += v.W - w
	}
	v.W = w
	return c, in, v
}

// barParts paints a progress or a meter of appearance none in its
// content box c (barRects).
func (p *painter) barParts(c geom.Rect, inner, bar, value *style.Computed, f float32, rtl, meter bool,
	alpha float32) {
	ri, rb, rv := barRects(c, inner, bar, value, f, rtl, meter)
	p.part(inner, ri, alpha)
	p.part(bar, rb, alpha)
	p.part(value, rv, alpha)
}

// rangeFlow is the way a range's value runs: down its box where it is
// vertical, from the end where reversed.
type rangeFlow struct{ vertical, reversed, fromRight bool }

// flowOfRange is the way st's range runs: along its inline axis, from
// its end in rtl and up a sideways-lr line; a range of its own look of
// appearance slider-vertical in horizontal lines runs up, as Chrome's.
func flowOfRange(st *style.Computed) rangeFlow {
	switch {
	case st.Mode.Vertical():
		return rangeFlow{vertical: true, reversed: (st.Dir == style.RTL) != st.Mode.Upward(),
			fromRight: st.Mode.BlockFromRight()}
	case st.Appearance == style.AppearanceSliderVertical:
		return rangeFlow{vertical: true, reversed: true}
	}
	return rangeFlow{reversed: st.Dir == style.RTL}
}

// in is r in the flow of a range whose content box is c: along it as x,
// across from its block's start as y.
func (fl rangeFlow) in(c, r geom.Rect) geom.Rect {
	if !fl.vertical {
		return r
	}
	t := geom.Rect{X: r.Y, Y: r.X, W: r.H, H: r.W}
	if fl.fromRight {
		t.Y = 2*c.X + c.W - r.X - r.W
	}
	return t
}

// out is r, of the flow of a range whose content box is c, on the page.
func (fl rangeFlow) out(c, r geom.Rect) geom.Rect {
	if !fl.vertical {
		return r
	}
	t := geom.Rect{X: r.Y, Y: r.X, W: r.H, H: r.W}
	if fl.fromRight {
		t.X = 2*c.X + c.W - r.Y - r.H
	}
	return t
}

// style is st read in fl: a vertical range's part with its width as its
// height and its block's start side as its top, so the parts of every
// range lay out as a horizontal one's.
func (fl rangeFlow) style(st *style.Computed) *style.Computed {
	if !fl.vertical || st == nil {
		return st
	}
	t := *st
	t.Size.Width, t.Size.Height = st.Size.Height, st.Size.Width
	t.MinSize.Width, t.MinSize.Height = st.MinSize.Height, st.MinSize.Width
	m, pd, b := st.Margin, st.Padding, st.Border
	if fl.fromRight {
		m.Left, m.Right, pd.Left, pd.Right, b.Left, b.Right = m.Right, m.Left, pd.Right, pd.Left, b.Right, b.Left
	}
	t.Margin = style.Sides[style.Length]{Top: m.Left, Right: m.Bottom, Bottom: m.Right, Left: m.Top}
	t.Padding = style.Sides[style.Length]{Top: pd.Left, Right: pd.Bottom, Bottom: pd.Right, Left: pd.Top}
	t.Border = style.Sides[float32]{Top: b.Left, Right: b.Bottom, Bottom: b.Right, Left: b.Top}
	return &t
}

// rangeLayout is where a range of appearance none lays its parts.
type rangeLayout struct {
	container, track, thumb geom.Rect
	// native is a thumb of Chrome's own look, a disc in the thumb rect.
	native bool
}

// rangeRects lays a range of appearance none and style st out in its
// content box c, as Chrome: its slider container filling c less its
// margins; the track across the container's content box, as thick as its
// thumb unless sized and centred across it; and the thumb at f along the
// track's content box less its own length, from its start and its
// margin further. A vertical range is laid out in its own flow.
func rangeRects(c geom.Rect, st *style.Computed, f float32) rangeLayout {
	fl := flowOfRange(st)
	var container, track, thumb *style.Computed
	if parts := st.Parts; parts != nil {
		container, track, thumb = fl.style(parts.Container), fl.style(parts.Track), fl.style(parts.Thumb)
	}
	fc := fl.in(c, c)
	native := thumb == nil || thumb.Appearance.Native() || thumb.Display == style.DisplayNone
	tw, th := float32(nativeThumb), float32(nativeThumb)
	if !native {
		tw, th = partSize(thumb)
	}
	box := inset(fc, container)
	tr := trackRect(partContent(container, box), track, th)
	in := partContent(track, tr)
	at := min(max(f, 0), 1)
	if fl.reversed {
		at = 1 - at
	}
	x := in.X + (in.W-tw)*at
	l := rangeLayout{container: fl.out(c, box), track: fl.out(c, tr), native: native}
	if native {
		l.thumb = fl.out(c, geom.Rect{X: x, Y: in.Y, W: tw, H: th})
		return l
	}
	ml, _ := thumb.Margin.Left.Resolve(in.W)
	mt, _ := thumb.Margin.Top.Resolve(in.W)
	l.thumb = fl.out(c, geom.Rect{X: x + ml, Y: in.Y + mt, W: tw, H: th})
	return l
}

// inset is r less st's margins, r where st is nil.
func inset(r geom.Rect, st *style.Computed) geom.Rect {
	if st == nil {
		return r
	}
	m := func(l style.Length) float32 { v, _ := l.Resolve(r.W); return v }
	l, t := m(st.Margin.Left), m(st.Margin.Top)
	return geom.Rect{X: r.X + l, Y: r.Y + t, W: max(r.W-l-m(st.Margin.Right), 0),
		H: max(r.H-t-m(st.Margin.Bottom), 0)}
}

// rangeParts paints a range of appearance none in its content box c
// (rangeRects): a thumb whose appearance isn't none is Chrome's, a disc
// in the range's accent, darker or lighter while hovered or pressed.
func (p *painter) rangeParts(c geom.Rect, st *style.Computed, s dom.State, f float32, alpha float32) {
	var container, track, thumb *style.Computed
	if parts := st.Parts; parts != nil {
		container, track, thumb = parts.Container, parts.Track, parts.Thumb
	}
	l := rangeRects(c, st, f)
	p.part(container, l.container, alpha)
	p.part(track, l.track, alpha)
	if !l.native {
		p.part(thumb, l.thumb, alpha)
		return
	}
	if thumb == nil || thumb.Display != style.DisplayNone {
		p.s.Circle(l.thumb.Center(), l.thumb.W/2, shaded(accentInk(st, 1), s.Shade(dom.ThumbHover, dom.ThumbActive), alpha))
	}
}

// shaded is a range's accent a in shade s, faded by alpha.
func shaded(a color.Color, s style.Shade, alpha float32) color.Color {
	return fade(style.AccentShade(a, s, style.LightRange(a)), alpha)
}

// trackRect is a range's track in its content box c: c's width less the
// track's side margins, its height its own or its content's, a thumb
// thumb high, centred down c with its margins.
func trackRect(c geom.Rect, track *style.Computed, thumb float32) geom.Rect {
	if track == nil {
		return geom.Rect{X: c.X, Y: c.Y + (c.H-thumb)/2, W: c.W, H: thumb}
	}
	m := func(l style.Length) float32 { v, _ := l.Resolve(c.W); return v }
	ml, mr, mt, mb := m(track.Margin.Left), m(track.Margin.Right), m(track.Margin.Top), m(track.Margin.Bottom)
	h := thumb + track.Border.Top + track.Border.Bottom + m(track.Padding.Top) + m(track.Padding.Bottom)
	if v, ok := track.Size.Height.Resolve(c.H); ok && track.Size.Height.Kind != style.Auto {
		h = v
		if track.BoxSizing == style.ContentBox {
			h += track.Border.Top + track.Border.Bottom + m(track.Padding.Top) + m(track.Padding.Bottom)
		}
	}
	return geom.Rect{X: c.X + ml, Y: c.Y + (c.H-h-mt-mb)/2 + mt, W: max(c.W-ml-mr, 0), H: h}
}

// partSize is the border box of a part of style st whose size its own
// width and height say, 0 where auto.
func partSize(st *style.Computed) (float32, float32) {
	pad := func(l style.Length) float32 { v, _ := l.Resolve(0); return v }
	pw := st.Border.Left + st.Border.Right + pad(st.Padding.Left) + pad(st.Padding.Right)
	ph := st.Border.Top + st.Border.Bottom + pad(st.Padding.Top) + pad(st.Padding.Bottom)
	side := func(l style.Length, pb float32) float32 {
		v, ok := l.Resolve(0)
		if !ok || l.Kind == style.Auto {
			return pb
		}
		if st.BoxSizing == style.ContentBox {
			v += pb
		}
		return max(v, pb)
	}
	return side(st.Size.Width, pw), side(st.Size.Height, ph)
}

// partInk is the background-color a part's rules give it, else ink: a
// meter of its own look takes its styled parts' colours, as Chrome's.
func partInk(st *style.Computed, ink color.Color, alpha float32) color.Color {
	if st != nil && st.Background.A > 0 {
		return fade(st.Background, alpha)
	}
	return ink
}

// meterBar is a meter's bar in its inner element's content box in: half
// as tall unless its own height is set, centred down it, as Chrome's.
func meterBar(in geom.Rect, bar *style.Computed) geom.Rect {
	h := in.H / 2
	if bar != nil && bar.Size.Height.Kind != style.Auto {
		if v, ok := bar.Size.Height.Resolve(in.H); ok {
			h = v
		}
	}
	return geom.Rect{X: in.X, Y: in.Y + (in.H-h)/2, W: in.W, H: h}
}
