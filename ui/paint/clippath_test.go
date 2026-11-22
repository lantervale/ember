package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func clipped(c style.ClipPath) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.ClipPath = c
		s.Background = color.White
	}
}

func circleOf(r float32) style.ClipPath {
	return style.ClipPath{Shape: style.ShapeCircle,
		Radius: [2]style.ShapeRadius{{Side: style.RadiusLength, Length: style.Px(r)}},
		At:     [2]style.Length{style.Percent(50), style.Percent(50)}}
}

// disc is a 100×100 box at 50, 50 cut to a circle of radius 40 at its
// centre, holding a child that covers it whole and spills past it.
func disc() (*box.Box, *box.Box, *box.Box) {
	child := el(-20, -20, 140, 140, bg(color.Black))
	d := el(50, 50, 100, 100, clipped(circleOf(40)), child)
	root := el(0, 0, 400, 300, nil, d)
	return root, d, child
}

func alphaAt(img image.Image, x, y int) uint32 {
	_, _, _, a := img.At(x, y).RGBA()
	return a >> 8
}

func TestAClipPathCutsTheElementAndAllItHoldsToItsShape(t *testing.T) {
	root, _, _ := disc()
	r := newRecorder()
	Paint(r, tree(root))
	masks := r.of("push-mask")
	require.Len(t, masks, 1, "one mask for the element and its child")
	assert.Equal(t, geom.Rect{X: 60, Y: 60, W: 80, H: 80}, masks[0].Rect, "the circle's bounds")
	img := masks[0].Tex.Image
	require.NotNil(t, img)
	w := img.Bounds().Dx()
	assert.Equal(t, uint32(255), alphaAt(img, w/2, w/2), "opaque at the centre")
	assert.Equal(t, uint32(0), alphaAt(img, 1, 1), "clear in the corners")
	for _, o := range r.of("rect") {
		assert.Equal(t, geom.Rect{X: 60, Y: 60, W: 80, H: 80}, o.Clip, "%v is drawn under the mask", o)
	}
	assert.Len(t, r.of("rect"), 2)
	assert.Equal(t, len(r.of("push", "push-round", "push-mask")), len(r.of("pop")), "every push popped")
}

func TestAClipPathsMaskIsMadeOnceForFramesAlike(t *testing.T) {
	a, b := newRecorder(), newRecorder()
	root, _, _ := disc()
	Paint(a, tree(root), Scale(2))
	Paint(b, tree(root), Scale(2))
	require.Len(t, a.of("push-mask"), 1)
	assert.Same(t, a.of("push-mask")[0].Tex, b.of("push-mask")[0].Tex)
	assert.Equal(t, 160, a.of("push-mask")[0].Tex.Image.Bounds().Dx(), "drawn at the paint's scale")
}

func TestAPointerOutsideAClipPathMissesWhatItCuts(t *testing.T) {
	root, d, child := disc()
	tr := tree(root)
	assert.Same(t, child, HitTest(tr, geom.V2(100, 100)), "in the circle, the child")
	assert.Same(t, child, HitTest(tr, geom.V2(100, 62)), "near its top, still")
	assert.Same(t, root, HitTest(tr, geom.V2(55, 55)), "in the box's corner, outside the circle, what is under it")
	assert.Same(t, root, HitTest(tr, geom.V2(40, 100)), "where the child spills out, too")
	assert.NotSame(t, d, HitTest(tr, geom.V2(65, 65)))
}

func TestAnEmptyClipPathShowsAndHitsNothing(t *testing.T) {
	child := el(0, 0, 100, 100, bg(color.Black))
	d := el(50, 50, 100, 100, clipped(circleOf(0)), child)
	root := el(0, 0, 400, 300, nil, d)
	r := newRecorder()
	Paint(r, tree(root))
	assert.Empty(t, r.of("rect", "push-mask"))
	assert.Same(t, root, HitTest(tree(root), geom.V2(100, 100)))
}

