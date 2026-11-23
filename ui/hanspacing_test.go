package ui_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
