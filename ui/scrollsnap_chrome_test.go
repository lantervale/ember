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

// scrollSnapSheet has snap containers of each kind: mandatory and by
// proximity, on y, x and both, start, end and center aligned, with
// scroll-padding, scroll-margin, a scroll-snap-stop and right to left.
const scrollSnapSheet = `
body { margin: 0; }
.s { height: 100px; width: 200px; overflow: auto; scroll-snap-type: y mandatory; margin-bottom: 4px; }
.it { height: 60px; scroll-snap-align: start; }
#s0 .i0 { margin-top: 30px; }
#s1 { scroll-snap-type: y proximity; height: 150px; }
#s1 .it { height: 300px; }
#s2 { scroll-padding: 10px 0 20px; }
#s2 .it { scroll-margin-top: 5px; }
#s2 .i3 { scroll-snap-stop: always; }
#s3 { scroll-snap-type: both mandatory; display: flex; flex-wrap: wrap; width: 100px; }
#s3 .it { flex: none; width: 70px; height: 70px; scroll-snap-align: end center; }
#s4 { direction: rtl; scroll-snap-type: x mandatory; display: flex; height: 40px; width: 100px; scroll-padding: 0 10%; }
#s4 .it { flex: none; width: 45px; height: 10px; }
#s5 .it { scroll-snap-align: center; }
#s6 { scroll-snap-type: inline mandatory; display: flex; height: 40px; width: 100px; scroll-padding-inline: 10px; }
#s6 .it { flex: none; width: 45px; height: 10px; scroll-snap-align: none start; }
#s7 { scroll-snap-type: y proximity; height: 150px; scroll-padding-top: 30px; }
#s7 .it { height: 300px; }
`

// snapStep is one scroll of a scroller: to an offset, by one, or an
// item brought into view by its nearest edge or to the view's start.
type snapStep struct {
	scroller int
	op       string
	at       geom.Vec2
	item     int
}

//nolint:gochecknoglobals // the steps both browsers run
var snapSteps = []snapStep{
	{0, "to", geom.V2(0, 95), 0}, {0, "to", geom.V2(0, 89), 0}, {0, "to", geom.V2(0, 120), 0},
	{0, "by", geom.V2(0, 10), 0}, {0, "by", geom.V2(0, 200), 0}, {0, "by", geom.V2(0, -10), 0},
	{0, "to", geom.V2(0, 1000), 0}, {0, "by", geom.V2(0, 100), 0},
	{1, "to", geom.V2(0, 250), 0}, {1, "to", geom.V2(0, 249), 0}, {1, "to", geom.V2(0, 340), 0},
	{1, "to", geom.V2(0, 160), 0}, {1, "by", geom.V2(0, 50), 0}, {1, "near", geom.Vec2{}, 2},
	{2, "to", geom.V2(0, 30), 0}, {2, "to", geom.V2(0, 100), 0}, {2, "to", geom.V2(0, 0), 0},
	{2, "by", geom.V2(0, 300), 0}, {2, "to", geom.V2(0, 400), 0}, {2, "by", geom.V2(0, -300), 0},
	{2, "to", geom.V2(0, 300), 0}, {2, "to", geom.V2(0, 0), 0}, {2, "near", geom.Vec2{}, 5},
	{2, "to", geom.V2(0, 400), 0}, {2, "near", geom.Vec2{}, 5}, {2, "to", geom.V2(0, 0), 0},
	{2, "start", geom.Vec2{}, 5},
	{3, "to", geom.V2(30, 30), 0}, {3, "to", geom.V2(100, 200), 0}, {3, "by", geom.V2(10, 10), 0},
	{3, "to", geom.V2(0, 180), 0}, {3, "by", geom.V2(0, 10), 0},
	{4, "to", geom.V2(-60, 0), 0}, {4, "by", geom.V2(10, 0), 0}, {4, "by", geom.V2(-100, 0), 0},
	{5, "to", geom.V2(0, 50), 0}, {5, "to", geom.V2(0, 1000), 0}, {5, "by", geom.V2(0, -30), 0},
	{6, "to", geom.V2(60, 0), 0}, {6, "by", geom.V2(-1, 0), 0}, {6, "near", geom.Vec2{}, 7},
	{7, "to", geom.V2(0, 225), 0}, {7, "to", geom.V2(0, 215), 0},
}

