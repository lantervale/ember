package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func TestElementAtIsTheInnermostElementUnderAPoint(t *testing.T) {
	word := ui.Span(ui.Text("door"))
	card := ui.Div(ui.Style("position: absolute; left: 10px; top: 20px; width: 200px; height: 100px"), word)
	over := ui.Div(ui.Style("position: absolute; left: 150px; top: 20px; width: 100px; height: 100px"))
	glass := ui.Div(ui.Style("position: absolute; inset: 0; pointer-events: none"))
	l := liveDoc(t, card, over, glass)

	assert.Same(t, word, l.doc.ElementAt(geom.V2(12, 25)),
		"the span, not the text in it nor the card round it, through the glass that takes no pointer")
	assert.Same(t, card, l.doc.ElementAt(geom.V2(100, 110)), "the card where it holds nothing")
	assert.Same(t, over, l.doc.ElementAt(geom.V2(160, 30)), "what is drawn over, as a click finds it")
	assert.Nil(t, l.doc.ElementAt(geom.V2(400, 300)), "nothing where no box takes the pointer")
	assert.Nil(t, l.doc.ElementAt(geom.V2(-5, 10)), "nor outside the Document")
	assert.Nil(t, ui.NewDocument().ElementAt(geom.V2(5, 5)), "nor before a layout")
}

func TestElementAtTakesWindowPointsUnderAZoom(t *testing.T) {
	card := ui.Div(ui.Style("position: absolute; left: 100px; top: 0; width: 50px; height: 50px"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Zoom(2)}, card)
	assert.Same(t, card, l.doc.ElementAt(geom.V2(250, 50)))
	assert.NotSame(t, card, l.doc.ElementAt(geom.V2(120, 20)))
}

func TestAFieldsSelectionIsReadAndSetByRange(t *testing.T) {
	field := ui.Input(ui.Value("lower vaults"))
	plain := ui.Div()
	l := liveDoc(t, field, plain)

	from, to := l.doc.SelectionRange(field)
	assert.Equal(t, [2]int{from, to}, [2]int{12, 12}, "a field starts with its caret at the end")

	l.doc.SetSelectionRange(field, 6, 99)
	from, to = l.doc.SelectionRange(field)
	assert.Equal(t, [2]int{6, 12}, [2]int{from, to}, "kept inside the text")

	l.doc.Focus(field)
	l.frames(2)
	in := l.win.Input()
	in.SetKey(input.KeyBackspace, true)
	l.frames(1)
	in.SetKey(input.KeyBackspace, false)
	l.frames(1)
	assert.Equal(t, "lower ", l.doc.Value(field), "the next key edits the range")

	l.doc.SetSelectionRange(plain, 1, 2)
	from, to = l.doc.SelectionRange(plain)
	assert.Equal(t, [2]int{0, 0}, [2]int{from, to}, "what is no text field has none")
}
