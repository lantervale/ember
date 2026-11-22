package css //nolint:testpackage // tests the unexported math parser (parseDim, Dim's parts)

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// resolved is src read and resolved against a 200px basis, an em of 10
// and a rem of 16, as the top of a calc() leaves it.
func resolved(t *testing.T, src string) float64 {
	t.Helper()
	d, err := parseDim(valueOf(t, src))
	require.NoError(t, err, src)
	return d.Absolute(&Units{Em: 10, Rem: 16}).Resolve(200)
}

func TestMathFunctionsWorkOutAsChromeDoes(t *testing.T) {
	cases := map[string]float64{
		"round(13px, 5px)":                      15,
		"round(nearest, 12.5px, 5px)":           15,
		"round(-12.5px, 5px)":                   -10,
		"round(up, 13px, 5px)":                  15,
		"round(down, -13px, 5px)":               -15,
		"round(to-zero, -13px, 5px)":            -10,
		"round(to-zero, 13px, 5px)":             10,
		"calc(round(2.5) * 1px)":                3,
		"calc(round(-2.5) * 1px)":               -2,
		"round(50%, 7px)":                       98,
		"round(up, 1em, 3px)":                   12,
		"mod(-7px, 5px)":                        3,
		"mod(7px, -5px)":                        -3,
		"rem(-7px, 5px)":                        -2,
		"rem(7px, -5px)":                        2,
		"mod(50%, 30px)":                        10,
		"abs(-50%)":                             100,
		"abs(calc(10px - 2em))":                 10,
		"calc(10px * sign(50% - 10px))":         10,
		"calc(10px * sign(-5%))":                -10,
		"calc(10px * sign(0))":                  0,
		"clamp(10px, 50px, none)":               50,
		"clamp(none, 50px, 20px)":               20,
		"clamp(none, 50px, none)":               50,
		"clamp(100px, 20%, 150px)":              100,
		"clamp(200px, 50%, 100px)":              200,
		"hypot(3px, 4px)":                       5,
		"hypot(30%, 40px)":                      math.Hypot(60, 40),
		"hypot(-3px)":                           3,
		"calc(1px * hypot(3, 4))":               5,
		"calc(10px * sin(30deg))":               5,
		"calc(10px * cos(0.5turn))":             -10,
		"calc(10px * tan(45deg))":               10,
		"calc(10px * sin(pi / 2))":              10,
		"calc(1px * asin(1) / 1deg)":            90,
		"calc(1px * acos(0) / 1deg)":            90,
		"calc(1px * atan(1) / 1deg)":            45,
		"calc(1px * atan2(1em, 10px) / 1deg)":   45,
		"calc(1px * atan2(-1, 0) / 1rad)":       -math.Pi / 2,
		"calc(1px * pow(2, 3))":                 8,
		"calc(1px * sqrt(16))":                  4,
		"calc(1px * exp(1))":                    math.E,
		"calc(1px * log(8, 2))":                 3,
		"calc(1px * log(e))":                    1,
		"calc(1px * e)":                         math.E,
		"calc(1px * PI)":                        math.Pi,
		"calc(10px / 1px * 1px)":                10,
		"calc(1em * 1em / 1px)":                 100,
		"calc(100px * atan2(1em, 10px) / 1deg)": 4500,
		"calc(50% * 2px / 1px)":                 200,
		"min(50%, 10px, 2em)":                   10,
		"max(10%, min(50%, 100px))":             100,
	}
	for src, want := range cases {
		got := resolved(t, src)
		assert.InDelta(t, want, got, 1e-9, src)
	}
}

func TestMathFunctionsKeepNaNInfinitiesAndSignedZeros(t *testing.T) {
	inf, ninf := math.Inf(1), math.Inf(-1)
	cases := map[string]float64{
		"calc(1px / 0)":                        inf,
		"calc(-1px / 0)":                       ninf,
		"calc(1px / -0)":                       ninf,
		"calc(1px / (-1 * 0))":                 ninf,
		"calc(infinity * 1px)":                 inf,
		"calc(-infinity * 1px)":                ninf,
		"calc(1px * Infinity)":                 inf,
		"calc(1px * tan(90deg))":               inf,
		"calc(1px * tan(-90deg))":              ninf,
		"calc(1px * tan(270deg))":              ninf,
		"calc(1px / sign(-0))":                 ninf,
		"calc(1px / mod(-0, 5))":               inf,
		"calc(1px / round(-0.3))":              ninf,
		"calc(1px / abs(-0))":                  inf,
		"calc(1px / rem(-0, 5))":               ninf,
		"round(infinity * 1px, 5px)":           inf,
		"calc(1px * round(up, 1, infinity))":   inf,
		"calc(1px / round(up, -1, infinity))":  ninf,
		"calc(1px / round(down, 1, infinity))": inf,
		"calc(1px * mod(5, infinity))":         5,
		"calc(1px * rem(-5, infinity))":        -5,
		"min(infinity * 1px, 5px)":             5,
	}
	for src, want := range cases {
		got := resolved(t, src)
		assert.Equal(t, want, got, src)
	}
	for _, src := range []string{"calc(NaN * 1px)", "calc(1px * nan)", "round(5px, 0px)", "mod(5px, 0px)",
		"rem(infinity * 1px, 5px)", "calc(1px * mod(-5, infinity))", "min(NaN * 1px, 5px)",
		"calc(1px * sqrt(-1))", "calc(1px * asin(2) / 1deg)", "round(infinity * 1px, infinity * 1px)"} {
		got := resolved(t, src)
		assert.True(t, math.IsNaN(got), "%s is %v", src, got)
	}
}

