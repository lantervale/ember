package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// styledPart is a part's style: its background-color, then what set changes.
func styledPart(bg color.Color, set func(*style.Computed)) *style.Computed {
	s := style.Initial()
	s.Background, s.Visible = bg, true
	if set != nil {
		set(&s)
	}
	return &s
}

func TestAControlOfAppearanceNoneDrawsNoMark(t *testing.T) {
	for _, typ := range []string{"checkbox", "radio"} {
		b := ctl(dom.Input, 16, 16, "type", typ)
		b.Node.SetState(dom.Checked, true)
		b.Style.Appearance = style.AppearanceNone
		r := paintOne(b)
		assert.Empty(t, r.of("line", "circle", "rect"), "%s: an empty box", typ)
	}
}

func TestACheckedControlOfItsOwnLookTakesItsAccent(t *testing.T) {
	red, pale := color.Hex("#ee0044"), color.Hex("#ffee88")
	b := ctl(dom.Input, 13, 13, "type", "checkbox")
	b.Node.SetState(dom.Checked, true)
	b.Style.Color, b.Style.AccentColor, b.Style.AccentAuto = white, red, false
	r := paintOne(b)
	fill := r.of("round")
	require.Len(t, fill, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 13, H: 13}, fill[0].Rect, "the box filled with the accent")
	assert.Equal(t, red, fill[0].Color)
	assert.Equal(t, color.White, r.of("line")[0].Color, "a white tick on a dark accent")

	b.Style.AccentColor = pale
	assert.Equal(t, darkTick, paintOne(b).of("line")[0].Color, "a dark tick on a light one")

	b.Node.SetState(dom.Checked, false)
	assert.Empty(t, paintOne(b).of("round"), "unchecked, the box keeps its look")

	radio := ctl(dom.Input, 16, 16, "type", "radio")
	radio.Node.SetState(dom.Checked, true)
	radio.Style.AccentColor, radio.Style.AccentAuto = red, false
	radio.Style.Border = style.Sides[float32]{Top: 1, Right: 1, Bottom: 1, Left: 1}
	r = paintOne(radio)
	require.NotEmpty(t, r.of("border"))
	assert.Equal(t, red, r.of("border")[0].Color, "the ring")
	assert.Equal(t, red, r.of("circle")[0].Color, "the dot")
}

func TestAProgressOfAppearanceNoneDrawsItsParts(t *testing.T) {
	b := ctl(dom.Progress, 200, 10, "value", "0.5")
	b.Style.Appearance = style.AppearanceNone
	padded := styledPart(blue, func(s *style.Computed) { s.Padding = style.All(style.Px(2)) })
	b.Style.Parts = &style.ControlParts{Bar: padded, Value: styledPart(red, nil)}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 200, H: 10}, rects[0].Rect, "the bar fills the box")
	assert.Equal(t, blue, rects[0].Color)
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 98, H: 6}, rects[1].Rect, "the value, half the bar's content box")
	assert.Equal(t, red, rects[1].Color)

	b.Node.RemoveAttr("value")
	assert.Len(t, paintOne(b).of("rect"), 1, "indeterminate: the bar alone")
}

func TestAMetersBarIsHalfItsHeightCentred(t *testing.T) {
	b := ctl(dom.Meter, 100, 16, "value", "0.25")
	b.Style.Appearance = style.AppearanceNone
	b.Style.Parts = &style.ControlParts{Bar: styledPart(blue, nil), Regions: [3]*style.Computed{styledPart(red, nil)}}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 14, W: 100, H: 8}, rects[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: 14, W: 25, H: 8}, rects[1].Rect)
}

func TestARangeOfAppearanceNoneDrawsItsTrackAndThumb(t *testing.T) {
	b := ctl(dom.Input, 200, 4, "type", "range", "value", "25")
	b.Style.Appearance = style.AppearanceNone
	b.Style.Parts = &style.ControlParts{
		Track: styledPart(red, func(s *style.Computed) { s.Size.Height = style.Px(4) }),
		Thumb: styledPart(blue, func(s *style.Computed) {
			s.Size = style.Size[style.Length]{Width: style.Px(16), Height: style.Px(16)}
			s.Margin.Top = style.Px(-6)
		}),
	}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 200, H: 4}, rects[0].Rect, "the track")
	assert.Equal(t, geom.Rect{X: 56, Y: 4, W: 16, H: 16}, rects[1].Rect, "the thumb at a quarter, less its width")

	b.Style.Parts.Thumb = nil
	r := paintOne(b)
	require.Len(t, r.of("circle"), 1, "Chrome's own thumb")
	assert.Equal(t, geom.V2(10+184*0.25+8, 10+8), r.of("circle")[0].At)
}

