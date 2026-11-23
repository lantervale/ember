package ui_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scroller is a page with a list 200×100 of ten 40-high buttons, inside
// an outer box 150 high followed by 300 points more, so both scroll.
type scroller struct {
	doc         *ui.Document
	w           *world.World
	in          *input.State
	outer, list *ui.Node
	items       []*ui.Node
}

func newScroller(t *testing.T, opts ...ui.DocumentOption) *scroller {
	t.Helper()
	s := &scroller{}
	s.list = ui.Div(ui.ID("list"), ui.Style("overflow: auto; width: 200px; height: 100px"))
	for i := range 10 {
		b := ui.Button(ui.Style("display: block; width: 180px; height: 40px"), ui.Text(fmt.Sprint("item ", i)))
		s.items = append(s.items, b)
		s.list.Append(b)
	}
	s.outer = ui.Div(ui.ID("outer"), ui.Style("overflow: auto; width: 240px; height: 150px"),
		s.list, ui.Div(ui.Style("height: 300px")))
	s.doc = ui.NewDocument(opts...)
	s.doc.Body().Append(s.outer)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(s.doc)
	s.w = ember.New().AddEntities(win)
	s.in = win.Input()
	s.w.RunFrames(2)
	return s
}

// scrolled is how far the list's content has moved up.
func (s *scroller) scrolled() float32 { return s.list.Box().Y - s.items[0].Box().Y }

func (s *scroller) outerScrolled() float32 { return s.outer.Box().Y - s.list.Box().Y }

// settle is enough frames for the longest of the user's glides, a
// notch's 12, to end.
const settle = 14

func (s *scroller) wheel(at geom.Vec2, d geom.Vec2) {
	s.in.SetMouse(at)
	s.w.RunFrames(1)
	s.in.AddWheel(d)
	s.w.RunFrames(settle)
}

func (s *scroller) key(k input.Key) {
	s.in.SetKey(k, true)
	s.w.RunFrames(1)
	s.in.SetKey(k, false)
	s.w.RunFrames(settle)
}

func TestTheWheelScrollsTheScrollerUnderThePointer(t *testing.T) {
	s := newScroller(t)
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.Positive(t, s.scrolled(), "a notch down")
	assert.Zero(t, s.outerScrolled(), "the inner one took it")
	first := s.scrolled()
	s.wheel(s.list.Box().Center(), geom.V2(0, -1))
	assert.InDelta(t, 0, s.scrolled(), 1e-3, "and back")
	assert.Greater(t, first, float32(20), "three lines a notch, as the old ember/ui")
}

func TestADocumentScalesWhatTheWheelScrolls(t *testing.T) {
	plain := newScroller(t)
	assert.InDelta(t, 1, plain.doc.WheelScroll(), 1e-6, "as the wheel says unless asked")
	plain.wheel(plain.list.Box().Center(), geom.V2(0, 1))
	notch := plain.scrolled()
	s := newScroller(t, ui.WheelScroll(2))
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.InDelta(t, 2*notch, s.scrolled(), 1e-3, "twice as far")
	s.doc.SetWheelScroll(0.5)
	assert.InDelta(t, 0.5, s.doc.WheelScroll(), 1e-6)
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.InDelta(t, 2.5*notch, s.scrolled(), 1e-3, "then half a notch more")
	s.doc.SetWheelScroll(0)
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.InDelta(t, 2.5*notch, s.scrolled(), 1e-3, "and not at all")
	assert.Zero(t, s.outerScrolled())
}

func TestANegativeWheelFactorReversesTheWheel(t *testing.T) {
	s := newScroller(t, ui.WheelScroll(-1))
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.Zero(t, s.scrolled(), "a notch that scrolled down scrolls up, and the list is at its start")
	assert.Zero(t, s.outerScrolled())
	s.wheel(s.list.Box().Center(), geom.V2(0, -1))
	assert.Positive(t, s.scrolled(), "the other way scrolls down")
	for range 20 {
		s.wheel(s.list.Box().Center(), geom.V2(0, -1))
	}
	assert.InDelta(t, 300, s.scrolled(), 1e-3, "the list at its end")
	assert.Positive(t, s.outerScrolled(), "the outer box took the rest, the same way")
}

