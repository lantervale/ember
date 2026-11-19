package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decoration is a decoration of lines in c, drawn by a box of the initial
// font: 16 points, whose ahem ascent is 12.8.
func decoration(lines style.DecorationLine, c color.Color) style.Decoration {
	return style.Decoration{TextDecoration: style.TextDecoration{Line: lines, Color: c}, Font: style.Initial().Font}
}

// decoratedText paints "hi", 16 wide on a baseline at 15, under ds.
func decoratedText(opts []Option, ds ...style.Decoration) *recorder {
	p := el(0, 0, 100, 20, func(s *style.Computed) { s.Color = white; s.Decorations = ds })
	p.Append(text(0, 0, "hi", p.Style))
	r := newRecorder()
	Paint(r, tree(p), append([]Option{Fonts(ahem{})}, opts...)...)
	return r
}

func TestAnUnderlineLiesUnderTheBaselineInItsOwnColour(t *testing.T) {
	r := decoratedText(nil, decoration(style.Underline, red))
	ops := r.of("rect", "text")
	require.Len(t, ops, 2)
	assert.Equal(t, "rect", ops[0].Kind, "under the text")
	assert.Equal(t, geom.Rect{X: 0, Y: 16, W: 16, H: 1}, ops[0].Rect,
		"a tenth of the font size thick in whole pixels, a gap of a pixel below the baseline")
	assert.Equal(t, red, ops[0].Color, "the decorating box's colour, not the text's")
}

func TestAnOverlineLiesOnTheTextsAscent(t *testing.T) {
	ops := decoratedText(nil, decoration(style.Overline, red)).of("rect", "text")
	require.Len(t, ops, 2)
	assert.Equal(t, "rect", ops[0].Kind, "under the text")
	assert.InDelta(t, 15-12.8-1, ops[0].Rect.Y, 1e-4)
	assert.InDelta(t, 1, ops[0].Rect.H, 1e-6)
}

func TestALineThroughIsDrawnOverTheText(t *testing.T) {
	ops := decoratedText(nil, decoration(style.LineThrough, red)).of("rect", "text")
	require.Len(t, ops, 2)
	assert.Equal(t, "text", ops[0].Kind)
	assert.InDelta(t, 15-12.8+2*12.8/3-0.8, ops[1].Rect.Y, 1e-4,
		"centred two thirds of the ascent down from its top")
	assert.Equal(t, red, ops[1].Color)
}

func TestTheLinesOfOneDecorationFrameTheText(t *testing.T) {
	r := decoratedText(nil, decoration(style.Underline|style.Overline|style.LineThrough, red))
	var kinds []string
	for _, o := range r.of("rect", "text") {
		kinds = append(kinds, o.Kind)
	}
	assert.Equal(t, []string{"rect", "rect", "text", "rect"}, kinds)
}

func TestAnInnerBoxsDecorationIsDrawnOverAnOuterOnes(t *testing.T) {
	ops := decoratedText(nil, decoration(style.Underline, red), decoration(style.Underline, blue)).of("rect")
	require.Len(t, ops, 2)
	assert.Equal(t, []color.Color{red, blue}, []color.Color{ops[0].Color, ops[1].Color})
}

func TestADecorationTakesItsThicknessAndPlaceFromTheDecoratingFont(t *testing.T) {
	d := decoration(style.Underline|style.Overline|style.LineThrough, red)
	d.Font.Size = 32
	ops := decoratedText(nil, d).of("rect")
	require.Len(t, ops, 3)
	assert.Equal(t, geom.Rect{X: 0, Y: 17, W: 16, H: 3}, ops[0].Rect, "3.2 thick in whole pixels, a gap of 2")
	assert.InDelta(t, 15-12.8-3, ops[1].Rect.Y, 1e-4, "over the text's own ascent")
	assert.InDelta(t, 15-12.8+2*25.6/3-1.6, ops[2].Rect.Y, 1e-4,
		"two thirds of the decorating font's ascent below the text's top, as Chrome")
}

func TestAGivenThicknessIsALengthOrAShareOfTheFontSize(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Thickness = style.Px(3)
	assert.Equal(t, geom.Rect{X: 0, Y: 17, W: 16, H: 3}, decoratedText(nil, d).of("rect")[0].Rect)
	d.Thickness = style.Length{Kind: style.Fixed, Percent: 25, IsPercent: true}
	assert.InDelta(t, 4, decoratedText(nil, d).of("rect")[0].Rect.H, 1e-6)
	d.Thickness = style.Px(0)
	assert.InDelta(t, 1, decoratedText(nil, d).of("rect")[0].Rect.H, 1e-6, "never under a pixel")
}