// A range's slider container fills its content box, the track centred in
// the container's content box.
func TestARangesContainerHoldsItsTrack(t *testing.T) {
	b := ctl(dom.Input, 200, 14, "type", "range", "value", "0")
	b.Style.Appearance = style.AppearanceNone
	b.Style.Parts = &style.ControlParts{
		Container: styledPart(green, func(s *style.Computed) { s.Padding = style.All(style.Px(3)) }),
		Track:     styledPart(red, func(s *style.Computed) { s.Size.Height = style.Px(8) }),
		Thumb: styledPart(blue, func(s *style.Computed) {
			s.Size = style.Size[style.Length]{Width: style.Px(20), Height: style.Px(20)}
		}),
	}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 3)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 200, H: 14}, rects[0].Rect, "the container")
	assert.Equal(t, geom.Rect{X: 13, Y: 13, W: 194, H: 8}, rects[1].Rect, "the track in its content box")
	assert.Equal(t, geom.Rect{X: 13, Y: 13, W: 20, H: 20}, rects[2].Rect, "the thumb at its start")
}

// A vertical range of appearance none lays its parts down its box: the
// track as wide as its thumb, the thumb from the top, from the bottom in
// rtl, its margin-left moving it across.
func TestAVerticalRangeLaysItsPartsDown(t *testing.T) {
	b := ctl(dom.Input, 8, 150, "type", "range", "value", "20")
	b.Style.Appearance, b.Style.Mode = style.AppearanceNone, style.VerticalLR
	b.Style.Parts = &style.ControlParts{
		Track: styledPart(red, func(s *style.Computed) { s.Size.Width = style.Px(8) }),
		Thumb: styledPart(blue, func(s *style.Computed) {
			s.Size = style.Size[style.Length]{Width: style.Px(20), Height: style.Px(10)}
			s.Margin.Left = style.Px(-6)
		}),
	}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 8, H: 150}, rects[0].Rect, "the track")
	assert.Equal(t, geom.Rect{X: 4, Y: 10 + 140*0.2, W: 20, H: 10}, rects[1].Rect, "the thumb a fifth down")

	b.Style.Dir = style.RTL
	assert.Equal(t, geom.Rect{X: 4, Y: 10 + 140*0.8, W: 20, H: 10}, paintOne(b).of("rect")[1].Rect, "up in rtl")
}

// The pointer over a range's thumb is over its track and container too,
// as over the elements of Chrome's shadow tree; over a progress's value,
// its bar and inner element.
func TestThePartsUnderThePointer(t *testing.T) {
	b := ctl(dom.Input, 200, 14, "type", "range", "value", "50")
	b.Style.Appearance = style.AppearanceNone
	b.Style.Parts = &style.ControlParts{
		Container: styledPart(green, func(s *style.Computed) { s.Padding = style.All(style.Px(3)) }),
		Track:     styledPart(red, func(s *style.Computed) { s.Size.Height = style.Px(8) }),
		Thumb: styledPart(blue, func(s *style.Computed) {
			s.Size = style.Size[style.Length]{Width: style.Px(20), Height: style.Px(20)}
			s.Margin.Top = style.Px(-6)
		}),
	}
	o := geom.V2(10, 10)
	for at, want := range map[geom.Vec2]dom.State{
		{X: 110, Y: 17}: dom.PartHover,
		{X: 110, Y: 26}: dom.PartHover,
		{X: 40, Y: 17}:  dom.ContainerHover | dom.TrackHover,
		{X: 40, Y: 11}:  dom.ContainerHover,
		{X: 40, Y: 40}:  0,
	} {
		assert.Equal(t, want, ControlPartsAt(b, o, at), "at %v", at)
	}
	assert.InDelta(t, 0.5, RangeAlong(b, o, geom.V2(110, 17)), 1e-6, "the thumb's middle on the value")
	assert.InDelta(t, 0, RangeAlong(b, o, geom.V2(20, 17)), 1e-6)

	own := ctl(dom.Input, 100, 16, "type", "range", "value", "50")
	assert.Equal(t, dom.PartHover, ControlPartsAt(own, o, geom.V2(60, 18)), "its own look's thumb")
	assert.Equal(t, dom.ContainerHover|dom.TrackHover, ControlPartsAt(own, o, geom.V2(20, 18)))

	bar := ctl(dom.Progress, 100, 10, "value", "0.5")
	bar.Style.Appearance = style.AppearanceNone
	assert.Equal(t, dom.PartHover, ControlPartsAt(bar, o, geom.V2(30, 15)), "its value")
	assert.Equal(t, dom.ContainerHover|dom.TrackHover, ControlPartsAt(bar, o, geom.V2(90, 15)), "its bar")
}

// An indeterminate progress of appearance none draws its value at no
// width but its own padding, borders and min-width, as Chrome's.
func TestAnIndeterminateValueKeepsItsOwnWidth(t *testing.T) {
	b := ctl(dom.Progress, 150, 14)
	b.Style.Appearance = style.AppearanceNone
	b.Style.Parts = &style.ControlParts{Bar: styledPart(blue, nil),
		Value: styledPart(red, func(s *style.Computed) { s.MinSize.Width = style.Px(30) })}
	rects := paintOne(b).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 30, H: 14}, rects[1].Rect)
}