// TestScrollSnapLandsWhereChromes scrolls the snap containers of the
// page step by step in both, as scrollTo, scrollBy and scrollIntoView,
// and compares each offset they land on, and those their first layout
// snaps them to.
func TestScrollSnapLandsWhereChromes(t *testing.T) {
	const scrollers = 8
	d := ui.NewDocument(ui.CSS(scrollSnapSheet))
	var boxes [][]*ui.Node
	var outer []*ui.Node
	for i := range scrollers {
		s := ui.Div(ui.ID(fmt.Sprint("s", i)), ui.Class("s"))
		var items []*ui.Node
		for j := range 10 {
			it := ui.Div(ui.Class("it", fmt.Sprint("i", j)))
			items = append(items, it)
			s.Append(it)
		}
		outer, boxes = append(outer, s), append(boxes, items)
	}
	d.Body().Append(outer...)

	var script strings.Builder
	script.WriteString(`const s = [...document.querySelectorAll(".s")];
const at = el => [el.scrollLeft, el.scrollTop];
const out = [s.map(at)];
const wait = () => new Promise(r => setTimeout(r, 150));
`)
	for _, st := range snapSteps {
		el := fmt.Sprintf("s[%d]", st.scroller)
		switch st.op {
		case "to":
			fmt.Fprintf(&script, "%s.scrollTo(%g, %g);\n", el, st.at.X, st.at.Y)
		case "by":
			fmt.Fprintf(&script, "%s.scrollBy(%g, %g);\n", el, st.at.X, st.at.Y)
		case "near":
			fmt.Fprintf(&script, "%s.children[%d].scrollIntoView({block: \"nearest\", inline: \"nearest\"});\n", el, st.item)
		case "start":
			fmt.Fprintf(&script, "%s.children[%d].scrollIntoView({block: \"start\", inline: \"nearest\"});\n", el, st.item)
		}
		fmt.Fprintf(&script, "await wait(); out.push([at(%s)]);\n", el)
	}
	script.WriteString("return out;")

	raw, err := chrome.Eval(d, geom.V2(600, 900), script.String())
	require.NoError(t, err)
	var want [][][2]float32
	require.NoError(t, json.Unmarshal(raw, &want))

	win := platform.NewWindow(platform.Size(600, 900))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	var got [][]geom.Vec2
	var first []geom.Vec2
	for _, s := range outer {
		first = append(first, d.Scroll(s))
	}
	got = append(got, first)
	for _, st := range snapSteps {
		s := outer[st.scroller]
		switch st.op {
		case "to":
			d.SetScroll(s, st.at)
		case "by":
			d.ScrollBy(s, st.at)
		case "near":
			d.ScrollIntoView(boxes[st.scroller][st.item])
		case "start":
			d.ScrollToStart(boxes[st.scroller][st.item])
		}
		w.RunFrames(2)
		got = append(got, []geom.Vec2{d.Scroll(s)})
	}

	require.Len(t, want, len(got))
	for i := range want {
		for j := range want[i] {
			what := "laid out"
			if i > 0 {
				st := snapSteps[i-1]
				what = fmt.Sprintf("step %d: scroller %d %s %v item %d", i-1, st.scroller, st.op, st.at, st.item)
			} else {
				what += fmt.Sprintf(", scroller %d", j)
			}
			assert.InDelta(t, want[i][j][0], got[i][j].X, 0.5, "%s: x", what)
			assert.InDelta(t, want[i][j][1], got[i][j].Y, 0.5, "%s: y", what)
		}
	}
	assert.Empty(t, d.Warnings())
}

