package paint //nolint:testpackage // white-box: paints parts of a box straight through the painter

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestATurnedBordersEdgesFadeOverAPixelAndItsMitresStayHard(t *testing.T) {
	st := style.Initial()
	st.BorderStyle = style.All(style.BorderSolid)
	st.BorderColor = style.All(color.RGB8(51, 68, 85))
	box4 := box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
	r := geom.Rect{X: 10, Y: 10, W: 60, H: 40}

	rec := newRecorder()
	(&painter{s: rec, scale: 1}).border(r, box4, radii{}, &st, 1)
	assert.Empty(t, rec.of("gradient"), "upright, its sides are drawn whole")

	rec = newRecorder()
	rec.xf = geom.Transform2DFrom(geom.Vec2{}, 0.5, 0, geom.V2(1, 1))
	(&painter{s: rec, scale: 2}).border(r, box4, radii{}, &st, 1)
	require.NotEmpty(t, rec.of("gradient"))
	var faded int
	for _, o := range rec.of("gradient") {
		for i, q := range o.Points {
			outside := max(r.X-q.X, q.X-r.X-r.W, r.Y-q.Y, q.Y-r.Y-r.H)
			inside := min(q.X-r.X-4, r.X+r.W-4-q.X, q.Y-r.Y-4, r.Y+r.H-4-q.Y)
			assert.LessOrEqual(t, outside, float32(0.25)+1e-4, "a quarter point, half a device pixel, past the box at most")
			assert.LessOrEqual(t, inside, float32(0.25)+1e-4, "and into its padding box")
			if o.Alpha[i] == 0 {
				faded++
				assert.True(t, abs(outside-0.25) < 1e-3 || abs(inside-0.25) < 1e-3,
					"faded out on the line half a pixel past an edge: %v", q)
			}
		}
	}
	assert.Equal(t, 4*2*2, faded, "each side's outer and inner edges fade, two corners each")
}

func TestASkewedBordersLevelEdgesStayAsDrawnSquare(t *testing.T) {
	st := style.Initial()
	st.BorderStyle = style.All(style.BorderSolid)
	st.BorderColor = style.All(color.RGB8(51, 68, 85))
	rec := newRecorder()
	rec.xf = geom.Transform2DFrom(geom.Vec2{}, 0, -0.35, geom.V2(1, 1))
	(&painter{s: rec, scale: 1}).border(geom.Rect{W: 60, H: 40}, box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4},
		radii{}, &st, 1)
	for _, o := range rec.of("gradient") {
		for i, q := range o.Points {
			if o.Alpha[i] == 0 {
				assert.True(t, q.X < 0 || q.X > 60 || q.X > 4 && q.X < 56, "only the slanted sides fade: %v", q)
			}
		}
	}
}

func TestATurnedNinePatchReachesPastItsBoxForItsClip(t *testing.T) {
	tex := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	eight := style.Sides[float32]{Top: 8, Right: 8, Bottom: 8, Left: 8}
	box := geom.Rect{X: 10, Y: 10, W: 100, H: 60}
	rec := newRecorder()
	rec.xf = geom.Transform2DFrom(geom.Vec2{}, 0.5, 0, geom.V2(1, 1))
	p := &painter{s: rec, scale: 1}
	p.ninePatch(box, tex, eight, eight, [2]style.ImageRepeat{}, color.White, true)
	require.Equal(t, "push", rec.ops[0].Kind)
	assert.Equal(t, box, rec.ops[0].Rect, "cut by its box, turned")
	lo, hi := geom.V2(1e9, 1e9), geom.V2(-1e9, -1e9)
	for _, o := range rec.of("image") {
		lo = geom.V2(min(lo.X, o.Rect.X), min(lo.Y, o.Rect.Y))
		hi = geom.V2(max(hi.X, o.Rect.X+o.Rect.W), max(hi.Y, o.Rect.Y+o.Rect.H))
		if o.Rect.X < box.X || o.Rect.Y < box.Y {
			assert.True(t, o.UV.W == 0 || o.UV.H == 0, "what reaches past is one texel's colour across")
		}
	}
	pad := p.slant()
	assert.InDelta(t, box.X-pad, lo.X, 1e-4)
	assert.InDelta(t, box.Y+box.H+pad, hi.Y, 1e-4)
}
