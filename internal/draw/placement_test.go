package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"math"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// quads are the glyph quads drawn on c: each one's rectangle and atlas
// UVs, in drawing order.
func quads(c *List) ([]geom.Rect, []geom.Vec4) {
	var rects []geom.Rect
	var uvs []geom.Vec4
	for i := 0; i+3 < len(c.Vertices); i += 4 {
		v := c.Vertices[i : i+4]
		if Kind(v[0].Shape.X) != KindText {
			continue
		}
		rects = append(
			rects,
			geom.Rect{X: v[0].Pos.X, Y: v[0].Pos.Y, W: v[2].Pos.X - v[0].Pos.X, H: v[2].Pos.Y - v[0].Pos.Y},
		)
		uvs = append(uvs, geom.Vec4{X: v[0].UV.X, Y: v[0].UV.Y, Z: v[2].UV.X, W: v[2].UV.Y})
	}
	return rects, uvs
}

// unrounded is the advance of r at px points as the font has it, with no
// rounding: what layout measures with.
func unrounded(t *testing.T, r rune, px float64) float32 {
	f, err := opentype.Parse(goregular.TTF)
	require.NoError(t, err)
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: px * 64, DPI: 72, Hinting: font.HintingNone})
	require.NoError(t, err)
	adv, ok := face.GlyphAdvance(r)
	require.True(t, ok)
	return float32(adv) / 64 / 64
}

func TestAdvancesAreTheFontsUnrounded(t *testing.T) {
	a := unrounded(t, 'i', 13)
	require.NotEqual(t, float32(math.Round(float64(a*2))/2), a, "a width that rounding to half points would change")
	assert.InDelta(t, a, Regular.Size(13).Measure("i").X, 1e-3)
	assert.InDelta(t, 10*a, Regular.Size(13).Measure("iiiiiiiiii").X, 1e-3, "so a run never drifts")
}

func TestGlyphQuadsSitOnWholeDevicePixels(t *testing.T) {
	var c List
	c.Text(geom.V2(0.3, 0.1), "Handgloves", Style{Size: 13})
	rects, _ := quads(&c)
	require.NotEmpty(t, rects)
	for _, r := range rects {
		assert.InDelta(
			t,
			math.Round(float64(r.X*oversample)),
			r.X*oversample,
			1e-3,
			"drawn 1:1, never blurred across pixels",
		)
		assert.InDelta(t, math.Round(float64(r.Y*oversample)), r.Y*oversample, 1e-3)
	}
}

func TestASubpixelOriginRastersTheGlyphAtThatPhase(t *testing.T) {
	at := func(x float32) (geom.Rect, geom.Vec4) {
		var c List
		c.Text(geom.V2(x, 0), "l", Style{Size: 16})
		r, uv := quads(&c)
		require.Len(t, r, 1)
		return r[0], uv[0]
	}
	r0, uv0 := at(0)
	rq, uvq := at(0.25)
	r1, uv1 := at(0.5)
	assert.NotEqual(t, uv0, uvq, "half a device pixel along: its own raster")
	assert.Equal(t, uv0, uv1, "a whole device pixel along: the same raster")
	assert.InDelta(t, r0.X+0.5, r1.X, 1e-4, "moved by that pixel")
	assert.True(t, rq.X == r0.X || rq.X == r1.X, "the phase is in the raster, the quad stays on the grid")
}

func TestABaselineStyleTakesTheOriginAsTheBaseline(t *testing.T) {
	var c List
	c.Text(geom.V2(0, 20), "x", Style{Size: 16, Baseline: true})
	r, _ := quads(&c)
	require.Len(t, r, 1)
	assert.InDelta(t, 20, r[0].Y+r[0].H, 0.5, "x stands on the baseline")

	var top List
	top.Text(geom.V2(0, 20), "x", Style{Size: 16})
	rt, _ := quads(&top)
	assert.InDelta(t, r[0].Y+Regular.Size(16).Ascent(), rt[0].Y, 0.5, "without it, the origin is the line's top")
}

func TestTheBaselineSnapsToTheDeviceGrid(t *testing.T) {
	var c, d List
	c.Text(geom.V2(0, 20.2), "x", Style{Size: 16, Baseline: true})
	d.Text(geom.V2(0, 20), "x", Style{Size: 16, Baseline: true})
	rc, _ := quads(&c)
	rd, _ := quads(&d)
	assert.Equal(t, rd[0].Y, rc[0].Y, "20.2 rounds to the nearest device pixel, 20")
}

func TestAFontWithoutLayoutTablesIsNotShaped(t *testing.T) {
	f := Regular.Size(16)
	assert.Empty(t, f.shapedRuns, "Go has no kern, GSUB or GPOS: nothing to shape")
	assert.InDelta(t, f.Measure("A").X+f.Measure("V").X, f.Measure("AV").X, 1e-4)
}

func TestTextOfAFractionalSizeKeepsItsOwnAdvances(t *testing.T) {
	const text = "A demonstration of what can be accomplished through"
	for _, size := range []float32{13.44, 13.5, 10.5625} {
		want := unrounded(t, 'i', float64(size))
		assert.InDelta(t, 10*want, Style{Size: size}.Measure("iiiiiiiiii").X, 1e-2, "%v points measured", size)
		var whole, c List
		whole.Text(geom.Vec2{}, text+" i", Style{Size: float32(math.Round(float64(size)))})
		c.Text(geom.Vec2{}, text+" i", Style{Size: size})
		rw, _ := quads(&whole)
		rects, _ := quads(&c)
		require.Len(t, rects, len(rw))
		last, lastWhole := rects[len(rects)-1], rw[len(rw)-1]
		ratio := size / float32(math.Round(float64(size)))
		assert.InDelta(t, lastWhole.X*ratio, last.X, 0.5, "%v points: the last letter where its own size puts it", size)
		assert.InDelta(t, lastWhole.H*ratio, last.H, 1e-2, "and as tall as its size")
	}
}

func TestTextOfAFractionalSizeIsDrawnAtItsSizeUnderAScale(t *testing.T) {
	var c List
	c.SetTransform(geom.Transform2D{X: geom.V2(2, 0), Y: geom.V2(0, 2)})
	c.Text(geom.Vec2{}, "iiiiiiiiiii", Style{Size: 13.44})
	rects, _ := quads(&c)
	require.Len(t, rects, 11)
	assert.InDelta(t, 2*10*unrounded(t, 'i', 13.44), rects[10].X-rects[0].X, 0.51, "ten advances of 13.44 points")
	assert.Equal(t, float32(27), c.Batches[len(c.Batches)-1].Atlas.size,
		"rasterized at the whole size nearest what the screen shows")
}
