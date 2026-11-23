//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
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

// snapAlignSheet has two-axis containers whose areas align differently
// on each axis, scroll-snap-align's first value the block axis and its
// second the inline, some under a scroll-padding of four lengths.
const snapAlignSheet = `
body { margin: 0; }
.g { width: 100px; height: 100px; overflow: auto; scrollbar-width: none; scroll-snap-type: both mandatory;
	margin-bottom: 4px; }
.pad { scroll-padding: 10px 20px 30px 5px; }
.x { scroll-snap-type: x mandatory; }
.y { scroll-snap-type: y mandatory; }
.room { position: relative; width: 600px; height: 600px; }
.a { position: absolute; width: 50px; height: 50px; }
`

//nolint:gochecknoglobals // the aligns both browsers lay out
var snapAligns = []string{"start", "end", "start end", "end start", "center start", "start center", "end center",
	"none start", "center"}

//nolint:gochecknoglobals // the scroll-snap-type classes both browsers lay out
var snapAlignTypes = []string{"", "x", "y"}

// snapAlignSteps are the scrolls of each container: scrollTo, else
// scrollBy.
//
//nolint:gochecknoglobals // where both browsers scroll each container
var snapAlignSteps = []snapStep{
	{0, "to", geom.V2(400, 400), 0}, {0, "to", geom.V2(200, 250), 0}, {0, "to", geom.V2(300, 50), 0},
	{0, "to", geom.V2(150, 200), 0}, {0, "to", geom.V2(0, 0), 0}, {0, "by", geom.V2(60, 60), 0},
	{0, "by", geom.V2(0, 120), 0}, {0, "by", geom.V2(120, 0), 0}, {0, "by", geom.V2(-40, -200), 0},
	{0, "to", geom.V2(500, 500), 0}, {0, "by", geom.V2(-30, -30), 0},
}

// TestSnapAlignsLandWhereChromes scrolls two-axis containers whose
// areas align by each pair of values, with and without scroll-padding,
// and compares where both land: the first value aligns the area
// vertically, the second horizontally, end against the padded end.
func TestSnapAlignsLandWhereChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(snapAlignSheet))
	var outer []*ui.Node
	var names []string
	for _, kind := range snapAlignTypes {
		for _, pad := range []bool{false, true} {
			for _, align := range snapAligns {
				room := ui.Div(ui.Class("room"))
				for _, at := range [][2]float32{{200, 300}, {350, 100}} {
					room.Append(ui.Div(ui.Class("a"),
						ui.Style(fmt.Sprintf("left: %gpx; top: %gpx; scroll-snap-align: %s", at[0], at[1], align))))
				}
				classes := []string{"g"}
				if pad {
					classes = append(classes, "pad")
				}
				if kind != "" {
					classes = append(classes, kind)
				}
				outer = append(outer, ui.Div(ui.Class(classes...), room))
				names = append(names, fmt.Sprintf("%v %q", classes, align))
			}
		}
	}
	d.Body().Append(outer...)

	var script strings.Builder
	script.WriteString(`const s = [...document.querySelectorAll(".g")];
const wait = () => new Promise(r => setTimeout(r, 10));
const out = [];
for (const g of s) {
	const row = [];
	for (const [by, x, y] of ` + stepsJS(snapAlignSteps) + `) {
		if (by) g.scrollBy(x, y); else g.scrollTo(x, y);
		await wait();
		row.push([g.scrollLeft, g.scrollTop]);
	}
	out.push(row);
}
return out;`)
	raw, err := chrome.Eval(d, geom.V2(200, 6000), script.String())
	require.NoError(t, err)
	var want [][][2]float32
	require.NoError(t, json.Unmarshal(raw, &want))

	win := platform.NewWindow(platform.Size(200, 6000))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	require.Len(t, want, len(outer))
	for i, g := range outer {
		for j, st := range snapAlignSteps {
			if st.op == "by" {
				d.ScrollBy(g, st.at)
			} else {
				d.SetScroll(g, st.at)
			}
			w.RunFrames(2)
			at := d.Scroll(g)
			what := fmt.Sprintf("%s, step %d %s %v", names[i], j, st.op, st.at)
			assert.InDelta(t, want[i][j][0], at.X, 0.5, "%s: x", what)
			assert.InDelta(t, want[i][j][1], at.Y, 0.5, "%s: y", what)
		}
	}
	assert.Empty(t, d.Warnings())
}

