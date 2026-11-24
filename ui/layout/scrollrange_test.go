package layout //nolint:testpackage // tests scrollRange, unexported

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

// A 120-point box whose 15-point gutters hold no bar, its content 300
// wide: Chrome scrolls it as though nothing took the padding box's room,
// and starts it right to left scrolled by those gutters.
func TestAStableGuttersSidewaysRangeIsChromes(t *testing.T) {
	for _, c := range []struct {
		name         string
		both, rtl    bool
		left, lo, hi float32
	}{
		{"stable", false, false, 0, 0, 180},
		{"both edges", true, false, 0, 0, 180},
		{"stable, right to left", false, true, -180, -195, -15},
		{"both edges, right to left", true, true, -180, -210, -30},
	} {
		st := style.Initial()
		st.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
		st.GutterStable, st.GutterBothEdges, st.ScrollbarWidth, st.ScrollbarHeight = true, c.both, 15, 15
		if c.rtl {
			st.Dir = style.RTL
		}
		gutter := float32(15)
		if c.both {
			gutter = 30
		}
		f := placed{size: size{w: 120, h: 80}, scrollbar: size{w: gutter},
			overflow: overflowRect{rect: rect{l: c.left, r: c.left + 300, b: 40}}}
		content, start := scrollRange(&st, f)
		port := 120 - (gutter - startGutter(&st))
		if start.X != c.lo || start.X+content.X-port != c.hi {
			t.Errorf("%s: %v..%v, want %v..%v", c.name, start.X, start.X+content.X-port, c.lo, c.hi)
		}
	}
	st := style.Initial()
	st.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	st.GutterStable, st.ScrollbarWidth = true, 15
	content, start := scrollRange(&st, placed{size: size{w: 120, h: 80}, scrollbar: size{w: 15},
		overflow: overflowRect{rect: rect{r: 100, b: 40}}})
	if content != geom.V2(100, 40) || start != (geom.Vec2{}) {
		t.Errorf("content that fits keeps its overflow: %v from %v", content, start)
	}
}
