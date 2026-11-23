package control

import "slices"

// A select with multiple or a size above 1 is a listbox, as Chrome's:
// its options stand in rows of the page, Size of them in view, and none
// is chosen until the player or the page picks some. The arrows, Home,
// End and Page Up and Down select the enabled option they reach alone;
// with Shift a multiple one selects the run from the anchor to it, with
// Command (Control off macOS) moves without selecting, and Space turns
// the option reached on or off. A press selects its option alone, with
// Command turns it on or off, with Shift selects the run from the anchor,
// and a drag the run it crosses. A run passes over disabled options.

// SetListBox shows s as a listbox of size rows, multiple or not, or as a
// pop-up list again, which keeps the first option selected, else
// chooses its first enabled one.
func (s *Select) SetListBox(on, multiple bool, size int) {
	switch {
	case on && !s.ListBox:
		s.Open = false
		s.picked = make([]bool, len(s.Options))
		if s.enabled(s.Selected) || s.Selected >= 0 && s.Selected < len(s.Options) {
			s.picked[s.Selected] = true
		}
		s.Active, s.Anchor = s.Selected, s.Selected
	case !on && s.ListBox:
		s.picked = nil
		if s.Selected < 0 {
			s.Selected = s.next(-1, 1)
		}
		s.Highlight = s.Selected
	}
	s.ListBox, s.Multiple, s.Size = on, on && multiple, size
	if on && !s.Multiple && s.Selected >= 0 {
		s.SetPicked(s.Selected)
	}
}

// Picked reports whether option i is selected: in a listbox, any of
// them; else the chosen one.
func (s *Select) Picked(i int) bool {
	if !s.ListBox {
		return i == s.Selected && i >= 0
	}
	return i >= 0 && i < len(s.picked) && s.picked[i]
}

// SetPicked selects options is alone, as the page does, a disabled one
// too: the last of them without multiple.
func (s *Select) SetPicked(is ...int) {
	if !s.ListBox {
		s.Selected = -1
		if len(is) > 0 {
			s.Selected = is[len(is)-1]
		}
		s.Highlight = s.Selected
		return
	}
	s.picked = make([]bool, len(s.Options))
	if !s.Multiple && len(is) > 1 {
		is = is[len(is)-1:]
	}
	for _, i := range is {
		if i >= 0 && i < len(s.picked) {
			s.picked[i] = true
		}
	}
	s.settle()
	s.Active, s.Anchor = s.Selected, s.Selected
}

// settle makes Selected the first option selected, -1 for none.
func (s *Select) settle() {
	if len(s.picked) < len(s.Options) {
		s.picked = append(s.picked, make([]bool, len(s.Options)-len(s.picked))...)
	}
	s.Selected = slices.Index(s.picked[:len(s.Options)], true)
	s.Highlight = s.Selected
}

// Key moves a listbox for i, Activate being Space; shift and command are
// the modifiers held.
func (s *Select) Key(i Intent, shift, command bool) Events {
	if s.Disabled || !s.ListBox {
		return 0
	}
	was := slices.Clone(s.picked)
	switch i {
	case Activate:
		if !s.enabled(s.Active) {
			return 0
		}
		s.Anchor = s.Active
		if s.Multiple {
			s.settle()
			s.picked[s.Active] = !s.picked[s.Active]
		} else {
			s.only(s.Active)
		}
	case Up, Down, Home, End, PageUp, PageDown:
		to := s.reach(i)
		if to < 0 {
			return 0
		}
		s.Active = to
		switch {
		case s.Multiple && command:
			return 0
		case s.Multiple && shift:
			if !s.enabled(s.Anchor) {
				s.Anchor = to
			}
			s.run(nil, s.Anchor, to, true)
		default:
			s.Anchor = to
			s.only(to)
		}
	case None, Back, Left, Right:
		return 0
	}
	s.settle()
	return s.changed(was)
}

// Press is a press on option i of a listbox, -1 for a group's label: its
// events come once the pointer lets go (Release).
func (s *Select) Press(i int, shift, command bool) {
	s.held, s.was = false, slices.Clone(s.picked)
	if s.Disabled || !s.ListBox || !s.enabled(i) {
		return
	}
	s.held, s.base = true, nil
	switch {
	case s.Multiple && command:
		s.settle()
		s.picked[i] = !s.picked[i]
		s.base = slices.Clone(s.picked)
		s.Anchor = i
	case s.Multiple && shift && s.enabled(s.Anchor):
		s.run(nil, s.Anchor, i, true)
	default:
		s.Anchor = i
		s.only(i)
	}
	s.Active = i
	s.settle()
}

// DragTo extends a held press to option i, past which the pointer went.
func (s *Select) DragTo(i int) {
	if !s.held || !s.enabled(i) || i == s.Active {
		return
	}
	s.Active = i
	if !s.Multiple {
		s.Anchor = i
		s.only(i)
	} else {
		s.run(s.base, s.Anchor, i, s.Picked(s.Anchor))
	}
	s.settle()
}

// Release ends a press: Input and Changed when it changed the selection.
func (s *Select) Release() Events {
	if !s.held {
		return 0
	}
	s.held = false
	return s.changed(s.was)
}

// only selects option i alone.
func (s *Select) only(i int) {
	s.picked = make([]bool, len(s.Options))
	s.picked[i] = true
}

// run sets the enabled options from a to b to on, the others as in base,
// off for a nil base.
func (s *Select) run(base []bool, a, b int, on bool) {
	s.picked = make([]bool, len(s.Options))
	copy(s.picked, base)
	for i := min(a, b); i <= max(a, b); i++ {
		if s.enabled(i) {
			s.picked[i] = on
		}
	}
}

func (s *Select) changed(was []bool) Events {
	for i := range s.Options {
		if s.Picked(i) != (i < len(was) && was[i]) {
			return Input | Changed
		}
	}
	return 0
}

// reach is the enabled option i moves a listbox's keys to from Active,
// as Chrome walks its rows, labels counted: Page Up and Down a page less
// a row on, else the last one short of it; -1 for none.
func (s *Select) reach(i Intent) int {
	rows := s.Rows()
	at := -1
	for k, r := range rows {
		if r.Option == s.Active && s.Active >= 0 {
			at = k
		}
	}
	page := max(s.Size-1, 1)
	switch i {
	case Down:
		return s.along(rows, at, 1, 1)
	case Up:
		if at < 0 {
			at = len(rows)
		}
		return s.along(rows, at, -1, 1)
	case Home:
		return s.along(rows, -1, 1, 1)
	case End:
		return s.along(rows, len(rows), -1, 1)
	case PageDown:
		return s.along(rows, 0, 1, page+at)
	case PageUp:
		if at < 0 {
			at = len(rows)
		}
		return s.along(rows, len(rows)-1, -1, page+len(rows)-1-at)
	case None, Activate, Back, Left, Right:
	}
	return -1
}

// along walks rows from row k by dir, skip rows on, and is the last
// enabled option it met, -1 for none.
func (s *Select) along(rows []Row, k, dir, skip int) int {
	good := -1
	for k += dir; k >= 0 && k < len(rows); k += dir {
		skip--
		if o := rows[k].Option; s.enabled(o) {
			good = o
			if skip <= 0 {
				break
			}
		}
	}
	return good
}
