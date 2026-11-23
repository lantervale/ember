package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
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

// barParts paints a progress or a meter of appearance none in its
// content box c as Chrome lays out their parts: the inner element
// filling c, the bar filling the inner's content box (a meter's half as
// tall, centred), and the value the bar's content box cut to f of its
// width, from its end in rtl.
func (p *painter) barParts(c geom.Rect, inner, bar, value *style.Computed, f float32, rtl, meter bool,
	alpha float32) {
	p.part(inner, c, alpha)
	in := partContent(inner, c)
	if meter {
		in = meterBar(in, bar)
	}
	p.part(bar, in, alpha)
	v := partContent(bar, in)
	w := v.W * min(max(f, 0), 1)
	if rtl {
		v.X += v.W - w
	}
	v.W = w
	p.part(value, v, alpha)
}

// rangeParts paints a range of appearance none in its content box c as
// Chrome lays out its parts: the track across c, as tall as its thumb
// unless sized and centred down it, and the thumb at f along the track's
// content box less its own width, from its top and its margin-top
// further; a thumb whose appearance isn't none is Chrome's, a disc in
// the range's accent.
func (p *painter) rangeParts(c geom.Rect, st *style.Computed, f float32, alpha float32) {
	parts := st.Parts
	var track, thumb *style.Computed
	if parts != nil {
		track, thumb = parts.Track, parts.Thumb
	}
	native := thumb == nil || thumb.Appearance.Native() || thumb.Display == style.DisplayNone
	tw, th := float32(nativeThumb), float32(nativeThumb)
	if !native {
		tw, th = partSize(thumb)
	}
	tr := trackRect(c, track, th)
	p.part(track, tr, alpha)
	in := partContent(track, tr)
	at := min(max(f, 0), 1)
	if st.Dir == style.RTL {
		at = 1 - at
	}
	x := in.X + (in.W-tw)*at
	if native {
		if thumb == nil || thumb.Display != style.DisplayNone {
			p.s.Circle(geom.V2(x+tw/2, in.Y+th/2), tw/2, accentInk(st, alpha))
		}
		return
	}
	ml, _ := thumb.Margin.Left.Resolve(in.W)
	mt, _ := thumb.Margin.Top.Resolve(in.W)
	p.part(thumb, geom.Rect{X: x + ml, Y: in.Y + mt, W: tw, H: th}, alpha)
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
