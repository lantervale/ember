package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"image/color"
	"testing"
)

func TestColours(t *testing.T) {
	for s, want := range map[string]color.NRGBA{
		"red":                      {255, 0, 0, 255},
		"RebeccaPurple":            {102, 51, 153, 255},
		"  DarkGrey ":              {169, 169, 169, 255},
		"transparent":              {},
		"#f80":                     {255, 136, 0, 255},
		"#F808":                    {255, 136, 0, 136},
		"#ff8800":                  {255, 136, 0, 255},
		"#ff880080":                {255, 136, 0, 128},
		"rgb(255, 136, 0)":         {255, 136, 0, 255},
		"rgb(100%,50%,0%)":         {255, 128, 0, 255},
		"rgb(255 136 0 / 50%)":     {255, 136, 0, 128},
		"rgba(255,136,0,0.5)":      {255, 136, 0, 128},
		"rgb(300, -5, 0)":          {255, 0, 0, 255},
		"hsl(0, 100%, 50%)":        {255, 0, 0, 255},
		"hsl(120, 100%, 25%)":      {0, 128, 0, 255},
		"hsl(210deg 50% 40%)":      {51, 102, 153, 255},
		"hsla(240, 100%, 50%, .5)": {0, 0, 255, 128},
		"hsl(-120, 100%, 50%)":     {0, 0, 255, 255},
		"hsl(0, 0%, 100%)":         {255, 255, 255, 255},
	} {
		got, ok := parseColor(s)
		if !ok || got != want {
			t.Errorf("%q: %v %v, want %v", s, got, ok, want)
		}
	}
	for _, s := range []string{
		"", "nocolour", "#12", "#12345", "#ggg", "rgb(1,2)", "rgb(a,b,c)", "hsl(0, 1, 1)", "var(--x)", "currentColor",
	} {
		if got, ok := parseColor(s); ok {
			t.Errorf("%q is read as %v", s, got)
		}
	}
}

func TestLengths(t *testing.T) {
	ctx := lengthContext{whole: 200, em: 10}
	for s, want := range map[string]float64{
		"12": 12, "12px": 12, "1.5\t": 1.5, "-3": -3, "1e1": 10,
		"72pt": 96, "6pc": 96, "1in": 96, "2.54cm": 96, "25.4mm": 96, "101.6Q": 96,
		"50%": 100, "2em": 20, "2ex": 10, "1rem": 10,
	} {
		if got, ok := length(s, ctx); !ok || !near(got, want, 1e-9) {
			t.Errorf("%q: %v %v, want %v", s, got, ok, want)
		}
	}
	for _, s := range []string{"", "px", "12 px px", "12furlongs", "auto"} {
		if got, ok := length(s, ctx); ok {
			t.Errorf("%q is read as %v", s, got)
		}
	}
}

func TestFractions(t *testing.T) {
	for s, want := range map[string]float64{"0.5": 0.5, "50%": 0.5, "2": 1, "-1": 0, ".25\n": 0.25} {
		if got, ok := fraction(s); !ok || got != want {
			t.Errorf("%q: %v %v, want %v", s, got, ok, want)
		}
	}
	if _, ok := fraction("half"); ok {
		t.Error("a word is read as a fraction")
	}
}

func TestTransforms(t *testing.T) {
	p := Point{10, 20}
	for s, want := range map[string]Point{
		"":                              {10, 20},
		"translate(5)":                  {15, 20},
		"translate(5, -5)":              {15, 15},
		"scale(2)":                      {20, 40},
		"scale(2 3)":                    {20, 60},
		"rotate(90)":                    {-20, 10},
		"rotate(90, 10, 10)":            {0, 10},
		"skewX(45)":                     {30, 20},
		"skewY(45)":                     {10, 30},
		"matrix(1 0 0 1 3 4)":           {13, 24},
		"matrix(0,1,-1,0,0,0)":          {-20, 10},
		"translate(10,0) scale(2)":      {30, 40}, // the last one applies first
		"scale(2),translate(10,0)":      {40, 40},
		"translate(4px, 6px)":           {14, 26},
		"rotate(90deg)":                 {-20, 10},
		"translateX(1) translateY(2)":   {11, 22},
		" translate( 5 )  scale ( 2 ) ": {25, 40},
	} {
		m, ok := parseTransform(s)
		if got := m.Apply(p); !ok || !near(got.X, want.X, 1e-9) || !near(got.Y, want.Y, 1e-9) {
			t.Errorf("%q: %v %v, want %v", s, got, ok, want)
		}
	}
	for _, s := range []string{"spin(3)", "translate(1", "rotate(1 2)", "matrix(1 2 3)", "scale()", "translate(5) junk"} {
		if m, ok := parseTransform(s); ok || m != Identity() {
			t.Errorf("%q is read as %v", s, m)
		}
	}
}

func TestMatrices(t *testing.T) {
	m := translate(5, 7).Mul(scale(2, 4))
	inv, ok := m.Inverse()
	if got := inv.Apply(m.Apply(Point{3, 9})); !ok || !near(got.X, 3, 1e-12) || !near(got.Y, 9, 1e-12) {
		t.Errorf("the inverse takes the point back to %v", got)
	}
	if _, ok = scale(0, 1).Inverse(); ok {
		t.Error("a flattening matrix has an inverse")
	}
	if got := m.stretch(); !near(got, 4, 1e-12) {
		t.Errorf("stretch %v, want 4", got)
	}
	rot, _ := parseTransform("rotate(30) scale(3)")
	if got := rot.stretch(); !near(got, 3, 1e-9) {
		t.Errorf("stretch of a turned scale %v, want 3", got)
	}
}
