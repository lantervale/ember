package css

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestTheSnapPropertiesComputeTheirShorthandsAndLogicalSides(t *testing.T) {
	box, area, rtl, plain := el(dom.Div, "", "a"), el(dom.Div, "", "b"), el(dom.Div, "", "b", "r"), el(dom.Div, "", "")
	s, warns := Parse("t.css", `.a { scroll-snap-type: both mandatory; scroll-padding: 10px 5% auto;
			scroll-padding-inline-start: 3px; scroll-behavior: smooth; overscroll-behavior: contain none; }
		.b { scroll-snap-type: inline; scroll-snap-align: end center; scroll-snap-stop: always;
			scroll-margin: 4px; scroll-margin-inline-start: 7px; overscroll-behavior-block: contain; }
		.r { direction: rtl; scroll-snap-align: start; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(box, area, rtl, plain), []*Sheet{s})
	want := style.Snap{Type: style.SnapX | style.SnapY, Mandatory: true,
		Padding: style.Sides[style.Length]{Top: style.Px(10), Right: style.Percent(5), Left: style.Px(3)},
		Smooth:  true, OverscrollX: style.OverscrollContain, OverscrollY: style.OverscrollNone}
	want.Padding.Right.IsPercent = true
	if got := styleOf(box).Snap; got != want {
		t.Errorf("the container: %+v, want %+v", got, want)
	}
	want = style.Snap{Type: style.SnapX, AlignY: style.SnapEnd, AlignX: style.SnapCenter, Always: true,
		Margin:      style.Sides[style.Length]{Top: style.Px(4), Right: style.Px(4), Bottom: style.Px(4), Left: style.Px(7)},
		OverscrollY: style.OverscrollContain}
	if got := styleOf(area).Snap; got != want {
		t.Errorf("an area: %+v, want %+v", got, want)
	}
	if got := styleOf(rtl).Snap; got.AlignX != style.SnapStart || got.AlignY != style.SnapStart ||
		got.Margin.Right != style.Px(7) || got.Margin.Left != style.Px(4) {
		t.Errorf("an rtl area takes its inline start on the right: %+v", got)
	}
	if got := styleOf(plain).Snap; got != (style.Snap{}) {
		t.Errorf("no rule: %+v", got)
	}
	for _, bad := range []string{"scroll-snap-type: mandatory", "scroll-snap-type: x y", "scroll-snap-align: left",
		"scroll-padding: -1px", "scroll-margin: 5%", "overscroll-behavior: auto auto auto", "scroll-behavior: fast"} {
		if _, w := Parse("t.css", ".x { "+bad+"; }"); len(w) == 0 {
			t.Errorf("%s: no warning", bad)
		}
	}
}
