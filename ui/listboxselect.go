package ui

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// A select with multiple or a size above 1 is a listbox (decision 0202):
// its options are rows of the page, which the pointer and the keys select
// as control.Select's listbox mode says, Command being Control off macOS.
// A press's change comes once it lets go, a key's at once; the option
// the keys reach scrolls into view.

// listBox is n's machine when n is a select shown as a listbox.
func (d *Document) listBox(n *Node) *control.Select {
	if s, ok := d.machine(n).(*control.Select); ok && s.ListBox {
		return s
	}
	return nil
}

// optionUnder is the option of the listbox lb in chain, the hovered
// elements innermost first: its index among lb's options, -1 for a
// group's label or none.
func optionUnder(lb *Node, chain []*Node) int {
	for _, n := range chain {
		if n.Tag() != dom.Option {
			continue
		}
		for i, o := range dom.Options(lb) {
			if o == n {
				return i
			}
		}
	}
	return -1
}

// commandHeld reports the platform's shortcut modifier held: Command on
// macOS, Control elsewhere.
func (d *Document) commandHeld(f inputfeed.Feed) bool {
	if d.platform == textedit.Mac {
		return f.RawDown(int(input.KeySuper))
	}
	return f.RawDown(int(input.KeyControl))
}

// pressListBox begins a press on the listbox lb, which took the focus.
func (d *Document) pressListBox(in *input.State, lb *Node) {
	s := d.listBox(lb)
	if s == nil {
		return
	}
	f := inputfeed.Of(in)
	s.Press(optionUnder(lb, d.hovered), f.RawDown(int(input.KeyShift)), d.commandHeld(f))
	d.listPressed = lb
	d.sync(lb, s)
}

// dragListBox extends the held press to the option under the pointer.
func (d *Document) dragListBox() {
	if s := d.listBox(d.listPressed); s != nil {
		if i := optionUnder(d.listPressed, d.hovered); i >= 0 {
			s.DragTo(i)
			d.sync(d.listPressed, s)
		}
	}
}

// releaseListBox ends the press, firing the change it made.
func (d *Document) releaseListBox() {
	lb := d.listPressed
	d.listPressed = nil
	if s := d.listBox(lb); s != nil {
		d.fire(lb, s.Release())
	}
}

// listBoxKey moves the focused listbox n for i, a key's or a pad button's
// (k KeyUnknown), with its modifiers, reporting whether it took it.
func (d *Document) listBoxKey(f inputfeed.Feed, n *Node, s *control.Select, i control.Intent) bool {
	if !s.Handles(i) {
		return false
	}
	d.fire(n, s.Key(i, f.RawDown(int(input.KeyShift)), d.commandHeld(f)))
	d.revealOption(n, s.Active)
	return true
}

// listBoxType goes to the option the text typed starts.
func (d *Document) listBoxType(n *Node, s *control.Select, text []rune) {
	for _, r := range text {
		if r != ' ' {
			d.fire(n, s.TypeAhead(string(r)))
			d.revealOption(n, s.Active)
		}
	}
}

// revealOption scrolls the listbox n to its option i.
func (d *Document) revealOption(n *Node, i int) {
	if opts := dom.Options(n); i >= 0 && i < len(opts) {
		d.sync(n, d.machine(n))
		d.ScrollIntoView(opts[i])
	}
}
