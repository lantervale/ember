package ui_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/world"
)

// snapPage is a page whose scroller, 200×100 inside an outer box that
// scrolls too, holds ten 60-high items, each a focusable button.
type snapPage struct {
	doc           *ui.Document
	w             *world.World
	in            *input.State
	outer, list   *ui.Node
	items, labels []*ui.Node
}

func newSnapPage(t *testing.T, sheet string) *snapPage {
	t.Helper()
	p := &snapPage{}
	p.list = ui.Div(ui.ID("list"))
	for i := range 10 {
		label := ui.I()
		it := ui.Button(ui.Class("it"), ui.Text(fmt.Sprint("item ", i)), label)
		p.items, p.labels = append(p.items, it), append(p.labels, label)
		p.list.Append(it)
	}
	p.outer = ui.Div(ui.ID("outer"), p.list, ui.Div(ui.Style("height: 300px")))
	p.doc = ui.NewDocument(ui.CSS(`
#outer { overflow: auto; width: 240px; height: 150px; }
#list { overflow: auto; width: 200px; height: 100px; }
.it { display: block; width: 180px; height: 60px; border: 0; padding: 0; scroll-snap-align: start; }
i { display: block; width: 1px; height: 1px; }
` + sheet))
	p.doc.Body().Append(p.outer)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(p.doc)
	p.w = ember.New().AddEntities(win)
	p.in = win.Input()
	p.w.RunFrames(2)
	return p
}

func (p *snapPage) at() float32 { return p.doc.Scroll(p.list).Y }

func (p *snapPage) wheel(d geom.Vec2) {
	p.in.SetMouse(p.list.Box().Center())
	p.w.RunFrames(1)
	p.in.AddWheel(d)
	p.w.RunFrames(1)
}

func TestTheWheelSnapsOnceItsGestureEnds(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-snap-type: y mandatory; }")
	p.wheel(geom.V2(0, 1))
	free := p.at()
	assert.Positive(t, free, "the wheel scrolls freely while it turns")
	p.w.RunFrames(40)
	assert.InDelta(t, 60, p.at(), 0.01, "then glides on to the next item ahead")
	p.wheel(geom.V2(0, -1))
	p.w.RunFrames(40)
	assert.InDelta(t, 0, p.at(), 0.01, "and back")
}

func TestAProximityScrollerSnapsOnlyNearASnapPosition(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-snap-type: y proximity; } .it { height: 200px; }")
	p.doc.SetScroll(p.list, geom.V2(0, 70))
	p.w.RunFrames(2)
	assert.InDelta(t, 70, p.at(), 0.01, "70 from 0 and 130 from 200: past a third of the view")
	p.doc.SetScroll(p.list, geom.V2(0, 170))
	p.w.RunFrames(2)
	assert.InDelta(t, 200, p.at(), 0.01, "within a third of it")
}

func TestPageKeysSnapToTheNextPositionAhead(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-snap-type: y mandatory; }")
	p.doc.Focus(p.items[0])
	p.w.RunFrames(2)
	p.in.SetKey(input.KeyPageDown, true)
	p.w.RunFrames(1)
	p.in.SetKey(input.KeyPageDown, false)
	p.w.RunFrames(2)
	assert.InDelta(t, 120, p.at(), 0.01, "a page of 100 lands on the item at 120, the nearest ahead")
	p.in.SetKey(input.KeyHome, true)
	p.w.RunFrames(1)
	p.in.SetKey(input.KeyHome, false)
	p.w.RunFrames(2)
	assert.InDelta(t, 0, p.at(), 0.01)
}

func TestASnapStopAlwaysHoldsTheScrollThatPassesIt(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-snap-type: y mandatory; } .it:nth-child(3) { scroll-snap-stop: always; }")
	p.doc.ScrollBy(p.list, geom.V2(0, 400))
	p.w.RunFrames(2)
	assert.InDelta(t, 120, p.at(), 0.01, "ScrollBy stops at the third item")
	p.doc.SetScroll(p.list, geom.V2(0, 0))
	p.doc.SetScroll(p.list, geom.V2(0, 400))
	p.w.RunFrames(2)
	assert.InDelta(t, 420, p.at(), 0.01, "SetScroll goes straight to the nearest, as scrollTo")
}

