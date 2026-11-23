package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/stretchr/testify/assert"
)

func fruits() *control.Select {
	return control.NewSelect(
		control.Option{Label: "Apple", Value: "a"},
		control.Option{Label: "Banana", Value: "b", Disabled: true},
		control.Option{Label: "Cherry", Value: "c"},
		control.Option{Label: "Coconut", Value: "co"},
	)
}

func TestASelectStartsOnItsFirstEnabledOption(t *testing.T) {
	s := fruits()
	assert.Equal(t, 0, s.Selected)
	assert.Equal(t, "a", s.Value())
	s = control.NewSelect(control.Option{Label: "x", Disabled: true}, control.Option{Label: "y"})
	assert.Equal(t, 1, s.Selected)
	assert.Equal(t, "y", s.Value(), "no Value: the label")
	assert.Equal(t, -1, control.NewSelect().Selected)
	assert.Empty(t, control.NewSelect(control.Option{Label: "x", Valued: true}).Value(), "an empty value attribute")
}

func TestAClosedSelectOpensOnActivateAndLeavesArrowsToTheFocus(t *testing.T) {
	s := fruits()
	assert.False(t, s.Handles(control.Down))
	assert.Equal(t, control.Events(0), s.Do(control.Activate))
	assert.True(t, s.Open)
	assert.Equal(t, 0, s.Highlight)
	assert.True(t, s.Handles(control.Down), "open, it takes the arrows")
}

func TestAnOpenSelectWalksEnabledOptionsAndChoosesOnActivate(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.Do(control.Down)
	assert.Equal(t, 2, s.Highlight, "skips the disabled Banana")
	s.Do(control.Down)
	s.Do(control.Down)
	assert.Equal(t, 3, s.Highlight, "stops at the last")
	s.Do(control.Home)
	assert.Equal(t, 0, s.Highlight)
	s.Do(control.End)
	assert.Equal(t, 3, s.Highlight)
	s.Do(control.Up)
	assert.Equal(t, control.Input|control.Changed, s.Do(control.Activate))
	assert.Equal(t, 2, s.Selected)
	assert.False(t, s.Open)

	s.Do(control.Activate)
	assert.Equal(t, 2, s.Highlight, "reopens on the chosen one")
	assert.Equal(t, control.Events(0), s.Do(control.Activate), "the same one again changes nothing")
}

func TestBackClosesASelectWithoutChoosing(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.Do(control.Down)
	assert.True(t, s.Handles(control.Back))
	assert.Equal(t, control.Events(0), s.Do(control.Back))
	assert.False(t, s.Open)
	assert.Equal(t, 0, s.Selected)
	assert.False(t, s.Handles(control.Back), "closed, Back goes up")
}

func TestClickingAnOptionChoosesItButNotADisabledOne(t *testing.T) {
	s := fruits()
	s.Click()
	assert.True(t, s.Open)
	assert.Equal(t, control.Events(0), s.ClickOption(1))
	assert.True(t, s.Open)
	assert.Equal(t, control.Input|control.Changed, s.ClickOption(3))
	assert.Equal(t, "co", s.Value())
	assert.False(t, s.Open)
	s.Click()
	s.ClickOutside()
	assert.False(t, s.Open)
}

func TestTypingJumpsToTheNextOptionStartingSo(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.Type('c', 0)
	assert.Equal(t, 2, s.Highlight)
	s.Type('c', 2)
	assert.Equal(t, 3, s.Highlight, "the same letter goes on to the next")
	s.Type('C', 4)
	assert.Equal(t, 2, s.Highlight, "and wraps, any case")
	s.Type('b', 6)
	assert.Equal(t, 2, s.Highlight, "disabled options are never highlighted")

	closed := fruits()
	assert.Equal(t, control.Input|control.Changed, closed.Type('c', 0), "closed, typing chooses")
	assert.Equal(t, 2, closed.Selected)
}

// pantry holds the options Chrome's type-ahead was measured on.
func pantry() *control.Select {
	return control.NewSelect(
		control.Option{Label: "Apple"}, control.Option{Label: "Avocado"}, control.Option{Label: "Banana"},
		control.Option{Label: "Bx", Disabled: true}, control.Option{Label: "Blueberry"},
		control.Option{Label: "Apricot"}, control.Option{Label: "  Cherry"}, control.Option{Label: "Ab"},
		control.Option{Label: "aa"},
	)
}

// TestTypingWithinASecondFindsWhatWasTyped follows Chrome's type-ahead:
// letters typed within a second of each other find the option starting
// with them all, from the selected one on; one letter repeated goes on to
// the next option starting with it; leading spaces are passed over.
func TestTypingWithinASecondFindsWhatWasTyped(t *testing.T) {
	typed := func(text string, gap float32) []int {
		s := pantry()
		var at []int
		for k, r := range []rune(text) {
			s.Type(r, 10+float32(k)*gap)
			at = append(at, s.Selected)
		}
		return at
	}
	assert.Equal(t, []int{1, 5, 7, 8, 0}, typed("aaaaa", 0.1), "one letter cycles")
	assert.Equal(t, []int{1, 5}, typed("ap", 0.1), "a prefix, from the selected option")
	assert.Equal(t, []int{1, 1}, typed("av", 0.1), "the selected option still starts so")
	assert.Equal(t, []int{2, 4, 2}, typed("bbb", 1.1), "past a second, a letter alone, over disabled options")
	assert.Equal(t, []int{1, 5}, typed("ap", 0.6), "within a second of the last letter")
	assert.Equal(t, []int{6}, typed("c", 0), "leading spaces")
	assert.Equal(t, []int{1, 7, 7}, typed("aba", 0.1), "nothing starts so: it stays")
	assert.Equal(t, []int{1, 5, 5}, typed("aab", 0.1))
	assert.Equal(t, []int{2, 4, 4, 4}, typed("blub", 0.1))
}

func TestADisabledSelectNeverOpens(t *testing.T) {
	s := fruits()
	s.Disabled = true
	s.Do(control.Activate)
	s.Click()
	assert.False(t, s.Open)
}

func weapons() *control.Select {
	s := control.NewSelect(
		control.Option{Label: "Sword"},
		control.Option{Label: "Wand", Group: 1},
		control.Option{Label: "Staff", Group: 1},
		control.Option{Label: "Bow"},
	)
	s.Groups = []control.Group{{Label: "Magic", At: 1}, {Label: "Empty", At: 3}, {Label: "Last", At: 4}}
	return s
}

func TestAGroupsLabelStandsBeforeItsOptionsInTheList(t *testing.T) {
	assert.Equal(t, []control.Row{
		{Option: 0}, {Option: -1, Group: 1}, {Option: 1, Group: 1}, {Option: 2, Group: 1},
		{Option: -1, Group: 2}, {Option: 3}, {Option: -1, Group: 3},
	}, weapons().Rows())
	assert.Equal(t, []control.Row{{Option: 0}}, control.NewSelect(control.Option{Label: "x"}).Rows())
}

func TestTheArrowsWalkOptionsAcrossGroupsNeverTheirLabels(t *testing.T) {
	s := weapons()
	s.Do(control.Activate)
	s.Do(control.Down)
	assert.Equal(t, 1, s.Highlight, "from Sword to Wand, past Magic")
	s.Do(control.Down)
	s.Do(control.Down)
	assert.Equal(t, 3, s.Highlight, "from Staff to Bow, past Empty")
	s.Do(control.Activate)
	assert.Equal(t, "Bow", s.Value())
}
