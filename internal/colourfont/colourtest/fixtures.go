package colourtest

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

// The shapes the fixtures' layers are cut from, Go Regular's: a square
// from (100, 0) to (1137, 1037), a disc from (174, 222) to (1062, 1110),
// a ring the same size and a triangle from (250, 0) to (1777, 1527).
const (
	Square   = '■'
	Disc     = '●'
	Ring     = '○'
	Triangle = '▲'
)

// The fixtures' palettes: the default one, one for light backgrounds
// and one for dark, of four entries each.
//
//nolint:gochecknoglobals // fixed fixtures
var (
	Default = []color.NRGBA{{0xe0, 0x30, 0x2a, 0xff}, {0x2a, 0x9d, 0x3a, 0xff}, {0x2a, 0x5a, 0xd0, 0xff},
		{0xf0, 0xb0, 0x20, 0xff}}
	Light = []color.NRGBA{{0x80, 0x10, 0x10, 0xff}, {0x10, 0x50, 0x20, 0xff}, {0x10, 0x20, 0x70, 0xff},
		{0x70, 0x40, 0x10, 0xff}}
	Dark = []color.NRGBA{{0xff, 0x90, 0xa0, 0xff}, {0xa0, 0xff, 0x80, 0xff}, {0x80, 0xc0, 0xff, 0xff},
		{0xff, 0xf0, 0x80, 0x80}}
)

// COLRv0 draws A, B and C in layers of the fixtures' palettes, B's ring
// in the text's colour.
func COLRv0() []byte {
	return Build(Options{
		Layers: map[rune][]Layer{
			'A': {{Square, 2}, {Disc, 0}},
			'B': {{Triangle, 1}, {Ring, Foreground}},
			'C': {{Disc, 3}, {Square, 0}, {Ring, 2}},
		},
		Palettes: [][]color.NRGBA{Default, Light, Dark},
		Types:    []uint32{0, 1, 2},
		Family:   "Layered",
	})
}

// COLRv1 draws A to L with each kind of paint of COLR version 1.
func COLRv1() []byte {
	three := Line{Stops: []Stop{{0, 0, 1}, {0.5, 3, 1}, {1, 2, 1}}}
	return Build(Options{
		Paints: map[rune]Paint{
			'A': Glyph{Square, Linear{Line: three, X0: 100, X1: 1137, X2: 100, Y2: 1000}},
			'B': Glyph{Disc, Radial{Line: Line{Reflect, three.Stops}, X0: 618, Y0: 666, X1: 618, Y1: 666, R1: 200}},
			'C': Glyph{Square, Sweep{Line: three, X: 618, Y: 518, Start: 0, End: 1}},
			'D': Composite{Source: Glyph{Disc, Solid{3, 1}}, Backdrop: Glyph{Square, Solid{2, 1}}, Mode: 23},
			'E': Rotate{Angle: 0.25, Around: true, CX: 618, CY: 518, Paint: Glyph{Square, Solid{1, 1}}},
			'F': Skew{X: 0.15, Around: true, CX: 618, Paint: Glyph{Square, Solid{2, 1}}},
			'G': Layers{Glyph{Triangle, Solid{3, 1}}, Translate{DX: 400, DY: 300, Paint: Glyph{Disc, Solid{0, 0.7}}}},
			'H': Layers{Glyph{Ring, Solid{Foreground, 1}},
				Scale{SX: 0.6, SY: 0.6, Around: true, CX: 618, CY: 518, Paint: ColrGlyph{'A'}}},
			'I': Transform{M: [6]float32{0.8, 0.2, -0.2, 0.8, 200, 0},
				Paint: Glyph{Triangle, Linear{Line: Line{Repeat, []Stop{{0, 1, 1}, {1, 3, 1}}},
					X0: 250, X1: 500, X2: 250, Y2: 500}}},
			'J': Glyph{Square, Var{Solid{Foreground, 0.5}}},
			'K': Glyph{Disc, Sweep{Line: Line{Stops: []Stop{{0, 0, 1}, {1, 2, 1}}}, X: 618, Y: 666, End: 0.5}},
			'L': Glyph{Square, Radial{Line: three, X0: 400, Y0: 300, R0: 100, X1: 800, Y1: 700, R1: 500}},
			'M': Scale{SX: 1.4, SY: 0.6, Paint: Glyph{Disc, Var{Linear{Line: three,
				X0: 174, Y0: 222, X1: 1062, Y1: 1110, X2: 1062, Y2: 222}}}},
			'N': Composite{Source: Glyph{Disc, Solid{0, 1}}, Backdrop: Glyph{Square, Solid{2, 1}}, Mode: 5},
		},
		Clips:    map[rune][4]int16{'H': {150, 50, 1100, 1000}},
		Palettes: [][]color.NRGBA{Default, Light, Dark},
		Types:    []uint32{0, 1, 2},
		Family:   "Painted",
	})
}

