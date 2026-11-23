package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"errors"
	"image/color"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

const xmlns = `xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"`

// picture parses body inside a root with a 100×100 view box.
func picture(t *testing.T, body string, opts ...Option) *Picture {
	t.Helper()
	p, err := NewPicture([]byte(`<svg `+xmlns+` viewBox="0 0 100 100">`+body+`</svg>`), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// shapes are the picture's shapes in paint order, groups opened.
func shapes(items []Item) []*Shape {
	var out []*Shape
	for _, it := range items {
		if it.Shape != nil {
			out = append(out, it.Shape)
		} else if it.Group != nil {
			out = append(out, shapes(it.Group.Items)...)
		}
	}
	return out
}

func only(t *testing.T, body string, opts ...Option) *Shape {
	t.Helper()
	all := shapes(picture(t, body, opts...).Items)
	if len(all) != 1 {
		t.Fatalf("%s: %d shapes, want 1", body, len(all))
	}
	return all[0]
}

//nolint:gochecknoglobals // the colours paints are expected to have
var (
	red   = color.NRGBA{255, 0, 0, 255}
	blue  = color.NRGBA{0, 0, 255, 255}
	green = color.NRGBA{0, 128, 0, 255}
	black = color.NRGBA{0, 0, 0, 255}
)

func TestShapesBecomePathsWithTheirPaints(t *testing.T) {
	for body, wantBounds := range map[string]Box{
		`<rect x="10" y="20" width="30" height="40"/>`:            {10, 20, 30, 40},
		`<rect width="50%" height="25%"/>`:                        {0, 0, 50, 25},
		`<rect x="10" y="10" width="30" height="40" rx="5"/>`:     {10, 10, 30, 40},
		`<circle cx="50" cy="50" r="20"/>`:                        {30, 30, 40, 40},
		`<circle cx="50%" cy="50%" r="10%"/>`:                     {40, 40, 20, 20},
		`<ellipse cx="50" cy="50" rx="20" ry="10"/>`:              {30, 40, 40, 20},
		`<line x1="10" y1="20" x2="30" y2="40" stroke="red"/>`:    {10, 20, 20, 20},
		`<polyline points="10,10 20,30 30,10"/>`:                  {10, 10, 20, 20},
		`<polygon points="10,10 20,30 30,10"/>`:                   {10, 10, 20, 20},
		`<path d="M10 10h20v20z"/>`:                               {10, 10, 20, 20},
		`<rect x="1em" width="2em" height="1ex" font-size="10"/>`: {10, 0, 20, 5},
		`<rect width="1in" height="12pt"/>`:                       {0, 0, 96, 16},
	} {
		s := only(t, body)
		got, _ := s.Path.Bounds()
		if !near(got.X, wantBounds.X, 0.01) || !near(got.Y, wantBounds.Y, 0.01) ||
			!near(got.W, wantBounds.W, 0.01) || !near(got.H, wantBounds.H, 0.01) {
			t.Errorf("%s: bounds %v, want %v", body, got, wantBounds)
		}
	}
	s := only(t, `<rect width="10" height="10"/>`)
	if s.Fill.Kind != Solid || s.Fill.Color != black || s.Fill.Opacity != 1 ||
		s.Stroke.Kind != NoPaint || s.StrokeWidth != 0 {
		t.Errorf("a plain shape: fill %+v stroke %+v", s.Fill, s.Stroke)
	}
	s = only(t, `<line x2="10" fill="red" stroke="blue"/>`)
	if s.Fill.Kind != NoPaint || s.Stroke.Color != blue || s.StrokeWidth != 1 {
		t.Errorf("a line: fill %+v stroke %+v", s.Fill, s.Stroke)
	}
	for _, body := range []string{
		`<rect width="0" height="10"/>`, `<circle r="0"/>`, `<path d=""/>`, `<path/>`, `<polygon points=""/>`,
		`<rect width="10" height="10" fill="none"/>`, `<rect width="10" height="10" fill="#0000"/>`,
		`<rect width="10" height="10" visibility="hidden"/>`,
		`<rect width="10" height="10" transform="scale(0)"/>`,
	} {
		if got := shapes(picture(t, body).Items); len(got) != 0 {
			t.Errorf("%s is drawn", body)
		}
	}
}

func TestPaintsAreRead(t *testing.T) {
	for attr, want := range map[string]Paint{
		`fill="red"`:                      {Kind: Solid, Color: red, Opacity: 1},
		`fill="#00f" fill-opacity=".5"`:   {Kind: Solid, Color: blue, Opacity: 0.5},
		`fill="#00f" fill-opacity="50%"`:  {Kind: Solid, Color: blue, Opacity: 0.5},
		`fill="currentColor"`:             {Kind: Solid, Color: black, Opacity: 1},
		`fill="currentColor" color="red"`: {Kind: Solid, Color: red, Opacity: 1},
		`fill="hsl(240, 100%, 50%)"`:      {Kind: Solid, Color: blue, Opacity: 1},
		`fill="url(#nowhere) red"`:        {Kind: Solid, Color: red, Opacity: 1},
		`fill="url(#nowhere)"`:            {},
		`fill="none"`:                     {},
		`fill="nonsense"`:                 {Kind: Solid, Color: black, Opacity: 1},
	} {
		all := shapes(picture(t, `<rect width="10" height="10" stroke="green" `+attr+`/>`).Items)
		if len(all) != 1 || all[0].Fill != want {
			t.Errorf("%s: %+v, want %+v", attr, all[0].Fill, want)
		}
	}
	s := only(t, `<rect width="10" height="10" fill="currentColor"/>`, CurrentColor(red))
	if s.Fill.Color != red {
		t.Errorf("currentColor with the option: %v", s.Fill.Color)
	}
	if got := picture(t, ``, CurrentColor(red)).CurrentColor(); got != red {
		t.Errorf("CurrentColor() = %v", got)
	}
}

func TestStrokesAreRead(t *testing.T) {
	s := only(t, `<path d="M0 0h10" stroke="red" stroke-width="3" stroke-linecap="round" stroke-linejoin="bevel"
		stroke-miterlimit="7" stroke-dasharray="4 2" stroke-dashoffset="1" stroke-opacity=".25" paint-order="stroke"/>`)
	if s.Stroke.Color != red || s.Stroke.Opacity != 0.25 || s.StrokeWidth != 3 ||
		s.Cap != RoundCap || s.Join != BevelJoin ||
		s.MiterLimit != 7 || !slices.Equal(s.Dashes, []float64{4, 2}) || s.DashOffset != 1 || !s.StrokeFirst {
		t.Errorf("stroke: %+v", s)
	}
	s = only(t, `<path d="M0 0h10" stroke="red"/>`)
	if s.StrokeWidth != 1 || s.Cap != ButtCap || s.Join != MiterJoin || s.MiterLimit != 4 ||
		s.Dashes != nil || s.StrokeFirst {
		t.Errorf("default stroke: %+v", s)
	}
	for attr, want := range map[string][]float64{
		`stroke-dasharray="5"`:       {5, 5},
		`stroke-dasharray="5,3,2"`:   {5, 3, 2, 5, 3, 2},
		`stroke-dasharray="none"`:    nil,
		`stroke-dasharray="0 0"`:     nil,
		`stroke-dasharray="5 -1"`:    nil,
		`stroke-dasharray="10% 1em"`: {10, 16},
	} {
		if s = only(t, `<path d="M0 0h10" stroke="red" `+attr+`/>`); !slices.Equal(s.Dashes, want) {
			t.Errorf("%s: %v, want %v", attr, s.Dashes, want)
		}
	}
	s = only(t, `<rect width="10" height="10" stroke="red" stroke-width="0"/>`)
	if s.Stroke.Kind != NoPaint {
		t.Errorf("a stroke of no width: %+v", s.Stroke)
	}
	s = only(t, `<rect width="10" height="10" stroke="red" stroke-width="10%"/>`)
	if s.StrokeWidth != 10 {
		t.Errorf("stroke-width 10%% of a 100×100 view box: %v", s.StrokeWidth)
	}
}

func TestAStyleAttributeWinsOverASheetOverAnAttribute(t *testing.T) {
	const css = `<style>rect { fill: blue } .c { stroke: blue } #i { stroke-width: 5 }</style>`
	s := only(t, css+`<rect width="10" height="10" fill="red"/>`)
	if s.Fill.Color != blue {
		t.Errorf("a sheet's rule over an attribute: %v", s.Fill.Color)
	}
	s = only(t, css+`<rect width="10" height="10" fill="red" style="fill: green"/>`)
	if s.Fill.Color != green {
		t.Errorf("a style attribute over a sheet's rule: %v", s.Fill.Color)
	}
	s = only(t, css+`<rect id="i" class="c" width="10" height="10" stroke="red" stroke-width="1"/>`)
	if s.Stroke.Color != blue || s.StrokeWidth != 5 {
		t.Errorf("class and id rules: %v %v", s.Stroke.Color, s.StrokeWidth)
	}
	s = only(t, `<style>#i { fill: red } rect { fill: blue } .c { fill: green }</style>`+
		`<rect id="i" class="c" width="1" height="1"/>`)
	if s.Fill.Color != red {
		t.Errorf("the most specific rule wins wherever it is: %v", s.Fill.Color)
	}
	s = only(t, `<style>.a { fill: red } .b { fill: blue }</style><rect class="b a" width="1" height="1"/>`)
	if s.Fill.Color != blue {
		t.Errorf("of two rules as specific the later wins: %v", s.Fill.Color)
	}
	s = only(t, `<style><![CDATA[ g > .a { fill: red } ]]></style><g><rect class="a" width="1" height="1"/></g>`)
	if s.Fill.Color != red {
		t.Errorf("a sheet in CDATA with a child selector: %v", s.Fill.Color)
	}
	s = only(t, `<defs><style type="text/css">.a{fill:red}</style></defs><rect class="a" width="1" height="1"/>`)
	if s.Fill.Color != red {
		t.Errorf("a sheet in defs: %v", s.Fill.Color)
	}
}

func TestPropertiesInheritOrDoNot(t *testing.T) {
	s := only(t, `<g fill="red" stroke="blue" stroke-width="3" fill-rule="evenodd" stroke-linecap="round" color="green">
		<g><rect width="10" height="10"/></g></g>`)
	if s.Fill.Color != red || s.Stroke.Color != blue || s.StrokeWidth != 3 || s.FillRule != EvenOdd || s.Cap != RoundCap {
		t.Errorf("inherited: %+v", s)
	}
	s = only(t, `<g fill="red"><rect width="10" height="10" fill="blue"/></g>`)
	if s.Fill.Color != blue {
		t.Errorf("an element's own value: %v", s.Fill.Color)
	}
	s = only(t, `<g fill="red"><rect width="10" height="10" fill="inherit"/></g>`)
	if s.Fill.Color != red {
		t.Errorf("inherit: %v", s.Fill.Color)
	}
	s = only(t, `<g color="red"><rect width="10" height="10" fill="currentColor" stroke="currentColor"/></g>`)
	if s.Fill.Color != red || s.Stroke.Color != red {
		t.Errorf("currentColor from a parent's color: %v %v", s.Fill.Color, s.Stroke.Color)
	}
	s = only(t, `<g font-size="10"><g font-size="2em"><rect width="1em" height="1"/></g></g>`)
	if b, _ := s.Path.Bounds(); b.W != 20 {
		t.Errorf("em of a font size in em: %v", b.W)
	}
	s = only(t, `<g transform="translate(10 0)"><rect width="1" height="1" transform="scale(2)"/></g>`)
	if got := s.Transform.Apply(Point{1, 1}); got != (Point{12, 2}) {
		t.Errorf("transforms nest: %v", got)
	}
	s = only(t, `<g visibility="hidden"><rect width="1" height="1" visibility="visible"/>`+
		`<rect width="2" height="2"/></g>`)
	if b, _ := s.Path.Bounds(); b.W != 1 {
		t.Errorf("visibility: the shape shown is %v wide", b.W)
	}
}

func TestDisplayNoneHidesAnElementAndItsChildren(t *testing.T) {
	// As Illustrator writes hidden layers: the children say inline, and stay hidden.
	p := picture(t, `<g display="none"><rect display="inline" width="1" height="1"/></g>
		<rect style="display:none" width="2" height="2"/>
		<g display="inline"><rect width="3" height="3"/></g>`)
	all := shapes(p.Items)
	if len(all) != 1 {
		t.Fatalf("%d shapes, want 1", len(all))
	}
	if b, _ := all[0].Path.Bounds(); b.W != 3 {
		t.Errorf("the shape shown is %v wide", b.W)
	}
}

func TestOpacityFoldsIntoOnePaintOrMakesAGroup(t *testing.T) {
	p := picture(t, `<rect width="1" height="1" opacity=".5" fill-opacity=".5"/>`)
	if s := p.Items[0].Shape; s == nil || s.Fill.Opacity != 0.25 {
		t.Errorf("one paint: %+v", p.Items[0])
	}
	p = picture(t, `<rect width="1" height="1" opacity=".5" fill="none" stroke="red"/>`)
	if s := p.Items[0].Shape; s == nil || s.Stroke.Opacity != 0.5 {
		t.Errorf("a stroke alone: %+v", p.Items[0])
	}
	p = picture(t, `<rect width="1" height="1" opacity=".5" stroke="red"/>`)
	if g := p.Items[0].Group; g == nil || g.Opacity != 0.5 || g.Items[0].Shape.Fill.Opacity != 1 {
		t.Errorf("fill and stroke: %+v", p.Items[0])
	}
	p = picture(t, `<g opacity=".5"><rect width="1" height="1"/><rect width="1" height="1"/></g>`)
	if g := p.Items[0].Group; g == nil || g.Opacity != 0.5 || len(g.Items) != 2 {
		t.Errorf("a group of two: %+v", p.Items[0])
	}
	p = picture(t, `<g opacity=".5"><g opacity=".5"><rect width="1" height="1"/><rect width="1" height="1"/></g></g>`)
	if g := p.Items[0].Group; g == nil || g.Opacity != 0.25 || len(g.Items) != 2 {
		t.Errorf("a group of one group: %+v", p.Items[0])
	}
	p = picture(t, `<g opacity="0"><rect width="1" height="1"/></g><g opacity=".5"></g><g>`+
		`<rect width="1" height="1"/></g>`)
	if len(p.Items) != 1 || p.Items[0].Shape == nil {
		t.Errorf("clear and empty groups: %+v", p.Items)
	}
}

func TestUseDrawsWhatItRefersTo(t *testing.T) {
	p := picture(t, `<defs><rect id="r" width="10" height="10"/></defs>
		<use href="#r" x="5" y="7" fill="red"/>
		<use xlink:href="#r" transform="translate(20 0)" fill="blue"/>`)
	all := shapes(p.Items)
	if len(all) != 2 {
		t.Fatalf("%d shapes, want 2", len(all))
	}
	if got := all[0].Transform.Apply(Point{}); got != (Point{5, 7}) || all[0].Fill.Color != red {
		t.Errorf("use with x and y: at %v, %v", got, all[0].Fill.Color)
	}
	if got := all[1].Transform.Apply(Point{}); got != (Point{20, 0}) || all[1].Fill.Color != blue {
		t.Errorf("use with a transform through xlink:href: at %v, %v", got, all[1].Fill.Color)
	}
	// Its own fill wins over the use's; a group brings its children.
	all = shapes(picture(t, `<g id="g"><rect fill="green" width="1" height="1"/><circle r="1"/></g>`+
		`<use href="#g" fill="red"/>`).Items)
	if len(all) != 4 || all[2].Fill.Color != green || all[3].Fill.Color != red {
		t.Errorf("use of a group: %d shapes", len(all))
	}
	for _, body := range []string{
		`<use href="#nowhere"/>`,
		`<use id="a" href="#a"/>`,
		`<g id="a"><use href="#a"/></g>`,
		`<defs><g id="a"><use href="#b"/></g><g id="b"><use href="#a"/></g></defs><use href="#a"/>`,
	} {
		if got := shapes(picture(t, body).Items); len(got) != 0 {
			t.Errorf("%s draws %d shapes", body, len(got))
		}
	}
}

func TestSwitchDrawsItsFirstChildWithoutACondition(t *testing.T) {
	all := shapes(picture(t, `<switch><foreignObject requiredExtensions="http://ns.adobe.com/"/>
		<g><rect width="3" height="3"/></g><rect width="5" height="5"/></switch>`).Items)
	if len(all) != 1 {
		t.Fatalf("%d shapes, want 1", len(all))
	}
	if b, _ := all[0].Path.Bounds(); b.W != 3 {
		t.Errorf("the shape shown is %v wide", b.W)
	}
}

func gradientOf(t *testing.T, body string) *Gradient {
	t.Helper()
	s := only(t, body)
	if s.Fill.Kind != GradientPaint {
		t.Fatalf("%s: fill %+v, want a gradient", body, s.Fill)
	}
	return s.Fill.Gradient
}

func TestLinearGradients(t *testing.T) {
	const rect = `<rect x="20" y="40" width="50" height="10" fill="url(#g)"/>`
	g := gradientOf(t, `<linearGradient id="g"><stop offset="0" stop-color="red"/>`+
		`<stop offset="100%" stop-color="blue" stop-opacity=".5"/></linearGradient>`+rect)
	if g.Radial || g.Spread != Pad || len(g.Stops) != 2 || g.Stops[0] != (Stop{0, red}) ||
		g.Stops[1] != (Stop{1, color.NRGBA{0, 0, 255, 128}}) {
		t.Errorf("stops: %+v", g)
	}
	// In bounding-box units the vector runs along the shape's top edge.
	if a, b := g.Transform.Apply(g.Start), g.Transform.Apply(g.End); a != (Point{20, 40}) || b != (Point{70, 40}) {
		t.Errorf("default vector: %v to %v", a, b)
	}
	g = gradientOf(t, `<linearGradient id="g" x1="0" y1="0" x2="0" y2="100%"><stop/><stop offset="1"/>`+
		`</linearGradient>`+rect)
	if a, b := g.Transform.Apply(g.Start), g.Transform.Apply(g.End); a != (Point{20, 40}) || b != (Point{20, 50}) {
		t.Errorf("a vertical vector: %v to %v", a, b)
	}
	g = gradientOf(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x1="10" x2="50%" `+
		`spreadMethod="reflect"><stop/><stop offset="1"/></linearGradient>`+rect)
	if a, b := g.Transform.Apply(g.Start), g.Transform.Apply(g.End); a != (Point{10, 0}) ||
		b != (Point{50, 0}) || g.Spread != Reflect {
		t.Errorf("user space: %v to %v, spread %v", a, b, g.Spread)
	}
	g = gradientOf(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x2="10" `+
		`gradientTransform="translate(5 5) rotate(90)" spreadMethod="repeat"><stop/><stop offset="1"/>`+
		`</linearGradient>`+rect)
	if b := g.Transform.Apply(g.End); !near(b.X, 5, 1e-9) || !near(b.Y, 15, 1e-9) || g.Spread != Repeat {
		t.Errorf("gradientTransform: the vector ends at %v", b)
	}
	// Offsets never go back, and are held to 0–1.
	g = gradientOf(t, `<linearGradient id="g"><stop offset=".5"/><stop offset=".2"/><stop offset="3"/>`+
		`</linearGradient>`+rect)
	if g.Stops[0].Offset != 0.5 || g.Stops[1].Offset != 0.5 || g.Stops[2].Offset != 1 {
		t.Errorf("offsets: %+v", g.Stops)
	}
	g = gradientOf(t, `<style>.s { stop-color: red }</style><linearGradient id="g"><stop class="s"/>`+
		`<stop offset="1" style="stop-color:#00f;stop-opacity:.5"/></linearGradient>`+rect)
	if g.Stops[0].Color != red || g.Stops[1].Color != (color.NRGBA{0, 0, 255, 128}) {
		t.Errorf("styled stops: %+v", g.Stops)
	}
}

func TestGradientsTakeFromTheOnesTheyReferTo(t *testing.T) {
	g := gradientOf(t, `<linearGradient id="stops">
		<stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>
		<linearGradient id="mid" href="#stops" gradientUnits="userSpaceOnUse" x2="40" spreadMethod="repeat"/>
		<linearGradient id="g" xlink:href="#mid" x1="20"/>
		<rect width="50" height="10" fill="url(#g)"/>`)
	if len(g.Stops) != 2 || g.Stops[1].Color != blue || g.Start != (Point{20, 0}) ||
		g.End != (Point{40, 0}) || g.Spread != Repeat {
		t.Errorf("chain: %+v", g)
	}
	g = gradientOf(t, `<linearGradient id="a" href="#b"><stop stop-color="red"/><stop offset="1"/></linearGradient>
		<linearGradient id="b" href="#a"/><rect width="50" height="10" fill="url(#b)"/>`)
	if len(g.Stops) != 2 {
		t.Errorf("a loop of references: %+v", g)
	}
	g = gradientOf(t, `<linearGradient id="l"><stop stop-color="red"/><stop offset="1"/></linearGradient>
		<radialGradient id="g" href="#l"/><rect width="50" height="10" fill="url(#g)"/>`)
	if !g.Radial || len(g.Stops) != 2 {
		t.Errorf("a radial gradient with a linear one's stops: %+v", g)
	}
}

func TestRadialGradients(t *testing.T) {
	const rect = `<rect x="20" y="40" width="40" height="20" fill="url(#g)"/>`
	g := gradientOf(t, `<radialGradient id="g"><stop/><stop offset="1"/></radialGradient>`+rect)
	if c := g.Transform.Apply(g.End); !g.Radial || c != (Point{40, 50}) ||
		g.Start != g.End || g.Radius != 0.5 || g.FocalRadius != 0 {
		t.Errorf("default circle: centre %v, %+v", c, g)
	}
	g = gradientOf(t, `<radialGradient id="g" gradientUnits="userSpaceOnUse" cx="30" cy="40" r="10%" fx="25" `+
		`fy="35" fr="2"><stop/><stop offset="1"/></radialGradient>`+rect)
	if g.End != (Point{30, 40}) || g.Start != (Point{25, 35}) || g.Radius != 10 || g.FocalRadius != 2 {
		t.Errorf("user space circle: %+v", g)
	}
}

func TestGradientsThatAreNone(t *testing.T) {
	const rect = `<rect width="50" height="10" fill="url(#g)" stroke="green"/>`
	for body, want := range map[string]Paint{
		`<linearGradient id="g"/>`: {},
		`<linearGradient id="g"><stop stop-color="red"/>` +
			`</linearGradient>`: {Kind: Solid, Color: red, Opacity: 1},
		`<linearGradient id="g" x2="0"><stop/><stop offset="1" stop-color="red"/>` +
			`</linearGradient>`: {Kind: Solid, Color: red, Opacity: 1},
		`<radialGradient id="g" r="0"><stop/><stop offset="1" stop-color="red"/>` +
			`</radialGradient>`: {Kind: Solid, Color: red, Opacity: 1},
		`<pattern id="g"/>`: {},
		`<rect id="g"/>`:    {},
	} {
		all := shapes(picture(t, body+rect).Items)
		if got := all[len(all)-1].Fill; got != want {
			t.Errorf("%s: %+v, want %+v", body, got, want)
		}
	}
	// A gradient in bounding-box units paints nothing on a shape with no area.
	all := shapes(picture(t, `<linearGradient id="g"><stop/><stop offset="1"/></linearGradient>`+
		`<path d="M0 5h50" fill="none" stroke="url(#g)"/>`).Items)
	if len(all) != 0 {
		t.Errorf("a flat line stroked with a bounding-box gradient is drawn")
	}
}

func TestIntrinsicSize(t *testing.T) {
	// As Chrome sizes an img of each: naturalWidth × naturalHeight, and what CSS gets.
	for attrs, want := range map[string]struct {
		w, h       float64
		hasW, hasH bool
		ratio      float64
	}{
		`width="40" height="30"`:                       {40, 30, true, true, 4.0 / 3},
		`width="40" height="30" viewBox="0 0 100 100"`: {40, 30, true, true, 1},
		`width="40"`:                        {40, 150, true, false, 0},
		`height="30"`:                       {300, 30, false, true, 0},
		`width="40" viewBox="0 0 200 100"`:  {40, 20, true, true, 2},
		`height="30" viewBox="0 0 200 100"`: {60, 30, true, true, 2},
		`viewBox="0 0 200 100"`:             {300, 150, false, false, 2},
		``:                                  {300, 150, false, false, 0},
		`width="100%" height="100%" viewBox="0 0 200 100"`: {300, 150, false, false, 2},
		`width="50%" height="100%"`:                        {300, 150, false, false, 0},
		`width="1in" height="10mm" viewBox="0 0 200 100"`:  {96, 37.795, true, true, 2},
		`width="2em" height="1em"`:                         {32, 16, true, true, 2},
		`width="40" height="30" viewBox="0 0 0 100"`:       {40, 30, true, true, 4.0 / 3},
		`width="-40" height="30" viewBox="0 0 100 100"`:    {0, 30, true, true, 1},
		`width="0" height="30"`:                            {0, 30, true, true, 0},
		`width="40px" height="30px" viewBox="0,0,80,60"`:   {40, 30, true, true, 4.0 / 3},
	} {
		p, err := NewPicture([]byte(`<svg ` + xmlns + ` ` + attrs + `/>`))
		if err != nil {
			t.Fatal(err)
		}
		w, h := p.Size()
		_, hasW := p.Width()
		_, hasH := p.Height()
		ratio, _ := p.Ratio()
		if !near(w, want.w, 0.01) || !near(h, want.h, 0.01) || hasW != want.hasW ||
			hasH != want.hasH || !near(ratio, want.ratio, 1e-9) {
			t.Errorf("%s: size %v×%v (%v, %v) ratio %v, want %+v", attrs, w, h, hasW, hasH, ratio, want)
		}
	}
}

func TestViewTransformFitsTheViewBox(t *testing.T) {
	// A 100×50 view box at (10, 20) in a 100×100 viewport: meet leaves 50 of height, slice hides 100 of width.
	for par, want := range map[string]Matrix{
		``:               {1, 0, 0, 1, -10, 5},
		`xMidYMid meet`:  {1, 0, 0, 1, -10, 5},
		`xMinYMin`:       {1, 0, 0, 1, -10, -20},
		`xMidYMin`:       {1, 0, 0, 1, -10, -20},
		`xMaxYMin`:       {1, 0, 0, 1, -10, -20},
		`xMinYMid`:       {1, 0, 0, 1, -10, 5},
		`xMaxYMid`:       {1, 0, 0, 1, -10, 5},
		`xMinYMax`:       {1, 0, 0, 1, -10, 30},
		`xMidYMax`:       {1, 0, 0, 1, -10, 30},
		`xMaxYMax`:       {1, 0, 0, 1, -10, 30},
		`xMinYMin slice`: {2, 0, 0, 2, -20, -40},
		`xMidYMin slice`: {2, 0, 0, 2, -70, -40},
		`xMaxYMin slice`: {2, 0, 0, 2, -120, -40},
		`xMinYMid slice`: {2, 0, 0, 2, -20, -40},
		`xMidYMid slice`: {2, 0, 0, 2, -70, -40},
		`xMaxYMid slice`: {2, 0, 0, 2, -120, -40},
		`xMinYMax slice`: {2, 0, 0, 2, -20, -40},
		`xMidYMax slice`: {2, 0, 0, 2, -70, -40},
		`xMaxYMax slice`: {2, 0, 0, 2, -120, -40},
		`none`:           {1, 0, 0, 2, -10, -40},
		`none slice`:     {1, 0, 0, 2, -10, -40},
		`defer xMinYMax`: {1, 0, 0, 1, -10, 30},
		`xMiddleYMid`:    {1, 0, 0, 1, -10, 5},
	} {
		p, err := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="10 20 100 50" preserveAspectRatio="` + par + `"/>`))
		if err != nil {
			t.Fatal(err)
		}
		if got := p.ViewTransform(100, 100); got != want {
			t.Errorf("%q: %v, want %v", par, got, want)
		}
	}
	// A tall viewport: meet leaves width, the x alignment shows.
	p, _ := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 50 100" preserveAspectRatio="xMaxYMid"/>`))
	if got := p.ViewTransform(100, 100); got != (Matrix{1, 0, 0, 1, 50, 0}) {
		t.Errorf("xMax in a wide viewport: %v", got)
	}
	// Without a view box the picture is stretched from its own size, an axis it gives none for kept as it is.
	for attrs, want := range map[string]Matrix{
		`width="40" height="30"`: {3, 0, 0, 4, 0, 0},
		`width="40"`:             {3, 0, 0, 1, 0, 0},
		`height="30"`:            {1, 0, 0, 4, 0, 0},
		``:                       {1, 0, 0, 1, 0, 0},
	} {
		p, _ = NewPicture([]byte(`<svg ` + xmlns + ` ` + attrs + `/>`))
		if got := p.ViewTransform(120, 120); got != want {
			t.Errorf("%q: %v, want %v", attrs, got, want)
		}
	}
}

func TestUnsupportedNamesWhatIsNotDrawn(t *testing.T) {
	for body, want := range map[string][]string{
		`<rect width="1" height="1"/>`:                         nil,
		`<text>hello</text>`:                                   nil,
		`<text>a <textPath href="#p">b</textPath></text>`:      nil,
		`<image href="a.png"/>`:                                {"external reference"},
		`<foreignObject/>`:                                     {"foreignObject"},
		`<rect width="1" height="1" clip-path="url(#c)"/>`:     nil,
		`<rect width="1" height="1" clip-path="circle(1px)"/>`: nil,
		`<rect width="1" height="1" clip-path="star(1px)"/>`:   {"clip-path"},
		`<g mask="url(#m)"/>`:                                  nil,
		`<g style="filter:url(#f)"/>`:                          nil,
		`<filter id="f"><feTurbulence/></filter><rect width="1" height="1" filter="url(#f)"/>`: {"feTurbulence"},
		`<filter id="f"><feBlend in2="BackgroundImage"/></filter><rect width="1" height="1" filter="url(#f)"/>`: {
			"BackgroundImage",
		},
		`<rect width="1" height="1" style="filter: wobble(2)"/>`:                            {"filter"},
		`<g filter="none" clip-path="none" mask="none"/>`:                                   nil,
		`<pattern id="p"/><rect width="1" height="1" fill="url(#p)"/>`:                      nil,
		`<symbol id="s"/><use href="#s"/>`:                                                  nil,
		`<svg><rect width="1" height="1"/></svg>`:                                           nil,
		`<path d="M0 0h5" stroke="red" marker-end="url(#m)"/>`:                              nil,
		`<g marker-start="url(#m)"><rect width="1" height="1"/></g>`:                        nil,
		`<use href="other.svg#a"/>`:                                                         {"external reference"},
		`<rect width="1" height="1" fill="url(other.svg#g)"/>`:                              {"external reference"},
		`<rect width="1" height="1"><animate attributeName="x"/></rect><animateTransform/>`: {"animation"},
		`<rect width="1" height="1" style="mix-blend-mode:multiply"/>`:                      nil,
		`<rect width="1" height="1" style="mix-blend-mode:plus-darker"/>`:                   {"mix-blend-mode"},
		`<rect width="1" height="1" vector-effect="non-scaling-stroke"/>`:                   nil,
		`<rect width="1" height="1" vector-effect="non-rotation"/>`:                         {"vector-effect"},
		`<style>@media print { a { b: c } } rect:hover { fill: ` +
			`red }</style>`: {"@media", "style selector"},
		`<g display="none"><text>hidden</text></g>`:                         nil,
		`<defs><text id="t">kept for later</text><clipPath id="c"/></defs>`: nil,
		`<text>a</text><image href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg'%3E` +
			`%3CforeignObject/%3E%3C/svg%3E"/>`: {"foreignObject"},
	} {
		if got := picture(t, body).Unsupported(); !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", body, got, want)
		}
	}
}