func TestAtItsEndTheWheelScrollsTheNextScrollerOut(t *testing.T) {
	s := newScroller(t)
	for range 20 {
		s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	}
	assert.InDelta(t, 300, s.scrolled(), 1e-3, "the list at its end")
	assert.Positive(t, s.outerScrolled(), "the outer box took the rest")
}

func TestShiftTurnsTheWheelSideways(t *testing.T) {
	wide := ui.Div(ui.Style("width: 600px; height: 20px"))
	box := ui.Div(ui.Style("overflow: auto; width: 200px; height: 60px"), wide)
	doc := ui.NewDocument()
	doc.Body().Append(box)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	in := win.Input()
	w.RunFrames(2)
	in.SetMouse(box.Box().Center())
	in.SetKey(input.KeyLeftShift, true)
	w.RunFrames(1)
	in.AddWheel(geom.V2(0, 1))
	w.RunFrames(settle)
	in.SetKey(input.KeyLeftShift, false)
	assert.Positive(t, box.Box().X-wide.Box().X)
}

func TestARightToLeftScrollerScrollsLeftFromZero(t *testing.T) {
	wide := ui.Div(ui.Style("width: 600px; height: 20px"))
	box := ui.Div(ui.Attr("dir", "rtl"), ui.Style("overflow: auto; width: 200px; height: 60px"), wide)
	doc := ui.NewDocument()
	doc.Body().Append(box)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	in := win.Input()
	w.RunFrames(2)
	start := wide.Box().X
	assert.Equal(t, geom.Vec2{}, doc.Scroll(box), "at its right end")
	assert.InDelta(t, box.Box().X+box.Box().W-600, start, 1e-3, "the content's right on the box's right")

	in.SetMouse(box.Box().Center())
	in.SetKey(input.KeyLeftShift, true)
	w.RunFrames(1)
	in.AddWheel(geom.V2(0, 1))
	w.RunFrames(settle)
	assert.InDelta(t, 0, doc.Scroll(box).X, 1e-3, "nothing past its right end")
	in.AddWheel(geom.V2(0, -1))
	w.RunFrames(settle)
	in.SetKey(input.KeyLeftShift, false)
	assert.Negative(t, doc.Scroll(box).X, "the wheel scrolls it left")
	assert.InDelta(t, start-doc.Scroll(box).X, wide.Box().X, 1e-3)

	doc.SetScroll(box, geom.V2(-9999, 0))
	assert.Equal(t, geom.V2(-400, 0), doc.Scroll(box), "kept within what scrolls, as Chrome's scrollLeft")
	doc.SetScroll(box, geom.V2(50, 0))
	assert.Equal(t, geom.Vec2{}, doc.Scroll(box))
}

func TestDraggingTheThumbScrollsAndClicksNothing(t *testing.T) {
	s := newScroller(t)
	l := s.list.Box()
	grab := geom.V2(l.X+l.W-10+2+3, l.Y+4+5)
	s.in.SetMouse(grab)
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.w.RunFrames(1)
	s.in.SetMouse(grab.Add(geom.V2(0, 30)))
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.w.RunFrames(2)
	travel := float32(100 - 25)
	assert.InDelta(t, 30/travel*300, s.scrolled(), 1, "the thumb's travel is the overflow's")
	var r world.Reader[ui.Clicked]
	assert.Zero(t, r.Count(s.w), "the bar took the press")
}

