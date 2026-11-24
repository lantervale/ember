package paint

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// ControlPartsAt is what the pointer at at is over of b, a range, a
// progress or a meter whose border box's top-left is at origin, as
// Chrome hovers the elements of its shadow tree: dom.ContainerHover over
// a range's slider container or a bar's inner element, with TrackHover
// over its track or bar, and ThumbHover over its thumb or value. A range
// of its own look has its track and thumb; any other box has no parts.
func ControlPartsAt(b *box.Box, origin, at geom.Vec2, opts ...Option) dom.State {
	n := b.Node
	if n == nil {
		return 0
	}
	p := &painter{}
	for _, o := range opts {
		o(p)
	}
	st := shown(b)
	e := entry{b: b, at: origin}
	c := content(e)
	var container, track, thumb geom.Rect
	switch {
	case n.Tag() == dom.Input && inputType(n) == rangeType:
		f := float32(p.rangeAt(n).Fraction())
		if st.Appearance.Native() {
			track, _, thumb = ownRange(c, st, f)
			container = c
			if thumb.Center().Sub(at).Len() > thumb.W/2 {
				thumb = geom.Rect{}
			}
			break
		}
		part := partRect(e, c)
		c.X, c.W = part.X, part.W
		l := rangeRects(c, st, f)
		container, track, thumb = l.container, l.track, l.thumb
	case n.Tag() == dom.Progress && !st.Appearance.Native():
		parts := partsOf(st)
		f, known := progressOf(p.machine(n), n)
		if !known {
			f = 0
		}
		container, track, thumb = barRects(partRect(e, c), parts.Inner, parts.Bar, parts.Value, f, st.Dir == style.RTL,
			false)
	case n.Tag() == dom.Meter && !st.Appearance.Native():
		m := meterOf(p.machine(n), n)
		parts := partsOf(st)
		container, track, thumb = barRects(partRect(e, c), parts.Inner, parts.Bar, parts.Regions[m.Region()],
			float32(m.Fraction()), st.Dir == style.RTL, true)
	default:
		return 0
	}
	switch {
	case thumb.Contains(at):
		return dom.PartHover
	case track.Contains(at):
		return dom.ContainerHover | dom.TrackHover
	case container.Contains(at):
		return dom.ContainerHover
	}
	return 0
}

// RangeAlong is where along its track the pointer at at puts the value
// of b, a range whose border box's top-left is at origin, from 0 at its
// start to 1 at its end, as Chrome moves it: the thumb's middle under
// the pointer, across a range of appearance none's track less the
// thumb's length, down a vertical range and from the end of a reversed
// one.
func RangeAlong(b *box.Box, origin, at geom.Vec2, opts ...Option) float64 {
	p := &painter{}
	for _, o := range opts {
		o(p)
	}
	st := shown(b)
	e := entry{b: b, at: origin}
	c := content(e)
	fl := flowOfRange(st)
	lo, length := c.X, c.W
	if !st.Appearance.Native() {
		part := partRect(e, c)
		c.X, c.W = part.X, part.W
		l := rangeRects(c, st, 0)
		in := partContent(fl.style(trackOf(st)), fl.in(c, l.track))
		tw := fl.in(c, l.thumb).W
		lo, length = in.X+tw/2, in.W-tw
	} else if fl.vertical {
		lo, length = c.Y, c.H
	}
	x := at.X
	if fl.vertical {
		x = at.Y
	}
	if length <= 0 {
		return 0
	}
	f := float64(min(max((x-lo)/length, 0), 1))
	if fl.reversed {
		f = 1 - f
	}
	return f
}

// trackOf is st's range track, nil where no rule styles it.
func trackOf(st *style.Computed) *style.Computed {
	if st.Parts == nil {
		return nil
	}
	return st.Parts.Track
}
