package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
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

func TestFontSmoothingSubpixelAntialiasedIsToldApartFromAuto(t *testing.T) {
	inner, plain := el(dom.Span, "inner"), el(dom.Span, "plain")
	outer := with(el(dom.Div, "outer"), inner, plain)
	Compute(page(outer), []*Sheet{sheet(t, `#outer { -webkit-font-smoothing: subpixel-antialiased }
#plain { -webkit-font-smoothing: auto }`)})
	assert.True(t, styleOf(outer).SubpixelText)
	assert.True(t, styleOf(inner).SubpixelText, "inherited")
	assert.False(t, styleOf(plain).SubpixelText)
	assert.False(t, styleOf(outer).ExactText || styleOf(outer).PixelText, "drawn as auto")
}

func TestTextRenderingIsReadAndInherited(t *testing.T) {
	inner, plain := el(dom.Span, "inner"), el(dom.Span, "plain")
	outer := with(el(dom.Div, "outer"), inner, plain)
	Compute(page(outer), []*Sheet{sheet(t, `#outer { text-rendering: optimizeLegibility }
#plain { text-rendering: geometricprecision }`)})
	assert.Equal(t, style.OptimizeLegibility, styleOf(outer).TextRendering)
	assert.Equal(t, style.OptimizeLegibility, styleOf(inner).TextRendering, "inherited")
	assert.Equal(t, style.GeometricPrecision, styleOf(plain).TextRendering)
	_, warns := Parse("t.css", "p { text-rendering: crisp }")
	assert.NotEmpty(t, warns, "only its keywords")
}
