package ui_test

import (
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
	host, child := hearKeys(p.host), hearKeys(p.child)
	p.tabIntoFrame(t)

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
	host, child := hearKeys(p.host), hearKeys(p.child)
	p.tabIntoFrame(t)

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
