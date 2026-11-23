package control

import (
	"strings"
	"unicode"
)

// Option is one choice of a select.
type Option struct {
	Label, Value string
	// Valued says Value is the option's even empty, as a value attribute
	// written empty.
	Valued   bool
	Disabled bool
	// Shown is the label as the player reads it, translated; Label when empty.
	Shown string
	// Group is 1 + the index in the select's Groups of the group the
	// option is in, 0 for none.
	Group int
}

// Group is an optgroup: a label the list shows, never chosen, before
// its options.
type Group struct {
	Label    string
	Disabled bool
	// Shown is the label as the player reads it, translated; Label when empty.
	Shown string
	// At is the option the label stands before: the group's first, or
	// the next one for an empty group.
	At int
}

// Text is what the list shows of the group.
func (g Group) Text() string {
	if g.Shown != "" {
		return g.Shown
	}
	return g.Label
}

// Row is a line of the open list: option Option, or a group's label
// when Option is -1. Group is 1 + the index of the group it belongs to,
// 0 for none.
type Row struct {
	Option, Group int
}

// Text is what the list shows of the option.
func (o Option) Text() string {
	if o.Shown != "" {
		return o.Shown
	}
	return o.Label
}

// value is what a form reads of the option: its Value, else its label.
func (o Option) value() string {
	if o.Value != "" || o.Valued {
		return o.Value
	}
	return o.Label
}

// Select is a select element, shown as a pop-up list: Activate or a
// click opens it on the selected option, the arrows walk its enabled
// options, Activate or a click chooses one, and Back or a click outside
// closes it as it was. Typing jumps to the next option starting with
// that letter, open or closed.
type Select struct {
	Options []Option
	// Groups are its optgroups, in order.
	Groups   []Group
	Disabled bool
	// Selected is the chosen option, -1 for none; Highlight the one the
	// open list is on.
	Selected, Highlight int
	Open                bool
}

func NewSelect(options ...Option) *Select {
	s := &Select{Options: options, Selected: -1}
	s.Selected = s.next(-1, 1)
	s.Highlight = s.Selected
	return s
}

// Rows are the open list's lines in order, each group's label before
// its options.
func (s *Select) Rows() []Row {
	rows := make([]Row, 0, len(s.Options)+len(s.Groups))
	g := 0
	label := func(before int) {
		for ; g < len(s.Groups) && s.Groups[g].At <= before; g++ {
			rows = append(rows, Row{Option: -1, Group: g + 1})
		}
	}
	for i, o := range s.Options {
		label(i)
		rows = append(rows, Row{Option: i, Group: o.Group})
	}
	label(len(s.Options))
	return rows
}

// Value is the selected option's value, "" for none.
func (s *Select) Value() string {
	if s.Selected < 0 || s.Selected >= len(s.Options) {
		return ""
	}
	return s.Options[s.Selected].value()
}

// Handles reports whether i is the select's: Activate always, and the
// arrows and Back only while it is open.
func (s *Select) Handles(i Intent) bool {
	if s.Disabled {
		return false
	}
	switch i {
	case Activate:
		return true
	case Up, Down, Home, End, PageUp, PageDown, Back:
		return s.Open
	case None, Left, Right:
	}
	return false
}

func (s *Select) Do(i Intent) Events {
	if !s.Handles(i) {
		return 0
	}
	if !s.Open {
		s.open()
		return 0
	}
	switch i {
	case Activate:
		return s.choose(s.Highlight)
	case Back:
		s.Open = false
	case Down:
		s.Highlight = s.next(s.Highlight, 1)
	case Up:
		s.Highlight = s.next(s.Highlight, -1)
	case Home, PageUp:
		s.Highlight = s.next(-1, 1)
	case End, PageDown:
		s.Highlight = s.next(len(s.Options), -1)
	case None, Left, Right: // Handles lets only these through
	}
	return 0
}

// Click opens or closes the list from the select itself.
func (s *Select) Click() Events {
	switch {
	case s.Disabled:
	case s.Open:
		s.Open = false
	default:
		s.open()
	}
	return 0
}

// ClickOption chooses option i of the open list, unless it is disabled.
func (s *Select) ClickOption(i int) Events {
	if !s.Open || !s.enabled(i) {
		return 0
	}
	return s.choose(i)
}

// ClickOutside closes the list as it was.
func (s *Select) ClickOutside() { s.Open = false }

// TypeAhead goes to the next enabled option after the current one whose
// label starts with text, any case, wrapping: the highlight while open,
// the choice while closed.
func (s *Select) TypeAhead(text string) Events {
	if s.Disabled || text == "" {
		return 0
	}
	from := s.Selected
	if s.Open {
		from = s.Highlight
	}
	n := len(s.Options)
	for k := 1; k <= n; k++ {
		i := (from + k + n) % n
		if s.enabled(i) && strings.HasPrefix(strings.ToLower(s.Options[i].Text()), strings.Map(unicode.ToLower, text)) {
			if s.Open {
				s.Highlight = i
				return 0
			}
			return s.pick(i)
		}
	}
	return 0
}

func (s *Select) open() {
	s.Open, s.Highlight = true, s.Selected
	if !s.enabled(s.Highlight) {
		s.Highlight = s.next(-1, 1)
	}
}

func (s *Select) choose(i int) Events {
	s.Open = false
	return s.pick(i)
}

func (s *Select) pick(i int) Events {
	if !s.enabled(i) || i == s.Selected {
		return 0
	}
	s.Selected, s.Highlight = i, i
	return Input | Changed
}

func (s *Select) enabled(i int) bool { return i >= 0 && i < len(s.Options) && !s.Options[i].Disabled }

// next is the first enabled option from i going by dir, or i if none.
func (s *Select) next(i, dir int) int {
	for j := i + dir; j >= 0 && j < len(s.Options); j += dir {
		if s.enabled(j) {
			return j
		}
	}
	if s.enabled(i) {
		return i
	}
	return -1
}
