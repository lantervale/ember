package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"image/color"
	"math"
	"slices"
	"testing"
)

func TestImportantDeclarationsWinOverTheOthers(t *testing.T) {
	for body, want := range map[string]color.NRGBA{
		`<style>rect { fill: red !important }</style><rect width="1" height="1" style="fill: blue"/>`:            red,
		`<style>#i { fill: red } rect { fill: blue !important }</style><rect id="i" width="1" height="1"/>`:      blue,
		`<style>rect { fill: red !important }</style><rect width="1" height="1" style="fill: blue !important"/>`: blue,
		`<rect width="1" height="1" style="fill: blue !important; fill: red"/>`:                                  blue,
	} {
		if got := only(t, body).Fill.Color; got != want {
			t.Errorf("%s: %v, want %v", body, got, want)
		}
	}
}

func TestARootTransformTurnsThePictureAboutItsCentre(t *testing.T) {
	p, err := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 10 10" transform="rotate(90)"/>`))
	if err != nil {
		t.Fatal(err)
	}
	m := p.ViewTransform(100, 100)
	if got := m.Apply(Point{0, 0}); math.Abs(got.X-100) > 1e-9 || math.Abs(got.Y) > 1e-9 {
		t.Errorf("the top-left corner turned a quarter about the centre: %v", got)
	}
	img := draw(t, ``)
	if img.Rect.Dx() != 100 {
		t.Fatal("no picture")
	}
	p, _ = NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 10 10" transform="scale(.5)">` +
		`<rect width="10" height="10" fill="red"/></svg>`))
	img, _ = p.Rasterise(100, 100)
	expect(t, img, "outside the halved picture", 10, 10, clearPx, 0)
	expect(t, img, "inside it", 30, 30, redPx, 0)
}

func TestClipPathsKeepWhatTheirShapesCover(t *testing.T) {
	img := draw(t, `<clipPath id="c"><rect x="20" y="20" width="20" height="20" fill="none"/></clipPath>`+
		`<rect width="100" height="100" fill="red" clip-path="url(#c)"/>`)
	expect(t, img, "inside the clip", 30, 30, redPx, 0)
	expect(t, img, "outside it", 50, 50, clearPx, 0)
	img = draw(t, `<clipPath id="c" clipPathUnits="objectBoundingBox"><rect width=".5" height=".5"/></clipPath>`+
		`<rect x="20" y="20" width="60" height="60" fill="red" clip-path="url(#c)"/>`)
	expect(t, img, "in the box's first quarter", 30, 30, redPx, 0)
	expect(t, img, "past it", 60, 60, clearPx, 0)
	img = draw(t, `<clipPath id="c"><path clip-rule="evenodd" d="M0 0h100v100H0zM40 40h20v20H40z"/></clipPath>`+
		`<rect width="100" height="100" fill="red" clip-path="url(#c)"/>`)
	expect(t, img, "the hole of an even-odd clip", 50, 50, clearPx, 0)
	expect(t, img, "round it", 20, 20, redPx, 0)
	img = draw(t, `<clipPath id="a"><rect width="50" height="100"/></clipPath>`+
		`<clipPath id="b" clip-path="url(#a)"><rect width="100" height="50"/></clipPath>`+
		`<rect width="100" height="100" fill="red" clip-path="url(#b)"/>`)
	expect(t, img, "in both clips", 25, 25, redPx, 0)
	expect(t, img, "in one alone", 75, 25, clearPx, 0)
	img = draw(t, `<clipPath id="c"><g><rect width="100" height="100"/></g></clipPath>`+
		`<rect width="100" height="100" fill="red" clip-path="url(#c)"/>`)
	expect(t, img, "a group in a clip path keeps nothing", 50, 50, clearPx, 0)
	img = draw(t, `<clipPath id="c"><rect width="10" height="10" clip-path="url(#c)"/></clipPath>`+
		`<rect width="100" height="100" fill="red" clip-path="url(#c)"/>`+
		`<rect width="100" height="100" fill="blue" clip-path="url(#none)"/>`)
	expect(t, img, "a clip naming nothing clips nothing", 50, 50, bluePx, 0)
}

