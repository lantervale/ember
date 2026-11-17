package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tallPage is a 400×300 window whose Document's viewport scrolls: a
// sticky header 20 high after a lead of 40 holding a button, a fixed badge, a list
// that scrolls on its own, 2000 points of filler and a button at the end.
type tallPage struct {
	doc                                   *ui.Document
	w                                     *world.World
	in                                    *input.State
	top, header, badge, list, filler, end *ui.Node
	clicks                                int
}

const tallSheet = `body { margin: 0; }
#lead { height: 40px; }
#header { position: sticky; top: 0; height: 20px; }
#badge { position: fixed; right: 30px; bottom: 10px; width: 50px; height: 20px; }
#list { overflow: auto; width: 200px; height: 100px; scrollbar-width: none; }
#filler { height: 2000px; }
#end { display: block; width: 100px; height: 30px; }
`

func newTallPage(t *testing.T, sheet string, opts ...ui.DocumentOption) *tallPage {
	t.Helper()
	p := &tallPage{
		top: ui.Button(ui.Text("Top")), header: ui.Div(ui.ID("header")), badge: ui.Div(ui.ID("badge")),
		list:   ui.Div(ui.ID("list"), ui.Div(ui.Style("height: 400px"))),
		filler: ui.Div(ui.ID("filler")),
	}
	p.end = ui.Button(ui.ID("end"), ui.Text("End"), ui.OnClick(func() { p.clicks++ }))
	p.doc = ui.NewDocument(append([]ui.DocumentOption{ui.CSS(tallSheet + sheet)}, opts...)...)
	p.doc.Body().Append(ui.Div(ui.ID("lead"), p.top), p.header, p.badge, p.list, p.filler, p.end)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(p.doc)
	p.w = ember.New().AddEntities(win)
	p.in = win.Input()
	p.w.RunFrames(2)
	return p
}

func newScrollingPage(t *testing.T, sheet string, opts ...ui.DocumentOption) *tallPage {
	t.Helper()
	return newTallPage(t, sheet, append([]ui.DocumentOption{ui.ScrollingViewport(), ui.OverlayScrollbars()}, opts...)...)
}

// scrolled is how far the page has moved up in the viewport.
func (p *tallPage) scrolled() float32 { return -p.doc.HTML().Box().Y }

func (p *tallPage) wheel(at geom.Vec2, lines float32) {
	p.in.SetMouse(at)
	p.w.RunFrames(1)
	p.in.AddWheel(geom.V2(0, lines))
	p.w.RunFrames(2)
}

func (p *tallPage) key(k input.Key) {
	p.in.SetKey(k, true)
	p.w.RunFrames(1)
	p.in.SetKey(k, false)
	p.w.RunFrames(1)
}

const pageHeight = 40 + 20 + 100 + 2000 + 30

func TestAScrollingViewportLeavesHTMLItsOwnHeight(t *testing.T) {
	p := newScrollingPage(t, "")
	assert.True(t, p.doc.ScrollingViewport())
	assert.Equal(t, geom.Rect{W: 400, H: pageHeight}, p.doc.HTML().Box())
	assert.Equal(t, geom.Rect{X: 320, Y: 270, W: 50, H: 20}, p.badge.Box())
}

func TestWithoutTheOptionAWindowsDocumentNeverScrollsAsAWhole(t *testing.T) {
	p := newTallPage(t, "")
	assert.False(t, p.doc.ScrollingViewport())
	p.wheel(geom.V2(300, 200), 3)
	p.key(input.KeyEnd)
	assert.Zero(t, p.scrolled())
}

func TestPercentHeightsOfHTMLAndBodyAreOfTheViewport(t *testing.T) {
	p := newScrollingPage(t, "html, body { height: 100%; }")
	assert.Equal(t, float32(300), p.doc.HTML().Box().H)
	assert.Equal(t, float32(300), p.doc.Body().Box().H)
	p.key(input.KeyEnd)
	assert.InDelta(t, pageHeight-300, p.scrolled(), 1e-3, "what overflows them still scrolls")
}

func TestTheWheelScrollsThePageFromAnywhereInTheViewport(t *testing.T) {
	p := newScrollingPage(t, "")
	p.wheel(geom.V2(300, 200), 1)
	first := p.scrolled()
	assert.Greater(t, first, float32(20), "three lines a notch")
	assert.InDelta(t, 60-first, p.list.Box().Y, 1e-3, "Box and BoxOf follow the page")
	box, ok := p.doc.BoxOf(p.filler)
	require.True(t, ok)
	assert.InDelta(t, 160-first, box.Y, 1e-3)
	p.wheel(geom.V2(300, 200), -1)
	assert.InDelta(t, 0, p.scrolled(), 1e-3)

	short := newScrollingPage(t, "#filler { height: 10px; } #low { position: absolute; top: 900px; height: 10px; }")
	short.doc.Body().Append(ui.Div(ui.ID("low")))
	short.w.RunFrames(2)
	require.Less(t, short.doc.HTML().Box().H, float32(300))
	short.wheel(geom.V2(300, 280), 1)
	assert.Positive(t, short.scrolled(), "below html's own box, the viewport still takes the wheel")
}

func TestAScrollerInThePageTakesTheWheelFirstThenThePage(t *testing.T) {
	p := newScrollingPage(t, "")
	at := p.list.Box().Center()
	p.wheel(at, 1)
	assert.Zero(t, p.scrolled())
	assert.Positive(t, p.list.Box().Y-p.list.Children()[0].Box().Y)
	for range 12 {
		p.in.AddWheel(geom.V2(0, 1))
		p.w.RunFrames(2)
	}
	assert.Positive(t, p.scrolled(), "at its end, the page takes the rest")
}