func TestASmoothScrollGlidesOnChromesCurve(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-behavior: smooth; }")
	p.doc.SetScroll(p.list, geom.V2(0, 400))
	assert.Zero(t, p.at(), "it glides from where it was")
	// 400 points take √400 frames of 1/60s: a third of a second, twenty
	// frames of the World's step; the first frame starts it.
	p.w.RunFrames(11)
	curve := css.CubicBezier(0.4, 0, 0, 1)
	assert.InDelta(t, 400*curve(0.5), p.at(), 0.5, "half way through its time")
	p.w.RunFrames(9)
	assert.Less(t, p.at(), float32(400))
	p.w.RunFrames(1)
	assert.InDelta(t, 400, p.at(), 0.01, "there in a third of a second")

	p.doc.ScrollIntoView(p.items[0])
	p.w.RunFrames(3)
	assert.Positive(t, p.at(), "ScrollIntoView glides too")
	p.w.RunFrames(30)
	assert.InDelta(t, 0, p.at(), 0.01)
}

func TestASmoothScrollGlidesToItsSnapPosition(t *testing.T) {
	p := newSnapPage(t, "#list { scroll-behavior: smooth; scroll-snap-type: y mandatory; }")
	p.doc.SetScroll(p.list, geom.V2(0, 200))
	p.w.RunFrames(40)
	assert.InDelta(t, 180, p.at(), 0.01)
}

func TestOverscrollBehaviorKeepsTheWheelFromTheScrollerRoundIt(t *testing.T) {
	for _, c := range []struct {
		sheet string
		outer bool
	}{{"", true}, {"#list { overscroll-behavior: contain; }", false}, {"#list { overscroll-behavior-y: none; }", false},
		{"#list { overscroll-behavior-x: contain; }", true}} {
		p := newSnapPage(t, c.sheet)
		p.doc.SetScroll(p.list, geom.V2(0, 1000))
		p.w.RunFrames(2)
		p.wheel(geom.V2(0, 1))
		p.w.RunFrames(2)
		assert.Equal(t, c.outer, p.doc.Scroll(p.outer).Y > 0, "%q: the outer box scrolled", c.sheet)
	}
}

func TestScrollStateAnswersSnappedAndScrolled(t *testing.T) {
	p := newSnapPage(t, `#list { scroll-snap-type: y mandatory; container-type: scroll-state; }
		.it { container-type: scroll-state; }
		@container scroll-state(snapped: y) { i { width: 2px; } }
		@container scroll-state(snapped: none) { i { height: 2px; } }
		.mk { width: 1px; height: 1px; }
		@container scroll-state(scrolled: bottom) { #mark { width: 3px; } }
		@container scroll-state(scrolled: top) { #mark { width: 4px; } }
		@container scroll-state(scrolled: none) { #mark { width: 5px; } }`)
	mark := ui.Div(ui.ID("mark"), ui.Class("mk"))
	p.list.Append(mark)
	p.w.RunFrames(2)
	snapped := func() []float32 {
		var out []float32
		for _, l := range p.labels[:4] {
			out = append(out, l.Box().W+l.Box().H*10)
		}
		return out
	}
	assert.Equal(t, []float32{12, 21, 21, 21}, snapped(), "the first item is snapped at the first layout")
	assert.InDelta(t, 5, mark.Box().W, 0.01, "nothing scrolled by the user yet")
	p.doc.SetScroll(p.list, geom.V2(0, 70))
	p.w.RunFrames(2)
	assert.Equal(t, []float32{21, 12, 21, 21}, snapped(), "the second, once scrolled to it")
	assert.InDelta(t, 5, mark.Box().W, 0.01, "SetScroll is no relative scroll")
	p.doc.ScrollBy(p.list, geom.V2(0, 10))
	p.w.RunFrames(2)
	assert.InDelta(t, 3, mark.Box().W, 0.01, "ScrollBy down")
	p.wheel(geom.V2(0, -1))
	p.w.RunFrames(40)
	assert.InDelta(t, 4, mark.Box().W, 0.01, "the wheel up")
	require.Empty(t, p.doc.Warnings())
}
