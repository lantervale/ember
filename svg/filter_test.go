package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"image/color"
	"slices"
	"testing"
)

func TestAFloodFillsTheFilterRegion(t *testing.T) {
	img := draw(t, `<filter id="f"><feFlood flood-color="blue"/></filter>
		<rect x="20" y="20" width="50" height="50" fill="red" filter="url(#f)"/>`)
	expect(t, img, "inside the shape", 40, 40, bluePx, 0)
	expect(t, img, "in the region's tenth past the box", 16, 16, bluePx, 0)
	expect(t, img, "past the region", 13, 40, clearPx, 0)
	img = draw(t, `<filter id="f" filterUnits="userSpaceOnUse" x="0" y="0" width="30" height="100">`+
		`<feFlood flood-color="blue" flood-opacity=".5"/></filter>
		<rect x="20" y="20" width="50" height="50" fill="red" filter="url(#f)"/>`)
	expect(t, img, "a user-space region", 5, 90, halfBlue, 1)
	expect(t, img, "past it", 40, 40, clearPx, 0)
}

func TestABlurSpreadsThePictureBothWays(t *testing.T) {
	img := draw(t, `<filter id="f"><feGaussianBlur stdDeviation="3"/></filter>
		<rect x="30" y="30" width="40" height="40" fill="red" filter="url(#f)"/>`)
	expect(t, img, "the middle", 50, 50, redPx, 2)
	left, right := img.RGBAAt(28, 50), img.RGBAAt(71, 50)
	if left.A == 0 || left.A == 255 || left != right {
		t.Errorf("the edges: %v and %v, want the same partial red", left, right)
	}
	expect(t, img, "far off", 10, 50, clearPx, 0)
	img = draw(t, `<filter id="f"><feGaussianBlur stdDeviation="3 0"/></filter>
		<rect x="30" y="30" width="40" height="40" fill="red" filter="url(#f)"/>`)
	expect(t, img, "no blur along y", 50, 29, clearPx, 0)
	if img.RGBAAt(29, 50).A == 0 {
		t.Error("the blur along x reaches nothing")
	}
}

func TestFiltersWorkInLinearLightUnlessToldOtherwise(t *testing.T) {
	const body = `<rect x="0" y="0" width="50" height="100" fill="red"/>` +
		`<rect x="50" y="0" width="50" height="100" fill="lime"/>`
	linear := draw(t, `<filter id="f"><feGaussianBlur stdDeviation="8"/></filter><g filter="url(#f)">`+body+`</g>`)
	encoded := draw(t, `<filter id="f" color-interpolation-filters="sRGB"><feGaussianBlur stdDeviation="8"/></filter>`+
		`<g filter="url(#f)">`+body+`</g>`)
	l, e := linear.RGBAAt(50, 50), encoded.RGBAAt(50, 50)
	if l.R <= e.R+40 || l.G <= e.G+40 {
		t.Errorf("between red and green: %v in linear light, %v in sRGB, want the first brighter", l, e)
	}
}

