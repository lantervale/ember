package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

func doubled(w float32, c color.Color, edit func(s *style.Computed)) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderDouble)
		s.BorderColor = style.All(c)
		s.Border = style.All(w)
		if edit != nil {
			edit(s)
		}
	}
}

func TestADoubleBorderIsTwoLinesOfAThirdOfItsWidthInDevicePixels(t *testing.T) {
	ink := color.RGB8(42, 37, 48)
	for _, c := range []struct {
		width, scale     float32
		outer, gap, line float32
	}{
		{6, 1, 2, 2, 2},
		{5, 1, 2, 1, 2},
		{10, 1, 3, 4, 3},
		{10, 2, 3.5, 3, 3.5},
		{2, 2, 0.5, 1, 0.5},
	} {
		r := newRecorder()
		Paint(r, tree(el(10, 10, 60, 40, doubled(c.width, ink, nil))), Scale(c.scale))
		rects := r.of("rect")
		require.Len(t, rects, 8, "%gpt at %gx: four sides, two lines each", c.width, c.scale)
		assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 60, H: c.outer}, rects[0].Rect, "%gpt: the outer top line", c.width)
		in := c.outer + c.gap
		assert.Equal(t, geom.Rect{X: 10 + in, Y: 10 + in, W: 60 - 2*in, H: c.line}, rects[4].Rect,
			"%gpt: the inner top line, past the gap", c.width)
	}
}

func TestADoubleBorderTooThinForAGapIsSolid(t *testing.T) {
	ink := color.RGB8(42, 37, 48)
	r := newRecorder()
	Paint(r, tree(el(10, 10, 60, 40, doubled(1, ink, nil))), Scale(2))
	rects := r.of("rect")
	require.Len(t, rects, 4)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 60, H: 1}, rects[0].Rect)
}

func TestARoundedDoubleBorderIsTwoRings(t *testing.T) {
	green := color.RGB8(42, 157, 74)
	r := newRecorder()
	Paint(r, tree(el(10, 10, 60, 40, doubled(9, green, rounded(14)))), Scale(1))
	rings := r.of("border")
	require.Len(t, rings, 2)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 60, H: 40}, rings[0].Rect)
	assert.Equal(t, float32(14), rings[0].Radius)
	assert.Equal(t, float32(3), rings[0].Width)
	assert.Equal(t, geom.Rect{X: 16, Y: 16, W: 48, H: 28}, rings[1].Rect)
	assert.Equal(t, float32(8), rings[1].Radius)
	assert.Equal(t, float32(3), rings[1].Width)
}

func TestADoubleSideBesideASolidOneLosesItsGapUpToTheirMitre(t *testing.T) {
	blue := color.RGB8(52, 33, 230)
	r := newRecorder()
	Paint(r, tree(el(10, 10, 60, 40, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderStyle.Top = style.BorderDouble
		s.BorderColor = style.All(blue)
		s.Border = style.All[float32](9)
	})), Scale(1))
	sides := r.sides()
	require.Len(t, sides, 5, "the top's two lines and the three solid sides, mitered")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 60, H: 3}, sides[0].Rect)
	assert.Equal(t, geom.Rect{X: 16, Y: 16, W: 48, H: 3}, sides[1].Rect)
}

func TestADoubleOutlineIsTwoRings(t *testing.T) {
	red := color.RGB8(224, 60, 49)
	r := newRecorder()
	Paint(r, tree(el(10, 10, 60, 40, func(s *style.Computed) {
		s.OutlineStyle, s.OutlineWidth, s.OutlineColor = style.BorderDouble, 6, red
	})), Scale(1))
	rings := r.of("border")
	require.Len(t, rings, 2)
	assert.Equal(t, geom.Rect{X: 4, Y: 4, W: 72, H: 52}, rings[0].Rect)
	assert.Equal(t, float32(2), rings[0].Width)
	assert.Equal(t, geom.Rect{X: 8, Y: 8, W: 64, H: 44}, rings[1].Rect)
	assert.Equal(t, float32(2), rings[1].Width)
}
