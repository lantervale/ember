package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestSolidSidesOfOtherColoursAreMitered(t *testing.T) {
	blue := color.RGB8(52, 33, 230)
	arrow := el(10, 10, 10, 10, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(color.Color{})
		s.BorderColor.Top = blue
		s.Border = style.All[float32](5)
	})
	arrow.Border = box.Edges{Top: 5, Right: 5, Bottom: 5, Left: 5}
	r := newRecorder()
	Paint(r, tree(arrow))
	sides := r.sides()
	require.Len(t, sides, 1, "a box of no size with one coloured side is a triangle")
	assert.Equal(t, blue, sides[0].Color)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 5}, sides[0].Rect)
	assert.Empty(t, r.of("rect"), "no side is a rectangle over its neighbours' corners")
	assert.Len(t, r.of("gradient"), 4, "both its edges smoothed, each side of the diagonal")

	frame := el(10, 10, 40, 30, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(blue)
		s.Border = style.All[float32](4)
	})
	frame.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
	r = newRecorder()
	Paint(r, tree(frame))
	assert.Empty(t, r.of("triangles"), "sides of one colour stay rectangles")
	assert.Len(t, r.of("rect"), 4)
}

func TestARoundedBorderOfOtherColoursIsARingPerColourCutAtItsMitres(t *testing.T) {
	box := el(10, 10, 60, 40, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.Sides[color.Color]{Top: red, Right: blue, Bottom: blue, Left: red}
		s.Border = style.All[float32](8)
		rounded(12)(s)
	})
	r := newRecorder()
	Paint(r, tree(box), Scale(1))
	rings := r.of("border")
	require.Len(t, rings, 2, "the top and left's colour round the box, the right and bottom's over it")
	assert.Equal(t, red, rings[0].Color)
	assert.Equal(t, blue, rings[1].Color)
	assert.Equal(t, float32(12), rings[1].Radius)
	masks := r.of("push-mask")
	require.Len(t, masks, 1, "the second ring cut to its sides")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 60, H: 40}, masks[0].Rect)
	assert.Empty(t, r.of("triangles"), "no square sides")

	r = newRecorder()
	bare := el(10, 10, 60, 40, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(color.Color{})
		s.BorderColor.Top = blue
		s.Border = style.All[float32](8)
		rounded(12)(s)
	})
	Paint(r, tree(bare), Scale(1))
	require.Len(t, r.of("border"), 1, "only the side that shows")
	assert.Len(t, r.of("push-mask"), 1, "cut to the top, the rest clear")
}

func TestSidesOfOtherColoursFadeToTheirBlendOnTheDiagonal(t *testing.T) {
	b := el(0, 0, 40, 30, func(s *style.Computed) {
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.Sides[color.Color]{Top: red, Right: red, Bottom: red, Left: blue}
		s.Border = style.All[float32](10)
	})
	r := newRecorder()
	Paint(r, tree(b), Scale(2))
	mid := halfway(red, blue)
	fades := r.of("gradient")
	require.Len(t, fades, 4, "the top and the bottom fade into the left, and the left into both")
	diagonal := 0
	for _, f := range fades {
		for i, q := range f.Points {
			if near(q.X, q.Y) || near(q.X, 30-q.Y) {
				assert.Equal(t, mid, f.Colors[i], "the diagonal at %v in the colours' blend", q)
				diagonal++
			}
		}
	}
	assert.Positive(t, diagonal)
	s := mid.SRGB()
	assert.InDelta(t, 0.5, s[0], 0.01, "halfway in encoded sRGB, as Chrome blends")
	assert.InDelta(t, 0.5, s[2], 0.01)
}

func near(a, b float32) bool { return a-b < 1e-4 && b-a < 1e-4 }
