package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/stretchr/testify/assert"
)

// gear is a listbox of five options, the fourth disabled, in rows of
// three.
func gear(multiple bool) *control.Select {
	s := control.NewSelect(
		control.Option{Label: "Sword"}, control.Option{Label: "Bow"}, control.Option{Label: "Wand"},
		control.Option{Label: "Skull", Disabled: true}, control.Option{Label: "Staff"},
	)
	s.SetListBox(true, multiple, 3)
	s.SetPicked()
	return s
}

func picked(s *control.Select) []int {
	var out []int
	for i := range s.Options {
		if s.Picked(i) {
			out = append(out, i)
		}
	}
	return out
}

func TestAListBoxStartsWithNothingSelected(t *testing.T) {
	s := gear(false)
	assert.Equal(t, -1, s.Selected, "unlike a pop-up list, no option is chosen for it")
	assert.Empty(t, picked(s))
	assert.False(t, s.Handles(control.Back))
	assert.True(t, s.Handles(control.Down), "the arrows are its own, never the focus's")
	assert.Equal(t, control.Events(0), s.Click(), "it never opens")
	assert.False(t, s.Open)
}

func TestAListBoxsArrowsSelectTheNextEnabledOption(t *testing.T) {
	s := gear(true)
	assert.Equal(t, control.Input|control.Changed, s.Key(control.Down, false, false))
	assert.Equal(t, []int{0}, picked(s), "from none, Down selects the first")
	s.Key(control.Down, false, false)
	s.Key(control.Down, false, false)
	assert.Equal(t, []int{2}, picked(s), "one at a time")
	s.Key(control.Down, false, false)
	assert.Equal(t, []int{4}, picked(s), "past the disabled Skull")
	assert.Equal(t, control.Events(0), s.Key(control.Down, false, false), "the last stays")
	s.Key(control.Home, false, false)
	assert.Equal(t, []int{0}, picked(s))
	s.Key(control.End, false, false)
	assert.Equal(t, []int{4}, picked(s))
	assert.Equal(t, 4, s.Selected)
}

func TestShiftAndTheArrowsSelectARunFromTheAnchor(t *testing.T) {
	s := gear(true)
	s.Press(1, false, false)
	s.Release()
	s.Key(control.Down, true, false)
	assert.Equal(t, []int{1, 2}, picked(s))
	s.Key(control.Up, true, false)
	s.Key(control.Up, true, false)
	assert.Equal(t, []int{0, 1}, picked(s), "the run turns round the anchor")
	s.Key(control.End, true, false)
	assert.Equal(t, []int{1, 2, 4}, picked(s), "a disabled option inside the run is left out")

	single := gear(false)
	single.Key(control.Down, false, false)
	single.Key(control.Down, true, false)
	assert.Equal(t, []int{1}, picked(single), "one alone without multiple")
}

func TestCommandAndTheArrowsMoveWithoutSelectingAndSpaceToggles(t *testing.T) {
	s := gear(true)
	s.Key(control.Down, false, false)
	assert.Equal(t, control.Events(0), s.Key(control.Down, false, true))
	assert.Equal(t, 1, s.Active)
	assert.Equal(t, []int{0}, picked(s))
	assert.Equal(t, control.Input|control.Changed, s.Key(control.Activate, false, false))
	assert.Equal(t, []int{0, 1}, picked(s))
	s.Key(control.Activate, false, false)
	assert.Equal(t, []int{0}, picked(s), "Space again turns it off")

	single := gear(false)
	single.Key(control.Down, false, false)
	single.Key(control.Down, false, true)
	assert.Equal(t, []int{1}, picked(single), "Command changes nothing without multiple")
}

func TestPageDownAndUpGoAPageLessARow(t *testing.T) {
	s := gear(false)
	s.Key(control.Home, false, false)
	s.Key(control.PageDown, false, false)
	assert.Equal(t, []int{2}, picked(s), "rows of three: two on")
	s.Key(control.PageDown, false, false)
	assert.Equal(t, []int{4}, picked(s), "the last enabled one short of a page")
	s.Key(control.PageUp, false, false)
	assert.Equal(t, []int{2}, picked(s))
}

