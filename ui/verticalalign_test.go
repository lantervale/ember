package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui"
)

func TestAlignedTextIsSelectedAndClickedWhereItShows(t *testing.T) {
	top, bottom := ui.Span(ui.Class("top"), ui.Text("up")), ui.Span(ui.Class("bottom"), ui.Text("down"))
	clicked := ""
	box := ui.Button(ui.Class("bottom"), ui.Text("go"), ui.OnClick(func(ui.Event) { clicked = "go" }))
	p := ui.P(ui.Text("line "), top, ui.Text(" "), bottom, ui.Text(" "), box)
	l := selectableDoc(t, `p { font-size: 10px; line-height: 80px; margin: 0; }
span, button { line-height: 12px; } .top { vertical-align: top; } .bottom { vertical-align: bottom; }`, p)
	line := p.Box()
	assert.InDelta(t, 80, line.H, 0.01)
	assert.InDelta(t, line.Y, top.Box().Y, 1.01, "the top span's text starts with the line")
	assert.InDelta(t, line.Y+line.H, bottom.Box().Y+bottom.Box().H, 1.01)
	assert.InDelta(t, line.Y+line.H, box.Box().Y+box.Box().H, 0.01)

	l.dragged(l.at(t, top, 0), l.at(t, top, 1))
	assert.Equal(t, "u", l.doc.Selection().Text)
	rects := l.doc.SelectionRects()
	require.Len(t, rects, 1)
	assert.LessOrEqual(t, rects[0].Y, top.Box().Y, "the selection covers the text where it shows")

	l.dragged(l.at(t, bottom, 1), l.at(t, bottom, 3))
	assert.Equal(t, "ow", l.doc.Selection().Text)
	rects = l.doc.SelectionRects()
	require.Len(t, rects, 1)
	assert.GreaterOrEqual(t, rects[0].Y+rects[0].H, bottom.Box().Y+bottom.Box().H)

	b := box.Box()
	l.down(b.Center())
	l.up()
	l.frames(1)
	assert.Equal(t, "go", clicked)
	assert.Empty(t, l.doc.Warnings())
}
