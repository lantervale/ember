package ui_test

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"weak"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/errlog"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/align"
	"github.com/lantervale/ember/world"
)

const frameHostSheet = `
body { margin: 0; user-select: text }
button { display: block; width: 100px; height: 30px }
input { display: block; width: 120px; height: 20px }
#slot { display: block; width: 300px; height: 150px; margin: 20px 0 0 40px; border: 0 }
#cover { position: absolute; left: 200px; top: 100px; width: 300px; height: 60px; background: #c33 }
#long { height: 2000px }`

const frameChildSheet = `
body { margin: 0; user-select: text }
button { display: block; width: 100px; height: 30px }
input { display: block; width: 120px; height: 20px }
#wide { width: 280px }
#tall { height: 600px }
@media (max-width: 400px) { #narrow { width: 50px } }`

// framePage is a Document with a button, a field, an iframe showing
// another Document, and a box lying over the right half of the iframe.
type framePage struct {
	host, child              *ui.Document
	slot, cover              *ui.Node
	before, after, hostField *ui.Node
	first, wide, field, last *ui.Node
	hostText, childText      *ui.Node
	clicks                   map[string]int
	win                      *platform.Window
	w                        *world.World
	in                       *input.State
}

func newFramePage(t *testing.T, opts ...ember.Option) *framePage {
	t.Helper()
	p := &framePage{clicks: map[string]int{}}
	count := func(name string) ui.Arg { return ui.OnClick(func() { p.clicks[name]++ }) }
	p.child = ui.NewDocument(ui.ScrollingViewport(), ui.CSS(frameChildSheet))
	p.first = ui.Button(ui.ID("first"), ui.Text("first"), count("first"))
	p.wide = ui.Button(ui.ID("wide"), ui.Text("wide"), count("wide"))
	p.field = ui.Input(ui.ID("field"))
	p.last = ui.Button(ui.ID("last"), ui.Text("last"), count("last"))
	p.childText = ui.P(ui.ID("words"), ui.Text("words of the frame"))
	p.child.Body().Append(p.first, p.wide, p.field, p.last, p.childText, ui.Div(ui.ID("narrow")), ui.Div(ui.ID("tall")))

	p.host = ui.NewDocument(ui.ScrollingViewport(), ui.CSS(frameHostSheet))
	p.before = ui.Button(ui.ID("before"), ui.Text("before"), count("before"))
	p.hostField = ui.Input(ui.ID("host-field"))
	p.slot = ui.Iframe(p.child, ui.ID("slot"))
	p.cover = ui.Button(ui.ID("cover"), ui.Text("cover"), ui.TabIndex(-1), count("cover"))
	p.after = ui.Button(ui.ID("after"), ui.Text("after"), count("after"))
	p.hostText = ui.P(ui.ID("host-words"), ui.Text("words of the page"))
	p.host.Body().Append(p.before, p.hostField, p.slot, p.cover, p.after, p.hostText, ui.Div(ui.ID("long")))

	p.win = platform.NewWindow(platform.Size(800, 500))
	p.win.Add(p.host)
	p.w = ember.New(opts...).AddEntities(p.win)
	p.in = p.win.Input()
	p.frames(2)
	return p
}

func (p *framePage) frames(n int) { p.w.RunFrames(n) }

// inFrame is the window's point of a point of the child Document.
func (p *framePage) inFrame(x, y float32) geom.Vec2 {
	r, _ := p.host.BoxOf(p.slot)
	return geom.V2(r.X+x, r.Y+y)
}

func (p *framePage) click(at geom.Vec2) {
	p.in.SetMouse(at)
	p.frames(1)
	p.in.SetMouseButton(input.MouseLeft, true)
	p.frames(1)
	p.in.SetMouseButton(input.MouseLeft, false)
	p.frames(2)
}

func (p *framePage) key(k input.Key, held ...input.Key) {
	for _, h := range held {
		p.in.SetKey(h, true)
	}
	p.in.SetKey(k, true)
	p.frames(1)
	p.in.SetKey(k, false)
	for _, h := range held {
		p.in.SetKey(h, false)
	}
	p.frames(2)
}

func centre(t *testing.T, d *ui.Document, n *ui.Node) geom.Vec2 {
	t.Helper()
	r, ok := d.BoxOf(n)
	if !ok {
		t.Fatalf("%s has no box", n.ID())
	}
	return r.Center()
}

