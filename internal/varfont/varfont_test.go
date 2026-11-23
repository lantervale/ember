package varfont_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/internal/varfont/varfonttest"
)

func parse(t *testing.T, ttf []byte) *varfont.Font {
	t.Helper()
	f, err := varfont.Parse(ttf)
	require.NoError(t, err)
	return f
}

func glyphOf(t *testing.T, r rune) uint16 {
	t.Helper()
	f, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	var buf sfnt.Buffer
	g, err := f.GlyphIndex(&buf, r)
	require.NoError(t, err)
	return uint16(g)
}

func TestAFontWithoutFvarHasNoAxesAndDrawsAsItIs(t *testing.T) {
	f := parse(t, goregular.TTF)
	assert.False(t, f.Variable())
	assert.Nil(t, f.Normalize([]float32{700}))
	g := glyphOf(t, 'A')

	s, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	var buf sfnt.Buffer
	adv, err := s.GlyphAdvance(&buf, sfnt.GlyphIndex(g), fixed.I(2048), font.HintingNone)
	require.NoError(t, err)
	assert.InDelta(t, float32(adv)/64, f.Advance(g, nil), 0.01)

	pts, ends, err := f.Outline(g, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, ends)
	segs, err := s.LoadGlyph(&buf, sfnt.GlyphIndex(g), fixed.I(2048), nil)
	require.NoError(t, err)
	var got []fixed.Point26_6
	varfont.Contours(pts, ends,
		func(x, y float32) { got = append(got, pt(x, y)) },
		func(x, y float32) { got = append(got, pt(x, y)) },
		func(cx, cy, x, y float32) { got = append(got, pt(cx, cy), pt(x, y)) })
	var want []fixed.Point26_6
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpQuadTo:
			want = append(want, seg.Args[0], seg.Args[1])
		case sfnt.SegmentOpMoveTo, sfnt.SegmentOpLineTo, sfnt.SegmentOpCubeTo:
			want = append(want, seg.Args[0])
		}
	}
	assert.Equal(t, want, got, "the same outline as sfnt's, y down")
}

func pt(x, y float32) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(math.Round(float64(x) * 64)), Y: fixed.Int26_6(math.Round(float64(-y) * 64))}
}

func TestTheAxesAndInstancesAreFvars(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	require.True(t, f.Variable())
	require.Len(t, f.Axes, 2)
	assert.Equal(t, varfont.Axis{Tag: "wght", Min: 100, Default: 400, Max: 900}, stripped(f.Axes[0]))
	assert.Equal(t, varfont.Axis{Tag: "wdth", Min: 50, Default: 100, Max: 150}, stripped(f.Axes[1]))
	assert.Equal(t, 1, f.Tag("wdth"))
	assert.Equal(t, -1, f.Tag("slnt"))
	require.Len(t, f.Instances, 3)
	assert.Equal(t, []float32{700, 75}, f.Instances[2].Coords)
}

func stripped(a varfont.Axis) varfont.Axis {
	return varfont.Axis{Tag: a.Tag, Min: a.Min, Default: a.Default, Max: a.Max, Hidden: a.Hidden}
}

func TestAPlaceIsNormalizedClampedAndMappedByAvar(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	assert.Nil(t, f.Normalize([]float32{400, 100}), "the default instance")
	assert.Nil(t, f.Normalize(nil))
	assert.Equal(t, varfont.Coords{-1, 0}, f.Normalize([]float32{50, 100}), "clamped")
	assert.Equal(t, varfont.Coords{1, 0.5}, f.Normalize([]float32{1000, 125}))
	assert.Equal(t, varfont.Coords{0.75, -0.5}, f.Normalize([]float32{650, 75}), "avar maps 0.5 to 0.75")
	assert.InDelta(t, 0.875, f.Normalize([]float32{775})[0], 1e-4, "between avar's points")

	linear := parse(t, varfonttest.Build(varfonttest.Options{NoAvar: true}))
	assert.Equal(t, varfont.Coords{0.5, 0}, linear.Normalize([]float32{650}))
}

func TestAdvancesFollowHVARAsThePhantomPointsDo(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	phantom := parse(t, varfonttest.Build(varfonttest.Options{NoHVAR: true}))
	for _, r := range "AVé .g" {
		g := glyphOf(t, r)
		base := f.Advance(g, nil)
		wide := float32(math.Round(float64(base * varfonttest.Wider)))
		assert.InDelta(t, base+2*varfonttest.Bolder, f.Advance(g, f.Normalize([]float32{900})), 1e-3, "%q", r)
		assert.InDelta(t, base-2*varfonttest.Lighter, f.Advance(g, f.Normalize([]float32{100})), 1e-3, "%q", r)
		assert.InDelta(t, base+wide, f.Advance(g, f.Normalize([]float32{400, 150})), 1e-3, "%q", r)
		assert.InDelta(t, base-wide/2, f.Advance(g, f.Normalize([]float32{400, 75})), 1e-3, "%q", r)
		for _, at := range [][]float32{{900}, {100}, {650, 125}, {250, 60}, {775, 150}} {
			c := f.Normalize(at)
			assert.InDelta(t, f.Advance(g, c), phantom.Advance(g, c), 1e-3, "%q at %v", r, at)
		}
	}
}

