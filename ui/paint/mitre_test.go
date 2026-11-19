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
	sides := r.of("triangles")
	require.Len(t, sides, 1, "a box of no size with one coloured side is a triangle")
	assert.Equal(t, blue, sides[0].Color)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 5}, sides[0].Rect)
	assert.Empty(t, r.of("rect"), "no side is a rectangle over its neighbours' corners")

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
