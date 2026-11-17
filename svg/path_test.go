package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"math"
	"slices"
	"testing"
)

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func samePoints(a, b []Point, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !near(a[i].X, b[i].X, eps) || !near(a[i].Y, b[i].Y, eps) {
			return false
		}
	}
	return true
}

func TestPathDataReadsEveryCommand(t *testing.T) {
	abs := parsePath("M10 10 L20 10 H30 V20 C30 30 20 30 20 20 S10 10 5 5 Q0 0 5 0 T10 10 A5 5 0 0 1 20 20 Z", 1000)
	want := []Verb{MoveTo, LineTo, LineTo, LineTo, CubicTo, CubicTo, QuadTo, QuadTo, CubicTo, CubicTo, Close}
	if !slices.Equal(abs.Verbs, want) {
		t.Fatalf("verbs %v, want %v", abs.Verbs, want)
	}
	rel := parsePath("m10 10 l10 0 h10 v10 c0 10 -10 10 -10 0 s-10 -10 -15 -15 q-5 -5 0 -5 t5 10 a5 5 0 0 1 10 10 z", 1000)
	if !slices.Equal(rel.Verbs, want) || !samePoints(rel.Points, abs.Points, 1e-9) {
		t.Errorf("relative commands gave\n%v\nwant\n%v", rel.Points, abs.Points)
	}
}

func TestPathDataRepeatsACommandOverFurtherNumbers(t *testing.T) {
	for d, want := range map[string]Path{
		"M0 0 10 0 10 10": {[]Verb{MoveTo, LineTo, LineTo}, []Point{{0, 0}, {10, 0}, {10, 10}}},
		"m1 1 1 0 0 1":    {[]Verb{MoveTo, LineTo, LineTo}, []Point{{1, 1}, {2, 1}, {2, 2}}},
		"M0 0L1 1 2 2":    {[]Verb{MoveTo, LineTo, LineTo}, []Point{{0, 0}, {1, 1}, {2, 2}}},
		"M0 0h1 2v1 2":    {[]Verb{MoveTo, LineTo, LineTo, LineTo, LineTo}, []Point{{0, 0}, {1, 0}, {3, 0}, {3, 1}, {3, 3}}},
		"M0 0C1 1 2 2 3 3 4 4 5 5 6 6": {
			[]Verb{MoveTo, CubicTo, CubicTo},
			[]Point{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}},
		},
		"M.5.5-1-1":         {[]Verb{MoveTo, LineTo}, []Point{{0.5, 0.5}, {-1, -1}}},
		"M1e1-.5e-1,2 3":    {[]Verb{MoveTo, LineTo}, []Point{{10, -0.05}, {2, 3}}},
		"M10 10L20 10ZL0 0": {[]Verb{MoveTo, LineTo, Close, MoveTo, LineTo}, []Point{{10, 10}, {20, 10}, {10, 10}, {0, 0}}},
	} {
		got := parsePath(d, 1000)
		if !slices.Equal(got.Verbs, want.Verbs) || !samePoints(got.Points, want.Points, 1e-12) {
			t.Errorf("%q: %v %v, want %v %v", d, got.Verbs, got.Points, want.Verbs, want.Points)
		}
	}
}

func TestPathDataReadsArcFlagsWrittenTogether(t *testing.T) {
	packed := parsePath("M0 0a5 5 0 0110 0", 100)
	spaced := parsePath("M0 0 a 5 5 0 0 1 10 0", 100)
	if len(packed.Verbs) < 2 || !slices.Equal(packed.Verbs, spaced.Verbs) || !samePoints(packed.Points, spaced.Points, 0) {
		t.Errorf("packed flags gave %v %v, spaced %v %v", packed.Verbs, packed.Points, spaced.Verbs, spaced.Points)
	}
}

func TestPathDataKeepsWhatComesBeforeAnError(t *testing.T) {
	for d, want := range map[string][]Verb{
		"":                          nil,
		"L10 10":                    nil,
		"10 10":                     nil,
		"M0 0 L10 10 L 20":          {MoveTo, LineTo},
		"M0 0 X 5 5":                {MoveTo},
		"M0 0 L1e999 0 L1 1":        {MoveTo},
		"M0 0 A5 5 0 2 1 10 0 L5 5": {MoveTo},
		"M0 0 Z 5 5":                {MoveTo, Close},
		"M0":                        nil,
	} {
		if got := parsePath(d, 1000); !slices.Equal(got.Verbs, want) {
			t.Errorf("%q: %v, want %v", d, got.Verbs, want)
		}
	}
}

func TestPathDataStopsAtItsLimit(t *testing.T) {
	if got := parsePath("M0 0 1 1 2 2 3 3 4 4 5 5", 3); len(got.Verbs) != 3 {
		t.Errorf("%d verbs, want 3", len(got.Verbs))
	}
}