func TestMathFunctionsAreTypedAsChromeTypesThem(t *testing.T) {
	types := map[string]dimType{
		"calc(1px / 4px)":         typeNumber,
		"calc(1em / 40px)":        typeNumber,
		"sign(50% - 10px)":        typeNumber,
		"atan2(1em, 10px)":        typeAngle,
		"calc(45deg + 0.5turn)":   typeAngle,
		"calc(asin(0.5))":         typeAngle,
		"min(10deg, 1rad)":        typeAngle,
		"calc(10deg * sign(1px))": typeAngle,
		"calc(10% + 10deg)":       typeAngle,
		"abs(-50%)":               typePercent,
		"round(15%, 1px)":         typeLength,
	}
	for src, want := range types {
		d, err := parseDim(valueOf(t, src))
		require.NoError(t, err, src)
		assert.Equal(t, want, d.typ, src)
	}
	for _, src := range []string{"clamp(10%, 0.5, 20%)", "calc(sqrt(4px * 4px))", "calc(1px * -pi)", "round(13px)",
		"calc(10deg + 5px)", "calc(1px + 2)", "calc(50% + 0.1)", "min(50%, 0.3)", "calc(2px * 3px)", "calc(1fr * 2)",
		"sin(10px)", "pow(2px, 2)", "atan2(1px, 1deg)", "round(sideways, 1px, 2px)", "mod(1px)", "clamp(none, 1px)",
		"clamp(1px, none, 2px)", "min()", "abs(1px, 2px)", "log(1, 2, 3)", "e"} {
		_, err := parseDim(valueOf(t, src))
		assert.Error(t, err, src)
	}
	d, _ := parseDim(valueOf(t, "sign(50% - 10px)"))
	assert.True(t, d.HasPercent(), "a number of a percentage waits for layout")
	assert.False(t, d.IsNumber())
}

func TestDeclarationsOfTheWrongTypeAreDroppedWithAWarning(t *testing.T) {
	for _, decl := range []string{"width: calc(1px + 2deg)", "width: sign(50% - 10px)", "width: min(1px, 2)",
		"width: round(13px)", "opacity: min(50%, 0.3)", "opacity: calc(10px)", "transform: rotate(calc(10deg + 5px))",
		"transform: rotate(atan2(50%, 10px))", "line-height: calc(1 + 10%)", "border-top-width: abs(-10%)",
		"width: 10deg", "width: calc(1px * -pi)"} {
		decls, warns := ParseInline(decl)
		assert.Empty(t, decls, decl)
		assert.NotEmpty(t, warns, decl)
	}
	for _, decl := range []string{"width: round(50%, 7px)", "opacity: sign(-3)", "opacity: calc(1px / 4px)",
		"transform: rotate(atan2(1em, 10px))", "line-height: sign(50% - 1px)", "width: calc(1px / 0)",
		"background-image: conic-gradient(red calc(10% + 10deg), blue)", "width: clamp(none, 50%, 20px)"} {
		decls, warns := ParseInline(decl)
		assert.Len(t, decls, 1, decl)
		assert.Empty(t, warns, decl)
	}
}

