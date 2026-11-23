//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// twoAxisSnapSheet has containers snapping on both axes whose areas
// are scattered, so the nearest position on each axis alone would land
// where neither area shows.
const twoAxisSnapSheet = `
body { margin: 0; }
.g { width: 100px; height: 100px; overflow: auto; scrollbar-width: none; scroll-snap-type: both mandatory;
	margin-bottom: 4px; }
.room { position: relative; width: 600px; height: 600px; }
.a { position: absolute; width: 100px; height: 100px; scroll-snap-align: start; }
#g2 .room { width: 700px; height: 700px; }
#g2 .a { scroll-snap-align: center; }
#g3 { scroll-snap-type: both proximity; }
`

// twoAxisAreas are the places of each container's areas.
//
//nolint:gochecknoglobals // the pages both browsers lay out
var twoAxisAreas = [][][2]float32{
	{{0, 0}, {300, 0}, {0, 300}},
	{{0, 0}, {0, 450}},
	{{0, 0}, {250, 0}, {500, 0}, {0, 250}, {250, 250}, {500, 250}, {0, 500}, {250, 500}, {500, 500}},
	{{0, 0}, {300, 300}},
}

//nolint:gochecknoglobals // the steps both browsers run
var twoAxisSteps = []snapStep{
	{0, "to", geom.V2(250, 250), 0}, {0, "to", geom.V2(0, 0), 0}, {0, "to", geom.V2(280, 40), 0},
	{0, "by", geom.V2(0, 200), 0}, {0, "to", geom.V2(0, 260), 0}, {0, "by", geom.V2(250, 0), 0},
	{0, "by", geom.V2(10, -280), 0}, {0, "to", geom.V2(160, 140), 0},
	{1, "to", geom.V2(250, 200), 0}, {1, "to", geom.V2(30, 380), 0}, {1, "to", geom.V2(400, 100), 0},
	{2, "to", geom.V2(200, 420), 0}, {2, "by", geom.V2(100, 0), 0}, {2, "to", geom.V2(380, 120), 0},
	{2, "to", geom.V2(250, 250), 0}, {2, "by", geom.V2(0, 100), 0}, {2, "by", geom.V2(-100, 0), 0},
	{3, "to", geom.V2(20, 280), 0}, {3, "to", geom.V2(290, 280), 0}, {3, "to", geom.V2(30, 20), 0},
}

// TestTwoAxisSnapsLandWhereChromes scrolls containers snapping on both
// axes in both, as scrollTo and scrollBy, and compares the offsets they
// land on: Chrome lands where the areas it snaps to on each axis are
// both seen, else on an area aligned on both nearest its corridor.
func TestTwoAxisSnapsLandWhereChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(twoAxisSnapSheet))
	var outer []*ui.Node
	for i, areas := range twoAxisAreas {
		room := ui.Div(ui.Class("room"))
		for _, at := range areas {
			room.Append(ui.Div(ui.Class("a"), ui.Style(fmt.Sprintf("left: %gpx; top: %gpx", at[0], at[1]))))
		}
		g := ui.Div(ui.ID(fmt.Sprint("g", i)), ui.Class("g"), room)
		outer = append(outer, g)
	}
	d.Body().Append(outer...)

	var script strings.Builder
	script.WriteString(`const s = [...document.querySelectorAll(".g")];
const at = el => [el.scrollLeft, el.scrollTop];
const out = [s.map(at)];
const wait = () => new Promise(r => setTimeout(r, 150));
`)
	for _, st := range twoAxisSteps {
		el := fmt.Sprintf("s[%d]", st.scroller)
		switch st.op {
		case "to":
			fmt.Fprintf(&script, "%s.scrollTo(%g, %g);\n", el, st.at.X, st.at.Y)
		case "by":
			fmt.Fprintf(&script, "%s.scrollBy(%g, %g);\n", el, st.at.X, st.at.Y)
		}
		fmt.Fprintf(&script, "await wait(); out.push([at(%s)]);\n", el)
	}
	script.WriteString("return out;")
	raw, err := chrome.Eval(d, geom.V2(300, 600), script.String())
	require.NoError(t, err)
	var want [][][2]float32
	require.NoError(t, json.Unmarshal(raw, &want))

	win := platform.NewWindow(platform.Size(300, 600))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	var first []geom.Vec2
	for _, s := range outer {
		first = append(first, d.Scroll(s))
	}
	got := [][]geom.Vec2{first}
	for _, st := range twoAxisSteps {
		s := outer[st.scroller]
		switch st.op {
		case "to":
			d.SetScroll(s, st.at)
		case "by":
			d.ScrollBy(s, st.at)
		}
		w.RunFrames(2)
		got = append(got, []geom.Vec2{d.Scroll(s)})
	}

	require.Len(t, want, len(got))
	for i := range want {
		for j := range want[i] {
			what := fmt.Sprintf("laid out, container %d", j)
			if i > 0 {
				st := twoAxisSteps[i-1]
				what = fmt.Sprintf("step %d: container %d %s %v", i-1, st.scroller, st.op, st.at)
			}
			assert.InDelta(t, want[i][j][0], got[i][j].X, 0.5, "%s: x", what)
			assert.InDelta(t, want[i][j][1], got[i][j].Y, 0.5, "%s: y", what)
		}
	}
	assert.Empty(t, d.Warnings())
}
