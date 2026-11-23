package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"math"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// topOnly is a w×24 box at (10, 10) with only a top border of width bw in
// style s.
func topOnly(w, bw float32, s style.BorderStyle) *box.Box {
	b := el(10, 10, w, 24, func(st *style.Computed) {
		st.Border = style.Sides[float32]{Top: bw}
		st.BorderStyle = style.Sides[style.BorderStyle]{Top: s}
		st.BorderColor = style.All(black)
	})
	return b
}

func TestADashedSideHasDashesFlushAtBothEndsAsChromes(t *testing.T) {
	for _, c := range []struct {
		w, bw, scale, dash float32
		n                  int
	}{
		{40, 1, 1, 3, 8}, {40, 2, 1, 6, 4}, {40, 3, 1, 6, 5}, {40, 4, 1, 8, 4},
		{40, 1, 2, 3, 8}, {73, 1, 2, 3, 15}, {40, 2, 2, 4, 7}, {120, 2, 2, 4, 20},
	} {
		dashes := paintOne(topOnly(c.w, c.bw, style.BorderDashed), Scale(c.scale)).of("round")
		require.Len(t, dashes, c.n, "%v px wide, %v long at %vx", c.bw, c.w, c.scale)
		assert.Equal(t, float32(10), dashes[0].Rect.X)
		last := dashes[c.n-1].Rect
		assert.InDelta(t, 10+c.w, last.X+last.W, 1e-3, "ends flush")
		for _, d := range dashes {
			assert.InDelta(t, c.dash, d.Rect.W, 1e-3)
			assert.Equal(t, c.bw, d.Rect.H)
			assert.Equal(t, float32(10), d.Rect.Y)
		}
	}
}

func TestDashesFallBetweenPixelsAsChromesGapsDo(t *testing.T) {
	dashes := paintOne(topOnly(40, 1, style.BorderDashed)).of("round")
	require.Len(t, dashes, 8)
	assert.InDelta(t, 10+16.0/7+3, dashes[1].Rect.X, 1e-3, "8 dashes of 3 leave 7 gaps of 16/7")
}

func TestAShortDashedSideHasTwoShortenedDashesOrNone(t *testing.T) {
	two := paintOne(topOnly(7, 1, style.BorderDashed)).of("round")
	require.Len(t, two, 2, "under two dashes and a gap: two, scaled to fit")
	assert.InDelta(t, 3*7.0/8, two[0].Rect.W, 1e-3)
	assert.InDelta(t, 17, two[1].Rect.X+two[1].Rect.W, 1e-3)
	solid := paintOne(topOnly(6, 1, style.BorderDashed))
	assert.Empty(t, solid.of("round"), "no room for two dashes: solid")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 6, H: 1}, solid.of("rect")[0].Rect)
}

func TestADottedSideHasRoundDotsAtBothEnds(t *testing.T) {
	for _, c := range []struct {
		w, bw, scale float32
		n            int
	}{{40, 4, 1, 6}, {40, 2, 2, 11}, {73, 4, 1, 10}} {
		dots := paintOne(topOnly(c.w, c.bw, style.BorderDotted), Scale(c.scale)).of("circle")
		require.Len(t, dots, c.n, "%v px wide, %v long at %vx", c.bw, c.w, c.scale)
		assert.InDelta(t, 10+c.bw/2, dots[0].At.X, 1e-3)
		assert.InDelta(t, 10+c.w-c.bw/2, dots[c.n-1].At.X, 0.1, "a hundredth of a pixel short a dot, as Chrome")
		assert.InDelta(t, c.bw/2, dots[0].Radius, 1e-3)
		assert.InDelta(t, 10+c.bw/2, dots[0].At.Y, 1e-3)
	}
}

func TestDotsUpToThreePixelsAreSquaresAtTheEndsAndCentredBetween(t *testing.T) {
	r := paintOne(topOnly(40, 1, style.BorderDotted), Scale(2))
	assert.Empty(t, r.of("circle"))
	squares := r.of("round")
	require.Len(t, squares, 21, "two device pixels wide, 80 long: one at each end, 19 between, as Chrome")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 1, H: 1}, squares[0].Rect)
	assert.Equal(t, geom.Rect{X: 11.5, Y: 10, W: 1, H: 1}, squares[1].Rect, "the leftover pixel split either side")
	assert.Equal(t, geom.Rect{X: 13.5, Y: 10, W: 1, H: 1}, squares[2].Rect)
	assert.Equal(t, geom.Rect{X: 49, Y: 10, W: 1, H: 1}, squares[20].Rect)
}

func TestSidesRunCornerToCornerSoCornersJoin(t *testing.T) {
	b := el(10, 10, 40, 24, func(st *style.Computed) {
		st.Border = style.All[float32](2)
		st.BorderStyle = style.All(style.BorderDashed)
		st.BorderColor = style.All(black)
	})
	var left []geom.Rect
	for _, o := range paintOne(b).of("round") {
		if o.Rect.X == 10 && o.Rect.W == 2 {
			left = append(left, o.Rect)
		}
	}
	require.NotEmpty(t, left)
	assert.Equal(t, float32(10), left[0].Y, "the left side's first dash starts in the corner")
	assert.InDelta(t, 34, left[len(left)-1].Y+left[len(left)-1].H, 1e-3)
}

func TestARoundedDashedBorderIsItsDashesCutToTheRing(t *testing.T) {
	b := el(10, 10, 40, 24, func(st *style.Computed) {
		st.Border = style.All[float32](2)
		st.BorderStyle = style.All(style.BorderDashed)
		st.BorderColor = style.All(black)
		c := style.Size[style.Length]{Width: style.Px(6), Height: style.Px(6)}
		st.Radius = style.Corners[style.Size[style.Length]]{TopLeft: c, TopRight: c, BottomRight: c, BottomLeft: c}
	})
	r := paintOne(b)
	assert.Empty(t, r.of("border"), "no solid ring")
	require.Len(t, r.of("push-mask"), 1, "one colour and style all round: one mask")
	assert.Len(t, r.of("rect"), 1)
}

func TestADashDownASideStartsAndEndsAtAQuarterOfAPixel(t *testing.T) {
	assert.Equal(t, float32(49.25), quarter(49+1.0/3), "three of four samples see an edge a third in")
	assert.Equal(t, float32(55.5), quarter(55.4))
	b := el(10, 10, 20, 31, func(st *style.Computed) {
		st.Border = style.Sides[float32]{Left: 1}
		st.BorderStyle = style.Sides[style.BorderStyle]{Left: style.BorderDashed}
		st.BorderColor = style.All(black)
	})
	dashes := paintOne(b, Scale(2)).of("round")
	require.Len(t, dashes, 7, "62 pixels: 7 dashes of 6, gaps of 10/3")
	for _, d := range dashes {
		assert.Zero(t, math.Mod(float64(d.Rect.Y*2*4), 1), "%v starts on a quarter pixel", d.Rect)
		assert.Zero(t, math.Mod(float64((d.Rect.Y+d.Rect.H)*2*4), 1), "%v ends on one", d.Rect)
	}
}
