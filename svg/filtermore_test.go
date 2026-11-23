package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"image/color"
	"testing"
)

func TestTurbulenceIsTheReferenceNoiseAsChromeReadsIt(t *testing.T) {
	img := draw(t, `<filter id="t" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB">`+
		`<feTurbulence baseFrequency=".07 0" stitchTiles="stitch"/></filter>
		<rect width="56" height="20" filter="url(#t)"/>`)
	// Chrome's red over white along a row, from x 41: stitched, the last
	// lattice cell bends back to the first.
	chrome := []int{255, 237, 219, 204, 192, 183, 177, 176, 179, 186, 197, 212, 228, 244, 255}
	for i, want := range chrome {
		px := img.RGBAAt(41+i, 5)
		if got := int(px.R) + 255 - int(px.A); got < want-2 || got > want+2 {
			t.Errorf("x %d: red over white %d, Chrome's %d", 41+i, got, want)
		}
	}
	noise := func(attrs string) color.RGBA {
		return draw(t, `<filter id="t" x="0" y="0" width="1" height="1"><feTurbulence `+attrs+`/></filter>
			<rect width="100" height="100" filter="url(#t)"/>`).RGBAAt(37, 61)
	}
	if noise(`baseFrequency=".1" seed="1"`) == noise(`baseFrequency=".1" seed="2"`) {
		t.Error("two seeds make the same noise")
	}
	if noise(`baseFrequency=".1" seed="2.9"`) != noise(`baseFrequency=".1" seed="2"`) {
		t.Error("a seed is not truncated")
	}
	if noise(`baseFrequency=".1"`) == noise(`baseFrequency=".1" type="fractalNoise"`) {
		t.Error("fractal noise is turbulence")
	}
	if px := noise(`baseFrequency="-1"`); px != clearPx {
		t.Errorf("a negative frequency: %v, want nothing", px)
	}
}

func TestAConvolutionWeighsTheNeighbours(t *testing.T) {
	const shape = `<rect x="40" y="40" width="20" height="20" fill="red" filter="url(#f)"/>`
	img := draw(t, `<filter id="f"><feConvolveMatrix order="3" kernelMatrix="0 0 0 0 0 0 0 0 1"/></filter>`+shape)
	expect(t, img, "moved a pixel, the kernel turned", 60, 60, redPx, 0)
	expect(t, img, "moved off", 40, 40, clearPx, 0)
	img = draw(t, `<filter id="f"><feConvolveMatrix order="3" kernelMatrix="1 1 1 1 1 1 1 1 1"/></filter>`+shape)
	expect(t, img, "an average at the corner", 39, 39, color.RGBA{28, 0, 0, 28}, 1)
	img = draw(t, `<filter id="f"><feConvolveMatrix order="3" kernelMatrix="0 0 0 0 1 0 0 0 0" bias=".5"`+
		` preserveAlpha="true"/></filter>`+shape)
	expect(t, img, "a bias in linear light, alpha kept", 50, 50, color.RGBA{255, 188, 188, 255}, 1)
	for _, bad := range []string{
		`order="0" kernelMatrix=""`, `order="2.5" kernelMatrix="1 1 1 1"`, `order="2" kernelMatrix="1 1 1"`,
		`order="1" kernelMatrix="1" divisor="0"`, `order="3" kernelMatrix="1 1 1 1 1 1 1 1 1" targetX="3"`,
	} {
		img = draw(t, `<filter id="f"><feConvolveMatrix `+bad+`/></filter>`+shape)
		expect(t, img, bad+" draws nothing", 50, 50, clearPx, 0)
	}
}

func TestADisplacementMapMovesByAnotherPicturesColours(t *testing.T) {
	img := draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3" color-interpolation-filters="sRGB">`+
		`<feFlood flood-color="#ff8080" result="map"/>`+
		`<feDisplacementMap in="SourceGraphic" in2="map" scale="20" xChannelSelector="R" yChannelSelector="G"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "each pixel reads half the scale to its right", 32, 50, redPx, 0)
	expect(t, img, "and not further", 52, 50, clearPx, 0)
}

func TestLightingLightsTheAlphaAsASurface(t *testing.T) {
	const shape = `<rect x="20" y="20" width="60" height="60" filter="url(#f)"/>`
	img := draw(t, `<filter id="f" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB">`+
		`<feDiffuseLighting lighting-color="#f80"><feDistantLight elevation="90"/></feDiffuseLighting></filter>`+shape)
	expect(t, img, "a flat surface lit from above", 50, 50, color.RGBA{255, 136, 0, 255}, 1)
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB">`+
		`<feDiffuseLighting><feDistantLight elevation="30"/></feDiffuseLighting></filter>`+shape)
	expect(t, img, "lit aslant", 50, 50, color.RGBA{128, 128, 128, 255}, 1)
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB">`+
		`<feSpecularLighting specularExponent="1"><fePointLight x="50" y="50" z="10"/></feSpecularLighting></filter>`+shape)
	under, aside := img.RGBAAt(50, 50), img.RGBAAt(22, 22)
	if under.A < 250 || aside.A >= under.A {
		t.Errorf("a shine under a point light: %v there, %v aside", under, aside)
	}
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1" color-interpolation-filters="sRGB">`+
		`<feDiffuseLighting><feSpotLight x="50" y="50" z="20" pointsAtX="50" pointsAtY="50" limitingConeAngle="20"/>`+
		`</feDiffuseLighting></filter>`+shape)
	expect(t, img, "in the spot's cone", 50, 50, whitePx, 1)
	expect(t, img, "out of it", 25, 25, blackPx, 0)
	img = draw(t, `<filter id="f" x="-1" y="-1" width="3" height="3" color-interpolation-filters="sRGB">`+
		`<feDiffuseLighting surfaceScale="20"><feDistantLight elevation="90"/></feDiffuseLighting></filter>`+shape)
	if edge := img.RGBAAt(20, 50); edge.R > 128 {
		t.Errorf("a steep edge lit from above: %v, want dark", edge)
	}
}

func TestAnFeImageDrawsAnElementOrAPictureInItsRegion(t *testing.T) {
	img := draw(t, `<circle id="dot" cx="5" cy="5" r="4" fill="blue"/>
		<filter id="f" x="0" y="0" width="1" height="1"><feImage href="#dot"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "the element, moved to the region's corner", 45, 45, bluePx, 0)
	expect(t, img, "the element where it is in the file", 5, 5, bluePx, 0)
	expect(t, img, "nothing else", 55, 55, clearPx, 0)
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1"><feImage preserveAspectRatio="none" `+
		`href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1 1'%3E%3Crect `+
		`width='1' height='1' fill='blue'/%3E%3C/svg%3E"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" filter="url(#f)"/>`)
	expect(t, img, "a picture filling the region", 58, 58, bluePx, 0)
	img = draw(t, `<filter id="f" x="0" y="0" width="1" height="1"><feImage href="#f"/></filter>
		<rect x="40" y="40" width="20" height="20" fill="red" filter="url(#f)"/>
		<g id="loop" filter="url(#g)"><rect width="10" height="10"/></g>
		<filter id="g"><feImage href="#loop"/></filter>`)
	expect(t, img, "no element that draws", 50, 50, clearPx, 0)
}