func TestAnIframeLaysItsDocumentOutInItsContentBox(t *testing.T) {
	p := newFramePage(t)
	body, _ := p.child.BoxOf(p.child.Body())
	if body.W != 300-15 {
		t.Fatalf("the frame's body is %v wide, want the slot's 300 less a scrollbar", body.W)
	}
	if narrow, _ := p.child.BoxOf(p.child.ByID("narrow")); narrow.W != 50 {
		t.Fatalf("@media (max-width: 400px) is not answered for the frame's box: %v", narrow.W)
	}
	if got := p.host.Frame(p.slot); got != p.child {
		t.Fatalf("Frame gives %p, want the child", got)
	}
	if !p.host.FrameShown(p.slot) {
		t.Fatal("the frame is in view and not said shown")
	}
}

func TestAnIframeResizedLaysItsDocumentOutAgain(t *testing.T) {
	p := newFramePage(t)
	p.slot.SetAttr("style", "width: 500px; height: 80px")
	p.frames(2)
	if body, _ := p.child.BoxOf(p.child.Body()); body.W != 500-15 {
		t.Fatalf("the frame's body is %v wide after the resize", body.W)
	}
	if narrow, _ := p.child.BoxOf(p.child.ByID("narrow")); narrow.W == 50 {
		t.Fatal("@media still answers for the old width")
	}
}

func TestAClickInAnIframeIsItsDocuments(t *testing.T) {
	p := newFramePage(t)
	p.click(p.inFrame(20, 10))
	if p.clicks["first"] != 1 || len(p.clicks) != 1 {
		t.Fatalf("clicks %v, want the frame's first button alone", p.clicks)
	}
}

func TestAnElementOverAnIframeShieldsIt(t *testing.T) {
	p := newFramePage(t)
	cover, _ := p.host.BoxOf(p.cover)
	wide, _ := p.child.BoxOf(p.wide)
	at := p.inFrame(wide.X+wide.W-20, wide.Y+wide.H/2)
	if !cover.Contains(at) {
		t.Fatalf("the cover %v does not lie over the frame's button at %v", cover, at)
	}
	p.click(at)
	if p.clicks["cover"] != 1 || p.clicks["wide"] != 0 {
		t.Fatalf("clicks %v, want the cover's alone", p.clicks)
	}
	if ui.HoveredElement(p.child) != nil {
		t.Fatal("the frame's button under the cover is hovered")
	}
	p.click(p.inFrame(wide.X+10, wide.Y+2))
	if p.clicks["wide"] != 1 {
		t.Fatalf("clicks %v: the part of the button the cover leaves takes none", p.clicks)
	}
}

func TestTheWheelScrollsAnIframeThenThePage(t *testing.T) {
	p := newFramePage(t)
	p.in.SetMouse(p.inFrame(20, 100))
	p.frames(2)
	slotAt := func() float32 { r, _ := p.host.BoxOf(p.slot); return r.Y }
	firstAt := func() float32 { r, _ := p.child.BoxOf(p.first); return r.Y }
	slot, first := slotAt(), firstAt()
	p.in.AddWheel(geom.V2(0, 2))
	p.frames(2)
	if firstAt() >= first || slotAt() != slot {
		t.Fatalf("a wheel over the frame moved the frame by %v and the page by %v", firstAt()-first, slotAt()-slot)
	}
	for first = firstAt() + 1; firstAt() < first; {
		first = firstAt()
		p.in.AddWheel(geom.V2(0, 3))
		p.frames(2)
	}
	if slotAt() >= slot {
		t.Fatal("the wheel does not reach the page once the frame is at its end")
	}
	p.in.SetMouse(p.inFrame(20, 100))
	p.frames(2)
	slot, first = slotAt(), firstAt()
	p.in.AddWheel(geom.V2(0, -1))
	p.frames(2)
	if firstAt() <= first || slotAt() != slot {
		t.Fatalf("a wheel back moved the frame by %v and the page by %v", firstAt()-first, slotAt()-slot)
	}
}

