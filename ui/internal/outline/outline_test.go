package outline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/outline"
)

func TestPathDataIsReadWholeAndWrittenAbsolute(t *testing.T) {
	for d, want := range map[string]string{
		"M0 0 L 10 10 Z":                         "M 0 0 L 10 10 Z",
		"M 0,0 l 10 10 h 5 z":                    "M 0 0 L 10 10 H 15 Z",
		"m 0 0 c 1 2 3 4 5 6 a 1 1 0 0 1 2 2":    "M 0 0 C 1 2 3 4 5 6 A 1 1 0 0 1 7 8",
		"M0 0 10 0 10 10":                        "M 0 0 L 10 0 L 10 10",
		"m1 1 1 0 0 1 z m 2 2 l1 1":              "M 1 1 L 2 1 L 2 2 Z M 3 3 L 4 4",
		"M0 0 s1 1 2 2 q 3 3 4 4 t 1 1 v 5 H -1": "M 0 0 S 1 1 2 2 Q 5 5 6 6 T 7 7 V 12 H -1",
		"M0 0a5 5 0 0110 0":                      "M 0 0 A 5 5 0 0 1 10 0",
	} {
		_, got, ok := outline.Parse(d)
		require.True(t, ok, d)
		assert.Equal(t, want, got, d)
	}
	for _, d := range []string{
		"", "  ", "M 0 0 L", "L 10 10", "M 0 0 L 10 10 garbage", "M0 0 Z 5 5", "M0 0 A 1 1 0 2 1 3 3",
	} {
		_, _, ok := outline.Parse(d)
		assert.False(t, ok, d)
	}
}

func TestContainsFollowsTheFillRuleAndTakesTheOutline(t *testing.T) {
	p, _, ok := outline.Parse("M0 0 H30 V30 H0 Z M10 10 H20 V20 H10 Z")
	require.True(t, ok)
	inner, ring, out := geom.V2(15, 15), geom.V2(5, 5), geom.V2(35, 5)
	assert.True(t, p.Contains(inner, false))
	assert.True(t, p.Contains(ring, false))
	assert.False(t, p.Contains(out, false))
	assert.False(t, p.Contains(inner, true), "evenodd leaves a hole")
	assert.True(t, p.Contains(ring, true))
	for _, edge := range []geom.Vec2{{X: 10, Y: 10}, {X: 15, Y: 20}, {X: 30, Y: 30}, {X: 0, Y: 12}} {
		assert.True(t, p.Contains(edge, true), "%v is on the outline", edge)
	}
}

func TestCurvesAreFlattenedCloseToThemselves(t *testing.T) {
	e := outline.Ellipse(geom.V2(50, 20), 40, 10)
	for q, in := range map[geom.Vec2]bool{{X: 50, Y: 20}: true, {X: 89.9, Y: 20}: true, {X: 90.1, Y: 20}: false,
		{X: 50, Y: 29.9}: true, {X: 50, Y: 30.1}: false, {X: 85, Y: 28}: false} {
		assert.Equal(t, in, e.Contains(q, false), "ellipse at %v", q)
	}
	b, ok := e.Bounds()
	require.True(t, ok)
	assert.InDelta(t, 10, b.X, 1e-3)
	assert.InDelta(t, 80, b.W, 1e-3)
	r := outline.RoundedRect(geom.Rect{W: 100, H: 50}, [4]geom.Vec2{{X: 20, Y: 10}, {}, {X: 10, Y: 10}, {}})
	for q, in := range map[geom.Vec2]bool{{X: 1, Y: 1}: false, {X: 20, Y: 1}: true, {X: 99, Y: 1}: true,
		{X: 99, Y: 49}: false, {X: 1, Y: 49}: true, {X: 50, Y: 25}: true} {
		assert.Equal(t, in, r.Contains(q, false), "rounded rect at %v", q)
	}
	assert.Empty(t, outline.RoundedRect(geom.Rect{W: 10}, [4]geom.Vec2{}).Contours, "an empty box has none")
	arc, _, ok := outline.Parse("M0 0 A10 10 0 0 1 20 0 Z")
	require.True(t, ok)
	assert.True(t, arc.Contains(geom.V2(10, -9.9), false), "a half disc above its chord")
	assert.False(t, arc.Contains(geom.V2(10, 1), false))
}

func TestAMaskIsOpaqueInsideAndClearOutside(t *testing.T) {
	p, _, ok := outline.Parse("M0 0 H40 V40 H0 Z M10 10 H30 V30 H10 Z")
	require.True(t, ok)
	r := geom.Rect{W: 40, H: 40}
	nonzero, evenodd := p.Mask(r, 80, 80, false), p.Mask(r, 80, 80, true)
	assert.Equal(t, uint8(255), nonzero.RGBAAt(40, 40).A, "nonzero fills the middle")
	assert.Equal(t, uint8(0), evenodd.RGBAAt(40, 40).A, "evenodd leaves it clear")
	assert.Equal(t, uint8(255), evenodd.RGBAAt(5, 5).A)
	assert.Equal(t, uint8(255), evenodd.RGBAAt(5, 5).R, "white")
	star := outline.Polygon([]geom.Vec2{{X: 20, Y: 0}, {X: 32, Y: 36}, {X: 1, Y: 14}, {X: 39, Y: 14}, {X: 8, Y: 36}})
	assert.Equal(t, uint8(255), star.Mask(r, 40, 40, false).RGBAAt(20, 20).A, "a pentagram's middle, nonzero")
	assert.Equal(t, uint8(0), star.Mask(r, 40, 40, true).RGBAAt(20, 20).A,
		"and its hole, evenodd, one contour crossing itself")
	assert.Equal(t, uint8(255), star.Mask(r, 40, 40, true).RGBAAt(20, 5).A)
	c := outline.Ellipse(geom.V2(20, 20), 20, 20).Mask(r, 40, 40, false)
	assert.Equal(t, uint8(0), c.RGBAAt(0, 0).A)
	assert.Equal(t, uint8(255), c.RGBAAt(20, 20).A)
}