func TestAClipPathsInkIsItsShapes(t *testing.T) {
	root, d, _ := disc()
	inked(tree(root))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 80, H: 80}, d.Ink)
}

func TestAPolygonsFillRuleCutsItsHole(t *testing.T) {
	pts := func(xy ...float32) [][2]style.Length {
		var out [][2]style.Length
		for i := 0; i < len(xy); i += 2 {
			out = append(out, [2]style.Length{style.Px(xy[i]), style.Px(xy[i+1])})
		}
		return out
	}
	// A pentagram: its middle is wound twice.
	star := style.ClipPath{Shape: style.ShapePolygon, Points: pts(50, 0, 79, 90, 2, 35, 98, 35, 21, 90)}
	for _, evenOdd := range []bool{false, true} {
		star.EvenOdd = evenOdd
		d := el(0, 0, 100, 100, clipped(star))
		root := el(0, 0, 200, 200, nil, d)
		want := d
		if evenOdd {
			want = root
		}
		assert.Same(t, want, HitTest(tree(root), geom.V2(50, 50)), "the middle, evenodd %v", evenOdd)
		assert.Same(t, d, HitTest(tree(root), geom.V2(50, 10)), "a point, evenodd %v", evenOdd)
	}
}

func TestAClipPathIsPlacedInItsReferenceBox(t *testing.T) {
	inset := style.ClipPath{Shape: style.ShapeInset}
	for which, want := range map[style.ShapeBox]geom.Rect{
		style.ShapeBoxUnset:   {X: 10, Y: 10, W: 100, H: 80},
		style.ShapeBorderBox:  {X: 10, Y: 10, W: 100, H: 80},
		style.ShapePaddingBox: {X: 14, Y: 14, W: 92, H: 72},
		style.ShapeContentBox: {X: 20, Y: 20, W: 80, H: 60},
		style.ShapeMarginBox:  {X: 5, Y: 5, W: 110, H: 90},
		style.ShapeFillBox:    {X: 20, Y: 20, W: 80, H: 60},
		style.ShapeStrokeBox:  {X: 10, Y: 10, W: 100, H: 80},
		style.ShapeViewBox:    {X: 10, Y: 10, W: 100, H: 80},
	} {
		inset.Box = which
		b := el(10, 10, 100, 80, clipped(inset))
		b.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
		b.Padding = box.Edges{Top: 6, Right: 6, Bottom: 6, Left: 6}
		b.Margin = box.Edges{Top: 5, Right: 5, Bottom: 5, Left: 5}
		r := newRecorder()
		Paint(r, tree(el(0, 0, 400, 300, nil, b)))
		require.Len(t, r.of("push-mask"), 1)
		assert.Equal(t, want, r.of("push-mask")[0].Rect, "box %d", which)
	}
}

func TestAClipPathStartsAStackingContext(t *testing.T) {
	s := style.Initial()
	assert.False(t, startsContext(&s))
	s.ClipPath = style.ClipPath{Box: style.ShapeBorderBox}
	assert.True(t, startsContext(&s), "a box alone")
	s.ClipPath = circleOf(4)
	assert.True(t, startsContext(&s))
}

func TestInsetsShortenTogetherWhereTheyMeet(t *testing.T) {
	c := style.ClipPath{Shape: style.ShapeInset,
		Inset: style.Sides[style.Length]{Top: style.Px(10), Right: style.Px(90), Bottom: style.Px(10), Left: style.Px(30)}}
	b := el(0, 0, 100, 50, clipped(c))
	s, ok := clipShapeOf(b, geom.Vec2{}, b.Style)
	require.True(t, ok)
	assert.Equal(t, geom.Rect{}, s.bounds, "30 and 90 over a width of 100 meet, and leave nothing")
	c.Inset.Right = style.Px(20)
	b = el(0, 0, 100, 50, clipped(c))
	s, _ = clipShapeOf(b, geom.Vec2{}, b.Style)
	assert.Equal(t, geom.Rect{X: 30, Y: 10, W: 50, H: 30}, s.bounds)
}