func TestTheWheelFactorOfTheOutermostDocumentHoldsInItsIframes(t *testing.T) {
	p := newFramePage(t)
	p.host.SetWheelScroll(-1)
	p.child.SetWheelScroll(3)
	p.in.SetMouse(p.inFrame(20, 100))
	p.frames(2)
	slotAt := func() float32 { r, _ := p.host.BoxOf(p.slot); return r.Y }
	firstAt := func() float32 { r, _ := p.child.BoxOf(p.first); return r.Y }
	slot, first := slotAt(), firstAt()
	p.in.AddWheel(geom.V2(0, -1))
	p.frames(2)
	moved := first - firstAt()
	if moved <= 0 || slotAt() != slot {
		t.Fatalf("a reversed wheel moved the frame by %v and the page by %v", -moved, slotAt()-slot)
	}
	p.host.SetWheelScroll(-2)
	p.in.AddWheel(geom.V2(0, -1))
	p.frames(2)
	if got := first - firstAt() - moved; got < 2*moved-0.01 || got > 2*moved+0.01 {
		t.Fatalf("at twice the factor the frame moved by %v after %v", got, moved)
	}
	for first = firstAt() + 1; firstAt() < first; {
		first = firstAt()
		p.in.AddWheel(geom.V2(0, -3))
		p.frames(2)
	}
	if slotAt() >= slot {
		t.Fatal("the reversed wheel does not reach the page once the frame is at its end")
	}
}

func TestTabWalksThroughAnIframeBothWays(t *testing.T) {
	p := newFramePage(t)
	focused := func() string {
		if n := p.host.ByID("slot"); p.host.Focused(n) {
			for _, c := range []*ui.Node{p.first, p.wide, p.field, p.last} {
				if p.child.Focused(c) {
					return "frame:" + c.ID()
				}
			}
			return "slot"
		}
		for _, n := range []*ui.Node{p.before, p.hostField, p.after} {
			if p.host.Focused(n) {
				return n.ID()
			}
		}
		return ""
	}
	p.host.Focus(p.before)
	p.frames(2)
	want := []string{"host-field", "frame:first", "frame:wide", "frame:field", "frame:last", "after", "before"}
	for i, w := range want {
		p.key(input.KeyTab)
		if got := focused(); got != w {
			t.Fatalf("Tab %d reached %q, want %q", i+1, got, w)
		}
	}
	back := []string{"after", "frame:last", "frame:field", "frame:wide", "frame:first", "host-field", "before"}
	for i, w := range back {
		p.key(input.KeyTab, input.KeyLeftShift)
		if got := focused(); got != w {
			t.Fatalf("Shift+Tab %d reached %q, want %q", i+1, got, w)
		}
	}
}

func TestTheKeysGoToTheFocusedDocumentOnly(t *testing.T) {
	p := newFramePage(t)
	p.click(centre(t, p.host, p.hostField))
	p.in.AddText('a')
	p.frames(2)
	field, _ := p.child.BoxOf(p.field)
	p.click(p.inFrame(field.X+5, field.Y+5))
	if !p.host.Focused(p.slot) || !p.child.Focused(p.field) || p.host.Focused(p.hostField) {
		t.Fatal("a click in the frame's field does not move the focus into the frame")
	}
	p.in.AddText('b')
	p.frames(2)
	if p.host.Value(p.hostField) != "a" || p.child.Value(p.field) != "b" {
		t.Fatalf("typed %q in the page and %q in the frame, want a and b", p.host.Value(p.hostField),
			p.child.Value(p.field))
	}
	p.click(centre(t, p.host, p.hostField).Add(geom.V2(40, 0)))
	if p.child.Focused(p.field) {
		t.Fatal("the frame's field keeps the focus the page took back")
	}
	p.in.AddText('c')
	p.frames(2)
	if p.host.Value(p.hostField) != "ac" || p.child.Value(p.field) != "b" {
		t.Fatalf("typed %q in the page and %q in the frame, want ac and b", p.host.Value(p.hostField),
			p.child.Value(p.field))
	}
}

func TestAFocusGivenInAnIframeTakesThePagesFocus(t *testing.T) {
	p := newFramePage(t)
	p.host.Focus(p.before)
	p.frames(2)
	p.child.Focus(p.field)
	p.frames(3)
	if !p.host.Focused(p.slot) || !p.child.Focused(p.field) {
		t.Fatal("the page's focus does not follow a Focus in the frame")
	}
}

