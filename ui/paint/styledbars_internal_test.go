package paint

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAStyledThumbIsSizedAndPlacedAsChromes(t *testing.T) {
	for _, c := range []struct {
		name                                     string
		start, track, port, content, scroll, min float32
		at, n                                    float32
		ok                                       bool
	}{
		{"its share of the track", 0, 88, 88, 200, 0, 26, 0, 39, true},
		{"at the end when scrolled down", 10, 88, 88, 200, 112, 26, 59, 39, true},
		{"no shorter than its least", 0, 64, 64, 2000, 0, 26, 0, 26, true},
		{"none when the content fits", 0, 88, 88, 80, 0, 26, 0, 0, false},
		{"the whole track when its least is longer", 0, 20, 20, 2000, 0, 26, 0, 20, true},
	} {
		at, n, ok := thumbAlong(c.start, c.track, c.port, c.content, c.scroll, c.min)
		if at != c.at || n != c.n || ok != c.ok {
			t.Errorf("%s: at %v, %v long, %v; want %v, %v, %v", c.name, at, n, ok, c.at, c.n, c.ok)
		}
	}
}

func TestStyledBarsLieAlongThePaddingBoxLessTheCorner(t *testing.T) {
	track := style.Initial()
	track.Margin.Top, track.Margin.Bottom = style.Px(6), style.Px(4)
	st := style.Initial()
	st.Overflow = style.Size[style.Overflow]{Width: style.Scroll, Height: style.Scroll}
	st.ScrollbarWidth, st.ScrollbarHeight = 12, 8
	st.Scrollbar = &style.ScrollbarParts{V: style.BarParts{Track: &track}}
	b := &box.Box{Kind: box.Element, Style: &st, Size: geom.V2(100, 80), Content: geom.V2(200, 300),
		Scrollbar: geom.V2(12, 8)}
	l := BarsOf(b, geom.V2(10, 20))
	if !l.V || !l.H || l.Parts == nil {
		t.Fatalf("bars %v %v, parts %v", l.V, l.H, l.Parts)
	}
	want := map[string][2]geom.Rect{
		"vertical bar":   {l.VBar, {X: 98, Y: 20, W: 12, H: 72}},
		"its track":      {l.VTrack, {X: 98, Y: 26, W: 12, H: 62}},
		"horizontal bar": {l.HBar, {X: 10, Y: 92, W: 88, H: 8}},
		"corner":         {l.Corner, {X: 98, Y: 92, W: 12, H: 8}},
	}
	for name, r := range want {
		if r[0] != r[1] {
			t.Errorf("%s: %v, want %v", name, r[0], r[1])
		}
	}
}
