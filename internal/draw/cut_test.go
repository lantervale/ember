package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

func near2(a, b geom.Vec2) bool { return a.Sub(b).Len() < 1e-4 }

func TestAMaskClipCutsItsBatchesUnderTheTransformItWasPushedIn(t *testing.T) {
	mask := &render.Texture{Image: image.NewAlpha(image.Rect(0, 0, 8, 8))}
	var c List
	c.SetTransform(geom.Transform2DFrom(geom.V2(100, 50), 0.5, 0, geom.V2(2, 2)))
	r := geom.Rect{X: 10, Y: 20, W: 40, H: 30}
	c.PushClipMask(r, mask)
	c.SetTransform(geom.Identity2D())
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	b := c.Batches[0]
	if b.Cut.Mask != mask || !b.Clipped {
		t.Fatalf("the batch is cut by the mask and clipped to its bounds: %+v", b)
	}
	xf := geom.Transform2DFrom(geom.V2(100, 50), 0.5, 0, geom.V2(2, 2))
	for local, uv := range map[geom.Vec2]geom.Vec2{
		{X: 10, Y: 20}: {}, {X: 50, Y: 50}: {X: 1, Y: 1}, {X: 30, Y: 35}: {X: 0.5, Y: 0.5},
	} {
		if got := b.Cut.ToMask.Xform(xf.Xform(local)); !near2(got, uv) {
			t.Errorf("%v of r, drawn at %v, is %v of the mask, want %v", local, xf.Xform(local), got, uv)
		}
	}
}

func TestAMaskClipNestsAsARoundedOneDoes(t *testing.T) {
	a := &render.Texture{Image: image.NewAlpha(image.Rect(0, 0, 4, 4))}
	b := &render.Texture{Image: image.NewAlpha(image.Rect(0, 0, 4, 4))}
	var c List
	c.PushClipMask(geom.Rect{W: 100, H: 100}, a)
	c.PushClip(geom.Rect{X: 10, Y: 10, W: 20, H: 20})
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if got := c.Batches[0].Cut.Mask; got != a {
		t.Fatalf("a square clip inside keeps the mask: %v", got)
	}
	c.PushClipRounded(geom.Rect{X: 10, Y: 10, W: 10, H: 10}, [4]float32{2, 2, 2, 2})
	c.Rect(geom.Rect{W: 5, H: 5}, color.Black)
	if last := c.Batches[len(c.Batches)-1]; last.Cut.Mask != a || last.Round.IsZero() {
		t.Fatalf("a rounded clip inside cuts with it: %+v", last)
	}
	c.PushClipMask(geom.Rect{X: 12, Y: 12, W: 4, H: 4}, b)
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if last := c.Batches[len(c.Batches)-1]; last.Cut.Mask != b || last.Round.IsZero() {
		t.Fatalf("an inner mask replaces the outer one, the rounding stays: %+v", last)
	}
	for range 3 {
		c.PopClip()
	}
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if last := c.Batches[len(c.Batches)-1]; last.Cut.Mask != a {
		t.Fatalf("popping brings the outer mask back: %+v", last)
	}
	c.PopClip()
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if last := c.Batches[len(c.Batches)-1]; !last.Cut.IsZero() || last.Clipped {
		t.Fatalf("and the last pop ends it: %+v", last)
	}
}

func TestAShiftMovesAMaskWithWhatItCuts(t *testing.T) {
	mask := &render.Texture{Image: image.NewAlpha(image.Rect(0, 0, 4, 4))}
	var c List
	c.PushClipMask(geom.Rect{X: 10, Y: 10, W: 20, H: 20}, mask)
	c.Rect(geom.Rect{X: 10, Y: 10, W: 20, H: 20}, color.White)
	c.Shift(geom.V2(5, -3))
	if got := c.Batches[0].Cut.ToMask.Xform(geom.V2(15, 7)); !near2(got, geom.Vec2{}) {
		t.Errorf("the mask's corner moved with the drawing: %v", got)
	}
}