func TestPageKeysScrollTheViewport(t *testing.T) {
	p := newScrollingPage(t, "")
	p.key(input.KeyPageDown)
	assert.InDelta(t, 300, p.scrolled(), 1e-3, "a page is the viewport")
	p.key(input.KeyEnd)
	assert.InDelta(t, pageHeight-300, p.scrolled(), 1e-3)
	p.key(input.KeyHome)
	assert.InDelta(t, 0, p.scrolled(), 1e-3)
}

func TestAFixedBoxStaysPutAndAStickyOneSticksToTheViewport(t *testing.T) {
	p := newScrollingPage(t, "")
	badge := p.badge.Box()
	p.key(input.KeyPageDown)
	assert.Equal(t, badge, p.badge.Box())
	assert.InDelta(t, 0, p.header.Box().Y, 1e-3, "stuck to the viewport's top")
	p.key(input.KeyHome)
	assert.InDelta(t, 40, p.header.Box().Y, 1e-3)
}

func TestAClassicViewportBarTakesItsRoomAndDrags(t *testing.T) {
	p := newTallPage(t, "", ui.ScrollingViewport())
	assert.InDelta(t, 385, p.filler.Box().W, 1e-3, "the bar keeps 15 points of the window")
	assert.InDelta(t, 385-30-50, p.badge.Box().X, 1e-3, "and of what a fixed box is placed in")
	grab := geom.V2(400-10+2+3, 4+5)
	p.in.SetMouse(grab)
	p.w.RunFrames(1)
	p.in.SetMouseButton(input.MouseLeft, true)
	p.w.RunFrames(1)
	p.in.SetMouse(grab.Add(geom.V2(0, 100)))
	p.w.RunFrames(1)
	p.in.SetMouseButton(input.MouseLeft, false)
	p.w.RunFrames(2)
	thumb := float32(300) * 300 / pageHeight
	assert.InDelta(t, 100/(300-thumb)*(pageHeight-300), p.scrolled(), 1, "the thumb's travel is the page's")
	var r world.Reader[ui.Clicked]
	assert.Zero(t, r.Count(p.w))

	fits := newTallPage(t, "#filler { height: 10px; }", ui.ScrollingViewport())
	assert.InDelta(t, 400, fits.filler.Box().W, 1e-3, "no bar while the page fits")
}

func TestOverflowOnHTMLOrBodyIsTheViewports(t *testing.T) {
	for name, sheet := range map[string]string{
		"html": "html { overflow: hidden; }",
		"body": "body { overflow: hidden; }",
	} {
		t.Run(name, func(t *testing.T) {
			p := newTallPage(t, sheet, ui.ScrollingViewport())
			assert.InDelta(t, 400, p.filler.Box().W, 1e-3, "no bar, so no room for one")
			assert.Equal(t, float32(pageHeight), p.doc.Body().Box().H, "the element itself clips nothing")
			p.wheel(geom.V2(300, 200), 3)
			p.key(input.KeyEnd)
			assert.Zero(t, p.scrolled(), "hidden: neither the wheel nor the keys scroll it")
		})
	}
	p := newTallPage(t, "html { overflow-y: scroll; } #filler { height: 10px; }", ui.ScrollingViewport())
	assert.InDelta(t, 385, p.filler.Box().W, 1e-3, "scroll keeps the bar's room though the page fits")
}

func TestAClickLandsOnWhatThePageScrolledUnderThePointer(t *testing.T) {
	p := newScrollingPage(t, "")
	p.key(input.KeyEnd)
	at := p.end.Box().Center()
	require.Less(t, at.Y, float32(300))
	p.in.SetMouse(at)
	p.w.RunFrames(1)
	p.in.SetMouseButton(input.MouseLeft, true)
	p.w.RunFrames(1)
	p.in.SetMouseButton(input.MouseLeft, false)
	p.w.RunFrames(2)
	assert.Equal(t, 1, p.clicks)
}

func TestFocusAndScrollIntoViewRevealThroughTheViewport(t *testing.T) {
	p := newScrollingPage(t, "")
	p.doc.Focus(p.top)
	p.w.RunFrames(1)
	p.key(input.KeyTab)
	require.True(t, p.doc.Focused(p.end))
	b := p.end.Box()
	assert.InDelta(t, 300, b.Y+b.H, 1e-3, "the button just shows at the viewport's bottom")

	p.doc.ScrollIntoView(p.list)
	p.w.RunFrames(2)
	assert.InDelta(t, 60, p.scrolled(), 1e-3, "by the least that shows the list")
}

func TestAutomationSeesAnElementWhereThePageScrolledIt(t *testing.T) {
	doc := ui.NewDocument(ui.ScrollingViewport(), ui.OverlayScrollbars(), ui.CSS("body { margin: 0; }"))
	low := ui.Button(ui.ID("low"), ui.Style("display: block; width: 80px; height: 30px"), ui.Text("Low"))
	doc.Body().Append(ui.Div(ui.Style("height: 1000px")), low)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(doc)
	auto := automation.New(automation.Recording())
	w := ember.New(ember.Automation(auto)).AddEntities(win)
	w.RunFrames(2)
	doc.ScrollIntoView(low)
	w.RunFrames(3)
	_, els := auto.Snapshot("", "")
	var got geom.Rect
	for _, e := range els {
		if e.ID == "low" {
			got = e.Bounds
		}
	}
	assert.Equal(t, geom.Rect{Y: 270, W: 80, H: 30}, got)
}