func TestOffsetsMorphologyAndShadowsMoveAndGrowThePicture(t *testing.T) {
	img := draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feOffset dx="10" dy="5"/></filter>
		<rect x="30" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "moved in", 55, 52, redPx, 0)
	expect(t, img, "moved out", 32, 32, clearPx, 0)
	img = draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feMorphology operator="dilate" radius="4"/></filter>
		<rect x="30" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "grown", 27, 40, redPx, 0)
	img = draw(t, `<filter id="f"><feMorphology radius="4"/></filter>
		<rect x="30" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "thinned", 32, 40, clearPx, 0)
	expect(t, img, "kept", 40, 40, redPx, 0)
	img = draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feDropShadow dx="20" dy="0" stdDeviation="0"`+
		` flood-color="blue"/></filter>
		<rect x="30" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "the picture over its shadow", 40, 40, redPx, 0)
	expect(t, img, "the shadow", 60, 40, bluePx, 0)
}

func TestColourPrimitivesChangeEachPixel(t *testing.T) {
	for _, c := range []struct {
		name, filter string
		want         color.RGBA
	}{
		{"saturate 0", `<feColorMatrix type="saturate" values="0" color-interpolation-filters="sRGB"/>`,
			color.RGBA{54, 54, 54, 255}},
		{"a matrix swapping red and blue", `<feColorMatrix values="0 0 1 0 0 0 1 0 0 0 1 0 0 0 0 0 0 0 1 0"/>`, bluePx},
		{"luminance to alpha", `<feColorMatrix type="luminanceToAlpha"/>`, color.RGBA{0, 0, 0, 54}},
		{"a table inverting", `<feComponentTransfer><feFuncR type="table" tableValues="1 0"/>` +
			`<feFuncG type="table" tableValues="1 0"/></feComponentTransfer>`, color.RGBA{0, 255, 0, 255}},
		{"a linear alpha", `<feComponentTransfer><feFuncA type="linear" slope=".5"/></feComponentTransfer>`, halfRed},
		{"a flood in the shape", `<feFlood flood-color="blue"/><feComposite in2="SourceGraphic" operator="in"/>`, bluePx},
		{"arithmetic", `<feFlood flood-color="blue" result="b"/>` +
			`<feComposite in="SourceGraphic" in2="b" operator="arithmetic" k2="1" k3="1"/>`, color.RGBA{255, 0, 255, 255}},
		{"a merge, the last on top", `<feFlood flood-color="blue" result="b"/>` +
			`<feMerge><feMergeNode in="SourceGraphic"/><feMergeNode in="b"/></feMerge>`, bluePx},
		{"a multiply", `<feFlood flood-color="#ff0" result="y"/><feBlend in="SourceGraphic" in2="y" mode="multiply"/>`,
			redPx},
		{"source alpha", `<feMerge><feMergeNode in="SourceAlpha"/></feMerge>`, blackPx},
		{"a primitive not drawn", `<feTurbulence baseFrequency=".1"/>`, clearPx},
	} {
		img := draw(t, `<filter id="f">`+c.filter+`</filter>
			<rect x="30" y="30" width="40" height="40" fill="red" filter="url(#f)"/>`)
		expect(t, img, c.name, 50, 50, c.want, 2)
	}
}

func TestPrimitiveRegionsAndUnitsFollowTheirAttributes(t *testing.T) {
	img := draw(t, `<filter id="f"><feFlood flood-color="blue" x="40" y="40" width="10" height="10"/></filter>
		<rect x="30" y="30" width="40" height="40" fill="red" filter="url(#f)"/>`)
	expect(t, img, "in the subregion", 45, 45, bluePx, 0)
	expect(t, img, "past it", 55, 55, clearPx, 0)
	img = draw(t, `<filter id="f" primitiveUnits="objectBoundingBox" width="2"><feOffset dx=".5"/></filter>
		<rect x="20" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "moved half its box", 45, 40, redPx, 0)
	img = draw(t, `<filter id="f"><feFlood flood-color="blue" result="b"/>`+
		`<feTile in="b"/></filter><rect x="30" y="30" width="40" height="40" fill="red" filter="url(#f)"/>`)
	expect(t, img, "a tile of the whole region", 50, 50, bluePx, 0)
}

func TestAFilterThatCannotBeRunHidesOrLeavesTheElement(t *testing.T) {
	img := draw(t, `<filter id="f"/><rect width="50" height="50" fill="red" filter="url(#f)"/>`)
	expect(t, img, "an empty filter hides it", 20, 20, clearPx, 0)
	img = draw(t, `<rect width="50" height="50" fill="red" filter="url(#nothing)"/>`)
	expect(t, img, "a filter that is not there leaves it", 20, 20, redPx, 0)
	img = draw(t, `<filter id="f"><feFlood/></filter>
		<g filter="url(#f)"><line x1="0" y1="0" x2="50" y2="0" stroke="red"/></g>`)
	expect(t, img, "a box with no height hides it", 20, 0, clearPx, 0)
}

func TestAFilterOverTooManyPixelsIsNotRun(t *testing.T) {
	p := picture(t, `<filter id="f"><feFlood flood-color="blue"/></filter>
		<rect width="100" height="100" fill="red" filter="url(#f)"/>`)
	img, err := p.Rasterise(2100, 2100)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, img, "drawn as if unfiltered", 1000, 1000, redPx, 0)
}

func TestCSSFilterFunctions(t *testing.T) {
	img := draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" style="filter: grayscale(1)"/>`)
	expect(t, img, "grayscale", 50, 50, color.RGBA{54, 54, 54, 255}, 1)
	img = draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" style="filter: invert(100%) opacity(.5)"/>`)
	expect(t, img, "invert then opacity", 50, 50, color.RGBA{0, 128, 128, 128}, 1)
	img = draw(t, `<rect x="30" y="30" width="20" height="20" fill="red" `+
		`style="filter: drop-shadow(20px 0 0 rgb(0, 0, 255))"/>`)
	expect(t, img, "a drop shadow, its colour in a function", 60, 40, bluePx, 0)
	img = draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" style="filter: blur(4px)"/>`)
	if a := img.RGBAAt(27, 50).A; a == 0 || a == 255 {
		t.Errorf("a blur reaching past the box: alpha %d", a)
	}
	p := picture(t, `<rect width="1" height="1" style="filter: blur(2px) brightness(1.2) hue-rotate(.5turn)"/>`)
	f := p.Items[0].Group.Filters[0]
	if kinds := []PrimitiveKind{f.Primitives[0].Kind, f.Primitives[1].Kind, f.Primitives[2].Kind}; !slices.Equal(kinds,
		[]PrimitiveKind{GaussianBlur, ComponentTransfer, ColorMatrix}) || f.Primitives[2].In[0] != 1 {
		t.Errorf("a list of functions: %v, each reading the last", f.Primitives)
	}
}