// picture is a w×w bitmap: a disc of the first colour on a square of
// the second, each letter its own pair.
func picture(w int, disc, square color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, w))
	in := w / 8
	draw.Draw(img, image.Rect(in, in, w-in, w-in), image.NewUniform(square), image.Point{}, draw.Src)
	c := float64(w) / 2
	r := float64(w) * 0.3
	for y := range w {
		for x := range w {
			dx, dy := float64(x)+0.5-c, float64(y)+0.5-c
			if dx*dx+dy*dy <= r*r {
				img.SetNRGBA(x, y, disc)
			}
		}
	}
	return PNG(img)
}

// bitmapLetters are the bitmap fixtures' letters and their colours.
//
//nolint:gochecknoglobals // fixed fixtures
var bitmapLetters = map[rune][2]color.NRGBA{
	'A': {Default[0], Default[2]},
	'B': {Default[3], Default[1]},
	'C': {Default[2], {0xf0, 0xb0, 0x20, 0x80}},
}

// emWide is the bitmap fixtures' letters each an em wide, as emoji.
//
//nolint:gochecknoglobals // fixed fixtures
var emWide = map[rune]uint16{'A': 2048, 'B': 2048, 'C': 2048}

// Sbix draws A, B and C, each an em wide, as PNGs of strikes of 20 and
// 64 pixels an em filling the em from the baseline up, as Apple's
// emoji; with outlines false, the font has no glyf table.
func Sbix(outlines bool) []byte {
	strikes := map[int]map[rune]Bitmap{}
	for _, ppem := range []int{20, 64} {
		strikes[ppem] = map[rune]Bitmap{}
		for r, c := range bitmapLetters {
			strikes[ppem][r] = Bitmap{PNG: picture(ppem, c[0], c[1])}
		}
	}
	return Build(Options{Strikes: strikes, NoOutlines: !outlines, Family: "Struck", Advances: emWide})
}

// CBDT draws A, B and C, each an em wide, as PNGs at 64 pixels an em
// filling the em, a fifth of it below the baseline, as Noto Color Emoji.
func CBDT(outlines bool) []byte {
	bitmaps := map[rune]Bitmap{}
	for r, c := range bitmapLetters {
		bitmaps[r] = Bitmap{PNG: picture(64, c[0], c[1]), Y: 51}
	}
	return Build(Options{Bitmaps: bitmaps, BitmapPPEM: 64, NoOutlines: !outlines, Family: "Bitmapped",
		Advances: emWide})
}

// SVG draws A, B and C as SVG documents, B's gzipped.
func SVG(gzip bool) []byte {
	doc := func(r rune, body string) string {
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg"><g id="glyph%d">%s</g></svg>`, GlyphID(r), body)
	}
	return Build(Options{
		SVG: map[rune]string{
			'A': doc('A', `<rect x="100" y="-1037" width="1037" height="1037" fill="#2a5ad0"/>`+
				`<circle cx="618" cy="-518" r="380" fill="#e0302a"/>`),
			'B': doc('B', `<defs><linearGradient id="g" x1="0" x2="1"><stop offset="0" stop-color="#2a9d3a"/>`+
				`<stop offset="1" stop-color="#f0b020"/></linearGradient></defs>`+
				`<path d="M250 0 L1777 0 L1013 -1527 Z" fill="url(#g)"/>`),
			'C': doc('C', `<circle cx="618" cy="-666" r="444" fill="none" stroke="currentColor" stroke-width="120"/>`+
				`<rect x="418" y="-866" width="400" height="400" fill="#2a5ad0" opacity="0.6"/>`),
		},
		Gzip:   gzip,
		Family: "Drawn",
	})
}