func TestAGlyphMovesByEachTupleScaledByItsRegion(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	g := glyphOf(t, 'H')
	base, ends, err := f.Outline(g, nil)
	require.NoError(t, err)

	wide, wideEnds, err := f.Outline(g, f.Normalize([]float32{400, 150}))
	require.NoError(t, err)
	assert.Equal(t, ends, wideEnds)
	for i, p := range base {
		assert.InDelta(t, p.X+float32(math.Round(float64(p.X*varfonttest.Wider))), wide[i].X, 1e-3, "point %d", i)
		assert.InDelta(t, p.Y, wide[i].Y, 1e-3)
	}

	// At 650 avar puts wght at 0.75: three quarters of wght 900's tuple,
	// and half of the intermediate one peaking at 0.5.
	mid, _, err := f.Outline(g, f.Normalize([]float32{650}))
	require.NoError(t, err)
	bold, _, err := f.Outline(g, f.Normalize([]float32{900}))
	require.NoError(t, err)
	for i, p := range base {
		tall := float32(math.Round(float64(p.Y*varfonttest.Taller))) / 2
		assert.InDelta(t, p.X+0.75*(bold[i].X-p.X), mid[i].X, 1e-3, "point %d", i)
		assert.InDelta(t, p.Y+0.75*(bold[i].Y-p.Y)+tall, mid[i].Y, 1e-3, "point %d", i)
	}
}

func TestPointsATupleLeavesOutAreInferredAlongTheirContour(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	g := glyphOf(t, 'O')
	base, _, err := f.Outline(g, nil)
	require.NoError(t, err)
	wide, _, err := f.Outline(g, f.Normalize([]float32{400, 150}))
	require.NoError(t, err)
	off := 0
	for i, p := range base {
		if p.On {
			continue
		}
		off++
		// wdth 150 names only the points on the curve; one between two is
		// stretched in proportion, as they are, to a rounding.
		assert.InDelta(t, p.X*(1+varfonttest.Wider), wide[i].X, 1, "point %d", i)
	}
	assert.Positive(t, off, "an O has control points")
}

func TestACompositeVariesItsComponentsAndMovesThemByItsOwnDeltas(t *testing.T) {
	f := parse(t, varfonttest.Weighty())
	c := f.Normalize([]float32{700, 150})
	e, _, err := f.Outline(glyphOf(t, 'e'), c)
	require.NoError(t, err)
	eAcute, ends, err := f.Outline(glyphOf(t, 'é'), c)
	require.NoError(t, err)
	baseE, _, err := f.Outline(glyphOf(t, 'e'), nil)
	require.NoError(t, err)
	baseAcute, _, err := f.Outline(glyphOf(t, 'é'), nil)
	require.NoError(t, err)
	require.Greater(t, len(eAcute), len(e))
	assert.Len(t, ends, 3, "the e's two contours and the accent's")
	// The e is the first component; its offset, stretched with the
	// glyph, is the same number of units past the plain e's as at rest.
	shift := (baseAcute[0].X - baseE[0].X) * varfonttest.Wider
	for i := range e {
		assert.InDelta(t, e[i].X+baseAcute[0].X-baseE[0].X+shift, eAcute[i].X, 1, "point %d", i)
		assert.InDelta(t, e[i].Y+baseAcute[0].Y-baseE[0].Y, eAcute[i].Y, 1e-3, "point %d", i)
	}
}

func TestBrokenTablesAreErrorsOrLeftOutNeverPanics(t *testing.T) {
	ttf := varfonttest.Weighty()
	for cut := 0; cut < len(ttf); cut += 97 {
		f, err := varfont.Parse(ttf[:cut])
		if err != nil {
			continue
		}
		for g := range uint16(min(f.NumGlyphs(), 40)) {
			c := f.Normalize([]float32{777, 66})
			f.Advance(g, c)
			_, _, _ = f.Outline(g, c)
		}
	}
	bad := append([]byte(nil), ttf...)
	for i := 300; i < len(bad); i += 13 {
		bad[i] ^= 0x5a
	}
	if f, err := varfont.Parse(bad); err == nil {
		for g := range uint16(f.NumGlyphs()) {
			c := f.Normalize([]float32{777, 66})
			f.Advance(g, c)
			_, _, _ = f.Outline(g, c)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add(varfonttest.Weighty())
	f.Fuzz(func(_ *testing.T, ttf []byte) {
		font, err := varfont.Parse(ttf)
		if err != nil {
			return
		}
		for g := range uint16(min(font.NumGlyphs(), 64)) {
			c := font.Normalize([]float32{600, 80})
			font.Advance(g, c)
			_, _, _ = font.Outline(g, c)
		}
	})
}
