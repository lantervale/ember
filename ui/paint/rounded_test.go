package paint //nolint:testpackage // white-box: the geometry of rounded sides

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestRoundedSidesMeetWhereTheMitreCrossesTheInnerCornersChord(t *testing.T) {
	r := geom.Rect{W: 100, H: 60}
	j := junctions(r, [4]float32{4, 4, 4, 4}, radii{20, 20, 20, 20})
	assert.InDelta(t, 4+16.0/2, j[0].X, 1e-4, "on the 45-degree line, half the inner radius past the inner corner")
	assert.InDelta(t, j[0].X, j[0].Y, 1e-4)
	assert.InDelta(t, 100-12, j[1].X, 1e-4)

	square := junctions(r, [4]float32{2, 8, 4, 12}, radii{})
	assert.Equal(t, geom.V2(12, 2), square[0], "a square inner corner is the junction")
	assert.Equal(t, geom.V2(92, 56), square[2])

	deep := junctions(geom.Rect{W: 20, H: 10}, [4]float32{2, 2, 2, 2}, radii{5, 5, 5, 5})
	for _, q := range deep {
		assert.InDelta(t, 5, q.Y, 5, "never past the middle")
		assert.True(t, q.X >= 0 && q.X <= 20)
	}
}

func TestChromePicksTheGapNearerItsOwnRoundAClosedPath(t *testing.T) {
	assert.InDelta(t, 4.2857, bestGap(172, 8, 4), 1e-3, "14 dashes leave a gap nearer 4 than 15 do")
	assert.InDelta(t, 3.4667, bestGap(172, 8, 3.2), 1e-3, "15 dashes when theirs is nearer")
	assert.InDelta(t, 2, bestGap(10, 8, 4), 1e-3, "one dash, its gap the rest")
}

func TestRoundedDashesStartAtTheTopLeftEndOfTheTopSide(t *testing.T) {
	r := geom.Rect{W: 64, H: 44}
	w := [4]float32{4, 4, 4, 4}
	p := dashPath(r, w, radii{18, 18, 18, 18}, 4, 4, 2, false)
	require.Len(t, p.Contours, 15, "172 points round: 14 dashes 8 long, and the half point past 172 starting one more")
	first := p.Contours[0]
	assert.InDelta(t, 2+16, (first[0].X+first[len(first)-1].X)/2, 0.01,
		"where the top-left corner's arc ends on the middle of the border")
	assert.InDelta(t, 2, (first[0].Y+first[len(first)-1].Y)/2, 0.05)

	dots := dashPath(r, w, radii{18, 18, 18, 18}, 4, 4, 2, true)
	require.NotEmpty(t, dots.Contours)
	b, _ := dots.Bounds()
	assert.InDelta(t, 0, b.Y, 0.05, "dots 4 across along the middle reach the top")

	thin := dashPath(r, w, radii{18, 18, 18, 18}, 1, 1, 2, true)
	assert.Greater(t, len(thin.Contours), 14, "a dot 2 device pixels across is a short dash as long as wide")
}

func TestRoundedSidesOfOtherStylesAreBandsOfTheirLook(t *testing.T) {
	grey := color.RGB8(128, 128, 128)
	w := [4]float32{8, 8, 8, 8}
	groove := lookOf(0, w, grey, style.BorderGroove, 1)
	require.Len(t, groove.bands, 2)
	darker, lighter := shades(grey)
	assert.Equal(t, darker, groove.bands[0].c, "a groove's top dark outside")
	assert.Equal(t, lighter, groove.bands[1].c)
	assert.Equal(t, [4]float32{4, 4, 4, 4}, groove.bands[0].to, "halfway across every side")
	assert.Equal(t, lighter, lookOf(1, w, grey, style.BorderGroove, 1).bands[0].c, "its right light outside")
	assert.Equal(t, lighter, lookOf(2, w, grey, style.BorderInset, 1).bands[0].c, "an inset's bottom light")

	double := lookOf(0, [4]float32{9, 3, 9, 3}, grey, style.BorderDouble, 2)
	require.Len(t, double.bands, 2)
	assert.Equal(t, [4]float32{3, 1, 3, 1}, double.bands[0].to, "the outer line cut in device pixels")
	assert.Equal(t, [4]float32{6, 2, 6, 2}, double.bands[1].from)
	assert.False(t, lookOf(0, w, grey, style.BorderSolid, 1).same(lookOf(1, w, grey, style.BorderInset, 1)))
}

func TestOnlyARoundedBorderSolidOrDoubleAlikeAllRoundIsARing(t *testing.T) {
	black := color.RGB(0, 0, 0)
	same := [4]color.Color{black, black, black, black}
	solid := [4]style.BorderStyle{style.BorderSolid, style.BorderSolid, style.BorderSolid, style.BorderSolid}
	assert.True(t, oneRing([4]float32{2, 2, 2, 2}, same, solid))
	assert.False(t, oneRing([4]float32{2, 3, 2, 2}, same, solid), "uneven")
	dashed := solid
	dashed[1] = style.BorderDashed
	assert.False(t, oneRing([4]float32{2, 2, 2, 2}, same, dashed))
}