func TestBlendModesMixAGroupWithWhatIsUnderIt(t *testing.T) {
	img := draw(t, `<rect width="100" height="100" fill="#ff0"/>
		<rect x="20" y="20" width="40" height="40" fill="#f0f" style="mix-blend-mode: multiply"/>`)
	expect(t, img, "multiply", 30, 30, redPx, 0)
	img = draw(t, `<rect width="100" height="100" fill="#f00"/>
		<rect x="20" y="20" width="40" height="40" fill="#0f0" style="mix-blend-mode: screen"/>`)
	expect(t, img, "screen", 30, 30, color.RGBA{255, 255, 0, 255}, 0)
	img = draw(t, `<rect width="100" height="100" fill="#fff"/>
		<g style="isolation: isolate">
			<rect x="20" y="20" width="40" height="40" fill="#f00" style="mix-blend-mode: difference"/>
		</g>`)
	expect(t, img, "an isolated group blends with nothing", 30, 30, redPx, 0)
	img = draw(t, `<rect width="100" height="100" fill="#fff"/>
		<g><rect x="20" y="20" width="40" height="40" fill="#f00" style="mix-blend-mode: difference"/></g>`)
	expect(t, img, "a plain group lets it reach the backdrop", 30, 30, color.RGBA{0, 255, 255, 255}, 0)
	for _, mode := range []BlendMode{BlendHue, BlendSaturation, BlendColor, BlendLuminosity} {
		px := blendPixel([4]float32{1, 0, 0, 1}, [4]float32{0.5, 0.5, 0.5, 1}, mode)
		if px[3] != 1 {
			t.Errorf("mode %d: alpha %v", mode, px[3])
		}
	}
}

func TestANonScalingStrokeKeepsItsWidthInPixels(t *testing.T) {
	p, err := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 10 10"><line x1="0" y1="5" x2="10" y2="5"
		stroke="red" stroke-width="2" vector-effect="non-scaling-stroke"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	img, err := p.Rasterise(100, 100)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, img, "on the line", 50, 50, redPx, 0)
	expect(t, img, "a pixel off it", 50, 51, clearPx, 0)
	img, err = p.RasteriseScaled(100, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, img, "a pixel off it at twice the pixels", 50, 51, redPx, 0)
}

func TestClipPathsOfBasicShapes(t *testing.T) {
	for _, c := range []struct {
		clip        string
		in, out     [2]int
		unsupported bool
	}{
		{"circle(20px)", [2]int{50, 50}, [2]int{25, 25}, false},
		{"circle(50% at 0 0)", [2]int{35, 35}, [2]int{65, 65}, false},
		{"ellipse(30px 10px)", [2]int{65, 50}, [2]int{50, 35}, false},
		{"inset(10px 5px round 4px)", [2]int{50, 45}, [2]int{32, 50}, false},
		{"polygon(evenodd, 0 0, 100% 0, 0 100%)", [2]int{35, 35}, [2]int{65, 65}, false},
		{"path('M0 0 H20 V20 Z')", [2]int{45, 35}, [2]int{35, 45}, false},
		{"xywh(0 0 50% 50%)", [2]int{35, 35}, [2]int{55, 55}, false},
		{"rect(0 auto 50% 0)", [2]int{65, 35}, [2]int{50, 55}, false},
		{"fill-box", [2]int{50, 50}, [2]int{25, 25}, false},
		{"circle(10px) view-box", [2]int{50, 50}, [2]int{65, 65}, false},
		{"circle(", [2]int{50, 50}, [2]int{}, true},
	} {
		body := `<rect x="30" y="30" width="40" height="40" fill="red" style="clip-path: ` + c.clip + `"/>`
		img := draw(t, body)
		expect(t, img, c.clip+", inside", c.in[0], c.in[1], redPx, 0)
		if !c.unsupported {
			expect(t, img, c.clip+", outside", c.out[0], c.out[1], clearPx, 0)
		}
	}
	img := draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" stroke="blue" stroke-width="10"
		style="clip-path: inset(0) stroke-box"/>`)
	expect(t, img, "a stroke box keeps the stroke", 27, 50, bluePx, 0)
	img = draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" stroke="blue" stroke-width="10"
		style="clip-path: inset(0) fill-box"/>`)
	expect(t, img, "a fill box cuts it", 27, 50, clearPx, 0)
	img = draw(t, `<rect width="100" height="100" fill="red" style="clip-path: circle(10px at 90px 90px) view-box"/>`)
	expect(t, img, "a view box", 90, 90, redPx, 0)
	expect(t, img, "outside it", 50, 50, clearPx, 0)
}

