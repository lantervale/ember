package ui_test

import (
	"image"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
)

// titleBar is a window's backend that counts what a header asks of it.
type titleBar struct {
	drags, doubles int
}

func (b *titleBar) StartDrag()                     { b.drags++ }
func (b *titleBar) DoubleClickTitleBar()           { b.doubles++ }
func (*titleBar) SetTitle(string)                  {}
func (*titleBar) SetCursor(platform.Cursor)        {}
func (*titleBar) Clipboard() string                { return "" }
func (*titleBar) SetClipboard(string)              {}
func (*titleBar) SetPosition(int, int)             {}
func (*titleBar) SetSize(int, int)                 {}
func (*titleBar) SetResizable(bool)                {}
func (*titleBar) SetAlwaysOnTop(bool)              {}
func (*titleBar) SetSizeLimits(_, _ geom.Vec2)     {}
func (*titleBar) SetAspectRatio(int, int)          {}
func (*titleBar) Maximize()                        {}
func (*titleBar) Minimize()                        {}
func (*titleBar) Restore()                         {}
func (*titleBar) SetFullscreen(bool)               {}
func (*titleBar) SetIcon([]image.Image)            {}
func (*titleBar) ScreenCursor() (float64, float64) { return 0, 0 }
func (*titleBar) WindowPosition() (int, int)       { return 0, 0 }

type header struct {
	t      *testing.T
	doc    *ui.Document
	win    *platform.Window
	w      *world.World
	bar    *titleBar
	clicks map[string]int
}

// newHeader is a 400×40 drag region at the window's top holding a button,
// a tab that clicks, a label, a no-drag box and a box no pointer touches,
// each 60 wide from the left, over a page that drags nothing.
func newHeader(t *testing.T) *header {
	t.Helper()
	h := &header{t: t, win: platform.NewWindow(platform.Size(400, 300), platform.OwnTitleBar()), bar: &titleBar{},
		clicks: map[string]int{}}
	h.doc = ui.NewDocument(ui.CSS(`body { margin: 0; }
#bar { display: flex; height: 40px; app-region: drag; }
#bar > * { width: 60px; height: 40px; padding: 0; border: none; }
#still { app-region: no-drag; }
#glass { pointer-events: none; }
#page { height: 200px; }`))
	count := func(id string) ui.Arg { return ui.OnClick(func() { h.clicks[id]++ }) }
	h.doc.Body().Append(
		ui.Header(ui.ID("bar"),
			ui.Button(ui.ID("button"), ui.Text("b"), count("button")),
			ui.Div(ui.ID("tab"), ui.TabIndex(0), ui.Span(ui.ID("name"), ui.Text("t")), count("tab")),
			ui.Span(ui.ID("label"), ui.Text("l")),
			ui.Div(ui.ID("still")),
			ui.Div(ui.ID("glass"))),
		ui.Div(ui.ID("page")))
	h.win.Add(h.doc)
	h.win.Attach(h.bar)
	h.w = ember.New().AddEntities(h.win)
	h.w.RunFrames(2)
	return h
}

func (h *header) press(x, y float32) {
	h.t.Helper()
	in := h.win.Input()
	in.SetMouse(geom.V2(x, y))
	h.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	h.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	h.w.RunFrames(2)
}

func TestAPressInADragRegionDragsTheWindow(t *testing.T) {
	h := newHeader(t)
	h.press(380, 20)
	assert.Equal(t, 1, h.bar.drags, "the bar's empty end")
	h.press(150, 20)
	assert.Equal(t, 2, h.bar.drags, "a label in the bar is the bar's")
	h.press(270, 20)
	assert.Equal(t, 3, h.bar.drags, "under a pointer-events: none box the bar still takes the press")
	h.press(200, 100)
	assert.Equal(t, 3, h.bar.drags, "the page drags nothing")
	assert.Zero(t, h.bar.doubles)
}

func TestWhatClicksInADragRegionStillClicks(t *testing.T) {
	h := newHeader(t)
	h.press(30, 20)
	h.press(90, 20)
	h.press(210, 20)
	assert.Equal(t, map[string]int{"button": 1, "tab": 1}, h.clicks, "a button, and a tabindex through its label")
	assert.Zero(t, h.bar.drags, "a control, what clicks and a no-drag box drag nothing")
}

func TestADoublePressInADragRegionIsTheTitleBars(t *testing.T) {
	h := newHeader(t)
	in := h.win.Input()
	in.SetMouse(geom.V2(380, 20))
	h.w.RunFrames(1)
	for range 2 {
		in.SetMouseButton(input.MouseLeft, true)
		h.w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		h.w.RunFrames(1)
	}
	assert.Equal(t, 1, h.bar.drags, "the first press drags")
	assert.Equal(t, 1, h.bar.doubles, "the second, at once, is a double click on the title bar")
	h.w.RunFrames(60)
	h.press(380, 20)
	assert.Equal(t, 2, h.bar.drags, "a press a second later drags again")
}

func TestADragPressLeavesTheFocusAndClicksNothing(t *testing.T) {
	h := newHeader(t)
	button := h.doc.ByID("button")
	h.doc.Focus(button)
	h.w.RunFrames(1)
	h.press(380, 20)
	assert.True(t, h.doc.Focused(button), "the focus stays where it was, as under a system title bar")
	assert.Empty(t, h.clicks)
}