// viewportSnapSheet snaps the page to its sections under a sticky
// header, its scroll-padding keeping them clear of it, and marks each
// section the page is snapped to.
const viewportSnapSheet = `
html { scroll-snap-type: y mandatory; scroll-padding-top: 40px; }
body { margin: 0; }
header { position: sticky; top: 0; height: 40px; background: #ccc; }
section { height: 300px; scroll-snap-align: start; container-type: scroll-state; }
section.tall { height: 700px; }
section.after { scroll-margin-top: 12px; }
i { display: block; width: 1px; height: 1px; }
@container scroll-state(snapped: y) { i { width: 2px; } }
@container scroll-state(snapped: x) { i { height: 2px; } }
`

// TestTheViewportSnapsAsChromes snaps a page's own scroll, its sections
// brought to its start under a sticky header as a fragment brings them,
// and answers which section it is snapped to at its first layout.
func TestTheViewportSnapsAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(viewportSnapSheet))
	var sections []*ui.Node
	d.Body().Append(ui.Header())
	for i := range 6 {
		var args []ui.Arg
		switch i {
		case 2:
			args = append(args, ui.Class("tall"))
		case 4:
			args = append(args, ui.Class("after"))
		}
		s := ui.Section(append(args, ui.I())...)
		sections = append(sections, s)
		d.Body().Append(s)
	}
	steps := []snapStep{{0, "to", geom.V2(0, 200), 0}, {0, "to", geom.V2(0, 1000), 0},
		{0, "to", geom.V2(0, 1290), 0}, {0, "start", geom.Vec2{}, 4}, {0, "start", geom.Vec2{}, 1},
		{0, "by", geom.V2(0, 30), 0}, {0, "near", geom.Vec2{}, 5}, {0, "by", geom.V2(0, -30), 0}}
	script := `const html = document.documentElement, ss = [...document.querySelectorAll("section")];
const out = [[html.scrollTop, ss.map(s => s.firstChild.getBoundingClientRect().width)]];
const wait = () => new Promise(r => setTimeout(r, 150));
`
	for _, st := range steps {
		switch st.op {
		case "to":
			script += fmt.Sprintf("html.scrollTo(0, %g);\n", st.at.Y)
		case "by":
			script += fmt.Sprintf("html.scrollBy(0, %g);\n", st.at.Y)
		case "near":
			script += fmt.Sprintf("ss[%d].scrollIntoView({block: \"nearest\"});\n", st.item)
		case "start":
			script += fmt.Sprintf("ss[%d].scrollIntoView();\n", st.item)
		}
		script += "await wait(); out.push([html.scrollTop]);\n"
	}
	script += "return out;"
	size := geom.V2(400, 500)
	// Headless Chrome answers scroll-state() from a snapshot it may take
	// after the script on a loaded machine: asked again until it says
	// some section is snapped.
	var want [][]json.RawMessage
	for range 4 {
		raw, err := chrome.Eval(d, size, script)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &want))
		if strings.Contains(string(want[0][1]), "2") {
			break
		}
	}

	win := platform.NewWindow(platform.Size(int(size.X), int(size.Y)))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	var marks []float32
	for _, s := range sections {
		marks = append(marks, s.Children()[0].Box().W)
	}
	got := []float32{d.Scroll(d.HTML()).Y}
	for _, st := range steps {
		switch st.op {
		case "to":
			d.SetScroll(d.HTML(), st.at)
		case "by":
			d.ScrollBy(d.HTML(), st.at)
		case "near":
			d.ScrollIntoView(sections[st.item])
		case "start":
			d.ScrollToStart(sections[st.item])
		}
		w.RunFrames(2)
		got = append(got, d.Scroll(d.HTML()).Y)
	}

	require.Len(t, want, len(got))
	var wantMarks []float32
	require.NoError(t, json.Unmarshal(want[0][1], &wantMarks))
	assert.Equal(t, wantMarks, marks, "the section snapped to at the first layout")
	for i := range want {
		var y float32
		require.NoError(t, json.Unmarshal(want[i][0], &y))
		what := "laid out"
		if i > 0 {
			what = fmt.Sprintf("step %d: %s %v section %d", i-1, steps[i-1].op, steps[i-1].at, steps[i-1].item)
		}
		assert.InDelta(t, y, got[i], 0.5, what)
	}
	assert.Empty(t, d.Warnings())
}