func TestADecorationIsWholeDevicePixels(t *testing.T) {
	ops := decoratedText([]Option{Scale(2)}, decoration(style.Underline, red)).of("rect")
	require.Len(t, ops, 1)
	assert.Equal(t, geom.Rect{X: 0, Y: 16, W: 16, H: 1.5}, ops[0].Rect, "3 of the 3.2 pixels, 2 below the baseline")
}

func TestADoubleLineIsTwoAThicknessAndAPixelApart(t *testing.T) {
	double := func(lines style.DecorationLine) []op {
		d := decoration(lines, red)
		d.Style = style.DecorationDouble
		ops := decoratedText(nil, d).of("rect")
		require.Len(t, ops, 2)
		return ops
	}
	under := double(style.Underline)
	assert.InDelta(t, 16, under[0].Rect.Y, 1e-4)
	assert.InDelta(t, 16+2.6, under[1].Rect.Y, 1e-4, "the second below")
	over := double(style.Overline)
	assert.InDelta(t, over[0].Rect.Y-2.6, over[1].Rect.Y, 1e-4, "the second above")
	through := double(style.LineThrough)
	assert.InDelta(t, through[0].Rect.Y+2, through[1].Rect.Y, 1e-4, "the second below, whole pixels")
}

func TestADottedLineIsSquaresThenRoundDotsFlushAtBothEnds(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Style, d.Thickness = style.DecorationDotted, style.Px(3)
	ops := decoratedText(nil, d).of("rect", "circle")
	require.Len(t, ops, 3)
	for i, x := range []float32{0, 6.5, 13} {
		assert.Equal(t, geom.Rect{X: x, Y: 17, W: 3, H: 3}, ops[i].Rect)
	}
	d.Thickness = style.Px(5)
	ops = decoratedText(nil, d).of("rect", "circle")
	require.Len(t, ops, 2)
	assert.Equal(t, "circle", ops[0].Kind, "round over 3 pixels thick")
	assert.Equal(t, geom.V2(2.5, 20.5), ops[0].At)
	assert.Equal(t, geom.V2(13.5, 20.5), ops[1].At)
	assert.InDelta(t, 2.5, ops[0].Radius, 1e-6)
}

func TestADashedLineIsDashesTwoThicknessesLongOrThreeWhenThin(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Style, d.Thickness = style.DecorationDashed, style.Px(3)
	ops := decoratedText(nil, d).of("rect")
	require.Len(t, ops, 2)
	assert.Equal(t, geom.Rect{X: 0, Y: 17, W: 6, H: 3}, ops[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: 17, W: 6, H: 3}, ops[1].Rect, "flush with the end")
	d.Thickness = style.Px(1)
	ops = decoratedText(nil, d).of("rect")
	require.Len(t, ops, 4)
	assert.InDelta(t, 3, ops[0].Rect.W, 1e-6)
	assert.InDelta(t, 13, ops[3].Rect.X, 1e-4)
}

func TestADecorationShowsUnderTransparentTextAndFadesWithOpacity(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Color = color.RGBA(0, 0, 0, 0)
		s.Opacity = 0.5
		s.Decorations = []style.Decoration{decoration(style.Underline|style.LineThrough, red)}
	})
	p.Append(text(0, 0, "hi", p.Style))
	r := newRecorder()
	Paint(r, tree(p), Fonts(ahem{}))
	assert.Empty(t, r.of("text"))
	ops := r.of("rect")
	require.Len(t, ops, 2)
	assert.InDelta(t, 0.5, ops[0].Color.A, 1e-6)
}

func TestATransparentDecorationAndAnEmptyFragmentDrawNoLine(t *testing.T) {
	assert.Empty(t, decoratedText(nil, decoration(style.Underline, color.RGBA(1, 0, 0, 0))).of("rect"))
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Decorations = []style.Decoration{decoration(style.Underline, red)}
	})
	run := text(0, 0, "", p.Style)
	p.Append(run)
	r := newRecorder()
	Paint(r, tree(p), Fonts(ahem{}))
	assert.Empty(t, r.of("rect"))
}

func TestWithoutFontsAnOverlineLiesOnTheFragmentsTop(t *testing.T) {
	p := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Decorations = []style.Decoration{decoration(style.Overline, red)}
	})
	p.Append(text(0, 0, "hi", p.Style))
	r := newRecorder()
	Paint(r, tree(p))
	assert.InDelta(t, -1, r.of("rect")[0].Rect.Y, 1e-6)
}