func TestAFilterUnderARotationWorksInTheElementsOwnSpace(t *testing.T) {
	img := draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feGaussianBlur stdDeviation="4 0"/></filter>
		<rect x="30" y="40" width="40" height="20" fill="red" transform="rotate(90 50 50)" filter="url(#f)"/>`)
	if a := img.RGBAAt(50, 28).A; a == 0 || a == 255 {
		t.Errorf("blurred along the element's x, now the picture's y: alpha %d past the end", a)
	}
	expect(t, img, "not along its y", 38, 50, clearPx, 0)
	expect(t, img, "the middle", 50, 50, redPx, 2)
	img = draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feMorphology operator="dilate" radius="6 0"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" transform="rotate(45 50 50)" filter="url(#f)"/>`)
	expect(t, img, "thickened along the element's x", 60, 60, redPx, 0)
	expect(t, img, "and not along its y", 40, 60, clearPx, 0)
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1"><feFlood flood-color="blue"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" transform="rotate(45 50 50)" filter="url(#f)"/>`)
	expect(t, img, "a region turned with it", 50, 37, bluePx, 0)
	expect(t, img, "its corner", 40, 40, clearPx, 0)
}

func TestInputsChromeDoesNotDrawAreTheResultBefore(t *testing.T) {
	for _, name := range []string{"BackgroundImage", "BackgroundAlpha", "FillPaint", "StrokePaint"} {
		img := draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3"><feOffset dx="10"/>`+
			`<feMerge><feMergeNode in="`+name+`"/></feMerge></filter>
			<rect x="30" y="30" width="20" height="20" fill="red" filter="url(#f)"/>`)
		expect(t, img, name+" after an offset", 55, 40, redPx, 0)
		expect(t, img, name+" after an offset, moved off", 35, 40, clearPx, 0)
		img = draw(t, `<filter id="f"><feMerge><feMergeNode in="`+name+`"/></feMerge></filter>
			<rect x="30" y="30" width="20" height="20" fill="red" stroke="blue" filter="url(#f)"/>`)
		expect(t, img, name+" first", 40, 40, redPx, 0)
	}
}

func TestCSSFunctionsAfterAFilterElementWorkInItsColourSpace(t *testing.T) {
	const defs = `<filter id="t" x="0" y="0" width="1" height="1"><feOffset/></filter>` +
		`<filter id="s" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB"><feOffset/></filter>`
	img := draw(t, defs+`<rect x="0" y="0" width="40" height="40" fill="#c63" style="filter: url(#t) grayscale(1)"/>
		<rect x="50" y="0" width="40" height="40" fill="#c63" style="filter: url(#s) grayscale(1)"/>
		<rect x="0" y="50" width="20" height="20" fill="#c63" style="filter: url(#t) drop-shadow(20px 0 0 #036)"/>
		<rect x="50" y="50" width="20" height="20" fill="#c63" style="filter: url(#t) grayscale(0) drop-shadow(20px 0 0 #036)"/>`)
	expect(t, img, "grey in linear light", 20, 20, color.RGBA{131, 131, 131, 255}, 1)
	expect(t, img, "grey in sRGB", 70, 20, color.RGBA{120, 120, 120, 255}, 1)
	expect(t, img, "a shadow whose colour is taken as linear", 30, 60, color.RGBA{0, 124, 170, 255}, 1)
	expect(t, img, "after a function after the filter", 80, 60, color.RGBA{0, 124, 170, 255}, 1)
}

