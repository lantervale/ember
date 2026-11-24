package paint

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// A 100-point box scrolling both ways, its bars 12 thick, each bar with
// a button of display: block in each of its four places, as Chrome lays
// them: two at each end, the track between.
func buttonedBars(t *testing.T, length style.Length) Bars {
	t.Helper()
	button := style.Initial()
	button.Display = style.Block
	button.Size = style.Size[style.Length]{Width: length, Height: length}
	st := style.Initial()
	st.Overflow = style.Size[style.Overflow]{Width: style.Scroll, Height: style.Scroll}
	st.ScrollbarWidth, st.ScrollbarHeight = 12, 12
	bar := style.BarParts{Buttons: [4]*style.Computed{&button, &button, &button, &button}}
	st.Scrollbar = &style.ScrollbarParts{V: bar, H: bar}
	b := &box.Box{Kind: box.Element, Style: &st, Size: geom.V2(100, 100), Content: geom.V2(300, 300),
		Scrollbar: geom.V2(12, 12)}
	return BarsOf(b, geom.V2(0, 0))
}

func TestStyledButtonsStandAtTheBarsEndsAsChromes(t *testing.T) {
	l := buttonedBars(t, style.Px(10))
	want := [4]geom.Rect{{X: 88, Y: 0, W: 12, H: 10}, {X: 88, Y: 10, W: 12, H: 10}, {X: 88, Y: 68, W: 12, H: 10},
		{X: 88, Y: 78, W: 12, H: 10}}
	if l.VButtons != want {
		t.Errorf("vertical buttons %v, want %v", l.VButtons, want)
	}
	if want := (geom.Rect{X: 88, Y: 20, W: 12, H: 48}); l.VTrack != want {
		t.Errorf("track %v, want %v", l.VTrack, want)
	}
	if want := (geom.Rect{X: 10, Y: 88, W: 10, H: 12}); l.HButtons[1] != want {
		t.Errorf("horizontal button going forward at the start %v, want %v", l.HButtons[1], want)
	}
	if auto := buttonedBars(t, style.Length{Kind: style.Auto}); auto.VButtons[1].H != autoButton {
		t.Errorf("an auto button is %v long, want %v", auto.VButtons[1].H, autoButton)
	}
	if long := buttonedBars(t, style.Px(30)); long.VButtons != [4]geom.Rect{} || long.VTrack != long.VBar {
		t.Errorf("buttons longer than the bar: %v, track %v", long.VButtons, long.VTrack)
	}
}

func TestThePartUnderThePointerIsTold(t *testing.T) {
	l := buttonedBars(t, style.Px(10))
	for at, want := range map[geom.Vec2]style.BarPart{
		geom.V2(94, 5):  style.BarBackStart,
		geom.V2(94, 15): style.BarForwardStart,
		geom.V2(94, 25): style.BarThumb,
		geom.V2(94, 60): style.BarForwardTrack,
		geom.V2(94, 85): style.BarForwardEnd,
		geom.V2(5, 94):  style.BarBackStart | style.BarHorizontal,
		geom.V2(94, 94): style.NoBarPart,
		geom.V2(50, 50): style.NoBarPart,
	} {
		if got := l.PartAt(at); got != want {
			t.Errorf("at %v: %v, want %v", at, got, want)
		}
	}
}

func TestAClickOnAButtonScrollsALine(t *testing.T) {
	l := buttonedBars(t, style.Px(10))
	var s Scrolling
	off, took := s.Input(l, Pointer{At: geom.V2(94, 85), Pressed: true, Down: true, Line: 40}, 16)
	if !took || off != geom.V2(0, 40) {
		t.Errorf("forward: %v, took %v", off, took)
	}
	l.Scroll = geom.V2(100, 0)
	if off, _ = s.Input(l, Pointer{At: geom.V2(5, 94), Pressed: true, Down: true, Line: 40}, 16); off != geom.V2(60, 0) {
		t.Errorf("back sideways: %v", off)
	}
}
