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

// controlParts gives a range, a progress or a meter of style c the parts
// its sheets style, each inheriting from the part it is in, as in
// Chrome's shadow tree. A range of appearance none whose height is auto
// is as tall as its track, which is as tall as its thumb unless sized.
func (r *restyler) controlParts(c *style.Computed, n *dom.Node, rem float32) {
	var p style.ControlParts
	part := func(name string, in *style.Computed) *style.Computed {
		if in == nil {
			in = c
		}
		return r.part(n, in, rem, name, 0)
	}
	t, _ := n.Attr("type")
	switch tag := n.Tag(); {
	case tag == dom.Progress:
		p.Inner = r.part(n, c, rem, progressInner, 0)
		p.Bar = part(progressBar, p.Inner)
		p.Value = part(progressValue, p.Bar)
	case tag == dom.Meter:
		p.Inner = r.part(n, c, rem, meterInner, 0)
		p.Bar = part(meterBar, p.Inner)
		for i, name := range [...]string{meterOptimum, meterSuboptimum, meterEvenLessGood} {
			p.Regions[i] = part(name, p.Bar)
		}
	case tag == dom.Input && strings.EqualFold(t, "range"):
		p.Track = r.part(n, c, rem, sliderTrack, 0)
		p.Thumb = part(sliderThumb, p.Track)
		if !c.Appearance.Native() && c.Size.Height.Kind == style.Auto {
			h := trackHeight(p.Track, p.Thumb)
			if c.BoxSizing == style.BorderBox {
				h += vertical(c)
			}
			c.Size.Height = style.Px(h)
		}
	default:
		return
	}
	if p != (style.ControlParts{}) {
		c.Parts = &p
	}
}

// trackHeight is the height of a range's track with its margins: its
// own, else its thumb's.
func trackHeight(track, thumb *style.Computed) float32 {
	h := float32(nativeThumb)
	if thumb != nil && !thumb.Appearance.Native() && thumb.Display != style.DisplayNone {
		h = outerHeight(thumb, 0)
	}
	if track == nil {
		return h
	}
	return outerHeight(track, h)
}

// outerHeight is the height of a part of style st with its margins, its
// content content high when its height is auto.
func outerHeight(st *style.Computed, content float32) float32 {
	h, ok := st.Size.Height.Resolve(0)
	switch {
	case !ok || st.Size.Height.Kind == style.Auto || st.Size.Height.Percent != 0:
		h = content + vertical(st)
	case st.BoxSizing == style.ContentBox:
		h += vertical(st)
	}
	top, _ := st.Margin.Top.Resolve(0)
	bottom, _ := st.Margin.Bottom.Resolve(0)
	return max(h, vertical(st)) + top + bottom
}

// vertical is st's padding and borders above and below its content.
func vertical(st *style.Computed) float32 {
	top, _ := st.Padding.Top.Resolve(0)
	bottom, _ := st.Padding.Bottom.Resolve(0)
	return top + bottom + st.Border.Top + st.Border.Bottom
}
