package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestTextStrokeAndFillAreInheritedAndFollowTheColor(t *testing.T) {
	inner, plain, own := el(dom.Span, "inner"), el(dom.Span, "plain"), el(dom.Span, "own")
	outer := with(el(dom.Div, "outer"), inner, plain, own)
	Compute(page(outer), []*Sheet{sheet(t, `#outer { -webkit-text-stroke: thin; color: #f00 }
#inner { color: #00f; -webkit-text-fill-color: transparent }
#plain { -webkit-text-stroke-width: medium; -webkit-text-stroke-color: #0f0 }
#own { -webkit-text-stroke: #0f0 2.5px; -webkit-text-fill-color: #ff0 }`)})
	red, blue, green := color.RGB8(255, 0, 0), color.RGB8(0, 0, 255), color.RGB8(0, 255, 0)
	assert.Equal(t, style.TextStroke{Width: 1, Color: red}, styleOf(outer).TextStroke)
	assert.Equal(t, red, styleOf(outer).Fill(), "currentcolor")
	assert.False(t, styleOf(outer).FillSet)
	assert.Equal(t, style.TextStroke{Width: 1, Color: blue}, styleOf(inner).TextStroke, "inherited as currentcolor")
	assert.Equal(t, color.Color{}, styleOf(inner).Fill())
	assert.Equal(t, style.TextStroke{Width: 3, Color: green}, styleOf(plain).TextStroke)
	assert.Equal(t, style.TextStroke{Width: 2.5, Color: green}, styleOf(own).TextStroke, "either order")
	assert.Equal(t, color.RGB8(255, 255, 0), styleOf(own).Fill())
	big := el(dom.P, "big")
	Compute(page(big), []*Sheet{sheet(t, `#big { -webkit-text-stroke: thick; font-size: 24px }`)})
	assert.InDelta(t, 5, styleOf(big).TextStroke.Width, 1e-5, "drawn 5 pixels wide whatever the font size")
	_, warns := Parse("t.css", "p { -webkit-text-stroke-width: -1px; -webkit-text-stroke: 1px 2px }")
	assert.Len(t, warns, 2)
}

func TestTheSelectionsStrokeAndFillAreItsOwnOrTheTexts(t *testing.T) {
	a, b, c := el(dom.Span, "a"), el(dom.Span, "b"), el(dom.Span, "c")
	Compute(page(with(el(dom.Div, "outer"), a, b, c)), []*Sheet{sheet(t, `#outer { -webkit-text-stroke: 2px #f00 }
#a::selection { color: #00f; -webkit-text-fill-color: transparent; -webkit-text-stroke-color: currentcolor }
#b::selection { -webkit-text-stroke-width: 4px }`)})
	assert.Equal(t, style.TextStroke{Width: 2, Color: color.RGB8(0, 0, 255)}, styleOf(a).SelectionStroke,
		"the text's width, the selection's colour")
	assert.Equal(t, style.TextStroke{Width: 4, Color: color.RGB8(255, 0, 0)}, styleOf(b).SelectionStroke)
	assert.False(t, styleOf(c).HasSelectionStroke)
}
