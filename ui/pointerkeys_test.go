package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// focusField clicks the field, which takes the focus.
func (l *live) focusField(t *testing.T, field *ui.Node) {
	t.Helper()
	in := l.win.Input()
	in.SetMouse(field.Box().Center())
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(1)
	require.True(t, l.doc.Focused(field))
}

func TestBlurTakesTheFocusAndCommitsTheField(t *testing.T) {
	field := ui.Input(ui.ID("address"))
	l := liveDoc(t, field)
	l.focusField(t, field)
	l.win.Input().AddText('a')
	l.frames(1)

	l.doc.Blur()
	l.frames(1)
	assert.False(t, l.doc.Focused(field))
	assert.Equal(t, []*ui.Node{field}, l.changes, "leaving it from code commits its edit, as a click away")
	l.frames(1)
	assert.False(t, l.win.Input().KeyboardCaptured(), "the keys are the game's again")

	l.doc.Focus(field)
	l.doc.Blur()
	l.frames(2)
	assert.False(t, l.doc.Focused(field), "a Blur drops a Focus asked before it")
}

func TestOnKeyHearsTheKeysOfAFocusedFieldBeforeIt(t *testing.T) {
	var heard []ui.Event
	field := ui.Input(ui.ID("console"), ui.OnKey(func(e ui.Event) {
		heard = append(heard, e)
		if e.Key == input.KeyEnter || e.Key == input.KeyB {
			e.PreventDefault()
		}
	}))
	var rooted []input.Key
	l := liveDoc(t, ui.Div(field, ui.OnKey(func(e ui.Event) { rooted = append(rooted, e.Key) })))
	var keys world.Reader[ui.KeyPressed]
	l.focusField(t, field)
	pressed := 0
	frames := func(n int) {
		for range n {
			l.frames(1)
			pressed += keys.Count(l.w)
		}
	}

	in := l.win.Input()
	in.SetKey(input.KeyA, true)
	in.AddText('a')
	frames(1)
	in.SetKey(input.KeyA, false)
	in.SetKey(input.KeyB, true)
	in.AddText('b')
	frames(1)
	in.SetKey(input.KeyB, false)
	frames(1)
	assert.Equal(t, "a", l.doc.Value(field), "b's keydown was prevented: it types nothing")
	require.Len(t, heard, 2)
	assert.Equal(t, input.KeyA, heard[0].Key)
	assert.Equal(t, "a", heard[0].Text)
	assert.Same(t, field, heard[0].Node)
	assert.Equal(t, []input.Key{input.KeyA, input.KeyB}, rooted, "it bubbles as a keydown")

	l.key(input.KeyEnter)
	assert.Empty(t, l.submits, "a prevented Enter submits nothing")
	l.key(input.KeyLeft)
	in.AddText('X')
	l.frames(1)
	assert.Equal(t, "Xa", l.doc.Value(field), "what isn't prevented still moves the caret")
	frames(1)
	assert.Zero(t, pressed, "the field's keys emit no KeyPressed")
}

func TestOnKeyPreventingTabOrEscapeKeepsTheFocus(t *testing.T) {
	field := ui.Input(ui.OnKey(func(e ui.Event) { e.PreventDefault() }))
	l := liveDoc(t, field, ui.Button(ui.Text("next")))
	l.focusField(t, field)
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(field))
	l.key(input.KeyEscape)
	assert.True(t, l.doc.Focused(field))
}

func TestSetValueFromAFieldsOwnCallbackSetsItWhileTypedIn(t *testing.T) {
	history := []string{"first", "second"}
	var field *ui.Node
	var doc *ui.Document
	at := len(history)
	field = ui.Input(ui.OnKey(func(e ui.Event) {
		if e.Key == input.KeyUp && at > 0 {
			at--
			doc.SetValue(field, history[at])
			e.PreventDefault()
		}
	}))
	l := liveDoc(t, field)
	doc = l.doc
	l.focusField(t, field)
	l.win.Input().AddText('x')
	l.frames(1)

	doc.SetValue(field, "from the game")
	l.frames(1)
	assert.Equal(t, "x", doc.Value(field), "the game's own SetValue still waits while it is typed in")

	l.key(input.KeyUp)
	assert.Equal(t, "second", doc.Value(field), "its callback sets it at once")
	l.win.Input().AddText('!')
	l.frames(1)
	assert.Equal(t, "second!", doc.Value(field), "the caret is at the end")
	l.key(input.KeyUp)
	assert.Equal(t, "first", doc.Value(field))
}