func TestEachDocumentSelectsItsOwnTextAndOneAtATime(t *testing.T) {
	p := newFramePage(t)
	drag := func(from, to geom.Vec2) {
		p.in.SetMouse(from)
		p.frames(1)
		p.in.SetMouseButton(input.MouseLeft, true)
		p.frames(1)
		p.in.SetMouse(to)
		p.frames(2)
		p.in.SetMouseButton(input.MouseLeft, false)
		p.frames(2)
	}
	words, _ := p.host.BoxOf(p.hostText)
	drag(geom.V2(words.X+1, words.Y+words.H/2), geom.V2(words.X+60, words.Y+words.H/2))
	if p.host.Selection().Empty() || !p.child.Selection().Empty() {
		t.Fatalf("a drag in the page selected %q there and %q in the frame", p.host.Selection().Text,
			p.child.Selection().Text)
	}
	inner, _ := p.child.BoxOf(p.childText)
	drag(p.inFrame(inner.X+1, inner.Y+inner.H/2), p.inFrame(inner.X+60, inner.Y+inner.H/2))
	if p.child.Selection().Empty() || !p.host.Selection().Empty() {
		t.Fatalf("a drag in the frame selected %q there and left %q in the page", p.child.Selection().Text,
			p.host.Selection().Text)
	}
	if got := p.child.Selection().Text; !strings.Contains(p.childText.Text(), got) {
		t.Fatalf("the frame selected %q", p.child.Selection().Text)
	}
}

func TestElementAtChainsThroughAnIframe(t *testing.T) {
	p := newFramePage(t)
	at := p.inFrame(20, 10)
	if got := p.host.ElementAt(at); got != p.slot {
		t.Fatalf("the page finds %v under a point of the frame", got)
	}
	inner, ok := p.host.FramePoint(p.slot, at)
	if !ok || inner != geom.V2(20, 10) {
		t.Fatalf("FramePoint gives %v %v", inner, ok)
	}
	if got := p.child.ElementAt(inner); got != p.first {
		t.Fatalf("the frame finds %v there, want its first button", got)
	}
	if _, ok = p.host.FramePoint(p.slot, geom.V2(1, 1)); ok {
		t.Fatal("a point outside the frame is said to be in it")
	}
}

func TestAnIframeZoomedWithItsPage(t *testing.T) {
	p := newFramePage(t)
	p.host.SetZoom(2)
	p.frames(3)
	if body, _ := p.child.BoxOf(p.child.Body()); body.W != 300-15 {
		t.Fatalf("zoomed, the frame's body is %v wide", body.W)
	}
	first, _ := p.child.BoxOf(p.first)
	slot, _ := p.host.BoxOf(p.slot)
	at := geom.V2(slot.X+(first.X+20)*2, slot.Y+(first.Y+10)*2)
	inner, _ := p.host.FramePoint(p.slot, at)
	if p.child.ElementAt(inner) != p.first {
		t.Fatalf("zoomed, FramePoint gives %v, not a point of the first button", inner)
	}
	p.click(at)
	if p.clicks["first"] != 1 {
		t.Fatalf("zoomed, clicks %v", p.clicks)
	}
}

func TestAutomationIDsOfAnIframeAreUnderIt(t *testing.T) {
	auto := automation.New(automation.Recording())
	p := newFramePage(t, ember.Automation(auto))
	p.frames(1)
	_, elements := auto.Snapshot("", "")
	var found *automation.Element
	for i, e := range elements {
		if strings.HasSuffix(e.ID, "slot/first") {
			found = &elements[i]
		}
	}
	if found == nil {
		ids := make([]string, len(elements))
		for i, e := range elements {
			ids[i] = e.ID
		}
		t.Fatalf("no element slot/first among %v", ids)
	}
	slot, _ := p.host.BoxOf(p.slot)
	first := p.first.Box()
	if want := (geom.Rect{X: slot.X + first.X, Y: slot.Y + first.Y, W: first.W, H: first.H}); found.Bounds != want {
		t.Fatalf("its bounds are %v, want %v in the window", found.Bounds, want)
	}
	auto.Queue(found.ID, automation.ActPress, nil)
	p.frames(3)
	if p.clicks["first"] != 1 {
		t.Fatalf("a press by its ID clicked %v", p.clicks)
	}
}

