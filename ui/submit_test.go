package ui_test

import (
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

const userValiditySheet = `
input { color: #00f; }
input:user-invalid { color: #f00; }
input:user-valid { color: #0f0; }
`

// submitted are the nodes of the Submitted events l heard.
func (l *live) submitted() []*ui.Node {
	var out []*ui.Node
	for _, s := range l.submits {
		out = append(out, s.Node)
	}
	return out
}

func TestASubmitButtonMarksItsFormsFieldsAndStopsOnTheFirstInvalid(t *testing.T) {
	name := ui.Input(ui.Attr("required", ""))
	code := ui.Input(ui.Attr("pattern", "[a-z]+"), ui.Value("9"))
	note := ui.Input(ui.Value("ok"))
	send := ui.Button(ui.Text("Send"))
	form := ui.Form(name, code, note, send)
	outside := ui.Input(ui.Attr("required", ""))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(userValiditySheet)}, form, outside)
	assert.Equal(t, paintedBlue, l.colorOf(t, name), "nothing before the submission")

	l.clickAt(send)
	assert.Equal(t, paintedRed, l.colorOf(t, name))
	assert.Equal(t, paintedRed, l.colorOf(t, code))
	assert.Equal(t, paintedGreen, l.colorOf(t, note))
	assert.Equal(t, paintedBlue, l.colorOf(t, outside), "a field of no form")
	assert.True(t, l.doc.Focused(name), "the first invalid field takes the focus")
	assert.Empty(t, l.submitted(), "an invalid form is not submitted")

	l.typed("ada")
	l.doc.SetValue(code, "abc")
	l.clickAt(send)
	assert.Equal(t, []*ui.Node{form}, l.submitted())
}

func TestEnterInAFieldClicksItsFormsSubmitButton(t *testing.T) {
	name := ui.Input()
	other := ui.Input(ui.Attr("required", ""))
	send := ui.Button(ui.Text("Send"))
	form := ui.Form(name, other, send)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(userValiditySheet)}, form)
	l.clickAt(name)
	l.typed("a")
	l.key(input.KeyEnter)
	assert.Contains(t, l.clicks, send, "Enter clicks the default button")
	assert.Equal(t, paintedRed, l.colorOf(t, other))
	assert.Equal(t, []*ui.Node{name}, l.submitted(), "the field's own Submitted, not the form's")
}

func TestEnterSubmitsAFormWithoutAButtonOnlyWhenItHasOneField(t *testing.T) {
	for _, c := range []struct {
		name   string
		fields int
		want   bool
	}{{"one field", 1, true}, {"two fields", 2, false}} {
		var fields []*ui.Node
		for range c.fields {
			fields = append(fields, ui.Input())
		}
		args := make([]ui.Arg, 0, len(fields)+1)
		for _, f := range fields {
			args = append(args, f)
		}
		args = append(args, ui.Input(ui.Type("checkbox")))
		form := ui.Form(args...)
		l := liveDoc(t, form)
		l.clickAt(fields[0])
		l.typed("a")
		l.key(input.KeyEnter)
		assert.Equal(t, c.want, len(l.submitted()) == 2, c.name)
	}
}

func TestADisabledDefaultButtonBlocksEnter(t *testing.T) {
	name := ui.Input()
	form := ui.Form(name, ui.Button(ui.Disabled(), ui.Text("Send")), ui.Button(ui.Text("Other")))
	l := liveDoc(t, form)
	l.clickAt(name)
	l.typed("a")
	l.key(input.KeyEnter)
	assert.Equal(t, []*ui.Node{name}, l.submitted())
}

func TestNovalidateSubmitsAnInvalidFormButMarksItsFields(t *testing.T) {
	name := ui.Input(ui.Attr("required", ""))
	form := ui.Form(ui.Attr("novalidate", ""), name)
	plain := ui.Input(ui.Attr("required", ""))
	skip := ui.Button(ui.Attr("formnovalidate", ""), ui.Text("Skip"))
	other := ui.Form(plain, skip)
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(userValiditySheet)}, form, other)
	l.doc.RequestSubmit(form)
	l.frames(1)
	assert.Equal(t, paintedRed, l.colorOf(t, name))
	l.clickAt(skip)
	assert.Equal(t, paintedRed, l.colorOf(t, plain))
	assert.Equal(t, []*ui.Node{form, other}, l.submitted())
}

func TestAnEmailsDomainIsReadInPunycodeUnlessTheProgramSetIt(t *testing.T) {
	typed := ui.Input(ui.Type("email"))
	given := ui.Input(ui.Type("email"), ui.Value("x@bücher.de"))
	set := ui.Input(ui.Type("email"))
	many := ui.Input(ui.Type("email"), ui.Attr("multiple", ""), ui.Value("x@bücher.de, y@é.fr"))
	local := ui.Input(ui.Type("email"), ui.Value("ü@a.de"))
	l := liveDoc(t, typed, given, set, many, local)
	l.doc.SetValue(set, "x@bücher.de")
	l.clickAt(typed)
	l.typed("x@ＡＢ.com")
	assert.True(t, l.doc.Valid(typed), "typed")
	assert.True(t, l.doc.Valid(given), "the value attribute")
	assert.False(t, l.doc.Valid(set), "the program's value, as a script's")
	assert.True(t, l.doc.Valid(many))
	assert.False(t, l.doc.Valid(local), "the local part stays ASCII")
	l.clickAt(set)
	l.typed("x")
	assert.True(t, l.doc.Valid(set), "the player's edit is read in punycode")
}