func TestSmoothCurvesMirrorTheControlPointBefore(t *testing.T) {
	p := parsePath("M0 0 C0 10 10 10 10 0 S20 -10 20 0", 100)
	if got := p.Points[4]; got != (Point{10, -10}) {
		t.Errorf("S after C starts from %v, want {10 -10}", got)
	}
	p = parsePath("M0 0 L10 0 S20 -10 20 0", 100)
	if got := p.Points[2]; got != (Point{10, 0}) {
		t.Errorf("S after L starts from %v, want the current point", got)
	}
	p = parsePath("M0 0 Q5 10 10 0 T20 0", 100)
	if got := p.Points[3]; got != (Point{15, -10}) {
		t.Errorf("T after Q turns round %v, want {15 -10}", got)
	}
	p = parsePath("M0 0 T20 0", 100)
	if got := p.Points[1]; got != (Point{0, 0}) {
		t.Errorf("T alone turns round %v, want the current point", got)
	}
}

func TestArcs(t *testing.T) {
	bounds := func(d string) Box {
		b, _ := parsePath(d, 100).Bounds()
		return b
	}
	for d, want := range map[string]Box{
		"M0 0 A10 10 0 0 1 20 0":   {0, -10, 20, 10}, // half a circle, over the top
		"M0 0 A10 10 0 0 0 20 0":   {0, 0, 20, 10},   // under
		"M0 0 A10 10 0 1 1 10 10":  {0, -10, 20, 20}, // three quarters
		"M0 0 A1 1 0 0 1 20 0":     {0, -10, 20, 10}, // radii too short grow to reach
		"M0 0 A10 5 90 0 1 10 0":   {0, -10, 10, 10}, // an ellipse turned a quarter
		"M0 0 A0 10 0 0 1 20 0":    {0, 0, 20, 0},    // no radius: a line
		"M0 0 A-10 -10 0 0 1 20 0": {0, -10, 20, 10}, // negative radii count as positive
		"M5 5 A10 10 0 0 1 5 5":    {5, 5, 0, 0},     // to where it is: nothing
	} {
		got := bounds(d)
		if !near(got.X, want.X, 0.01) || !near(got.Y, want.Y, 0.01) ||
			!near(got.W, want.W, 0.01) || !near(got.H, want.H, 0.01) {
			t.Errorf("%q: bounds %v, want %v", d, got, want)
		}
	}
	p := parsePath("M0 0 A10 10 0 0 1 20 0", 100)
	if last := p.Points[len(p.Points)-1]; last != (Point{20, 0}) {
		t.Errorf("an arc ends at %v, want exactly {20 0}", last)
	}
}

func TestBoundsMeasuresCurvesWhereTheyTurn(t *testing.T) {
	b, ok := parsePath("M0 0 C0 100 100 100 100 0", 10).Bounds()
	if !ok || !near(b.H, 75, 1e-9) || b.W != 100 {
		t.Errorf("bounds %v, want 100 wide and 75 high", b)
	}
	b, _ = parsePath("M0 0 Q50 100 100 0", 10).Bounds()
	if !near(b.H, 50, 1e-9) {
		t.Errorf("a quadratic's bounds %v, want 50 high", b)
	}
	if _, ok = (Path{}).Bounds(); ok {
		t.Error("an empty path has bounds")
	}
}

func TestBasicShapesArePaths(t *testing.T) {
	r := rectPath(10, 20, 30, 40, 0, 0)
	if !slices.Equal(r.Verbs, []Verb{MoveTo, LineTo, LineTo, LineTo, Close}) ||
		!samePoints(r.Points, []Point{{10, 20}, {40, 20}, {40, 60}, {10, 60}}, 0) {
		t.Errorf("rect: %v %v", r.Verbs, r.Points)
	}
	round := rectPath(0, 0, 100, 40, 80, 80)
	if b, _ := round.Bounds(); !near(b.W, 100, 1e-9) || !near(b.H, 40, 1e-9) {
		t.Errorf("a round rect's bounds %v", b)
	}
	// Radii are held to half the sides: the corner (0, 0) is cut by a 50×20 ellipse.
	if round.Points[0] != (Point{50, 0}) {
		t.Errorf("a round rect starts at %v, want {50 0}", round.Points[0])
	}
	if got := rectPath(0, 0, -5, 10, 0, 0); len(got.Verbs) != 0 {
		t.Error("a rect of negative width is drawn")
	}
	c := ellipsePath(Point{50, 50}, 20, 10)
	if b, _ := c.Bounds(); !near(b.X, 30, 1e-3) || !near(b.Y, 40, 1e-3) || !near(b.W, 40, 1e-3) || !near(b.H, 20, 1e-3) {
		t.Errorf("ellipse bounds %v", b)
	}
	if got := ellipsePath(Point{}, 0, 5); len(got.Verbs) != 0 {
		t.Error("an ellipse of no radius is drawn")
	}
	poly := pointsPath("0,0 10,0 10,10 5", true, 100)
	if !slices.Equal(poly.Verbs, []Verb{MoveTo, LineTo, LineTo, Close}) {
		t.Errorf("polygon: %v", poly.Verbs)
	}
	line := pointsPath("0,0 10,0", false, 100)
	if !slices.Equal(line.Verbs, []Verb{MoveTo, LineTo}) {
		t.Errorf("polyline: %v", line.Verbs)
	}
}
