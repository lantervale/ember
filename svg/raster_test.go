package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
)

// draw rasterises body, in a 100×100 view box, at 100×100: a unit a pixel.
func draw(t *testing.T, body string) *image.RGBA {
	t.Helper()
	img, err := picture(t, body).Rasterise(100, 100)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func expect(t *testing.T, img *image.RGBA, what string, x, y int, want color.RGBA, slack int) {
	t.Helper()
	got := img.RGBAAt(x, y)
	for _, d := range [4]int{
		int(got.R) - int(want.R), int(got.G) - int(want.G), int(got.B) - int(want.B), int(got.A) - int(want.A),
	} {
		if d > slack || d < -slack {
			t.Errorf("%s: pixel (%d, %d) is %v, want %v", what, x, y, got, want)
			return
		}
	}
}

//nolint:gochecknoglobals // the colours pixels are expected to have
var (
	clearPx  = color.RGBA{}
	redPx    = color.RGBA{255, 0, 0, 255}
	bluePx   = color.RGBA{0, 0, 255, 255}
	blackPx  = color.RGBA{0, 0, 0, 255}
	whitePx  = color.RGBA{255, 255, 255, 255}
	halfRed  = color.RGBA{128, 0, 0, 128}
	halfBlue = color.RGBA{0, 0, 128, 128}
)

func TestFillsCoverPixelsByTheirArea(t *testing.T) {
	img := draw(t, `<rect x="10.5" y="20" width="30.25" height="40" fill="red"/>`)
	expect(t, img, "inside", 25, 40, redPx, 0)
	expect(t, img, "outside", 5, 40, clearPx, 0)
	expect(t, img, "below", 25, 60, clearPx, 0)
	expect(t, img, "half a pixel at the left edge", 10, 40, halfRed, 1)
	expect(t, img, "three quarters at the right edge", 40, 40, color.RGBA{191, 0, 0, 191}, 1)
	img = draw(t, `<rect x="10" y="10.5" width="30" height="30" fill="red"/>`)
	expect(t, img, "half a pixel at the top edge", 20, 10, halfRed, 1)
	img = draw(t, `<path d="M0 0 L100 0 L0 100 Z" fill="blue"/>`)
	expect(t, img, "on a diagonal", 50, 49, halfBlue, 2)
	expect(t, img, "under the diagonal", 20, 20, bluePx, 0)
	expect(t, img, "over the diagonal", 80, 80, clearPx, 0)
}

func TestFillRules(t *testing.T) {
	// Two squares turning the same way, one inside the other.
	const d = `d="M10 10H90V90H10Z M30 30H70V70H30Z"`
	img := draw(t, `<path `+d+`/>`)
	expect(t, img, "nonzero, in the inner square", 50, 50, blackPx, 0)
	img = draw(t, `<path `+d+` fill-rule="evenodd"/>`)
	expect(t, img, "evenodd, in the inner square", 50, 50, clearPx, 0)
	expect(t, img, "evenodd, in the ring", 20, 50, blackPx, 0)
	// Turning opposite ways the inner one is a hole under both rules.
	img = draw(t, `<path d="M10 10H90V90H10Z M30 30V70H70V30Z"/>`)
	expect(t, img, "nonzero, in a hole wound the other way", 50, 50, clearPx, 0)
	expect(t, img, "nonzero, in the ring", 20, 50, blackPx, 0)
	// An open subpath fills as if closed.
	img = draw(t, `<path d="M10 10H90V90"/>`)
	expect(t, img, "an open path's fill", 70, 30, blackPx, 0)
	expect(t, img, "outside an open path's fill", 30, 70, clearPx, 0)
}

func TestAPartCoveredTwiceIsCoveredOnce(t *testing.T) {
	// Two rectangles of one path sharing an edge that crosses pixels: the
	// edge pixels are half covered, not fully as when signed areas are summed.
	img := draw(t, `<path d="M10.5 10H50V50H10.5Z M10.5 20H40V40H10.5Z" fill="red"/>`)
	expect(t, img, "the shared edge", 10, 30, halfRed, 1)
	expect(t, img, "the single edge", 10, 15, halfRed, 1)
	expect(t, img, "the overlap", 30, 30, redPx, 0)
	// A stroke that crosses itself, half clear: one even tone.
	img = draw(t, `<path d="M20 20H80V80H50V10" fill="none" stroke="red" stroke-width="10" stroke-opacity=".5"/>`)
	expect(t, img, "where the stroke crosses itself", 50, 20, halfRed, 1)
	expect(t, img, "on the stroke", 70, 20, halfRed, 1)
	expect(t, img, "inside its sharp corner", 77, 23, halfRed, 1)
}

func TestCaps(t *testing.T) {
	butt := draw(t, `<path d="M30 50H70" stroke="red" stroke-width="20"/>`)
	expect(t, butt, "butt, on the line", 31, 50, redPx, 0)
	expect(t, butt, "butt, past the end", 28, 50, clearPx, 0)
	expect(t, butt, "butt, at the side", 50, 41, redPx, 0)
	expect(t, butt, "butt, past the side", 50, 39, clearPx, 0)
	square := draw(t, `<path d="M30 50H70" stroke="red" stroke-width="20" stroke-linecap="square"/>`)
	expect(t, square, "square, in the cap's corner", 21, 41, redPx, 0)
	expect(t, square, "square, past the cap", 18, 50, clearPx, 0)
	expect(t, square, "square, the far cap", 78, 58, redPx, 0)
	round := draw(t, `<path d="M30 50H70" stroke="red" stroke-width="20" stroke-linecap="round"/>`)
	expect(t, round, "round, in the cap", 21, 50, redPx, 0)
	expect(t, round, "round, outside the cap's curve", 21, 41, clearPx, 0)
	expect(t, round, "round, the far cap", 78, 50, redPx, 0)
	expect(t, round, "round, past the cap", 18, 50, clearPx, 0)
	// A path of no length is a dot under round and square caps, nothing under butt.
	for d, shown := range map[string]bool{"M50 50z": true, "M50 50h0": true, "M50 50L50 50": true, "M50 50": false} {
		for linecap, dot := range map[string]bool{"round": true, "square": true, "butt": false} {
			img := draw(t, `<path d="`+d+`" stroke="red" stroke-width="20" stroke-linecap="`+linecap+`"/>`)
			want := clearPx
			if shown && dot {
				want = redPx
			}
			expect(t, img, d+" with "+linecap+" caps", 50, 50, want, 0)
			if linecap == "round" {
				expect(t, img, d+" round dot's corner", 41, 41, clearPx, 0)
			}
		}
	}
}

func TestJoins(t *testing.T) {
	// A right angle at (60, 20), 10 wide: the miter's point is (65, 15).
	const corner = `<path d="M20 20H60V60" fill="none" stroke="red" stroke-width="10" `
	for attrs, want := range map[string]color.RGBA{
		``:                        redPx,
		`stroke-linejoin="miter"`: redPx,
		`stroke-linejoin="bevel"`: clearPx,
		`stroke-linejoin="round"`: clearPx,
		`stroke-miterlimit="1.5"`: redPx, // a right angle's miter is √2 long
		`stroke-miterlimit="1.4"`: clearPx,
		`stroke-linejoin="miter" stroke-miterlimit="1"`: clearPx,
	} {
		img := draw(t, corner+attrs+`/>`)
		expect(t, img, attrs+": at the miter's point", 64, 15, want, 0)
		expect(t, img, attrs+": inside the bevel", 62, 18, redPx, 0)
		expect(t, img, attrs+": inside the corner", 57, 23, redPx, 0)
		expect(t, img, attrs+": in the angle", 50, 30, clearPx, 0)
	}
	img := draw(t, corner+`stroke-linejoin="round"/>`)
	expect(t, img, "round: in the rounding", 63, 17, redPx, 0)
	// A closed path joins its end to its start.
	img = draw(t, `<path d="M20 20H60V60Z" fill="none" stroke="red" stroke-width="10"/>`)
	expect(t, img, "closed: the first corner's miter", 12, 16, redPx, 0)
	img = draw(t, `<path d="M20 20H60V60L20 20" fill="none" stroke="red" stroke-width="10"/>`)
	expect(t, img, "open: no join where the ends meet", 12, 16, clearPx, 0)
	// A line there and back has caps, and a join at the turn.
	img = draw(t, `<path d="M30 50H70H30" fill="none" stroke="red" stroke-width="10" stroke-linejoin="round"/>`)
	expect(t, img, "a turn straight back, rounded", 73, 50, redPx, 0)
	expect(t, img, "its start, butt", 27, 50, clearPx, 0)
}

func TestStrokesFollowTheTransform(t *testing.T) {
	// Stretched four times across: the stroke is wider across than down.
	img := draw(t, `<path d="M5 10V90H20" fill="none" stroke="red" stroke-width="4" transform="scale(4 1)"/>`)
	expect(t, img, "the upright stroke, 16 wide", 13, 50, redPx, 0)
	expect(t, img, "the upright stroke, 16 wide", 27, 50, redPx, 0)
	expect(t, img, "beside it", 29, 50, clearPx, 0)
	expect(t, img, "the flat stroke, 4 high", 60, 89, redPx, 0)
	expect(t, img, "over it", 60, 87, clearPx, 0)
}

func TestDashes(t *testing.T) {
	img := draw(t, `<path d="M10 50H90" stroke="red" stroke-width="10" stroke-dasharray="20 10"/>`)
	for x, on := range map[int]bool{12: true, 28: true, 32: false, 38: false, 42: true, 58: true, 65: false, 72: true} {
		want := clearPx
		if on {
			want = redPx
		}
		expect(t, img, "dashes", x, 50, want, 0)
	}
	img = draw(t, `<path d="M10 50H90" stroke="red" stroke-width="10" stroke-dasharray="20 10" `+
		`stroke-dashoffset="15"/>`)
	expect(t, img, "offset dashes: the first one's last 5", 12, 50, redPx, 0)
	expect(t, img, "offset dashes: the first gap", 20, 50, clearPx, 0)
	expect(t, img, "offset dashes: the second dash", 30, 50, redPx, 0)
	// A dash goes round a corner with its join.
	img = draw(t, `<path d="M20 20H60V60" fill="none" stroke="red" stroke-width="10" stroke-dasharray="60 100"/>`)
	expect(t, img, "a dash round a corner", 64, 15, redPx, 0)
	expect(t, img, "its end", 60, 45, clearPx, 0)
	// Dots: dashes of no length with round caps.
	img = draw(t, `<path d="M10 50H90" stroke="red" stroke-width="10" stroke-dasharray="0 20" `+
		`stroke-linecap="round"/>`)
	expect(t, img, "the first dot, at the start", 11, 50, redPx, 0)
	expect(t, img, "a dot", 30, 50, redPx, 0)
	expect(t, img, "between dots", 40, 50, clearPx, 0)
}

func TestOpacity(t *testing.T) {
	const two = `<rect x="10" y="10" width="50" height="50" fill="red"/>` +
		`<rect x="30" y="30" width="50" height="50" fill="blue"/>`
	img := draw(t, `<g opacity=".5">`+two+`</g>`)
	expect(t, img, "a faded group: red alone", 20, 20, halfRed, 1)
	expect(t, img, "a faded group: blue hides red, then fades", 40, 40, halfBlue, 1)
	img = draw(t, `<g fill-opacity=".5">`+two+`</g>`)
	expect(t, img, "faded fills: blue over red", 40, 40, color.RGBA{64, 0, 128, 191}, 1)
	img = draw(t, `<rect x="10" y="10" width="50" height="50" fill="red" stroke="blue" stroke-width="10" `+
		`opacity=".5"/>`)
	expect(t, img, "a faded shape: its stroke hides its fill, then fades", 12, 30, halfBlue, 1)
	img = draw(t, `<rect x="10" y="10" width="50" height="50" fill="red" stroke="blue" stroke-width="10" `+
		`stroke-opacity=".5"/>`)
	expect(t, img, "a faded stroke over its fill", 12, 30, color.RGBA{128, 0, 128, 255}, 1)
	// Groups nested deeper than layers go still fade.
	img = draw(t, `<g opacity=".5"><g opacity=".5"><g opacity=".5"><g opacity=".5"><g opacity=".5">`+
		`<g opacity=".5">`+two+
		`<rect/></g><rect/></g><rect/></g><rect/></g><rect/></g><rect/></g>`)
	if a := img.RGBAAt(20, 20).A; a < 3 || a > 5 {
		t.Errorf("six groups at half: alpha %d, want 4", a)
	}
	img = draw(t, `<rect width="100" height="100" fill="white"/><rect width="100" height="100" fill="#000" `+
		`opacity=".5"/>`)
	expect(t, img, "black at half over white", 50, 50, color.RGBA{128, 128, 128, 255}, 1)
}

func TestPaintOrder(t *testing.T) {
	const r = `<rect x="20" y="20" width="60" height="60" fill="red" stroke="blue" stroke-width="20" `
	expect(t, draw(t, r+`/>`), "the stroke over the fill", 25, 50, bluePx, 0)
	expect(t, draw(t, r+`paint-order="stroke"/>`), "the fill over the stroke", 25, 50, redPx, 0)
	expect(t, draw(t, r+`paint-order="stroke"/>`), "the stroke outside the fill", 15, 50, bluePx, 0)
}

func TestLinearGradientsArePainted(t *testing.T) {
	const stops = `<stop offset="0" stop-color="#000"/><stop offset="1" stop-color="#fff"/></linearGradient>` +
		`<rect width="100" height="100" fill="url(#g)"/>`
	img := draw(t, `<linearGradient id="g">`+stops)
	for _, x := range []int{0, 25, 50, 75, 99} {
		v := uint8(math.Round((float64(x) + 0.5) / 100 * 255))
		expect(t, img, "left to right", x, 50, color.RGBA{v, v, v, 255}, 1)
	}
	img = draw(t, `<linearGradient id="g" x2="0" y2="1">`+stops)
	expect(t, img, "top to bottom", 50, 25, color.RGBA{65, 65, 65, 255}, 1)
	img = draw(t, `<linearGradient id="g" gradientTransform="rotate(90)">`+stops)
	expect(t, img, "turned by gradientTransform", 50, 25, color.RGBA{65, 65, 65, 255}, 1)
	img = draw(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x1="20" x2="40">`+stops)
	expect(t, img, "pad before", 10, 50, blackPx, 0)
	expect(t, img, "between", 30, 50, color.RGBA{134, 134, 134, 255}, 1)
	expect(t, img, "pad after", 60, 50, whitePx, 0)
	img = draw(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x1="20" x2="40" `+
		`spreadMethod="repeat">`+stops)
	expect(t, img, "repeat", 45, 50, color.RGBA{70, 70, 70, 255}, 1)
	expect(t, img, "repeat before", 5, 50, color.RGBA{70, 70, 70, 255}, 1)
	img = draw(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x1="20" x2="40" `+
		`spreadMethod="reflect">`+stops)
	expect(t, img, "reflect", 45, 50, color.RGBA{185, 185, 185, 255}, 1)
	expect(t, img, "reflect twice", 65, 50, color.RGBA{70, 70, 70, 255}, 1)
	// Colours meet half way without premultiplying: clear red to blue is half-clear purple.
	img = draw(t, `<linearGradient id="g">
		<stop stop-color="red" stop-opacity="0"/><stop offset="1" stop-color="blue"/></linearGradient>
		<rect width="100" height="100" fill="url(#g)" fill-opacity=".5"/>`)
	expect(t, img, "half way from clear red to blue, at half", 50, 50, color.RGBA{32, 0, 32, 64}, 2)
	// A hard stop.
	img = draw(t, `<linearGradient id="g">
		<stop offset=".5" stop-color="red"/><stop offset=".5" stop-color="blue"/></linearGradient>
		<rect width="100" height="100" fill="url(#g)"/>`)
	expect(t, img, "before a hard stop", 48, 50, redPx, 0)
	expect(t, img, "after a hard stop", 51, 50, bluePx, 0)
	// On a stroke.
	img = draw(t, `<linearGradient id="g" gradientUnits="userSpaceOnUse" x2="100">
		<stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>
		<path d="M0 50H100" stroke="url(#g)" stroke-width="10"/>`)
	expect(t, img, "a gradient stroke", 25, 50, color.RGBA{190, 0, 65, 255}, 1)
}

func TestRadialGradientsArePainted(t *testing.T) {
	const stops = `<stop offset="0" stop-color="#fff"/><stop offset="1" stop-color="#000"/></radialGradient>` +
		`<rect width="100" height="100" fill="url(#g)"/>`
	img := draw(t, `<radialGradient id="g">`+stops)
	expect(t, img, "the centre", 50, 50, color.RGBA{253, 253, 253, 255}, 2)
	expect(t, img, "half way out", 75, 50, color.RGBA{125, 125, 125, 255}, 2)
	expect(t, img, "past the circle", 5, 5, blackPx, 0)
	// A focal point off centre: white moves there, the circle's edge stays.
	img = draw(t, `<radialGradient id="g" fx=".25" fy=".5">`+stops)
	expect(t, img, "the focal point", 25, 50, color.RGBA{252, 252, 252, 255}, 4)
	expect(t, img, "the centre is no longer white", 50, 50, color.RGBA{170, 170, 170, 255}, 3)
	expect(t, img, "the edge", 99, 50, color.RGBA{2, 2, 2, 255}, 3)
	// A focal radius: white inside it.
	img = draw(t, `<radialGradient id="g" fr=".25">`+stops)
	expect(t, img, "inside the focal circle", 60, 50, whitePx, 0)
	expect(t, img, "half way from it to the edge", 87, 50, color.RGBA{128, 128, 128, 255}, 5)
	img = draw(t, `<radialGradient id="g" gradientUnits="userSpaceOnUse" cx="0" cy="0" r="100" `+
		`gradientTransform="scale(1 .5)">`+stops)
	expect(t, img, "squashed by gradientTransform", 0, 25, color.RGBA{125, 125, 125, 255}, 3)
	expect(t, img, "squashed by gradientTransform", 50, 0, color.RGBA{126, 126, 126, 255}, 3)
}

func TestTheViewBoxIsFittedWhenDrawing(t *testing.T) {
	p, err := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 10 5"><rect width="10" height="5" ` +
		`fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	img, _ := p.Rasterise(100, 100)
	expect(t, img, "meet: over the box", 50, 20, clearPx, 0)
	expect(t, img, "meet: in the box", 50, 50, redPx, 0)
	expect(t, img, "meet: its corner", 0, 25, redPx, 0)
	wide, _ := p.Rasterise(200, 50)
	expect(t, wide, "meet: beside the box", 40, 25, clearPx, 0)
	expect(t, wide, "meet: in the box", 100, 25, redPx, 0)
	// Draw fits into a rectangle of an image, over what is there, and stays in it.
	dst := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for i := range dst.Pix {
		dst.Pix[i] = 255
	}
	p.Draw(dst, image.Rect(20, 20, 60, 40))
	expect(t, dst, "Draw: in the rectangle", 40, 30, redPx, 0)
	expect(t, dst, "Draw: outside it", 10, 30, whitePx, 0)
	expect(t, dst, "Draw: outside it", 40, 41, whitePx, 0)
	p.Draw(dst, image.Rect(80, 80, 200, 140))
	expect(t, dst, "Draw: a rectangle past the image", 99, 99, redPx, 0)
	// A picture with no view box is a unit a pixel of its own size.
	p, _ = NewPicture([]byte(`<svg ` + xmlns + ` width="10" height="10"><rect width="5" height="5" ` +
		`fill="red"/></svg>`))
	img, _ = p.Rasterise(100, 100)
	expect(t, img, "no view box: stretched from its size", 49, 49, redPx, 0)
	expect(t, img, "no view box: stretched from its size", 51, 51, clearPx, 0)
	p, _ = NewPicture([]byte(`<svg ` + xmlns + ` width="0" height="10"><rect width="5" height="5" ` +
		`fill="red"/></svg>`))
	img, _ = p.Rasterise(100, 100)
	expect(t, img, "a picture of no width", 1, 1, clearPx, 0)
}

func TestAlphaIsPremultiplied(t *testing.T) {
	img := draw(t, `<circle cx="50" cy="50" r="40" fill="#ff8000" fill-opacity=".5" stroke="#fff" `+
		`stroke-width="3" stroke-opacity=".3"/>`)
	for i := 0; i < len(img.Pix); i += 4 {
		if a := img.Pix[i+3]; img.Pix[i] > a || img.Pix[i+1] > a || img.Pix[i+2] > a {
			t.Fatalf("pixel %d: %v is brighter than its alpha", i/4, img.Pix[i:i+4])
		}
	}
	expect(t, img, "half-clear orange", 50, 50, color.RGBA{128, 64, 0, 128}, 1)
}

// distance is how far p is from the segment a–b.
func distance(p, a, b Point) float64 {
	d := b.sub(a)
	t := math.Max(0, math.Min(1, p.sub(a).dot(d)/d.dot(d)))
	return p.sub(a.add(d.mul(t))).len()
}

// A stroke with round joins and caps is every point within half its
// width of the path: random paths that cross themselves, turn sharply
// and have segments shorter than the stroke is wide, checked pixel by
// pixel against that.
func TestARoundStrokeIsWhatLiesNearThePath(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for trial := range 300 {
		n := 2 + rng.IntN(7)
		pts := make([]Point, n)
		for i := range pts {
			pts[i] = Point{10 + 80*rng.Float64(), 10 + 80*rng.Float64()}
			if i > 0 && rng.IntN(3) == 0 {
				pts[i] = pts[i-1].add(Point{4 * rng.Float64(), 4 * rng.Float64()})
			}
		}
		closed := rng.IntN(2) == 0
		hw := 0.5 + 9*rng.Float64()
		var path Path
		path.moveTo(pts[0])
		for _, p := range pts[1:] {
			path.lineTo(p)
		}
		segs := n - 1
		if closed {
			path.close()
			segs = n
		}
		pic := &Picture{ViewBox: Box{0, 0, 100, 100}, Items: []Item{{Shape: &Shape{
			Path: path, Transform: Identity(), Stroke: Paint{Kind: Solid, Color: black, Opacity: 1},
			StrokeWidth: 2 * hw, Cap: RoundCap, Join: RoundJoin, MiterLimit: 4,
		}}}}
		img, err := pic.Rasterise(100, 100)
		if err != nil {
			t.Fatal(err)
		}
		for y := range 100 {
			for x := range 100 {
				p := Point{float64(x) + 0.5, float64(y) + 0.5}
				dist := math.Inf(1)
				for i := range segs {
					dist = math.Min(dist, distance(p, pts[i], pts[(i+1)%n]))
				}
				a := img.RGBAAt(x, y).A
				if dist < hw-0.75 && a != 255 || dist > hw+0.75 && a != 0 {
					t.Fatalf("trial %d: %v closed=%v hw=%.2f: pixel (%d, %d), %.2f from the path, has alpha %d",
						trial, pts, closed, hw, x, y, dist, a)
				}
			}
		}
	}
}

// Under miter and bevel joins a stroke still covers each segment's own
// rectangle, and stays within the miter limit of the path.
func TestAnyStrokeCoversItsSegments(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for trial := range 300 {
		n := 3 + rng.IntN(5)
		pts := make([]Point, n)
		for i := range pts {
			pts[i] = Point{20 + 60*rng.Float64(), 20 + 60*rng.Float64()}
		}
		hw := 0.5 + 4*rng.Float64()
		s := &Shape{
			Transform: Identity(), Stroke: Paint{Kind: Solid, Color: black, Opacity: 1},
			StrokeWidth: 2 * hw, Cap: Cap(rng.IntN(3)), Join: Join(rng.IntN(3)), MiterLimit: 1 + 3*rng.Float64(),
		}
		s.Path.moveTo(pts[0])
		for _, p := range pts[1:] {
			s.Path.lineTo(p)
		}
		segs := n - 1
		if rng.IntN(2) == 0 {
			s.Path.close()
			segs = n
		}
		img, err := (&Picture{ViewBox: Box{0, 0, 100, 100}, Items: []Item{{Shape: s}}}).Rasterise(100, 100)
		if err != nil {
			t.Fatal(err)
		}
		for y := range 100 {
			for x := range 100 {
				p := Point{float64(x) + 0.5, float64(y) + 0.5}
				inside, dist := false, math.Inf(1)
				for i := range segs {
					a, b := pts[i], pts[(i+1)%n]
					d := b.sub(a)
					along := p.sub(a).dot(d) / d.len()
					across := math.Abs(p.sub(a).cross(d) / d.len())
					inside = inside || along > 0.75 && along < d.len()-0.75 && across < hw-0.75
					dist = math.Min(dist, distance(p, a, b))
				}
				a := img.RGBAAt(x, y).A
				if inside && a != 255 {
					t.Fatalf("trial %d: %v %+v: pixel (%d, %d) inside a segment has alpha %d", trial, pts, s, x, y, a)
				}
				if dist > hw*math.Max(s.MiterLimit, math.Sqrt2)+0.75 && a != 0 {
					t.Fatalf("trial %d: %v %+v: pixel (%d, %d), %.2f from the path, has alpha %d", trial, pts, s, x, y, dist, a)
				}
			}
		}
	}
}

func TestCurvesAreFlattenedWithinTolerance(t *testing.T) {
	// A circle of radius 40 drawn 10 times larger: every flattened point
	// is on it, and no chord strays more than the tolerance in pixels.
	path := ellipsePath(Point{50, 50}, 40, 40)
	subs := flatten(path, 10, 0)
	if len(subs) != 1 || !subs[0].closed {
		t.Fatalf("%d subpaths", len(subs))
	}
	pts := subs[0].pts
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		mid := lerp(p, q, 0.5)
		// The cubics themselves are up to a tenth of a pixel off the circle.
		if sag := (40 - mid.sub(Point{50, 50}).len()) * 10; sag > tolerance*1.5 || sag < -0.12 {
			t.Fatalf("chord %d strays %.3f pixels", i, sag)
		}
	}
	if len(pts) > 400 {
		t.Errorf("%d points for a circle of radius 400 pixels", len(pts))
	}
	// A stroked curve takes the steps of its outer edge.
	tight := parsePath("M0 0 C0 1 1 1 1 0", 10)
	if plain, wide := flatten(tight, 1, 0), flatten(tight, 1, 50); len(wide[0].pts) < 4*len(plain[0].pts) {
		t.Errorf("%d points for a wide stroke, %d for a fill", len(wide[0].pts), len(plain[0].pts))
	}
	// Coordinates no picture has still flatten, into a bounded number of points.
	huge := parsePath("M0 0 C1e300 1e300 -1e300 1e300 1e300 0 Q1e308 1e308 0 0", 10)
	if got := flatten(huge, 1e10, 1e10); len(got[0].pts) > 600 {
		t.Errorf("%d points for two huge curves", len(got[0].pts))
	}
}

func TestShapesFarOutsideOrBrokenDrawNothingAndDoNotPanic(t *testing.T) {
	for _, body := range []string{
		`<rect x="1e30" width="1e30" height="1e30"/>`,
		`<path d="M-1e300 -1e300 L1e300 1e300 L0 1e300Z" stroke="red" stroke-width="1e300"/>`,
		`<circle r="1e200" stroke="red" stroke-width="1e-200" stroke-dasharray="1e-300"/>`,
		`<path d="M0 0 L50 50" stroke="red" transform="matrix(1e200 1e200 1e200 1e200 0 0)"/>`,
		`<path d="M0 0 L50 50" stroke="red" transform="scale(1e-320)"/>`,
		`<path d="M0 0 A1e-320 1e-320 0 0 1 50 50 A1e300 1e300 1e300 1 1 0 0" stroke="red" stroke-linejoin="round"/>`,
		`<radialGradient id="g" r="1e-320" fr="1e300" fx="1e300"><stop/><stop offset="1"/></radialGradient>` +
			`<rect width="100" height="100" fill="url(#g)"/>`,
		`<linearGradient id="g" x2="1e-320"><stop/><stop offset="1"/></linearGradient>` +
			`<rect width="100" height="100" fill="url(#g)"/>`,
	} {
		draw(t, body)
	}
}

func TestAModelMadeByHandIsDrawnOrLeftOut(t *testing.T) {
	stroke := Paint{Kind: Solid, Color: black, Opacity: 1}
	line := Path{Verbs: []Verb{MoveTo, LineTo}, Points: []Point{{10, 50}, {90, 50}}}
	empty := Paint{Kind: GradientPaint, Gradient: &Gradient{}, Opacity: 1}
	for name, s := range map[string]*Shape{
		"points missing":  {Path: Path{Verbs: []Verb{MoveTo, CubicTo}, Points: []Point{{1, 1}}}, Fill: stroke},
		"no move first":   {Path: Path{Verbs: []Verb{LineTo}, Points: []Point{{1, 1}}}, Fill: stroke},
		"an unknown verb": {Path: Path{Verbs: []Verb{MoveTo, 9}, Points: []Point{{1, 1}}}, Fill: stroke},
		"odd dashes":      {Path: line, Stroke: stroke, StrokeWidth: 4, Dashes: []float64{3}},
		"negative dashes": {Path: line, Stroke: stroke, StrokeWidth: 4, Dashes: []float64{-3, 1}},
		"no dash offset":  {Path: line, Stroke: stroke, StrokeWidth: 4, Dashes: []float64{3, 1}, DashOffset: math.NaN()},
		"no transform":    {Path: line, Stroke: stroke, StrokeWidth: 4},
		"no stops":        {Path: line, StrokeWidth: 4, Stroke: empty},
		"endless": {Path: line, StrokeWidth: 4, Stroke: Paint{Kind: GradientPaint, Opacity: 1, Gradient: &Gradient{
			Stops: []Stop{{0, red}, {1, blue}}, End: Point{math.Inf(1), 0}, Spread: Repeat, Transform: Identity(),
		}}},
	} {
		if name != "no transform" {
			s.Transform = Identity()
		}
		p := &Picture{ViewBox: Box{0, 0, 100, 100}, Items: []Item{{Shape: s}, {}, {Group: &Group{Opacity: 0.5}}}}
		if _, err := p.Rasterise(100, 100); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOnePictureIsDrawnFromSeveralGoroutines(t *testing.T) {
	data, err := os.ReadFile("testdata/own-gradients.svg")
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPicture(data)
	if err != nil {
		t.Fatal(err)
	}
	want, err := p.Rasterise(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			got, drawErr := p.Rasterise(64, 64)
			if drawErr != nil || !bytes.Equal(got.Pix, want.Pix) {
				t.Errorf("a picture drawn beside others differs (%v)", drawErr)
			}
		})
	}
	wg.Wait()
}

func TestAPictureWithNoViewBoxKeepsItsUnitACSSPixelOnAScaledScreen(t *testing.T) {
	bare, err := NewPicture([]byte(`<svg ` + xmlns + `><rect width="10" height="10" fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	one, _ := bare.Rasterise(40, 40)
	expect(t, one, "a unit a pixel", 5, 5, redPx, 0)
	expect(t, one, "a unit a pixel: past the square", 15, 15, clearPx, 0)
	two, err := bare.RasteriseScaled(40, 40, 2)
	if err != nil {
		t.Fatal(err)
	}
	expect(t, two, "a unit two pixels", 15, 15, redPx, 0)
	expect(t, two, "a unit two pixels: past the square", 25, 25, clearPx, 0)

	wide, err := NewPicture([]byte(`<svg ` + xmlns + ` width="20"><rect width="10" height="10" fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	img, _ := wide.RasteriseScaled(80, 40, 2)
	expect(t, img, "the axis with a size is stretched to it", 35, 15, redPx, 0)
	expect(t, img, "the other keeps its unit", 35, 25, clearPx, 0)

	boxed, err := NewPicture([]byte(`<svg ` + xmlns + ` viewBox="0 0 10 10"><rect width="5" height="10" ` +
		`fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	img, _ = boxed.RasteriseScaled(40, 40, 2)
	expect(t, img, "a view box is fitted whatever the scale", 15, 20, redPx, 0)
	expect(t, img, "a view box is fitted whatever the scale", 25, 20, clearPx, 0)
	if _, err = boxed.RasteriseScaled(MaxSide+1, 1, 2); err != ErrTooLarge { //nolint:errorlint // the sentinel itself
		t.Errorf("over MaxSide: %v", err)
	}
}
