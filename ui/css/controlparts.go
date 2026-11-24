package css

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// nativeThumb is the side of Chrome's own range thumb, which a range
// drawn from its CSS keeps unless its ::-webkit-slider-thumb's
// appearance is none.
const nativeThumb = 16

// rangeType is a range input's type attribute.
const rangeType = "range"

// controlParts gives a range, a progress or a meter of style c the parts
// its sheets style, each inheriting from the part it is in, as in
// Chrome's shadow tree. A range of appearance none whose height is auto
// is at least as tall as its track, which is as tall as its thumb unless
// sized: its min-height, so its ::before and ::after can make it taller.
func (r *restyler) controlParts(c *style.Computed, n *dom.Node, rem float32) {
	var p style.ControlParts
	st := n.State()
	in := func(hover, active dom.State) partState {
		var s partState
		if st.Has(hover) {
			s |= partHover
		}
		if st.Has(active) {
			s |= partActive
		}
		return s
	}
	outer, middle, inner := in(dom.ContainerHover, dom.ContainerActive), in(dom.TrackHover, dom.TrackActive),
		in(dom.ThumbHover, dom.ThumbActive)
	part := func(name string, in *style.Computed, s partState) *style.Computed {
		if in == nil {
			in = c
		}
		return r.part(n, in, rem, name, s)
	}
	t, _ := n.Attr("type")
	switch tag := n.Tag(); {
	case tag == dom.Progress:
		p.Inner = r.part(n, c, rem, progressInner, outer)
		p.Bar = part(progressBar, p.Inner, middle)
		p.Value = part(progressValue, p.Bar, inner)
	case tag == dom.Meter:
		p.Inner = r.part(n, c, rem, meterInner, outer)
		p.Bar = part(meterBar, p.Inner, middle)
		for i, name := range [...]string{meterOptimum, meterSuboptimum, meterEvenLessGood} {
			p.Regions[i] = part(name, p.Bar, inner)
		}
	case tag == dom.Input && strings.EqualFold(t, rangeType):
		p.Container = r.part(n, c, rem, sliderContainer, outer)
		p.Track = part(sliderTrack, p.Container, middle)
		p.Thumb = part(sliderThumb, p.Track, inner)
		if !c.Appearance.Native() {
			minAcross(c, p.Container, p.Track, p.Thumb)
		}
	default:
		return
	}
	if p != (style.ControlParts{}) {
		c.Parts = &p
	}
}

// minAcross makes a range of appearance none and style c whose size
// across its track is auto at least as thick as its slider container,
// round its track, whose thickness is its thumb's unless sized: its
// min-height, its min-width where its lines are vertical.
func minAcross(c, container, track, thumb *style.Computed) {
	x := across{c.Mode.Vertical()}
	if x.size(c).Kind != style.Auto {
		return
	}
	t := float32(nativeThumb)
	if thumb != nil && !thumb.Appearance.Native() && thumb.Display != style.DisplayNone {
		t = x.outer(thumb, 0)
	}
	if track != nil {
		t = x.outer(track, t)
	}
	if container != nil {
		t = x.outer(container, t)
	}
	if c.BoxSizing == style.BorderBox {
		t += x.edges(c)
	}
	least := &c.MinSize.Height
	if x.vertical {
		least = &c.MinSize.Width
	}
	if m, ok := least.Resolve(0); least.Kind == style.Auto || ok && least.Percent == 0 && m < t {
		*least = style.Px(t)
	}
}

// across reads a part's box across a range's track: down a horizontal
// range, along a vertical one's width.
type across struct{ vertical bool }

// size is st's height, its width across a vertical track.
func (x across) size(st *style.Computed) style.Length {
	if x.vertical {
		return st.Size.Width
	}
	return st.Size.Height
}

// outer is the thickness of a part of style st with its margins, its
// content content thick when its size is auto.
func (x across) outer(st *style.Computed, content float32) float32 {
	l := x.size(st)
	v, ok := l.Resolve(0)
	switch {
	case !ok || l.Kind == style.Auto || l.Percent != 0:
		v = content + x.edges(st)
	case st.BoxSizing == style.ContentBox:
		v += x.edges(st)
	}
	a, b := st.Margin.Top, st.Margin.Bottom
	if x.vertical {
		a, b = st.Margin.Left, st.Margin.Right
	}
	ma, _ := a.Resolve(0)
	mb, _ := b.Resolve(0)
	return max(v, x.edges(st)) + ma + mb
}

// edges is st's padding and borders on either side of its content.
func (x across) edges(st *style.Computed) float32 {
	if x.vertical {
		l, _ := st.Padding.Left.Resolve(0)
		r, _ := st.Padding.Right.Resolve(0)
		return l + r + st.Border.Left + st.Border.Right
	}
	t, _ := st.Padding.Top.Resolve(0)
	b, _ := st.Padding.Bottom.Resolve(0)
	return t + b + st.Border.Top + st.Border.Bottom
}