func TestOverlayScrollbarsTakeNoRoomAndScrollAsClassicOnes(t *testing.T) {
	classic, s := newScroller(t), newScroller(t, ui.OverlayScrollbars())
	assert.True(t, s.doc.OverlayScrollbars())
	fill := func(s *scroller) float32 {
		row := ui.Div(ui.Style("height: 1px"))
		s.list.Append(row)
		s.w.RunFrames(2)
		return row.Box().W
	}
	assert.InDelta(t, 185, fill(classic), 0.01, "a classic bar keeps 15 points of the list")
	assert.InDelta(t, 200, fill(s), 0.01, "an overlay one none")

	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.Positive(t, s.scrolled(), "the wheel scrolls it")
	s.wheel(s.list.Box().Center(), geom.V2(0, -1))

	l := s.list.Box()
	grab := geom.V2(l.X+l.W-10+2+3, l.Y+4+5)
	s.in.SetMouse(grab)
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.w.RunFrames(1)
	s.in.SetMouse(grab.Add(geom.V2(0, 30)))
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.w.RunFrames(2)
	assert.Greater(t, s.scrolled(), float32(100), "its thumb drags, over the content")
	var r world.Reader[ui.Clicked]
	assert.Zero(t, r.Count(s.w), "and takes the press from the button under it")
}

func TestAClickOnTheTrackPagesBySevenEighthsOfTheView(t *testing.T) {
	s := newScroller(t)
	l := s.list.Box()
	s.in.SetMouse(geom.V2(l.X+l.W-10+2+3, l.Y+l.H-8))
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.w.RunFrames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.w.RunFrames(settle)
	assert.InDelta(t, 87, s.scrolled(), 1e-3, "seven eighths of the view, as Chrome's")
}

func TestPageAndEndKeysScrollTheFocusedElementsScroller(t *testing.T) {
	s := newScroller(t)
	s.doc.Focus(s.items[0])
	s.w.RunFrames(1)
	require.Zero(t, s.scrolled())
	s.key(input.KeyPageDown)
	assert.InDelta(t, 87, s.scrolled(), 1e-3, "a page is seven eighths of the view, as Chrome's")
	s.key(input.KeyEnd)
	assert.InDelta(t, 300, s.scrolled(), 1e-3)
	s.key(input.KeyHome)
	assert.InDelta(t, 0, s.scrolled(), 1e-3)
}

func TestFocusByKeyboardScrollsTheElementIntoViewByItsNearestEdge(t *testing.T) {
	s := newScroller(t)
	s.doc.Focus(s.items[0])
	s.w.RunFrames(1)
	for range 3 {
		s.key(input.KeyTab)
	}
	l, b := s.list.Box(), s.items[3].Box()
	assert.InDelta(t, l.Y+l.H, b.Y+b.H, 1e-3, "item 3 just shows at the bottom")
	for range 3 {
		s.key(input.KeyTab)
	}
	l, b = s.list.Box(), s.items[6].Box()
	assert.InDelta(t, l.Y+l.H, b.Y+b.H, 1e-3)
	s.in.SetKey(input.KeyLeftShift, true)
	for range 6 {
		s.key(input.KeyTab)
	}
	s.in.SetKey(input.KeyLeftShift, false)
	l, b = s.list.Box(), s.items[0].Box()
	assert.InDelta(t, l.Y, b.Y, 1e-3, "back up: item 0 at the top")
}

func TestWithNothingFocusedPageKeysScrollWhatIsUnderThePointer(t *testing.T) {
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.InDelta(t, 87, s.scrolled(), 1e-3, "the list under the pointer")
	s.key(input.KeyEnd)
	assert.InDelta(t, 300, s.scrolled(), 1e-3)

	o := s.outer.Box()
	s.in.SetMouse(geom.V2(o.X+o.W-20, o.Y+o.H-10))
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.Positive(t, s.outerScrolled(), "beside the list: the outer box")
}

func TestAFocusedScrollContainerPagesItself(t *testing.T) {
	s := newScroller(t)
	s.list.SetAttr("tabindex", "0")
	s.w.RunFrames(1)
	s.doc.Focus(s.list)
	s.w.RunFrames(1)
	s.key(input.KeyPageDown)
	assert.InDelta(t, 87, s.scrolled(), 1e-3)
}