func TestAnIframeInAnIframe(t *testing.T) {
	clicks := 0
	sheet := ui.CSS(`body { margin: 0 } button { display: block; width: 60px; height: 20px }
		iframe { display: block; border: 0; margin: 10px; width: 200px; height: 80px }`)
	inner := ui.NewDocument(sheet)
	deep := ui.Button(ui.ID("deep"), ui.Text("deep"), ui.OnClick(func() { clicks++ }))
	inner.Body().Append(deep)
	middle := ui.NewDocument(sheet)
	nested := ui.Iframe(inner, ui.ID("nested"))
	next := ui.Button(ui.ID("next"), ui.Text("next"))
	middle.Body().Append(nested, next)
	outer := ui.NewDocument(sheet, ui.CSS(`#frame { width: 400px; height: 200px }`))
	start := ui.Button(ui.ID("start"), ui.Text("start"))
	frame := ui.Iframe(middle, ui.ID("frame"))
	outer.Body().Append(start, frame)
	win := platform.NewWindow(platform.Size(800, 500))
	win.Add(outer)
	w := ember.New().AddEntities(win)
	in := win.Input()
	w.RunFrames(3)

	if body, _ := inner.BoxOf(inner.Body()); body.W != 200 {
		t.Fatalf("the inner frame's body is %v wide", body.W)
	}
	in.SetMouse(geom.V2(10+10+5, 20+10+10+5))
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(2)
	if clicks != 1 {
		t.Fatalf("the inner button was clicked %d times", clicks)
	}
	tab := func() {
		in.SetKey(input.KeyTab, true)
		w.RunFrames(1)
		in.SetKey(input.KeyTab, false)
		w.RunFrames(2)
	}
	outer.Focus(start)
	w.RunFrames(2)
	tab()
	if !inner.Focused(deep) || !middle.Focused(nested) || !outer.Focused(frame) {
		t.Fatal("Tab does not walk into the inner frame")
	}
	tab()
	if !middle.Focused(next) || inner.Focused(deep) || !outer.Focused(frame) {
		t.Fatal("Tab does not walk out of the inner frame into the one around it")
	}
	tab()
	if !outer.Focused(start) || middle.Focused(next) {
		t.Fatal("Tab does not walk out of the outer frame")
	}
}

func TestAnIframeTakenOutLetsGoOfItsDocument(t *testing.T) {
	p := newFramePage(t)
	field, _ := p.child.BoxOf(p.field)
	p.click(p.inFrame(field.X+5, field.Y+5))
	painted := ui.Painted(p.child)
	p.host.Body().Remove(p.slot)
	p.frames(3)
	if p.host.FrameShown(p.slot) || ui.Framed(p.child) {
		t.Fatal("the Document of an iframe taken out is still tied to it")
	}
	if p.child.Focused(p.field) {
		t.Fatal("it keeps its focus")
	}
	p.in.AddText('x')
	p.frames(3)
	if p.child.Value(p.field) != "" || ui.Painted(p.child) != painted {
		t.Fatalf("it still types (%q) or paints (%d more)", p.child.Value(p.field), ui.Painted(p.child)-painted)
	}
}

func TestSetFrameSwapsTheDocumentAnIframeShows(t *testing.T) {
	p := newFramePage(t)
	other := ui.NewDocument(ui.CSS(`body { margin: 0 }`))
	other.Body().Append(ui.P(ui.Text("another")))
	p.host.SetFrame(p.slot, other)
	p.frames(3)
	if p.host.Frame(p.slot) != other || !p.host.FrameShown(p.slot) || ui.Framed(p.child) || ui.Painted(other) == 0 {
		t.Fatal("SetFrame does not swap the Document shown")
	}
	p.host.SetFrame(p.slot, nil)
	p.frames(2)
	if ui.Framed(other) || p.host.FrameShown(p.slot) {
		t.Fatal("SetFrame(nil) leaves the Document tied")
	}
}

func TestAnIframeOutOfViewIsNotDrawn(t *testing.T) {
	p := newFramePage(t)
	p.slot.SetAttr("style", "margin-top: 1500px")
	p.frames(3)
	painted := ui.Painted(p.child)
	p.child.Body().Append(ui.P(ui.Text("more")))
	p.frames(3)
	if p.host.FrameShown(p.slot) || ui.Painted(p.child) != painted {
		t.Fatalf("out of view, the frame is drawn: shown %v, %d paints", p.host.FrameShown(p.slot),
			ui.Painted(p.child)-painted)
	}
	p.slot.SetAttr("style", "display: none")
	p.frames(3)
	if p.host.FrameShown(p.slot) || ui.Painted(p.child) != painted {
		t.Fatal("display: none, the frame is drawn")
	}
	p.slot.SetStyle("")
	p.frames(3)
	if !p.host.FrameShown(p.slot) || ui.Painted(p.child) == painted {
		t.Fatal("back in view, the frame is not drawn")
	}
}

