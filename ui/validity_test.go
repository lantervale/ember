package ui_test

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// colorOf is n's color as the last frame styled it.
func (l *live) colorOf(t *testing.T, n *ui.Node) color.Color {
	t.Helper()
	st := dom.Data(n).Style
	require.NotNil(t, st)
	return st.Color
}

//nolint:gochecknoglobals // the colours the sheets give
var (
	paintedRed   = color.Hex("#f00")
	paintedGreen = color.Hex("#0f0")
	paintedBlue  = color.Hex("#00f")
)

func TestTypingRestylesAFieldByItsValidity(t *testing.T) {
	name := ui.Input(ui.Attr("required", ""), ui.Attr("pattern", "[a-z]+"), ui.Placeholder("Name"))
	tip := ui.Span(ui.Text("!"))
	form := ui.Form(name, tip)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
input:invalid { color: #f00; }
input:valid { color: #0f0; }
input:placeholder-shown { background-color: #00f; }
:invalid + span { color: #f00; }
span { color: #0f0; }
form:has(:invalid) { color: #00f; }
form { color: #0f0; }
`)}, form)
	assert.Equal(t, paintedRed, l.colorOf(t, name), "required and empty")
	assert.Equal(t, paintedRed, l.colorOf(t, tip))
	assert.Equal(t, paintedBlue, l.colorOf(t, form))
	assert.False(t, l.doc.Valid(name))
	assert.False(t, l.doc.Valid(form))
	assert.Equal(t, paintedBlue, dom.Data(name).Style.Background)

	l.clickAt(name)
	l.typed("ada")
	assert.Equal(t, paintedGreen, l.colorOf(t, name), "the pattern holds")
	assert.Equal(t, paintedGreen, l.colorOf(t, tip), "the sibling restyles")
	assert.Equal(t, paintedGreen, l.colorOf(t, form), "and the form through :has()")
	assert.True(t, l.doc.Valid(form))
	assert.NotEqual(t, paintedBlue, dom.Data(name).Style.Background, "no placeholder shows")

	l.typed("1")
	assert.Equal(t, paintedRed, l.colorOf(t, name), "the pattern breaks")
	assert.Equal(t, paintedBlue, l.colorOf(t, form))
}

func TestUserInvalidWaitsForTheFieldsChange(t *testing.T) {
	field := ui.Input(ui.Type("email"))
	other := ui.Input()
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
input { color: #00f; }
input:user-invalid { color: #f00; }
input:user-valid { color: #0f0; }
`)}, field, other)
	l.clickAt(field)
	l.typed("ada")
	assert.Equal(t, paintedBlue, l.colorOf(t, field), "still typing")
	assert.False(t, l.doc.Valid(field))
	l.clickAt(other)
	assert.Equal(t, paintedRed, l.colorOf(t, field), "changed and left")
	l.clickAt(field)
	l.typed("@example.com")
	l.key(input.KeyEnter)
	assert.Equal(t, paintedGreen, l.colorOf(t, field))
}

func TestMinlengthWaitsForThePlayersEdit(t *testing.T) {
	field := ui.Input(ui.Attr("minlength", "4"), ui.Value("ab"))
	l := liveDoc(t, field)
	assert.True(t, l.doc.Valid(field), "a value the program set breaks no length")
	l.clickAt(field)
	l.typed("c")
	assert.False(t, l.doc.Valid(field), "the player's edit does")
	l.typed("d")
	assert.True(t, l.doc.Valid(field))
	l.doc.SetValue(field, "x")
	l.frames(1)
	assert.True(t, l.doc.Valid(field), "the program's value again")
}

func TestANumberIsInRangeOrOut(t *testing.T) {
	age := ui.Input(ui.Type("number"), ui.Attr("min", "0"), ui.Attr("max", "120"), ui.Attr("step", "1"))
	other := ui.Input()
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
input:in-range { color: #0f0; }
input:out-of-range { color: #f00; }
`)}, age, other)
	assert.Equal(t, paintedGreen, l.colorOf(t, age))
	l.clickAt(age)
	l.typed("130")
	assert.Equal(t, paintedRed, l.colorOf(t, age))
	assert.False(t, l.doc.Valid(age))
	l.clickAt(other)
	assert.Equal(t, paintedGreen, l.colorOf(t, age), "left, it settles on its max")
	l.doc.SetValue(age, "1.5")
	l.frames(1)
	assert.Equal(t, paintedGreen, l.colorOf(t, age))
	assert.False(t, l.doc.Valid(age), "off its steps")
}

func TestCheckedStatesRestyle(t *testing.T) {
	all := ui.Input(ui.Type("checkbox"))
	agree := ui.Input(ui.Type("checkbox"), ui.Attr("required", ""))
	a := ui.Input(ui.Type("radio"), ui.Attr("name", "size"), ui.Attr("required", ""))
	b := ui.Input(ui.Type("radio"), ui.Attr("name", "size"))
	pick := ui.Select(ui.Attr("required", ""),
		ui.Option(ui.Value(""), ui.Text("Choose")), ui.Option(ui.Text("One")))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
:invalid { color: #f00; }
:valid { color: #0f0; }
:indeterminate { color: #00f; }
option:checked { color: #00f; }
`)}, all, agree, a, b, pick)
	l.doc.SetIndeterminate(all, true)
	l.frames(1)
	assert.True(t, l.doc.Indeterminate(all))
	assert.Equal(t, paintedBlue, l.colorOf(t, all))
	assert.Equal(t, paintedBlue, l.colorOf(t, a), "a group none of which is checked")
	assert.Equal(t, paintedRed, l.colorOf(t, agree))
	assert.Equal(t, paintedRed, l.colorOf(t, pick), "its placeholder option is chosen")
	assert.Equal(t, paintedBlue, l.colorOf(t, pick.Children()[0]))

	l.clickAt(all)
	assert.False(t, l.doc.Indeterminate(all), "a click settles it")
	assert.Equal(t, paintedGreen, l.colorOf(t, all))
	l.clickAt(agree)
	assert.Equal(t, paintedGreen, l.colorOf(t, agree))
	l.clickAt(b)
	assert.Equal(t, paintedGreen, l.colorOf(t, a), "the group is answered")
	l.doc.SetSelected(pick, 1)
	l.frames(1)
	assert.Equal(t, paintedGreen, l.colorOf(t, pick))
	assert.Equal(t, paintedBlue, l.colorOf(t, pick.Children()[1]))
}

func TestADisabledFieldsetDisablesItsFieldsButItsLegends(t *testing.T) {
	inLegend, inside := ui.Input(), ui.Input()
	fs := ui.Fieldset(ui.Disabled(), ui.Legend(inLegend), inside)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
input:disabled { color: #f00; }
input:enabled { color: #0f0; }
`)}, fs)
	assert.Equal(t, paintedRed, l.colorOf(t, inside))
	assert.Equal(t, paintedGreen, l.colorOf(t, inLegend))
	l.clickAt(inside)
	assert.False(t, l.doc.Focused(inside), "it takes no focus")
	l.clickAt(inLegend)
	assert.True(t, l.doc.Focused(inLegend))

	fs.RemoveAttr("disabled")
	l.frames(2)
	assert.Equal(t, paintedGreen, l.colorOf(t, inside))
}

func TestDirFollowsTheAttributeAboveAndWhatIsTyped(t *testing.T) {
	field := ui.Input(ui.Attr("dir", "auto"))
	word := ui.Span(ui.Text("word"))
	box := ui.Div(word)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`
:dir(rtl) { color: #f00; }
:dir(ltr) { color: #0f0; }
`)}, field, box)
	box.SetAttr("dir", "rtl")
	l.frames(1)
	assert.Equal(t, paintedRed, l.colorOf(t, word), "a dir above restyles it")
	assert.Equal(t, paintedGreen, l.colorOf(t, field))
	l.clickAt(field)
	l.typed("שלום")
	assert.Equal(t, paintedRed, l.colorOf(t, field))
}