func TestAPagingKeyIsTheGamesOnlyWhenNothingScrolled(t *testing.T) {
	s := newScroller(t)
	s.in.SetMouse(s.list.Box().Center())
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyPageDown, true)
	s.w.RunFrames(1)
	assert.False(t, s.in.Down(input.KeyPageDown), "it scrolled the list: the game doesn't see it")
	s.in.SetKey(input.KeyPageDown, false)
	s.w.RunFrames(2)

	s.in.SetMouse(geom.V2(390, 290))
	s.w.RunFrames(1)
	s.in.SetKey(input.KeyHome, true)
	s.w.RunFrames(1)
	assert.True(t, s.in.Down(input.KeyHome), "nothing there to scroll home: the game has it")
	s.in.SetKey(input.KeyHome, false)
}

func TestADocumentSaysAndSetsWhereAScrollerIsScrolled(t *testing.T) {
	s := newScroller(t)
	assert.Equal(t, geom.Vec2{}, s.doc.Scroll(s.list))
	s.wheel(s.list.Box().Center(), geom.V2(0, 1))
	assert.Equal(t, geom.V2(0, s.scrolled()), s.doc.Scroll(s.list), "what the wheel scrolled is read")
	assert.Positive(t, s.scrolled())

	s.doc.SetScroll(s.list, geom.V2(0, 120))
	assert.Equal(t, geom.V2(0, 120), s.doc.Scroll(s.list), "read at once")
	s.w.RunFrames(2)
	assert.InDelta(t, 120, s.scrolled(), 1e-3)
	assert.InDelta(t, 0, s.outerScrolled(), 1e-3, "no other scroller moves")

	s.doc.SetScroll(s.list, geom.V2(-50, 9999))
	assert.Equal(t, geom.V2(0, 300), s.doc.Scroll(s.list), "kept within what scrolls")
	s.w.RunFrames(2)
	assert.InDelta(t, 300, s.scrolled(), 1e-3)

	s.doc.SetScroll(s.items[0], geom.V2(0, 40))
	s.w.RunFrames(2)
	assert.Equal(t, geom.Vec2{}, s.doc.Scroll(s.items[0]), "an element that scrolls nothing stays at zero")
	assert.InDelta(t, 300, s.scrolled(), 1e-3)
}

func TestTheViewportsScrollIsTheRootElements(t *testing.T) {
	p := newScrollingPage(t, "")
	p.doc.SetScroll(p.doc.HTML(), geom.V2(0, 500))
	p.w.RunFrames(2)
	assert.InDelta(t, 500, p.scrolled(), 1e-3)
	assert.Equal(t, geom.V2(0, 500), p.doc.Scroll(p.doc.HTML()))

	p.key(input.KeyEnd)
	assert.Equal(t, geom.V2(0, pageHeight-300), p.doc.Scroll(p.doc.HTML()), "the keys' scroll is read too")

	p.doc.SetScroll(p.doc.HTML(), geom.V2(0, -10))
	p.w.RunFrames(2)
	assert.InDelta(t, 0, p.scrolled(), 1e-3)
}

func TestAScrollSetBeforeTheFirstLayoutIsKept(t *testing.T) {
	doc := ui.NewDocument(ui.ScrollingViewport(), ui.OverlayScrollbars(), ui.CSS("body { margin: 0; }"))
	low := ui.Div(ui.Style("height: 30px"))
	doc.Body().Append(ui.Div(ui.Style("height: 1000px")), low, ui.Div(ui.Style("height: 1000px")))
	doc.SetScroll(doc.HTML(), geom.V2(0, 1000))
	assert.Equal(t, geom.V2(0, 1000), doc.Scroll(doc.HTML()))
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	assert.InDelta(t, 0, low.Box().Y, 1e-3)
	doc.SetScroll(doc.HTML(), geom.V2(0, 1e6))
	w.RunFrames(2)
	assert.Equal(t, geom.V2(0, 1730), doc.Scroll(doc.HTML()), "and one past the end ends at it once laid out")
}