func TestAQuietIframeReplays(t *testing.T) {
	p := newFramePage(t)
	p.in.SetMouse(geom.V2(700, 400))
	p.frames(4)
	host, child := ui.Painted(p.host), ui.Painted(p.child)
	p.frames(5)
	if ui.Painted(p.child) != child {
		t.Fatalf("a frame nothing changed in painted %d times", ui.Painted(p.child)-child)
	}
	if ui.Painted(p.host) == host {
		t.Fatal("a page with an element over its frame is not painted each frame")
	}
	p.host.Body().Remove(p.cover)
	p.frames(4)
	host = ui.Painted(p.host)
	p.frames(5)
	if ui.Painted(p.host) != host || ui.Painted(p.child) != child {
		t.Fatalf("with nothing over the frame, the page painted %d times and the frame %d", ui.Painted(p.host)-host,
			ui.Painted(p.child)-child)
	}
	p.first.SetText("changed")
	p.frames(2)
	if ui.Painted(p.host) != host || ui.Painted(p.child) == child {
		t.Fatalf("a change in the frame painted the page %d times and the frame %d", ui.Painted(p.host)-host,
			ui.Painted(p.child)-child)
	}
}

func TestAPageWithAnIframeIsAlignedAndWarnsOfNothing(t *testing.T) {
	p := newFramePage(t)
	for name, d := range map[string]*ui.Document{"page": p.host, "frame": p.child} {
		if faults := align.Check(d); len(faults) > 0 {
			t.Errorf("%s: %v", name, faults)
		}
		if warns := d.Warnings(); len(warns) > 0 {
			t.Errorf("%s warns: %v", name, warns)
		}
	}
}

func TestAnIframesBoxIsReplaced(t *testing.T) {
	doc := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { margin: 0 } .w { width: 400px } .h { height: 50% }
		.r { aspect-ratio: 2 / 1; width: 100px }
		#half { height: 200px } .none { display: none }`))
	plain := ui.Iframe(nil)
	attrs := ui.Iframe(nil, ui.Attr("width", "120"), ui.Attr("height", "40"))
	percent := ui.Iframe(nil, ui.Attr("width", "50%"))
	bare := ui.Iframe(nil, ui.Attr("frameborder", "0"))
	wide := ui.Iframe(nil, ui.Class("w"))
	half := ui.Iframe(nil, ui.Class("h"), ui.Style("display: block"))
	ratio := ui.Iframe(nil, ui.Class("r"))
	none := ui.Iframe(nil, ui.Class("none"))
	doc.Body().Append(ui.Div(plain), ui.Div(attrs), ui.Div(percent), ui.Div(bare), ui.Div(wide),
		ui.Div(ui.ID("half"), half), ui.Div(ratio), ui.Div(none))
	win := platform.NewWindow(platform.Size(800, 600))
	win.Add(doc)
	ember.New().AddEntities(win).RunFrames(2)
	for _, c := range []struct {
		name string
		n    *ui.Node
		w, h float32
	}{
		{"plain", plain, 300, 150}, {"width and height attributes", attrs, 120, 40},
		{"a percentage width attribute", percent, 400, 150}, {"frameborder 0", bare, 300, 150},
		{"a CSS width keeps the height", wide, 400, 150}, {"a percentage height", half, 300, 100},
		{"aspect-ratio", ratio, 100, 50},
	} {
		r, ok := doc.BoxOf(c.n)
		if !ok || r.W != c.w || r.H != c.h {
			t.Errorf("%s: content box %v by %v, want %v by %v", c.name, r.W, r.H, c.w, c.h)
		}
	}
	if r := plain.Box(); r.W != 304 || r.H != 154 {
		t.Errorf("the border box is %v by %v, want 304 by 154", r.W, r.H)
	}
	if r := bare.Box(); r.W != 300 || r.H != 150 {
		t.Errorf("frameborder 0 leaves a border: %v by %v", r.W, r.H)
	}
	if _, ok := doc.BoxOf(none); ok {
		t.Error("display: none has a box")
	}
}

func TestADocumentAnIframeLetGoOfIsCollected(t *testing.T) {
	p := newFramePage(t)
	held := func() weak.Pointer[ui.Document] {
		child := ui.NewDocument(ui.CSS(`body { margin: 0 }`))
		child.Body().Append(ui.Button(ui.Text("gone")))
		p.host.SetFrame(p.slot, child)
		p.frames(3)
		if !p.host.FrameShown(p.slot) {
			t.Fatal("the Document is not shown")
		}
		return weak.Make(child)
	}()
	p.child, p.first, p.wide, p.field, p.last, p.childText = nil, nil, nil, nil, nil, nil
	p.host.SetFrame(p.slot, nil)
	p.frames(3)
	for range 5 {
		runtime.GC()
	}
	if held.Value() != nil {
		t.Fatal("a Document its iframe shows no more is still held")
	}
}

