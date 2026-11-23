package ui_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

// heardKeys records the elements a Document's OnKey and OnBack on its
// body heard a key and a Back on.
type heardKeys struct{ keys, backs []*ui.Node }

func hearKeys(d *ui.Document) *heardKeys {
	h := &heardKeys{}
	ui.OnKey(func(e ui.Event) { h.keys = append(h.keys, e.Node) }).Apply(d.Body())
	ui.OnBack(func(e ui.Event) { h.backs = append(h.backs, e.Node) }).Apply(d.Body())
	return h
}

// tabIntoFrame focuses the frame's first button by Tab, as a keyboard
// does.
func (p *framePage) tabIntoFrame(t *testing.T) {
	t.Helper()
	p.host.Focus(p.before)
	p.frames(2)
	p.key(input.KeyTab)
	p.key(input.KeyTab)
	if !p.child.Focused(p.first) {
		t.Fatal("two Tabs from the page's first button don't reach the frame's")
	}
}

func TestAKeyInAnIframeGoesOnToThePageRoundIt(t *testing.T) {
	p := newFramePage(t)
	p.tabIntoFrame(t)
	host, child := hearKeys(p.host), hearKeys(p.child)

	p.key(input.KeyK)
	if len(child.keys) != 1 || child.keys[0] != p.first {
		t.Fatalf("the frame heard the key on %v, want its focused button", child.keys)
	}
	if len(host.keys) != 1 || host.keys[0] != p.slot {
		t.Fatalf("the page heard the key on %v, want the iframe", host.keys)
	}

	ui.OnKey(func(e ui.Event) { e.StopPropagation() }).Apply(p.first)
	p.key(input.KeyK)
	if len(child.keys) != 1 || len(host.keys) != 1 {
		t.Fatalf("a key stopped in the frame reached %d of the frame's and %d of the page's handlers, want none",
			len(child.keys)-1, len(host.keys)-1)
	}
}

func TestBackInAnIframeGoesOnToThePageRoundIt(t *testing.T) {
	p := newFramePage(t)
	p.tabIntoFrame(t)
	host, child := hearKeys(p.host), hearKeys(p.child)

	p.key(input.KeyEscape)
	if len(child.backs) != 1 || child.backs[0] != p.first {
		t.Fatalf("the frame heard Back on %v, want its focused button", child.backs)
	}
	if len(host.backs) != 1 || host.backs[0] != p.slot {
		t.Fatalf("the page heard Back on %v, want the iframe", host.backs)
	}

	words, _ := p.child.BoxOf(p.childText)
	p.click(p.inFrame(words.X+5, words.Y+5))
	if !p.host.Focused(p.slot) || p.child.Focused(p.first) {
		t.Fatal("a click on the frame's words leaves the focus on its button")
	}
	p.key(input.KeyEscape)
	if len(host.backs) != 2 || host.backs[1] != p.slot {
		t.Fatalf("with nothing focused in the frame the page heard Back on %v, want the iframe", host.backs)
	}
}

func TestTheArrowsWalkOutOfAnIframeAndIntoIt(t *testing.T) {
	p := newFramePage(t)
	p.tabIntoFrame(t)

	p.key(input.KeyUp)
	if !p.host.Focused(p.hostField) || p.child.Focused(p.first) {
		t.Fatal("Up from the frame's first button doesn't reach the page's field above the frame")
	}

	p.host.Focus(p.after)
	p.frames(2)
	p.key(input.KeyUp)
	if !p.host.Focused(p.slot) || !p.child.Focused(p.last) {
		t.Fatal("Up from the page's button under the frame doesn't reach the frame's last button")
	}
	p.key(input.KeyDown)
	if !p.host.Focused(p.after) || p.child.Focused(p.last) {
		t.Fatal("Down from the frame's last button doesn't reach the page's button under the frame")
	}
}

