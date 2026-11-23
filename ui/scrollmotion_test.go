package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
)

// notch is how far a notch of the wheel scrolls the list of newScroller:
// three of its lines.
func notch(t *testing.T) float32 {
	t.Helper()
	s := newScroller(t, ui.InstantScrolls())
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	return s.scrolled()
}

func TestANotchGlidesOnChromesWheelCurve(t *testing.T) {
	d := notch(t)
	require.Less(t, d, float32(120), "a notch short of 120 points glides for 12 frames")
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.AddWheel(geom.V2(0, 1))
	s.w.RunFrames(1)
	assert.Zero(t, s.doc.Scroll(s.list).Y, "it starts from where it was the next frame")
	s.w.RunFrames(7)
	curve := css.CubicBezier(0.42, 0, 0.58, 1)
	assert.InDelta(t, d*curve(0.5), s.doc.Scroll(s.list).Y, 0.01, "half way through its time, half way")
	s.w.RunFrames(5)
	assert.Less(t, s.doc.Scroll(s.list).Y, d)
	s.w.RunFrames(1)
	assert.InDelta(t, d, s.doc.Scroll(s.list).Y, 0.01, "there in 12 frames")
}

func TestNotchesInARowGoOnFromWhereTheGlideIsGoing(t *testing.T) {
	d := notch(t)
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.AddWheel(geom.V2(0, 1))
	s.w.RunFrames(4)
	mid := s.doc.Scroll(s.list).Y
	s.in.AddWheel(geom.V2(0, 1))
	s.w.RunFrames(1)
	assert.GreaterOrEqual(t, s.doc.Scroll(s.list).Y, mid, "no step back")
	s.w.RunFrames(14)
	assert.InDelta(t, 2*d, s.doc.Scroll(s.list).Y, 0.01, "two notches from where it began")
}

func TestKeysGlideForNineFrames(t *testing.T) {
	s := newScroller(t)
	s.doc.Focus(s.items[0])
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyPageDown, true)
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyPageDown, false)
	s.w.RunFrames(5)
	curve := css.CubicBezier(0.42, 0, 0.58, 1)
	assert.InDelta(t, 87*curve(4.0/9), s.doc.Scroll(s.list).Y, 0.01)
	s.w.RunFrames(5)
	assert.InDelta(t, 87, s.doc.Scroll(s.list).Y, 0.01)
}

func TestATrackpadScrollsAtOnceByItsPoints(t *testing.T) {
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.AddPreciseWheel(geom.V2(0, 2.5))
	s.w.RunFrames(1)
	assert.InDelta(t, 25, s.doc.Scroll(s.list).Y, 0.01, "25 points, the frame it moved")
}

func TestInstantScrollsMoveAtOnce(t *testing.T) {
	d := notch(t)
	s := newScroller(t, ui.InstantScrolls())
	assert.True(t, s.doc.InstantScrolls())
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.AddWheel(geom.V2(0, 1))
	s.w.RunFrames(1)
	assert.InDelta(t, d, s.doc.Scroll(s.list).Y, 0.01, "a notch, the frame it turned")
	s.in.SetKey(input.KeyPageDown, true)
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyPageDown, false)
	assert.InDelta(t, d+87, s.doc.Scroll(s.list).Y, 0.01, "a page key too")

	s.doc.SetInstantScrolls(false)
	assert.False(t, s.doc.InstantScrolls())
	s.in.AddWheel(geom.V2(0, -1))
	s.w.RunFrames(1)
	assert.InDelta(t, d+87, s.doc.Scroll(s.list).Y, 0.01, "gliding again")
}

func TestArrowsAndSpaceScrollAPageWithNothingFocused(t *testing.T) {
	p := newScrollingPage(t, "")
	p.in.SetMouse(geom.V2(200, 150))
	p.w.RunFrames(1)
	p.key(input.KeyDown)
	assert.InDelta(t, 40, p.scrolled(), 1e-3, "an arrow scrolls 40 points")
	p.key(input.KeySpace)
	assert.InDelta(t, 40+262, p.scrolled(), 1e-3, "Space a page")
	p.in.SetKey(input.KeyLeftShift, true)
	p.key(input.KeySpace)
	p.in.SetKey(input.KeyLeftShift, false)
	assert.InDelta(t, 40, p.scrolled(), 1e-3, "Shift+Space back")
	p.key(input.KeyUp)
	assert.InDelta(t, 0, p.scrolled(), 1e-3)

	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyDown, true)
	s.w.RunFrames(1)
	assert.True(t, s.in.Down(input.KeyDown), "a game's screen leaves the arrows to the game")
	s.in.SetKey(input.KeyDown, false)
	s.w.RunFrames(settle)
	assert.Zero(t, s.scrolled())
}
