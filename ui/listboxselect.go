package ui

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/paint"
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

// listHold is how the pointer went while a press on a listbox is held:
// where it was last, whether it moved over the listbox, which starts its
// autoscroll, and the clock when it last scrolled it.
type listHold struct {
	at         geom.Vec2
	moved      bool
	scrolledAt float32
}

// pressListBox begins a press at at on the listbox lb, which took the
// focus.
func (d *Document) pressListBox(in *input.State, lb *Node, at geom.Vec2) {
	s := d.listBox(lb)
	if s == nil {
		return
	}
	f := inputfeed.Of(in)
	s.Press(optionUnder(lb, d.hovered), f.RawDown(int(input.KeyShift)), d.commandHeld(f))
	d.listPressed, d.listHeld = lb, listHold{at: at, scrolledAt: d.clock}
	d.sync(lb, s)
}

// dragListBox scrolls the held listbox while the pointer at is near or
// past its top or bottom edge, and as the pointer moves extends the
// press to the option under it: as Chrome's, an option the scroll brings
// under a still pointer waits for it to move.
func (d *Document) dragListBox(at geom.Vec2) {
	lb, h := d.listPressed, &d.listHeld
	moved := at != h.at
	h.at = at
	if moved && contains(d.hovered, lb) {
		h.moved = true
	}
	d.autoscrollListBox(lb, at)
	s := d.listBox(lb)
	if s == nil || !moved {
		return
	}
	if i := optionUnder(lb, d.hovered); i >= 0 {
		s.DragTo(i)
		d.sync(lb, s)
	}
}

// listBelt is how near its top or bottom edge, in points, a held press's
// pointer scrolls a listbox, as Chrome's autoscroll: each frame, at 60 a
// second, by a point more than the pointer's rounded place lies into the
// belt at the top, two at the bottom, past the edge counted.
const listBelt = 20

// autoscrollListBox scrolls lb for the pointer at, once it moved over lb
// with the press held.
func (d *Document) autoscrollListBox(lb *Node, at geom.Vec2) {
	h := &d.listHeld
	dt := min(max(d.clock-h.scrolledAt, 0), 0.05)
	h.scrolledAt = d.clock
	if !h.moved || d.tree == nil {
		return
	}
	b := d.tree.Of[lb]
	if b == nil {
		return
	}
	r, y := b.Rect(), float32(math.Round(float64(at.Y)))
	var by float32
	switch {
	case y < r.Y+listBelt:
		by = y - (r.Y + listBelt + 1)
	case y > r.Y+r.H-listBelt:
		by = y - (r.Y + r.H - listBelt - 2)
	default:
		return
	}
	bars := paint.BarsOf(b, origin(b))
	off := bars.Scroll
	off.Y += by * dt * 60
	d.scrollerOf(lb).glide = nil
	d.setScroll(b, bars.Clamp(off))
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

// selectType finds the option of the select n the text typed starts, a
// listbox's scrolled into view.
func (d *Document) selectType(n *Node, s *control.Select, text []rune) {
	for _, r := range text {
		if r == ' ' {
			continue
		}
		d.fire(n, s.Type(r, d.clock))
		if s.ListBox {
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