func TestAModalDialogInAnIframeHoldsOnlyItsDocument(t *testing.T) {
	p := newFramePage(t)
	dialog := ui.Dialog(ui.Button(ui.ID("ok"), ui.Text("ok")))
	p.child.Body().Append(dialog)
	p.frames(1)
	p.child.ShowModal(dialog)
	p.frames(2)

	p.click(centre(t, p.host, p.before))
	if p.clicks["before"] != 1 {
		t.Fatal("a modal dialog in the frame keeps the page round it from its clicks, where Chrome's leaves it")
	}
	p.click(p.inFrame(5, 5))
	if p.clicks["first"] != 0 {
		t.Fatal("a click in the frame under its modal dialog reached the button behind it")
	}
}

// typeKey presses k, typing r, then lets it go.
func (p *framePage) typeKey(k input.Key, r rune) {
	p.in.SetKey(k, true)
	p.in.AddText(r)
	p.frames(1)
	p.in.SetKey(k, false)
	p.frames(1)
}

func TestAKeyInAFieldOfAnIframeReachesTheFieldThenThePage(t *testing.T) {
	p := newFramePage(t)
	host, child := hearKeys(p.host), hearKeys(p.child)
	var order []string
	ui.OnKey(func(e ui.Event) {
		order = append(order, "field")
		if e.Key == input.KeyX {
			e.PreventDefault()
		}
	}).Apply(p.field)
	ui.OnKey(func(ui.Event) { order = append(order, "page") }).Apply(p.host.Body())
	field, _ := p.child.BoxOf(p.field)
	p.click(p.inFrame(field.X+5, field.Y+5))
	if !p.child.Focused(p.field) {
		t.Fatal("a click on the frame's field doesn't focus it")
	}

	p.typeKey(input.KeyX, 'x')
	p.typeKey(input.KeyY, 'y')
	if got := p.child.Value(p.field); got != "y" {
		t.Fatalf("the frame's field holds %q, want the y its OnKey let through", got)
	}
	if want := []string{"field", "page", "field", "page"}; fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("the keys were heard by %v, want the field's OnKey, then the page's, for each", order)
	}
	if len(child.keys) != 2 || child.keys[0] != p.field || len(host.keys) != 2 || host.keys[1] != p.slot {
		t.Fatalf("the frame's body heard %v and the page's %v, want the field, then the iframe", child.keys, host.keys)
	}

	ui.OnKey(func(e ui.Event) { e.StopPropagation() }).Apply(p.field)
	p.typeKey(input.KeyZ, 'z')
	if len(host.keys) != 2 || p.child.Value(p.field) != "yz" {
		t.Fatalf("a key the field stopped reached the page (%d more) or wasn't typed (%q)", len(host.keys)-2,
			p.child.Value(p.field))
	}
}

func TestAPointerCapturedInAnIframeFollowsItOverThePage(t *testing.T) {
	p := newFramePage(t)
	var moves int
	var ups []*ui.Node
	ui.OnPointerDown(func(e ui.Event) { p.child.SetPointerCapture(e.Node) }).Apply(p.first)
	ui.OnPointerMove(func(ui.Event) {
		if p.child.PointerCapture() == p.first {
			moves++
		}
	}).Apply(p.first)
	ui.OnPointerUp(func(e ui.Event) { ups = append(ups, e.Node) }).Apply(p.first)

	button, _ := p.child.BoxOf(p.first)
	p.in.SetMouse(p.inFrame(button.X+5, button.Y+5))
	p.frames(1)
	p.in.SetMouseButton(input.MouseLeft, true)
	p.frames(1)
	if p.child.PointerCapture() != p.first {
		t.Fatal("a press in the frame doesn't let its button take the pointer")
	}
	p.in.SetMouse(centre(t, p.host, p.after))
	p.frames(1)
	p.in.SetMouse(centre(t, p.host, p.before))
	p.frames(1)
	p.in.SetMouseButton(input.MouseLeft, false)
	p.frames(2)
	if moves < 2 || len(ups) != 1 || ups[0] != p.first {
		t.Fatalf("held over the page, the frame's button heard %d moves and its release on %v", moves, ups)
	}
	if p.clicks["after"] != 0 || p.clicks["before"] != 0 {
		t.Fatal("a release over the page after a press captured in the frame clicked the page's button")
	}
}