// stepsJS is steps as a JavaScript array of [by, x, y].
func stepsJS(steps []snapStep) string {
	var b strings.Builder
	b.WriteString("[")
	for i, st := range steps {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "[%t, %g, %g]", st.op == "by", st.at.X, st.at.Y)
	}
	b.WriteString("]")
	return b.String()
}

// scatteredSnapSheet has two-axis containers whose areas, all smaller
// than the snapport, are scattered at random, every fifth by proximity.
const scatteredSnapSheet = `
body { margin: 0; }
.g { width: 100px; height: 100px; overflow: auto; scrollbar-width: none; scroll-snap-type: both mandatory;
	margin-bottom: 4px; }
.p { scroll-snap-type: both proximity; }
.room { position: relative; width: 600px; height: 600px; }
.a { position: absolute; }
`

// TestScatteredSnapsLandWhereChromes lays out containers of areas
// scattered at random in both browsers, compares where their first
// layout snaps them, then scrolls each by diagonal scrollBys and a few
// scrollTos and compares where each lands.
func TestScatteredSnapsLandWhereChromes(t *testing.T) {
	const containers, steps = 90, 8
	r := rand.New(rand.NewPCG(1, 7))
	aligns := []string{"start", "end", "center", "start end", "end start", "center start"}
	d := ui.NewDocument(ui.CSS(scatteredSnapSheet))
	var outer []*ui.Node
	var names []string
	var scrolls [][]snapStep
	for i := range containers {
		room := ui.Div(ui.Class("room"))
		var areas []string
		for range 2 + r.IntN(6) {
			x, y := r.IntN(55)*10, r.IntN(55)*10
			w, h := 20+r.IntN(8)*10, 20+r.IntN(8)*10
			align := aligns[r.IntN(len(aligns))]
			room.Append(ui.Div(ui.Class("a"), ui.Style(fmt.Sprintf(
				"left: %dpx; top: %dpx; width: %dpx; height: %dpx; scroll-snap-align: %s", x, y, w, h, align))))
			areas = append(areas, fmt.Sprintf("%d,%d %dx%d %s", x, y, w, h, align))
		}
		classes := []string{"g"}
		if i%5 == 4 {
			classes = append(classes, "p")
		}
		outer = append(outer, ui.Div(ui.Class(classes...), room))
		names = append(names, fmt.Sprint(classes, areas))
		var ss []snapStep
		for range steps {
			if r.IntN(4) == 0 {
				ss = append(ss, snapStep{0, "to", geom.V2(float32(r.IntN(50)*10), float32(r.IntN(50)*10)), 0})
			} else {
				ss = append(ss, snapStep{0, "by", geom.V2(float32(r.IntN(41)*10-200), float32(r.IntN(41)*10-200)), 0})
			}
		}
		scrolls = append(scrolls, ss)
	}
	d.Body().Append(outer...)

	var script strings.Builder
	script.WriteString(`const s = [...document.querySelectorAll(".g")];
const wait = () => new Promise(r => setTimeout(r, 10));
const steps = [`)
	for _, ss := range scrolls {
		script.WriteString(stepsJS(ss) + ",\n")
	}
	script.WriteString(`];
const out = [];
for (let i = 0; i < s.length; i++) {
	const g = s[i];
	const row = [[g.scrollLeft, g.scrollTop]];
	for (const [by, x, y] of steps[i]) {
		if (by) g.scrollBy(x, y); else g.scrollTo(x, y);
		await wait();
		row.push([g.scrollLeft, g.scrollTop]);
	}
	out.push(row);
}
return out;`)
	raw, err := chrome.Eval(d, geom.V2(200, 9600), script.String())
	require.NoError(t, err)
	var want [][][2]float32
	require.NoError(t, json.Unmarshal(raw, &want))

	win := platform.NewWindow(platform.Size(200, 9600))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	require.Len(t, want, len(outer))
	for i, g := range outer {
		at := d.Scroll(g)
		assert.InDelta(t, want[i][0][0], at.X, 0.5, "%s laid out: x", names[i])
		assert.InDelta(t, want[i][0][1], at.Y, 0.5, "%s laid out: y", names[i])
		for j, st := range scrolls[i] {
			if st.op == "by" {
				d.ScrollBy(g, st.at)
			} else {
				d.SetScroll(g, st.at)
			}
			w.RunFrames(2)
			at = d.Scroll(g)
			what := fmt.Sprintf("%s, step %d %s %v", names[i], j, st.op, st.at)
			assert.InDelta(t, want[i][j+1][0], at.X, 0.5, "%s: x", what)
			assert.InDelta(t, want[i][j+1][1], at.Y, 0.5, "%s: y", what)
		}
	}
	assert.Empty(t, d.Warnings())
}