func TestABasicShapesCornersTakeRadiiOfTheirOwn(t *testing.T) {
	for _, c := range []struct {
		clip            string
		round, straight [2]int
	}{
		{"inset(0 round 0 20px)", [2]int{68, 31}, [2]int{31, 31}},
		{"inset(0 round 0 0 20px)", [2]int{68, 68}, [2]int{68, 31}},
		{"inset(0 round 0 0 0 20px)", [2]int{31, 68}, [2]int{68, 68}},
		{"inset(0 round 20px 0 0 0)", [2]int{31, 31}, [2]int{31, 68}},
		{"inset(0 round 0 50% / 0 10%)", [2]int{68, 31}, [2]int{31, 31}},
		{"xywh(0 0 100% 100% round 40px 0)", [2]int{35, 35}, [2]int{68, 31}},
	} {
		img := draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" style="clip-path: `+c.clip+`"/>`)
		expect(t, img, c.clip+", a round corner", c.round[0], c.round[1], clearPx, 0)
		expect(t, img, c.clip+", a straight one", c.straight[0], c.straight[1], redPx, 0)
	}
	img := draw(t, `<rect x="30" y="30" width="40" height="40" fill="red" style="clip-path: inset(0 round 30px 30px 0 0)"/>`)
	expect(t, img, "radii past a side are scaled down together", 32, 46, redPx, 0)
	expect(t, img, "to meet", 31, 31, clearPx, 0)
}

func TestTextSpacingDecorationsAndPaths(t *testing.T) {
	width := func(body string) int {
		img := draw(t, body)
		right := 0
		for x := range 100 {
			for y := range 100 {
				if img.RGBAAt(x, y).A > 0 {
					right = x
				}
			}
		}
		return right
	}
	plain := width(`<text x="0" y="50" font-size="10">a b c</text>`)
	if spaced := width(`<text x="0" y="50" font-size="10" letter-spacing="4">a b c</text>`); spaced < plain+16 {
		t.Errorf("letter-spacing: %d pixels wide, %d without", spaced, plain)
	}
	if spaced := width(`<text x="0" y="50" font-size="10" word-spacing="10">a b c</text>`); spaced < plain+19 {
		t.Errorf("word-spacing: %d pixels wide, %d without", spaced, plain)
	}
	img := draw(t, `<text x="10" y="50" font-size="20" text-decoration="underline" fill="blue">.... ....</text>`)
	under := uint8(0)
	for y := 50; y < 56; y++ {
		under = max(under, img.RGBAAt(42, y).A)
	}
	if under < 60 {
		t.Errorf("an underline under the space: alpha %d", under)
	}
	img = draw(t, `<text x="10" y="50" font-size="20" fill="blue"><tspan style="text-decoration: line-through">--- `+
		`</tspan><tspan fill="red">---</tspan></text>`)
	expect(t, img, "a line through the space", 42, 44, bluePx, 30)
	img = draw(t, `<path id="p" d="M10 90 V10"/>
		<text font-size="20" fill="blue"><textPath href="#p">llll</textPath></text>`)
	ink := 0
	for y := range 100 {
		for x := range 100 {
			if img.RGBAAt(x, y).A > 0 {
				ink++
				if x > 15 || y < 9 {
					t.Fatalf("a glyph off its path, at (%d, %d)", x, y)
				}
			}
		}
	}
	if ink == 0 {
		t.Error("nothing set along the path")
	}
	img = draw(t, `<path id="p" d="M10 50 H90" pathLength="10"/>
		<text font-size="10" fill="blue"><textPath href="#p" startOffset="5" method="stretch" side="right"
			spacing="auto">ll</textPath></text>`)
	for x := range 100 {
		for y := range 100 {
			if img.RGBAAt(x, y).A > 0 && (x < 49 || y > 51) {
				t.Fatalf("an offset of the path's own length, method, side and spacing as Chrome: ink at (%d, %d)", x, y)
			}
		}
	}
	img = draw(t, `<path id="p" d="M10 50 H20"/>
		<text font-size="20" fill="blue"><textPath href="#p">mmmmmm</textPath></text>`)
	for x := 35; x < 100; x++ {
		if img.RGBAAt(x, 45).A > 0 {
			t.Fatalf("a glyph past the path's end, at x %d", x)
		}
	}
}

func TestAUTF16FileIsRead(t *testing.T) {
	const file = `<?xml version="1.0" encoding="UTF-16"?>` +
		`<svg ` + xmlns + ` viewBox="0 0 1 1"><rect width="1" height="1"/></svg>`
	units := make([]byte, 0, 2*len(file)+2)
	be := []byte{0xfe, 0xff}
	le := []byte{0xff, 0xfe}
	for _, r := range file {
		be = append(be, 0, byte(r))
		le = append(le, byte(r), 0)
	}
	units = append(units, le[2:]...)
	for name, data := range map[string][]byte{"big-endian": be, "little-endian": le, "with no mark": units} {
		p, err := NewPicture(data)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(p.Items) != 1 {
			t.Errorf("%s: %d items", name, len(p.Items))
		}
	}
}
