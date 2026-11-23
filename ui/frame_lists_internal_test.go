package ui

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// framedSelect is a select at the foot of a short iframe, a button of
// the page round it lying under where its list opens.
type framedSelect struct {
	outer, child *Document
	sel, frame   *Node
	below        *Node
	win          *platform.Window
	w            *world.World
	changed      int
	clicked      int
}

func newFramedSelect(t *testing.T) *framedSelect {
	t.Helper()
	f := &framedSelect{}
	f.child = NewDocument(CSS(`body { margin: 0; } select { position: absolute; left: 10px; top: 30px; width: 120px; }`))
	f.sel = Select(Option(Text("one")), Option(Text("two")), Option(Text("three")), Option(Text("four")),
		OnChange(func() { f.changed++ }))
	f.child.Body().Append(f.sel)
	f.outer = NewDocument(CSS(`body { margin: 0; }
iframe { position: absolute; left: 50px; top: 40px; width: 200px; height: 60px; border: 0; }
.below { position: absolute; left: 50px; top: 100px; width: 200px; height: 100px; }`))
	f.frame = Iframe(f.child)
	f.below = Button(Class("below"), Text("below"), OnClick(func() { f.clicked++ }))
	f.outer.Body().Append(f.frame, f.below)
	f.win = platform.NewWindow(platform.Size(400, 300))
	f.win.Add(f.outer)
	f.w = ember.New().AddEntities(f.win)
	f.win.Input().SetMouse(geom.V2(390, 290))
	f.w.RunFrames(4)
	return f
}

func (f *framedSelect) click(at geom.Vec2) {
	in := f.win.Input()
	in.SetMouse(at)
	f.w.RunFrames(2)
	in.SetMouseButton(input.MouseLeft, true)
	f.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	f.w.RunFrames(2)
}

func (f *framedSelect) open(t *testing.T) *control.Select {
	t.Helper()
	r, ok := f.child.BoxOf(f.sel)
	require.True(t, ok)
	f.click(geom.V2(50+r.X+r.W/2, 40+r.Y+r.H/2))
	m, _ := f.child.machine(f.sel).(*control.Select)
	require.NotNil(t, m)
	require.True(t, m.Open, "a click opens it")
	return m
}

func TestAFramedSelectsListShowsWholeOverThePage(t *testing.T) {
	f := newFramedSelect(t)
	f.open(t)
	require.Len(t, f.outer.lists.rects, 1, "drawn by the window's Document")
	list := f.outer.lists.rects[0]
	frame, _ := f.outer.BoxOf(f.frame)
	assert.Greater(t, list.Y+list.H, frame.Y+frame.H, "past the frame's foot")
	assert.Less(t, list.Y+list.H, float32(300), "inside the window")
	assert.InDelta(t, 40+30+24, list.Y, 4, "under the select")
}

func TestAFramedSelectsListTakesThePointerOverThePage(t *testing.T) {
	f := newFramedSelect(t)
	m := f.open(t)
	list := f.outer.lists.rects[0]
	row := (list.H - 2) / 4
	f.click(geom.V2(list.X+20, list.Y+1+3.5*row))
	assert.False(t, m.Open, "picked, it closes")
	assert.Equal(t, 3, m.Selected, "the fourth option, below the frame's foot")
	assert.Equal(t, 1, f.changed)
	assert.Zero(t, f.clicked, "the button under the list never hears it")
	assert.Empty(t, f.outer.lists.rects, "nothing over the page once closed")
}

func TestAPressOnThePageClosesAFramedList(t *testing.T) {
	f := newFramedSelect(t)
	m := f.open(t)
	f.click(geom.V2(380, 280))
	assert.False(t, m.Open, "a press outside it closes it")
	assert.Zero(t, f.changed)
}

func TestAFramedListIsDrawnEveryFrameOverAReplayedPage(t *testing.T) {
	f := newFramedSelect(t)
	f.open(t)
	before := repaintsIn(f.w)
	f.w.RunFrames(4)
	assert.Equal(t, before, repaintsIn(f.w), "the pages replay")
	assert.Len(t, f.outer.lists.rects, 1, "the list over them, frame after frame")
}
