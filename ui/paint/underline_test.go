package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// underlined is ahem whose post table puts its underline 0.15 em below
// the baseline, 0.05 em thick, and whose em box is 0.8 em above it.
type underlined struct{ ahem }

func (underlined) Metrics(f style.Font) box.Metrics {
	return box.Metrics{Ascent: 0.8 * f.Size, Descent: 0.2 * f.Size, Underline: 0.15 * f.Size,
		UnderlineThickness: 0.05 * f.Size, Underlined: true, EmAscent: 0.8}
}

// underlineRect is the rect the underline d draws under "hi" on a
// baseline at 15, at 16 points.
func underlineRect(t *testing.T, d style.Decoration, opts ...Option) (float32, float32) {
	t.Helper()
	p := el(0, 0, 100, 20, func(s *style.Computed) { s.Color = white; s.Decorations = []style.Decoration{d} })
	p.Append(text(0, 0, "hi", p.Style))
	r := newRecorder()
	Paint(r, tree(p), append([]Option{Fonts(underlined{})}, opts...)...)
	ops := r.of("rect")
	require.Len(t, ops, 1)
	return ops[0].Rect.Y, ops[0].Rect.H
}

func TestAnUnderlineOffsetTakesThePlaceOfItsGap(t *testing.T) {
	d := decoration(style.Underline, red)
	for _, c := range []struct {
		offset style.Length
		y      float32
		why    string
	}{
		{style.Length{}, 16, "auto: half the thickness below the baseline"},
		{style.Px(0), 15, "0: on the baseline, no gap"},
		{style.Px(3), 18, "a length below it"},
		{style.Px(2.6), 18, "in whole pixels"},
		{style.Length{Kind: style.Fixed, Percent: 25, IsPercent: true}, 19, "a percentage of the decorating font's size"},
		{style.Px(-2), 13, "above it"},
	} {
		d.Offset = c.offset
		y, _ := underlineRect(t, d)
		assert.InDelta(t, c.y, y, 1e-4, c.why)
	}
}

func TestAnUnderlineFromTheFontLiesWhereItsPostTableSays(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Position = style.UnderFromFont
	y, h := underlineRect(t, d)
	assert.InDelta(t, 15+2, y, 1e-4, "2.4 below the baseline in whole pixels")
	assert.InDelta(t, 1, h, 1e-4, "auto thickness, a tenth of the size")
	d.Offset = style.Px(3)
	y, _ = underlineRect(t, d)
	assert.InDelta(t, 15+5, y, 1e-4, "the offset added before rounding")
	y, _ = underlineRect(t, d, Scale(2))
	assert.InDelta(t, 15+5.5, y, 1e-4, "10.8 device pixels rounded")
}

func TestAnUnderlineUnderLiesAPixelBelowTheEmBox(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Position = style.UnderUnder | style.UnderLeft
	y, _ := underlineRect(t, d)
	assert.InDelta(t, 15+4, y, 1e-4, "the 3.2 below the baseline floored, and a pixel")
	d.Offset = style.Px(1.9)
	y, _ = underlineRect(t, d)
	assert.InDelta(t, 15+6, y, 1e-4)
}

func TestAThicknessFromTheFontIsItsUnderlines(t *testing.T) {
	d := decoration(style.Underline, red)
	d.Thickness = style.Length{Kind: style.None}
	d.Font.Size = 60
	y, h := underlineRect(t, d)
	assert.InDelta(t, 3, h, 1e-4, "0.05 em")
	assert.InDelta(t, 15+2, y, 1e-4, "a gap of half of it, whole")
	y, h = underlineRect(t, d, Scale(2))
	assert.InDelta(t, 3, h, 1e-4)
	assert.InDelta(t, 15+1.5, y, 1e-4)
}

func TestVerticalLinesAreUnderlinedOffTheirCentralBaseline(t *testing.T) {
	p := &painter{fonts: underlined{}}
	st := style.Initial()
	st.Mode = style.VerticalRL
	d := decoration(style.Underline, red)
	dy, step := p.underline(&st, &d, 2)
	assert.InDelta(t, -4.8+9, dy, 1e-4, "half the em and a pixel left of the central baseline, 4.8 above the baseline")
	assert.Positive(t, step)
	d.Position = style.UnderRight
	dy, step = p.underline(&st, &d, 2)
	assert.InDelta(t, -4.8-9-2, dy, 1e-4, "as far right of it")
	assert.Negative(t, step, "a double line's second further right")
	d.Position, d.RightInVertical = 0, true
	right, _ := p.underline(&st, &d, 2)
	assert.InDelta(t, dy, right, 1e-4, "Japanese and Korean underline on the right")
	d.Position = style.UnderLeft
	left, _ := p.underline(&st, &d, 2)
	assert.InDelta(t, -4.8+9, left, 1e-4, "unless left")
	st.Orientation = style.OrientSideways
	d.Position = style.UnderRight
	sideways, _ := p.underline(&st, &d, 2)
	assert.InDelta(t, 1, sideways, 1e-4, "sideways glyphs sit on their alphabetic baseline, left and right unread")
}