func TestMasksShowWhatIsBrightOrOpaque(t *testing.T) {
	img := draw(t, `<mask id="m"><rect width="50" height="100" fill="white"/>`+
		`<rect x="50" width="50" height="100" fill="black"/></mask>`+
		`<rect width="100" height="100" fill="red" mask="url(#m)"/>`)
	expect(t, img, "under white", 25, 50, redPx, 0)
	expect(t, img, "under black", 75, 50, clearPx, 0)
	img = draw(t, `<mask id="m" style="mask-type: alpha">`+
		`<rect width="100" height="100" fill="black" fill-opacity=".5"/></mask>`+
		`<rect width="100" height="100" fill="red" mask="url(#m)"/>`)
	expect(t, img, "an alpha mask half clear", 50, 50, halfRed, 1)
	img = draw(t, `<mask id="m" x="0" y="0" width=".5" height="1"><rect width="100" height="100" fill="white"/></mask>`+
		`<rect width="100" height="100" fill="red" mask="url(#m)"/>`)
	expect(t, img, "inside the mask's region", 25, 50, redPx, 0)
	expect(t, img, "outside it", 75, 50, clearPx, 0)
}

func TestPatternsRepeatTheirTile(t *testing.T) {
	img := draw(t, `<pattern id="p" patternUnits="userSpaceOnUse" width="20" height="20">`+
		`<rect width="10" height="10" fill="red"/></pattern><rect width="100" height="100" fill="url(#p)"/>`)
	for _, at := range [][2]int{{5, 5}, {25, 45}, {85, 65}} {
		expect(t, img, "a tile's square", at[0], at[1], redPx, 0)
	}
	expect(t, img, "between them", 15, 15, clearPx, 0)
	s := only(t, `<pattern id="p" width=".5" height=".25"><rect width="1" height="1"/></pattern>`+
		`<rect x="10" y="20" width="40" height="40" fill="url(#p)"/>`)
	if s.Fill.Kind != PatternPaint || s.Fill.Pattern.Tile != (Box{0, 0, 20, 10}) ||
		s.Fill.Pattern.Transform != translate(10, 20) {
		t.Errorf("a pattern in bounding-box units: %+v", s.Fill.Pattern)
	}
	all := shapes(picture(t, `<pattern id="p" width="1" height="1"><rect width="1" height="1" fill="url(#p)"/></pattern>`+
		`<rect width="40" height="40" fill="url(#p) red"/>`).Items)
	if len(all) != 0 {
		t.Errorf("a pattern painting with itself paints nothing: %+v", all[0].Fill)
	}
}

func TestNestedSvgsAndSymbolsFitTheirViewBoxes(t *testing.T) {
	img := draw(t, `<svg x="10" y="10" width="20" height="20" viewBox="0 0 2 2"><rect width="1" height="1" fill="red"/>`+
		`<circle cx="1" cy="1" r="5" fill="blue" fill-opacity=".5"/></svg>`)
	expect(t, img, "the view box's unit, ten pixels", 15, 15, color.RGBA{128, 0, 127, 255}, 2)
	expect(t, img, "cut to the viewport", 35, 35, clearPx, 0)
	img = draw(t, `<symbol id="s" viewBox="0 0 1 1"><rect width="1" height="1" fill="red"/></symbol>`+
		`<use href="#s" x="50" y="50" width="20" height="20"/>`)
	expect(t, img, "a symbol where its use puts it", 60, 60, redPx, 0)
	expect(t, img, "nowhere else", 20, 20, clearPx, 0)
	expect(t, img, "a symbol is not drawn by itself", 75, 75, clearPx, 0)
}

func TestMarkersGoAtTheVertices(t *testing.T) {
	vs := vertices(parsePath("M0 0L10 0L10 10", 100))
	if len(vs) != 3 || vs[0].angle() != 0 || vs[1].angle() != 45 || vs[2].angle() != 90 {
		t.Errorf("an open path's vertices: %+v", vs)
	}
	vs = vertices(parsePath("M0 0h10v10z", 100))
	if len(vs) != 4 || math.Abs(vs[0].angle()-(-67.5)) > 1e-9 && math.Abs(vs[0].angle()-112.5) > 1e-9 {
		t.Errorf("a closed path's first vertex bisects its closing segment and its first: %+v", vs[0])
	}
	img := draw(t, `<marker id="m" markerWidth="10" markerHeight="10" refX="5" refY="5" markerUnits="userSpaceOnUse">`+
		`<rect width="10" height="10" fill="red"/></marker>`+
		`<polyline points="20,20 50,20 50,80" fill="none" stroke="none" marker-mid="url(#m)"/>`)
	expect(t, img, "the middle vertex", 50, 20, redPx, 0)
	expect(t, img, "not the ends", 20, 20, clearPx, 0)
	expect(t, img, "not the ends", 50, 80, clearPx, 0)
}

