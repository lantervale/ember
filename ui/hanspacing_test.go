package ui_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestTheSelectionFollowsTrimmedPunctuationAndAutospace(t *testing.T) {
	face, err := os.ReadFile("../internal/shape/testdata/go-han.ttf")
	require.NoError(t, err)
	sheet := selectable + `@font-face { font-family: Han; src: url(han.ttf); }
p { font: 20px Han; white-space: nowrap; text-autospace: normal; }
.all { text-spacing-trim: space-all; text-autospace: no-autospace; }`
	plain, spaced := ui.P(ui.Class("all"), ui.Text("」「漢A")), ui.P(ui.Text("」「漢A"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Stylesheet(fstest.MapFS{
		"page.css": {Data: []byte(sheet)}, "han.ttf": {Data: face},
	}, "page.css")}, plain, spaced)
	for i, d := range []float32{0, 0, -10, -7.5} {
		assert.InDelta(t, l.at(t, plain, i).X+d, l.at(t, spaced, i).X, 0.01, "character %d", i)
	}
	width := func(n *ui.Node) float32 {
		l.doc.Select(ui.Position{Node: textOf(n), Offset: 0}, ui.Position{Node: textOf(n), Offset: 4})
		rects := l.doc.SelectionRects()
		require.Len(t, rects, 1)
		return rects[0].W
	}
	assert.InDelta(t, width(plain)-7.5, width(spaced), 0.01, "half the bracket off, an eighth of an em on")
	assert.Empty(t, l.doc.Warnings())
}

func TestAFieldsCaretFollowsTrimmedPunctuationAndAutospace(t *testing.T) {
	face, err := os.ReadFile("../internal/shape/testdata/go-han.ttf")
	require.NoError(t, err)
	caret := func(class, value string, at int) float32 {
		d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(`
@font-face { font-family: Han; src: url(han.ttf); }
body { margin: 0; } input, textarea { font: 20px Han; padding: 0; border: none; width: 300px; }
.auto { text-autospace: normal; } .all { text-spacing-trim: space-all; }`)}, "han.ttf": {Data: face}}, "page.css"))
		in := ui.Input(ui.Class(class), ui.Attr("value", value))
		if strings.Contains(value, "\n") {
			in = ui.Textarea(ui.Class(class), ui.Text(value))
		}
		d.Body().Append(in)
		d.Layout(geom.V2(400, 100))
		d.Focus(in)
		d.SetSelectionRange(in, at, at)
		d.Layout(geom.V2(400, 100))
		return ui.FieldCaretRect(d, in).X
	}
	assert.InDelta(t, caret("all", "」「漢", 3)-10, caret("", "」「漢", 3), 0.01, "the opening bracket half wide")
	assert.InDelta(t, caret("", "漢A漢", 2)+5, caret("auto", "漢A漢", 2), 0.01,
		"an eighth of an em after the ideograph and after the letter")
	assert.InDelta(t, caret("all", "字\n」「漢", 5)-10, caret("", "字\n」「漢", 5), 0.01, "along a textarea's line")
}
