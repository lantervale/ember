package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

// A box at 0,0, 100 points square, scrolling down, its vertical bar 12
// thick with a 10-point button at each end: its track runs from 10 to
// 90, its thumb 26 long at its top.
func buttonedBox(t *testing.T) (*ui.Node, *input.State, func(int)) {
	t.Helper()
	s := ui.Div(ui.Class("s"), ui.Div(ui.Style("height: 300px")))
	d := ui.NewDocument(ui.CSS(`body { margin: 0; }
.s { width: 100px; height: 100px; overflow-y: scroll; }
.s::-webkit-scrollbar { width: 12px; }
.s::-webkit-scrollbar-thumb { background-color: #00f; }
.s::-webkit-scrollbar-button:start:decrement, .s::-webkit-scrollbar-button:end:increment {
  display: block; height: 10px; }`))
	d.Body().Append(s)
	win := platform.NewWindow(platform.Size(200, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	return s, win.Input(), w.RunFrames
}

func TestThePartUnderThePointerAndThePartPressedAreKept(t *testing.T) {
	s, in, run := buttonedBox(t)
	in.SetMouse(geom.V2(94, 20))
	run(1)
	assert.Equal(t, style.BarThumb, dom.Data(s).BarHover)
	in.SetMouseButton(input.MouseLeft, true)
	run(1)
	assert.Equal(t, style.BarThumb, dom.Data(s).BarPressed)
	in.SetMouse(geom.V2(150, 150))
	run(1)
	assert.Equal(t, style.BarThumb, dom.Data(s).BarHover, "a thumb dragged stays the part hovered, as Chrome's")
	in.SetMouseButton(input.MouseLeft, false)
	run(1)
	assert.Equal(t, style.NoBarPart, dom.Data(s).BarPressed)
	assert.Equal(t, style.NoBarPart, dom.Data(s).BarHover, "off the bars, no part")
	in.SetMouse(geom.V2(94, 30))
	run(1)
	assert.Equal(t, style.BarBackTrack, dom.Data(s).BarHover, "the drag took the thumb to the end")
}

func TestAPressOnAStyledButtonScrollsALine(t *testing.T) {
	s, in, run := buttonedBox(t)
	in.SetMouse(geom.V2(94, 95))
	run(1)
	in.SetMouseButton(input.MouseLeft, true)
	run(1)
	in.SetMouseButton(input.MouseLeft, false)
	run(20)
	assert.InDelta(t, 40, dom.Data(s).Scroll.Y, 1e-3, "forty points, as Chrome's line")
}

func TestAnInactiveWindowRestylesTheBarsWindowInactiveRules(t *testing.T) {
	s := ui.Div(ui.Class("s"), ui.Div(ui.Style("height: 300px")))
	d := ui.NewDocument(ui.CSS(`.s { width: 100px; height: 100px; overflow-y: scroll; }
.s::-webkit-scrollbar { width: 12px; }
.s::-webkit-scrollbar-thumb { background-color: #0000ff; }
.s::-webkit-scrollbar-thumb:vertical:window-inactive { background-color: #ff0000; }`))
	d.Body().Append(s)
	win := platform.NewWindow(platform.Size(200, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	thumb := func() color.Color { return dom.Data(s).Style.Scrollbar.V.Thumb.Background }
	assert.Equal(t, color.Hex("#0000ff"), thumb())
	win.SetFocused(false)
	w.RunFrames(1)
	assert.Equal(t, color.Hex("#ff0000"), thumb())
}