// collected reports whether what held points at went, frames run between
// the collections.
func (p *framePage) collected(held ...weak.Pointer[ui.Document]) bool {
	for range 10 {
		runtime.GC()
		left := 0
		for _, h := range held {
			if h.Value() != nil {
				left++
			}
		}
		if left == 0 {
			return true
		}
		p.frames(1)
	}
	return false
}

func TestAnIframeTakenOutOfItsDocumentLetsGoOfTheOneItShowed(t *testing.T) {
	for _, c := range []struct {
		name   string
		remove func(p *framePage, around *ui.Node)
	}{
		{"the element", func(p *framePage, _ *ui.Node) { p.slot.Parent().Remove(p.slot) }},
		{"an element around it", func(p *framePage, around *ui.Node) { p.host.Body().Remove(around) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newFramePage(t)
			around := ui.Div(ui.ID("around"))
			p.host.Body().Remove(p.host.ByID("long"))
			p.host.Body().Remove(p.slot)
			around.Append(p.slot)
			p.host.Body().Append(around)
			p.frames(2)
			if !p.host.FrameShown(p.slot) {
				t.Fatal("the Document is not shown")
			}
			p.click(p.inFrame(10, 70))
			child, field := p.child, p.field
			if !child.Focused(field) {
				t.Fatal("the field of the frame did not take the focus")
			}
			held := weak.Make(p.child)
			c.remove(p, around)
			p.frames(2)
			if p.host.Frame(p.slot) != nil {
				t.Error("an iframe out of its Document still holds the one it showed")
			}
			if child.Focused(field) {
				t.Error("the Document let go of kept its focus")
			}
			p.child, p.first, p.wide, p.field, p.last, p.childText = nil, nil, nil, nil, nil, nil
			if !p.collected(held) {
				t.Fatal("the Document of an iframe taken out of its Document is still held")
			}
		})
	}
}

func TestAnIframePutBackKeepsItsDocumentOnlyWithinTheFrame(t *testing.T) {
	p := newFramePage(t)
	p.host.Body().Remove(p.host.ByID("long"))
	p.host.Body().Remove(p.slot)
	p.host.Body().Append(p.slot)
	p.frames(2)
	if p.host.Frame(p.slot) != p.child || !p.host.FrameShown(p.slot) {
		t.Fatal("an iframe moved within a frame lost its Document")
	}
	p.host.Body().Remove(p.slot)
	p.frames(1)
	p.host.Body().Append(p.slot)
	p.frames(2)
	if p.host.Frame(p.slot) != nil {
		t.Fatal("an iframe put back a frame later still shows the Document it had")
	}
	p.host.SetFrame(p.slot, p.child)
	p.frames(2)
	if !p.host.FrameShown(p.slot) {
		t.Fatal("an iframe put back does not show the Document it is given")
	}
}

func TestAnIframeMovedToAnotherDocumentShowsNothing(t *testing.T) {
	for _, first := range []string{"the one it leaves", "the one it joins"} {
		t.Run(first+" drawn first", func(t *testing.T) {
			p := newFramePage(t)
			other := ui.NewDocument()
			if first == "the one it joins" {
				p.w.RemoveEntity(p.host)
				p.win.Add(other, p.host)
			} else {
				p.win.Add(other)
			}
			p.frames(2)
			p.host.Body().Remove(p.slot)
			other.Body().Append(p.slot)
			p.frames(3)
			if other.Frame(p.slot) != nil || other.FrameShown(p.slot) {
				t.Fatal("an iframe moved to another Document still shows the one it had")
			}
			other.SetFrame(p.slot, p.child)
			p.frames(2)
			if !other.FrameShown(p.slot) {
				t.Fatal("an iframe moved to another Document does not show the one it is given")
			}
		})
	}
}

