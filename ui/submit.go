package ui

import (
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// RequestSubmit submits the form n, as the DOM's requestSubmit and as its
// submit button does: every control of the form matches :user-valid or
// :user-invalid from then on; unless the form has novalidate, an invalid
// control stops the submission and takes the focus, the first in tree
// order; else the form emits Submitted (OnSubmit on the form). ui sends
// nothing anywhere: what a submission does is the program's.
func (d *Document) RequestSubmit(n *Node) {
	if n != nil && n.Tag() == dom.Form {
		d.submit(n, nil)
	}
}

// submit runs the submission of form by submitter, nil for none.
func (d *Document) submit(form, submitter *Node) {
	var first *Node
	d.html.Walk(func(c *Node) bool {
		if m := d.machine(c); m != nil && formOwner(c) == form {
			if c.Tag() != dom.Button && !submits(c) {
				d.used(c, changed)
			}
			if first == nil && d.fieldStates(c, m, d.groupNow)&dom.Invalid != 0 {
				first = c
			}
		}
		return !c.IsText()
	})
	if !has(form, "novalidate") && (submitter == nil || !has(submitter, "formnovalidate")) && first != nil {
		d.moveFocus(func() bool { return d.focus.Set(first, true) })
		return
	}
	d.events = append(d.events, Submitted{form})
}

// submitClick is a click on n: a submit button submits its form.
func (d *Document) submitClick(n *Node) {
	if submits(n) && !disabledField(n) {
		if f := formOwner(n); f != nil {
			d.submit(f, n)
		}
	}
}

// implicitSubmit is Enter in the one-line field n, as HTML's implicit
// submission: its form's first submit button is clicked, unless it is
// disabled; a form without one is submitted when n is its only field
// that blocks implicit submission.
func (d *Document) implicitSubmit(n *Node) {
	form := formOwner(n)
	if form == nil {
		return
	}
	var button *Node
	fields := 0
	d.html.Walk(func(c *Node) bool {
		if formOwner(c) == form && d.machine(c) != nil {
			if button == nil && submits(c) {
				button = c
			}
			if c.Tag() == dom.Input && blocksImplicit[inputType(c)] {
				fields++
			}
		}
		return !c.IsText()
	})
	switch {
	case button != nil:
		if !disabledField(button) {
			d.fire(button, control.Clicked)
			d.submitClick(button)
		}
	case fields == 1:
		d.submit(form, nil)
	}
}

// blocksImplicit are the input types of which a form with no submit
// button must have one alone for Enter to submit it.
//
//nolint:gochecknoglobals // lookup table
var blocksImplicit = map[string]bool{
	typeText: true, typeSearch: true, typeURL: true, "tel": true, typeEmail: true, typePassword: true, "date": true,
	"month": true, "week": true, "time": true, "datetime-local": true, typeNumber: true,
}