func TestAControlClickOnMacOSIsAContextClick(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	var at []geom.Vec2
	button := ui.Button(ui.Text("Slot"), ui.Style("width: 80px; height: 40px"),
		ui.OnContextMenu(func(e ui.Event) { at = append(at, e.At) }))
	l := liveDoc(t, button)
	var h heard
	in := l.win.Input()
	in.SetKey(input.KeyLeftControl, true)
	where := button.Box().Center().Add(geom.V2(5, 3))
	h.press(l, input.MouseLeft, where)
	h.release(l, input.MouseLeft)
	in.SetKey(input.KeyLeftControl, false)
	require.Len(t, h.contexts, 1)
	assert.Equal(t, where, h.contexts[0].At, "the event carries where it was")
	assert.Equal(t, []geom.Vec2{where}, at)
	assert.Empty(t, l.clicks, "a Control click is no click")
	assert.False(t, l.doc.Focused(button), "nor does it focus")

	ui.SetPlatform(t, textedit.Others)
	in.SetKey(input.KeyLeftControl, true)
	h.press(l, input.MouseLeft, where)
	h.release(l, input.MouseLeft)
	in.SetKey(input.KeyLeftControl, false)
	assert.Len(t, h.contexts, 1, "elsewhere Control and the left button click")
	assert.Len(t, l.clicks, 1)
}

func TestARightClickCarriesWhereItWas(t *testing.T) {
	button := ui.Button(ui.Text("Slot"), ui.Style("width: 80px; height: 40px"), ui.OnContextMenu(func() {}))
	l := liveDoc(t, button)
	var h heard
	where := button.Box().Center().Add(geom.V2(-7, 2))
	h.press(l, input.MouseRight, where)
	h.release(l, input.MouseRight)
	require.Len(t, h.contexts, 1)
	assert.Equal(t, where, h.contexts[0].At)
}

func TestPointerCallbacksAndCaptureDragAnEdge(t *testing.T) {
	var log []string
	var moves []geom.Vec2
	var doc *ui.Document
	edge := ui.Div(ui.ID("edge"), ui.Style("width: 6px; height: 200px"))
	panel := ui.Div(ui.Style("display: flex"), edge, ui.Div(ui.Style("width: 300px; height: 200px")),
		ui.OnPointerDown(func(e ui.Event) {
			log = append(log, "down")
			assert.Equal(t, input.MouseLeft, e.Button)
			if e.Node == edge {
				doc.SetPointerCapture(edge)
			}
		}),
		ui.OnPointerMove(func(e ui.Event) {
			if doc.PointerCapture() == edge {
				moves = append(moves, e.At)
			}
		}),
		ui.OnPointerUp(func(e ui.Event) {
			log = append(log, "up")
			assert.Same(t, edge, e.Node, "the release goes to the element holding the pointer")
		}))
	l := liveDoc(t, panel)
	doc = l.doc
	var pr world.Reader[ui.PointerPressed]
	var rr world.Reader[ui.PointerReleased]

	from := edge.Box().Center()
	far := geom.V2(500, 300)
	in := l.win.Input()
	in.SetMouse(from)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	assert.Equal(t, 1, pr.Count(l.w))
	assert.Same(t, edge, doc.PointerCapture())
	in.SetMouse(geom.V2(200, 100))
	l.frames(1)
	in.SetMouse(far)
	l.frames(1)
	assert.Equal(t, []geom.Vec2{geom.V2(200, 100), far}, moves, "held, its moves reach the edge past its box")
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(1)
	assert.Equal(t, []string{"down", "up"}, log)
	assert.Equal(t, 1, rr.Count(l.w))
	assert.Nil(t, doc.PointerCapture(), "let go after the release")
	in.SetMouse(far.Add(geom.V2(1, 1)))
	l.frames(1)
	assert.Len(t, moves, 2, "over the body, which is the game's, no move is heard")
}
