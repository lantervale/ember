package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
)

func TestFontSmoothingNoneMakesPixelTextInheritedAndAutoUndoesIt(t *testing.T) {
	inner, plain := el(dom.Span, "inner"), el(dom.Span, "plain")
	outer := with(el(dom.Div, "outer"), inner, plain)
	Compute(page(outer),
		[]*Sheet{sheet(t, "#outer { -webkit-font-smoothing: none } #plain { -webkit-font-smoothing: auto }")})
	assert.True(t, styleOf(outer).PixelText)
	assert.True(t, styleOf(inner).PixelText, "inherited")
	assert.False(t, styleOf(plain).PixelText, "auto smooths again")
	_, warns := Parse("t.css", "p { -webkit-font-smoothing: blurry }")
	assert.NotEmpty(t, warns, "only its keywords")
}

func TestFontSmoothingAntialiasedMakesExactTextInherited(t *testing.T) {
	inner, plain, pixel, sub := el(dom.Span, "inner"), el(dom.Span, "plain"), el(dom.Span, "pixel"), el(dom.Span, "sub")
	outer := with(el(dom.Div, "outer"), inner, plain, pixel, sub)
	Compute(page(outer), []*Sheet{sheet(t, `#outer { -webkit-font-smoothing: antialiased }
#plain { -webkit-font-smoothing: auto } #pixel { -webkit-font-smoothing: none }
#sub { -webkit-font-smoothing: subpixel-antialiased }`)})
	assert.True(t, styleOf(outer).ExactText)
	assert.False(t, styleOf(outer).PixelText)
	assert.True(t, styleOf(inner).ExactText, "inherited")
	assert.False(t, styleOf(plain).ExactText, "auto darkens again")
	assert.False(t, styleOf(sub).ExactText, "subpixel-antialiased is the default look")
	assert.True(t, styleOf(pixel).PixelText)
	assert.False(t, styleOf(pixel).ExactText, "unsmoothed is not both")
}