func TestPercentagesInMathFunctionsWaitForWhatTheyAreOf(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	Compute(root, []*Sheet{sheet(t, `#box { font-size: 20px;
line-height: min(150%, 40px); word-spacing: max(50%, 2px); vertical-align: min(50%, 4px);
width: round(50%, 7px); margin-left: calc(10px * sign(50% - 25px));
background-image: linear-gradient(red min(50%, 30px), blue), conic-gradient(red calc(25% + 90deg), blue) }`)})
	s := styleOf(box)
	assert.InDelta(t, 30, s.LineHeight.Px, 1e-4, "line-height is of the font size")
	assert.InDelta(t, 10, s.WordGap(), 1e-4, "word-spacing is of the font size")
	assert.InDelta(t, 4, s.VerticalShift.At(100), 1e-4)
	assert.InDelta(t, 2, s.VerticalShift.At(4), 1e-4, "vertical-align is of the line height")
	assert.InDelta(t, 98, s.Size.Width.At(200), 1e-4)
	assert.InDelta(t, 10, s.Margin.Left.At(200), 1e-4)
	assert.InDelta(t, -10, s.Margin.Left.At(20), 1e-4)
	stops := s.BackgroundImages[0].Gradient.Stops
	assert.InDelta(t, 30, stops[0].At.At(100), 1e-4)
	assert.InDelta(t, 20, stops[0].At.At(40), 1e-4, "a stop is of the gradient line")
	assert.InDelta(t, 50, s.BackgroundImages[1].Gradient.Stops[0].At.Percent, 1e-4, "a conic stop is of a turn")
}

func TestACalcHasNoLimitOfSteps(t *testing.T) {
	src := "calc(0px"
	for i := 1; i <= 40; i++ {
		src += " + min(1%, " + strconv.Itoa(i) + "px)"
	}
	src += ")"
	box := el(dom.Div, "box")
	root := page(box)
	Compute(root, []*Sheet{sheet(t, "#box { width: "+src+" }")})
	w := styleOf(box).Size.Width
	require.NotZero(t, w.Calc)
	want := float32(0)
	for i := 1; i <= 40; i++ {
		want += min(4, float32(i))
	}
	assert.InDelta(t, want, w.At(400), 1e-3)
}

func TestInfiniteLengthsAreTheLongestAndNaNIsZero(t *testing.T) {
	box := el(dom.Div, "box")
	root := page(box)
	Compute(root, []*Sheet{sheet(t, `#box { max-width: calc(infinity * 1px); margin-left: calc(-1px / 0);
width: calc(NaN * 1px); min-width: min(NaN * 1%, 5px); opacity: calc(infinity); padding-top: calc(1px * sign(-0)) }`)})
	s := styleOf(box)
	assert.Equal(t, float32(style.MaxLength), s.MaxSize.Width.Px)
	assert.Equal(t, float32(-style.MaxLength), s.Margin.Left.Px)
	assert.Zero(t, s.Size.Width.Px)
	assert.Zero(t, s.MinSize.Width.At(100))
	assert.Equal(t, float32(1), s.Opacity)
	assert.Zero(t, s.Padding.Top.Px)
}

func TestMathFunctionsInBasicShapesWaitForTheReferenceBox(t *testing.T) {
	s := gridOf(t, "font-size: 10px; clip-path: inset(min(10%, 5px) round(up, 20%, 3px) 0 clamp(1px, 50%, 2em))")
	in := s.ClipPath.Inset
	assert.InDelta(t, 5, in.Top.At(100), 1e-4)
	assert.InDelta(t, 2, in.Top.At(20), 1e-4)
	assert.InDelta(t, 21, in.Right.At(100), 1e-4)
	assert.InDelta(t, 20, in.Left.At(100), 1e-4)
	s = gridOf(t, "clip-path: xywh(min(10%, 5px) 0 max(50%, 30px) 10px)")
	assert.InDelta(t, 100-5-50, s.ClipPath.Inset.Right.At(100), 1e-4, "xywh()'s right is what x and the width leave")
	assert.InDelta(t, 40-4-30, s.ClipPath.Inset.Right.At(40), 1e-4)
	s = gridOf(t, "clip-path: polygon(min(50%, 20px) 0, 100% abs(-50%), 0 calc(100% * sign(50% - 10px)))")
	assert.InDelta(t, 20, s.ClipPath.Points[0][0].At(100), 1e-4)
	assert.InDelta(t, 50, s.ClipPath.Points[1][1].At(100), 1e-4)
	assert.InDelta(t, -10, s.ClipPath.Points[2][1].At(10), 1e-4)
	s = gridOf(t, "clip-path: circle(hypot(30%, 40px) at round(50%, 8px) 50%)")
	assert.InDelta(t, 104, s.ClipPath.At[0].At(200), 1e-4, "halfway rounds up")
}

func TestMathFunctionsReadContainerUnits(t *testing.T) {
	d, err := parseDim(valueOf(t, "round(up, calc(50cqw + 1em * sign(10cqh - 5px)), 7px)"))
	require.NoError(t, err)
	assert.NotZero(t, d.rel&containerUnits, "the units a container measures are read")
	u := &Units{Em: 10}
	u.Rel[unitCqw], u.Rel[unitCqh] = 3, 2
	assert.InDelta(t, 161, d.Absolute(u).Resolve(0), 1e-9)
}