func TestADocumentThatGoesTakesWhatItsIframesShowWithIt(t *testing.T) {
	p := newFramePage(t)
	host, child := weak.Make(p.host), weak.Make(p.child)
	p.w.RemoveEntity(p.host)
	p.frames(2)
	p.host, p.slot, p.cover, p.before, p.after, p.hostField, p.hostText = nil, nil, nil, nil, nil, nil, nil
	p.child, p.first, p.wide, p.field, p.last, p.childText = nil, nil, nil, nil, nil, nil
	if !p.collected(host, child) {
		t.Fatalf("held after the Document left its window: the Document %v, the one its iframe showed %v",
			host.Value() != nil, child.Value() != nil)
	}
}

func TestAGuardedIframeOutlivesAPanicOfItsDocument(t *testing.T) {
	for _, c := range []struct {
		name  string
		build func(p *framePage, armed *bool) *ui.Document
		arm   func(p *framePage)
	}{
		{"as it draws", func(_ *framePage, armed *bool) *ui.Document {
			doc := ui.NewDocument(ui.CSS(`body { margin: 0 } canvas { display: block; width: 50px; height: 50px }`))
			canvas := ui.Canvas(func(c *twod.Canvas) {
				c.PushClip(geom.Rect{W: 10, H: 10})
				c.Push("inside")
				if *armed {
					panic("drawn wrong")
				}
				c.Pop()
				c.PopClip()
			})
			doc.Body().Append(canvas, ui.Button(ui.ID("lower"), ui.Text("lower")))
			return doc
		}, func(p *framePage) { p.host.Frame(p.slot).Invalidate(p.host.Frame(p.slot).Body().Children()[0]) }},
		{"as it is clicked", func(_ *framePage, armed *bool) *ui.Document {
			doc := ui.NewDocument(ui.CSS(`body { margin: 0 } button { display: block; width: 100px; height: 30px }`))
			doc.Body().Append(ui.Button(ui.ID("trap"), ui.Text("trap"), ui.OnClick(func() {
				if *armed {
					panic("clicked wrong")
				}
			})))
			return doc
		}, func(p *framePage) { p.click(p.inFrame(10, 10)) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newFramePage(t)
			armed := false
			ui.Guarded().Apply(p.slot)
			p.host.Body().Remove(p.cover)
			p.host.SetFrame(p.slot, c.build(p, &armed))
			p.frames(3)
			if !p.host.FrameShown(p.slot) || p.host.FrameBroken(p.slot) != nil {
				t.Fatalf("the guarded frame is not shown, or broken before anything: %v", p.host.FrameBroken(p.slot))
			}
			armed = true
			c.arm(p)
			p.frames(3)
			broken := p.host.FrameBroken(p.slot)
			var raised *ui.FrameError
			if !errors.As(broken, &raised) || !strings.Contains(fmt.Sprint(raised.Value), "wrong") ||
				!strings.Contains(string(raised.Stack), "frame_test.go") {
				t.Fatalf("the Document around a frame that panicked is told %v", broken)
			}
			if p.host.Frame(p.slot) != nil || p.host.FrameShown(p.slot) {
				t.Error("the iframe still shows the Document that panicked")
			}
			if got := strings.Join(errlog.Lines(), "\n"); !strings.Contains(got, "wrong") ||
				!strings.Contains(got, "frame_test.go") {
				t.Errorf("the error log does not hold the panic and where it came from: %s", got)
			}
			after, _ := p.host.BoxOf(p.after)
			p.click(after.Center())
			if p.clicks["after"] != 1 {
				t.Errorf("after the panic a button of the Document around heard %d clicks", p.clicks["after"])
			}
			if errs := align.Check(p.host); len(errs) > 0 {
				t.Errorf("the Document around is out of line: %v", errs)
			}
			next := ui.NewDocument(ui.CSS(`body { margin: 0 }`))
			next.Body().Append(ui.Button(ui.ID("next"), ui.Text("next")))
			p.host.SetFrame(p.slot, next)
			p.frames(2)
			if p.host.FrameBroken(p.slot) != nil || !p.host.FrameShown(p.slot) {
				t.Error("an iframe given another Document after a panic does not show it")
			}
		})
	}
}

func TestAPanicOfAFramedDocumentGoesOnWithoutAGuard(t *testing.T) {
	p := newFramePage(t)
	child := ui.NewDocument()
	child.Body().Append(ui.Canvas(func(*twod.Canvas) { panic("unguarded") }))
	p.host.SetFrame(p.slot, child)
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "unguarded") {
			t.Fatalf("the panic of a Document in an iframe with no guard came out as %v", r)
		}
	}()
	p.frames(3)
	t.Fatal("the panic was swallowed")
}