func TestFilesThatAreNoPicture(t *testing.T) {
	for _, data := range []string{
		``, `hello`, `<html><body/></html>`, `<svg`, "\xff\xfe<\x00s\x00v\x00g\x00",
		`<svg xmlns="http://example.com/other"/>`, `<?xml version="1.0"?>`,
	} {
		var format *FormatError
		if _, err := NewPicture([]byte(data)); !errors.As(err, &format) {
			t.Errorf("%q: %v, want a FormatError", data, err)
		}
	}
	for _, data := range []string{
		`<svg/>`, `<svg><g></svg>`, `<svg xmlns="http://www.w3.org/2000/svg"></svg> trailing <junk`,
		`<?xml version="1.0" encoding="ISO-8859-1"?><svg>` +
			`<title>caf` + "\xe9" + `</title><rect width="1" height="1"/></svg>`,
		`<s:svg xmlns:s="http://www.w3.org/2000/svg"><s:rect width="1" height="1"/></s:svg>`,
	} {
		if _, err := NewPicture([]byte(data)); err != nil {
			t.Errorf("%q: %v", data, err)
		}
	}
}

func TestEntitiesOfTheDoctypeAreExpanded(t *testing.T) {
	// As Illustrator writes its namespaces and styles.
	p, err := NewPicture([]byte(`<?xml version="1.0"?>
		<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
			<!ENTITY ns_svg "http://www.w3.org/2000/svg">
			<!ENTITY st0 "fill:#FF0000;">
			<!ENTITY nested "&st0;">
		]>
		<svg xmlns="&ns_svg;"><rect style="&st0;" width="1" height="1"/><rect style="&nested;" width="1" height="1"/>
		<style>circle { &st0; }</style><circle r="1"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	all := shapes(p.Items)
	if len(all) != 3 || all[0].Fill.Color != red || all[1].Fill.Color != black || all[2].Fill.Color != red {
		t.Errorf("entities: %d shapes", len(all))
	}
}

func TestLimits(t *testing.T) {
	doc := func(body string) []byte { return []byte(`<svg ` + xmlns + `>` + body + `</svg>`) }
	tooComplex := map[string][]byte{
		"elements": doc(strings.Repeat(`<g/>`, MaxElements)),
		"depth":    doc(strings.Repeat(`<g>`, MaxDepth) + strings.Repeat(`</g>`, MaxDepth)),
		"shapes":   doc(`<rect id="r" width="1" height="1"/>` + strings.Repeat(`<use href="#r"/>`, MaxShapes)),
		"segments": doc(`<path id="p" d="M0 0` + strings.Repeat(` 1 1`, 50_000) + `"/>` +
			strings.Repeat(`<use href="#p"/>`, MaxSegments/50_000)),
		// Ten uses of ten uses…: a few hundred bytes asking for 10¹⁰ shapes.
		"use bomb": doc(func() string {
			s := `<rect id="u0" width="1" height="1"/>`
			for i := 1; i <= 10; i++ {
				s += `<g id="u` + itoa(i) + `">` + strings.Repeat(`<use href="#u`+itoa(i-1)+`"/>`, 10) + `</g>`
			}
			return s
		}()),
		"use depth": doc(func() string {
			var s strings.Builder
			s.WriteString(`<rect id="d0" width="1" height="1"/>`)
			for i := 1; i <= MaxUseDepth+1; i++ {
				s.WriteString(`<use id="d` + itoa(i) + `" href="#d` + itoa(i-1) + `"/>`)
			}
			return s.String()
		}()),
		// A billion laughs: entities are not expanded inside one another, and what is expanded is counted.
		"entities": []byte(`<!DOCTYPE svg [<!ENTITY a "` + strings.Repeat("x", 1000) + `">]><svg ` + xmlns + `>` +
			strings.Repeat(`<g id="&a;&a;&a;&a;"/>`, 1000) + `</svg>`),
	}
	for name, data := range tooComplex {
		if _, err := NewPicture(data); !errors.Is(err, ErrTooComplex) {
			t.Errorf("%s: %v, want ErrTooComplex", name, err)
		}
	}
	laughs := `<!DOCTYPE svg [<!ENTITY a "ha"><!ENTITY b "&a;&a;&a;&a;&a;&a;&a;&a;"><!ENTITY c "&b;&b;&b;&b;&b;&b;&b;&b;">
		<!ENTITY d "&c;&c;&c;&c;&c;&c;&c;&c;"><!ENTITY e "&d;&d;&d;&d;&d;&d;&d;&d;">]>
		<svg ` + xmlns + `><rect id="&e;&e;&e;&e;" width="1" height="1"/></svg>`
	p, err := NewPicture([]byte(laughs))
	if err != nil || len(p.Items) != 1 {
		t.Errorf("nested entities: %v", err)
	}
	if _, err = NewPicture(make([]byte, MaxBytes+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("a file over MaxBytes: %v", err)
	}
	p = picture(t, `<rect width="100" height="100"/>`)
	for _, size := range [][2]int{{0, 10}, {10, -1}, {MaxSide + 1, 1}, {1, MaxSide + 1}, {MaxSide, MaxSide}} {
		if _, err = p.Rasterise(size[0], size[1]); !errors.Is(err, ErrTooLarge) {
			t.Errorf("Rasterise(%d, %d): %v, want ErrTooLarge", size[0], size[1], err)
		}
	}
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}

func TestLoadPictureReadsAFileSystem(t *testing.T) {
	fsys := fstest.MapFS{
		"icon.svg": {Data: []byte(`<svg ` + xmlns + ` width="24" height="24"><rect width="1" height="1"/></svg>`)},
		"bad.svg":  {Data: []byte(`nothing`)},
	}
	p, err := LoadPicture(fsys, "icon.svg", CurrentColor(red))
	if err != nil || len(p.Items) != 1 || p.CurrentColor() != red {
		t.Errorf("LoadPicture: %v", err)
	}
	if _, err = LoadPicture(fsys, "missing.svg"); err == nil {
		t.Error("a missing file loads")
	}
	var format *FormatError
	if _, err = LoadPicture(fsys, "bad.svg"); !errors.As(err, &format) || !strings.Contains(err.Error(), "bad.svg") {
		t.Errorf("a bad file: %v", err)
	}
}
