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

func clippedAt(at style.Position, r style.ClipRect) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.Position = at
		s.Clip = r
		s.Background = color.White
	}
}

// clipBox is a 100×100 box at 50, 50 positioned at and clipped by r,
// holding a child that covers it whole and spills past it, and a fixed
// box inside it.
func clipBox(at style.Position, r style.ClipRect) (*box.Box, *box.Box, *box.Box) {
	child := el(-20, -20, 140, 140, bg(color.Black))
	fixed := el(40, 40, 50, 50, func(s *style.Computed) {
		s.Position = style.FixedPosition
		s.Background = color.Black
	})
	c := el(50, 50, 100, 100, clippedAt(at, r), child, fixed)
	return el(0, 0, 400, 300, nil, c), c, child
}

func TestAClipCutsAPositionedBoxAndAllItHoldsToItsRect(t *testing.T) {
	r := style.ClipRect{Set: true, Edges: style.Sides[float32]{Top: 10, Right: 60, Bottom: 70, Left: 20}}
	root, _, _ := clipBox(style.Absolute, r)
	rec := newRecorder()
	Paint(rec, tree(root))
	rects := rec.of("rect")
	require.Len(t, rects, 3)
	for _, o := range rects {
		assert.Equal(t, geom.Rect{X: 70, Y: 60, W: 40, H: 60}, o.Clip, "%v is cut, the fixed box too", o)
	}
	assert.Equal(t, len(rec.of("push", "push-round", "push-mask")), len(rec.of("pop")), "every push popped")
}

func TestAClipsAutoEdgesAreTheBorderBoxs(t *testing.T) {
	r := style.ClipRect{Set: true, Edges: style.Sides[float32]{Top: -50, Left: -50},
		Auto: style.Sides[bool]{Right: true, Bottom: true}}
	root, _, _ := clipBox(style.Absolute, r)
	rec := newRecorder()
	Paint(rec, tree(root))
	for _, o := range rec.of("rect") {
		assert.Equal(t, geom.Rect{X: 0, Y: 0, W: 150, H: 150}, o.Clip, "%v", o)
	}
}

func TestAnEmptyClipShowsNothing(t *testing.T) {
	r := style.ClipRect{Set: true, Edges: style.Sides[float32]{Top: 96, Right: 96, Bottom: 96, Left: 96}}
	root, c, _ := clipBox(style.Absolute, r)
	rec := newRecorder()
	Paint(rec, tree(root))
	for _, o := range rec.of("rect") {
		assert.Zero(t, o.Clip.W*o.Clip.H, "%v", o)
	}
	assert.NotSame(t, c, HitTest(tree(root), geom.V2(100, 100)))
}

func TestAClipCutsOnlyPositionedBoxes(t *testing.T) {
	r := style.ClipRect{Set: true, Edges: style.Sides[float32]{Right: 1, Bottom: 1}}
	root, _, child := clipBox(style.Static, r)
	rec := newRecorder()
	Paint(rec, tree(root))
	for _, o := range rec.of("rect") {
		assert.NotEqual(t, geom.Rect{X: 50, Y: 50, W: 1, H: 1}, o.Clip, "%v", o)
	}
	assert.Same(t, child, HitTest(tree(root), geom.V2(60, 60)))
}

func TestAPointerOutsideAClipMissesWhatItCuts(t *testing.T) {
	r := style.ClipRect{Set: true, Edges: style.Sides[float32]{Top: 10, Right: 60, Bottom: 70, Left: 20}}
	root, _, child := clipBox(style.FixedPosition, r)
	tr := tree(root)
	assert.Same(t, child, HitTest(tr, geom.V2(80, 80)), "in the rect, the child")
	assert.Same(t, root, HitTest(tr, geom.V2(130, 130)), "in the box, outside the rect, what is under it")
	assert.Same(t, root, HitTest(tr, geom.V2(40, 40)), "where the child spills out, too")
}
