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

// FormValue is what a form reads of the option: its Value, else its label.
func (o Option) FormValue() string {
	if o.Value != "" || o.Valued {
		return o.Value
	}
	return o.Label
}

// Select is a select element, shown as a pop-up list: Activate or a
// click opens it on the selected option, the arrows walk its enabled
// options, Activate or a click chooses one, and Back or a click outside
// closes it as it was. Typing finds an option by what was typed, open or
// closed (Type).
type Select struct {
	Options []Option
	// Groups are its optgroups, in order.
	Groups   []Group
	Disabled bool
	// Selected is the chosen option, -1 for none, a listbox's first
	// selected; Highlight the one the open list is on.
	Selected, Highlight int
	Open                bool
	// ListBox shows the options as rows of the page, Size of them in
	// view; Multiple lets it select several (listbox.go).
	ListBox, Multiple bool
	Size              int
	// Active is a listbox's option the keys move from, Anchor the one a
	// run starts at; -1 for none.
	Active, Anchor int
	// picked marks a listbox's selected options; was is them as a press
	// began, base them as a Command press left them, and held a press
	// on an option not let go yet.
	picked, was, base []bool
	held              bool
	// last is a listbox's selection as the player's last press or key
	// left it, nil before any.
	last []bool
	// typed is what type-ahead has, lowered, its last letter at typedAt;
	// repeat the letter it cycles on, 0 for none.
	typed   []rune
	typedAt float32
	repeat  rune
}

func NewSelect(options ...Option) *Select {
	s := &Select{Options: options, Selected: -1, Active: -1, Anchor: -1}
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
	return s.Options[s.Selected].FormValue()
}

// Handles reports whether i is the select's: Activate always, and the
// arrows and Back only while it is open; a listbox's arrows always, never
// Back.
func (s *Select) Handles(i Intent) bool {
	if s.Disabled {
		return false
	}
	if s.ListBox {
		return i != Back && i != Left && i != Right && i != None
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
	if s.ListBox {
		return s.Key(i, false, false)
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

// Click opens or closes the list from the select itself; a listbox
// never opens.
func (s *Select) Click() Events {
	switch {
	case s.Disabled, s.ListBox:
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

// typeAheadTime is how long, in seconds, a letter typed after another
// goes on with it, as Chrome's kTypeAheadTimeout.
const typeAheadTime = 1

// Type finds an option by r typed at now, in seconds, as Chrome's
// type-ahead: the letters typed within a second of each other are a
// prefix the selected option or the next enabled one after it starts
// with, any case and leading spaces passed over, while one letter typed
// again goes on to the next option starting with it, wrapping. What it
// finds is the highlight while open, the choice while closed, a
// listbox's selection, which it holds alone (typedChange).
func (s *Select) Type(r rune, now float32) Events {
	if s.Disabled {
		return 0
	}
	r = unicode.ToLower(r)
	if now-s.typedAt > typeAheadTime {
		s.typed = s.typed[:0]
	}
	s.typed, s.typedAt = append(s.typed, r), now
	prefix, start := s.typed, 1
	switch {
	case r == s.repeat:
		prefix = s.typed[len(s.typed)-1:]
	case len(s.typed) > 1:
		s.repeat, start = 0, 0
	default:
		s.repeat = r
	}
	from := s.Selected
	if s.Open {
		from = s.Highlight
	}
	want, n := string(prefix), len(s.Options)
	for k := range n {
		i := (max(from, 0) + start + k) % n
		label := strings.ToLower(strings.TrimLeftFunc(s.Options[i].Text(), unicode.IsSpace))
		if s.enabled(i) && strings.HasPrefix(label, want) {
			switch {
			case s.Open:
				s.Highlight = i
				return 0
			case s.ListBox:
				s.Active, s.Anchor = i, i
				s.only(i)
				s.settle()
				return s.typedChange()
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