func TestAPressSelectsCommandTogglesShiftRuns(t *testing.T) {
	s := gear(true)
	s.Press(1, false, false)
	assert.Equal(t, control.Input|control.Changed, s.Release())
	s.Press(1, false, false)
	assert.Equal(t, control.Events(0), s.Release(), "the same selection again changes nothing")
	s.Press(0, false, true)
	s.Release()
	assert.Equal(t, []int{0, 1}, picked(s))
	s.Press(0, false, true)
	s.Release()
	assert.Equal(t, []int{1}, picked(s), "Command again turns it off")
	s.Press(4, true, false)
	s.Release()
	assert.Equal(t, []int{0, 1, 2, 4}, picked(s), "from the anchor, the last pressed, past the disabled")
	s.Press(3, false, false)
	assert.Equal(t, control.Events(0), s.Release(), "a disabled option takes no press")
	s.Press(2, false, false)
	s.Release()
	assert.Equal(t, []int{2}, picked(s))
}

func TestADragSelectsTheRunItCrosses(t *testing.T) {
	s := gear(true)
	s.Press(0, false, false)
	s.DragTo(1)
	s.DragTo(2)
	assert.Equal(t, []int{0, 1, 2}, picked(s))
	s.DragTo(1)
	assert.Equal(t, []int{0, 1}, picked(s), "back again")
	s.DragTo(3)
	assert.Equal(t, []int{0, 1}, picked(s), "over a disabled option it holds")
	assert.Equal(t, control.Input|control.Changed, s.Release(), "one change for the whole drag")
	s.DragTo(2)
	assert.Equal(t, []int{0, 1}, picked(s), "let go, it moves nothing")

	single := gear(false)
	single.Press(0, false, false)
	single.DragTo(2)
	assert.Equal(t, []int{2}, picked(single), "without multiple the one under the pointer")
}

func TestTypingSelectsTheOptionStartingSoAlone(t *testing.T) {
	s := gear(true)
	s.SetPicked(0, 1)
	assert.Equal(t, control.Changed, s.Type('w', 0), "a change alone, as Chrome's")
	assert.Equal(t, []int{2}, picked(s))
	assert.Equal(t, 2, s.Active)
	s.SetPicked(1, 4)
	s.Type('s', 2)
	assert.Equal(t, []int{4}, picked(s), "from the first option selected")
	s.Type('b', 4)
	assert.Equal(t, control.Changed, s.Type('b', 6), "the same option found again")
	assert.Equal(t, control.Events(0), s.Type('x', 8), "nothing found")

	s.Press(0, false, false)
	s.Release()
	assert.Equal(t, control.Input|control.Changed, s.Type('w', 10), "after a press, input and change")
	assert.Equal(t, control.Events(0), s.Type('w', 12), "for a new selection only")
}

func TestSetPickedHoldsOneWithoutMultiple(t *testing.T) {
	s := gear(true)
	s.SetPicked(4, 1)
	assert.Equal(t, []int{1, 4}, picked(s))
	assert.Equal(t, 1, s.Selected, "the first selected")
	single := gear(false)
	single.SetPicked(1, 4)
	assert.Equal(t, []int{4}, picked(single), "the last, as selected attributes do")
	single.SetPicked()
	assert.Equal(t, -1, single.Selected)
}

func TestAListBoxShownAsAPopUpListKeepsItsFirstChoice(t *testing.T) {
	s := gear(true)
	s.SetPicked(1, 2)
	s.SetListBox(false, false, 0)
	assert.Equal(t, 1, s.Selected)
	assert.Equal(t, []int{1}, picked(s))
	s = gear(false)
	s.SetListBox(false, false, 0)
	assert.Equal(t, 0, s.Selected, "a pop-up list chooses its first enabled option")
}

func TestAListBoxsGroupLabelsArePassedOver(t *testing.T) {
	s := weapons()
	s.SetListBox(true, true, 4)
	s.SetPicked()
	s.Key(control.Down, false, false)
	s.Key(control.Down, false, false)
	assert.Equal(t, []int{1}, picked(s), "from Sword to Wand, past Magic")
	s.Press(-1, false, false)
	assert.Equal(t, control.Events(0), s.Release(), "a label takes no press")
}

func TestADisabledListBoxTakesNothing(t *testing.T) {
	s := gear(true)
	s.Disabled = true
	assert.Equal(t, control.Events(0), s.Key(control.Down, false, false))
	s.Press(0, false, false)
	assert.Equal(t, control.Events(0), s.Release())
	assert.False(t, s.Handles(control.Down))
}
