package css //nolint:testpackage // white-box: reads computed styles

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestABackgroundIsClippedToText(t *testing.T) {
	s := computed(t, "background: linear-gradient(red, blue); background-clip: text")
	assert.Equal(t, style.UnderText, s.BackgroundLayer(0).Clip)
	s = computed(t, "background: linear-gradient(red, blue) text")
	assert.Equal(t, style.UnderText, s.BackgroundLayer(0).Clip, "the shorthand's text clips")
	assert.Equal(t, style.InitialBox, s.BackgroundLayer(0).Origin, "and leaves the origin")
	s = computed(t, "background: linear-gradient(red, blue) content-box text")
	assert.Equal(t, style.UnderContent, s.BackgroundLayer(0).Origin, "a box beside it is the origin")
	assert.Equal(t, style.UnderText, s.BackgroundLayer(0).Clip)
	s = computed(t, "background: linear-gradient(red, blue); -webkit-background-clip: text")
	assert.Equal(t, style.UnderText, s.BackgroundLayer(0).Clip, "Chrome's prefixed spelling")
}

func TestTextFillColourFillsTheGlyphs(t *testing.T) {
	p, c := el(dom.Div, "p"), el(dom.Span, "c")
	p.Append(c)
	Compute(page(p), []*Sheet{sheet(t, `
#p { color: red; -webkit-text-fill-color: blue; text-decoration: underline }
#c { color: lime }`)})
	assert.Equal(t, color.RGB(0, 0, 1), styleOf(p).Fill())
	assert.Equal(t, color.RGB(0, 0, 1), styleOf(c).Fill(), "inherited as the colour")
	assert.Equal(t, color.RGB(0, 0, 1), styleOf(p).TextDecoration.Color,
		"a decoration of currentColor drawn in the fill, as Chrome draws it")

	p = el(dom.Div, "p")
	Compute(page(p), []*Sheet{sheet(t, `#p { color: red }`)})
	assert.Equal(t, color.RGB(1, 0, 0), styleOf(p).Fill(), "the colour without one")
}
