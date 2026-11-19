package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestScrollIntoViewShowsAnElementLaidOutLater(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"))
	for range 10 {
		sc.Append(ui.Div(ui.Style("height: 40px")))
	}
	l := liveDoc(t, sc)
	last := ui.Div(ui.Style("height: 40px"), ui.Text("added"))
	sc.Append(last)
	l.doc.ScrollIntoView(last)
	l.frames(3)
	view, box := sc.Box(), last.Box()
	assert.GreaterOrEqual(t, box.Y, view.Y-0.5, "its top is in the scroller's view")
	assert.LessOrEqual(t, box.Y+box.H, view.Y+view.H+0.5, "and its bottom")
}

func TestScrollIntoViewOfARemovedElementDoesNothing(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"))
	for range 10 {
		sc.Append(ui.Div(ui.Style("height: 40px")))
	}
	l := liveDoc(t, sc)
	first := sc.Children()[0].Box()
	l.doc.ScrollIntoView(ui.Div())
	l.frames(2)
	assert.Equal(t, first, sc.Children()[0].Box())
}

// startPage is a scrolling page 400×300 of a 1000-point block, a button
// 30 high, and 1000 points more.
func startPage(t *testing.T) (*live, *ui.Node) {
	t.Helper()
	low := ui.Button(ui.Style("display: block; width: 80px; height: 30px"), ui.Text("Low"))
	l := liveDocWith(t, []ui.DocumentOption{ui.ScrollingViewport(), ui.OverlayScrollbars(),
		ui.CSS("body { margin: 0; }")}, ui.Div(ui.Style("height: 1000px")), low, ui.Div(ui.Style("height: 1000px")))
	return l, low
}

func TestScrollToStartPutsAnElementAtTheTopOfTheView(t *testing.T) {
	l, low := startPage(t)
	l.doc.ScrollIntoView(low)
	l.frames(2)
	assert.InDelta(t, 330, low.Box().Y, 1e-3, "ScrollIntoView still scrolls by the least: the window is 360 high")

	l.doc.ScrollToStart(low)
	l.frames(2)
	assert.InDelta(t, 0, low.Box().Y, 1e-3)
	assert.Equal(t, geom.V2(0, 1000), l.doc.Scroll(l.doc.HTML()))

	l.doc.ScrollToStart(low)
	l.frames(2)
	assert.InDelta(t, 0, low.Box().Y, 1e-3, "asked again, it stays")
}

func TestScrollToStartStopsAtTheEndOfTheScroll(t *testing.T) {
	last := ui.Div(ui.Style("height: 40px"))
	l := liveDocWith(t, []ui.DocumentOption{ui.ScrollingViewport(), ui.OverlayScrollbars(),
		ui.CSS("body { margin: 0; }")}, ui.Div(ui.Style("height: 1000px")), last)
	l.doc.ScrollToStart(last)
	l.frames(2)
	assert.InDelta(t, 320, last.Box().Y, 1e-3, "the page ends at the window's bottom, 360 down")
}

func TestScrollToStartGoesThroughEveryScrollerRoundTheElement(t *testing.T) {
	s := newScroller(t)
	s.doc.ScrollToStart(s.items[5])
	s.w.RunFrames(2)
	assert.InDelta(t, 200, s.scrolled(), 1e-3)
	assert.InDelta(t, 0, s.outerScrolled(), 1e-3, "the list is at the outer box's top already")

	s.doc.ScrollToStart(s.items[9])
	s.w.RunFrames(2)
	assert.InDelta(t, 300, s.scrolled(), 1e-3, "the list ends there")
	assert.InDelta(t, 60, s.outerScrolled(), 1e-3, "and the outer box brings the item to its own top")
	assert.InDelta(t, s.outer.Box().Y, s.items[9].Box().Y, 1e-3)
}

func TestScrollToStartLeavesTheOtherAxisToTheNearestEdge(t *testing.T) {
	wide := ui.Div(ui.Style("overflow: auto; width: 200px; height: 100px"))
	inner := ui.Div(ui.Style("width: 1000px; height: 400px; position: relative"))
	seen := ui.Div(ui.Style("position: absolute; left: 50px; top: 200px; width: 40px; height: 20px"))
	far := ui.Div(ui.Style("position: absolute; left: 600px; top: 300px; width: 40px; height: 20px"))
	inner.Append(seen, far)
	wide.Append(inner)
	l := liveDocWith(t, []ui.DocumentOption{ui.OverlayScrollbars()}, wide)
	l.doc.ScrollToStart(seen)
	l.frames(2)
	assert.Equal(t, geom.V2(0, 200), l.doc.Scroll(wide), "what shows sideways is left where it is")
	l.doc.ScrollToStart(far)
	l.frames(2)
	assert.Equal(t, geom.V2(440, 300), l.doc.Scroll(wide), "and what does not comes in by its nearest edge")
}

func TestScrollToStartWaitsForAnElementLaidOutLater(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"))
	for range 10 {
		sc.Append(ui.Div(ui.Style("height: 40px")))
	}
	l := liveDoc(t, sc)
	added := ui.Div(ui.Style("height: 40px"), ui.Text("added"))
	sc.Append(added, ui.Div(ui.Style("height: 400px")))
	l.doc.ScrollToStart(added)
	l.frames(3)
	assert.InDelta(t, sc.Box().Y, added.Box().Y, 1e-3)

	l.doc.ScrollToStart(ui.Div())
	l.frames(2)
	assert.InDelta(t, sc.Box().Y, added.Box().Y, 1e-3, "an element of no Document scrolls nothing")
}