func TestImagesOfDataURLs(t *testing.T) {
	const png = `data:image/png;base64,` +
		`iVBORw0KGgoAAAANSUhEUgAAAAIAAAABCAIAAAB7QOjdAAAAFElEQVR4nAAHAPj/Av8AAAAA/wMABw4CAZBteiQAAAAASUVORK5CYII=`
	img := draw(t, `<image width="100" height="50" preserveAspectRatio="none" image-rendering="pixelated" `+
		`href="`+png+`"/>`)
	expect(t, img, "the left pixel", 20, 25, redPx, 0)
	expect(t, img, "the right pixel", 80, 25, bluePx, 0)
	expect(t, img, "below the image", 50, 75, clearPx, 0)
	p := picture(t, `<image href="`+png+`"/>`)
	if len(p.Items) != 1 || p.Items[0].Image == nil || p.Items[0].Image.Transform != Identity() {
		t.Errorf("an image with no size takes its own: %+v", p.Items)
	}
	img = draw(t, `<image width="100" height="100" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' `+
		`viewBox='0 0 2 2'%3E%3Crect width='1' height='1' fill='red'/%3E%3C/svg%3E"/>`)
	expect(t, img, "an SVG image's picture", 25, 25, redPx, 0)
	expect(t, img, "and only it", 75, 75, clearPx, 0)
	for _, url := range []string{"data:,hello", "data:image/png;base64,!!", "a.png"} {
		if broken := picture(t, `<image width="10" height="10" href="`+url+`"/>`); len(broken.Items) != 0 {
			t.Errorf("%s: %+v", url, broken.Items)
		}
	}
}

func TestDataURLs(t *testing.T) {
	for url, want := range map[string]string{
		"data:,a%20b":                 "a b",
		"data:text/plain;base64,aGk=": "hi",
		"DATA:;base64,aG k":           "hi",
		"data:,100%":                  "100%",
	} {
		if got, ok := dataURL(url); !ok || string(got) != want {
			t.Errorf("%s: %q %v, want %q", url, got, ok, want)
		}
	}
	if _, ok := dataURL("http://example.com/a.png"); ok {
		t.Error("a URL that is no data: URL is read")
	}
}

func TestTextIsDrawnInGosGlyphs(t *testing.T) {
	p := picture(t, `<text x="10" y="50" font-size="20" fill="red">Hi <tspan fill="blue">there</tspan></text>`)
	all := shapes(p.Items)
	if len(all) != 2 || all[0].Fill.Color != red || all[1].Fill.Color != blue {
		t.Fatalf("a shape for each run of one style: %d", len(all))
	}
	first, _ := all[0].Path.Bounds()
	second, _ := all[1].Path.Bounds()
	if first.X < 10 || first.X > 13 || first.Y+first.H > 50.5 || second.X <= first.X+first.W {
		t.Errorf("glyphs from the pen on the baseline, run after run: %+v %+v", first, second)
	}
	mid := shapes(picture(t, `<text x="50" y="50" text-anchor="middle">Hello</text>`).Items)
	box, _ := mid[0].Path.Bounds()
	if c := box.X + box.W/2; math.Abs(c-50) > 1 {
		t.Errorf("a middle anchor centres the line: %v", c)
	}
	xs := shapes(picture(t, `<text x="10 40 70" y="50">abc</text>`).Items)
	box, _ = xs[0].Path.Bounds()
	if box.W < 60 {
		t.Errorf("each character at its own x: %+v", box)
	}
	if got := shapes(picture(t, `<text x="10" y="50">  </text>`).Items); len(got) != 0 {
		t.Errorf("white space alone draws nothing: %d", len(got))
	}
}

func TestReferencesInACycleDrawNothing(t *testing.T) {
	for _, body := range []string{
		`<mask id="m"><rect width="10" height="10" fill="white" mask="url(#m)"/></mask>` +
			`<rect width="10" height="10" mask="url(#m)"/>`,
		`<marker id="m"><path d="M0 0h5" stroke="red" marker-end="url(#m)"/></marker>` +
			`<path d="M0 0h5" stroke="red" marker-end="url(#m)"/>`,
	} {
		p := picture(t, body)
		if _, err := p.Rasterise(20, 20); err != nil {
			t.Errorf("%s: %v", body, err)
		}
	}
	if got := picture(t, `<rect width="1" height="1" clip-path="url(#c)"/>`).Unsupported(); !slices.Equal(got, nil) {
		t.Errorf("a clip naming nothing: %v", got)
	}
}
