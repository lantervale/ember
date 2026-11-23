package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"image/color"
	"slices"
	"testing"
)

func TestAMaskOfACSSGradient(t *testing.T) {
	for _, c := range []struct {
		style       string
		shown, gone [][2]int
	}{
		{"mask: linear-gradient(to right, black 40%, transparent 60%)", [][2]int{{20, 50}}, [][2]int{{80, 50}}},
		{"mask-image: linear-gradient(90deg, black 30%, transparent 30% 60%, black 60%)",
			[][2]int{{20, 50}, {80, 50}}, [][2]int{{45, 50}}},
		{"mask-image: linear-gradient(to bottom right, black 50%, transparent 50%)",
			[][2]int{{10, 10}, {90, 5}}, [][2]int{{90, 90}, {55, 55}}},
		{"mask-image: radial-gradient(circle 20px, black 99%, transparent)", [][2]int{{50, 50}, {50, 32}},
			[][2]int{{50, 25}, {10, 10}}},
		{"mask-image: radial-gradient(closest-side at 25% 50%, black 99%, transparent)",
			[][2]int{{25, 50}, {48, 50}, {25, 2}}, [][2]int{{55, 50}, {2, 2}}},
		{"mask-image: repeating-linear-gradient(90deg, black 0 10px, transparent 10px 20px)",
			[][2]int{{5, 50}, {25, 50}, {85, 50}}, [][2]int{{15, 50}, {35, 50}, {95, 50}}},
		{"mask: linear-gradient(black, black) luminance", nil, [][2]int{{50, 50}}},
		{"mask: linear-gradient(white, white) luminance", [][2]int{{50, 50}}, nil},
	} {
		img := draw(t, `<rect width="100" height="100" fill="red" style="`+c.style+`"/>`)
		for _, at := range c.shown {
			expect(t, img, c.style+": shown", at[0], at[1], redPx, 2)
		}
		for _, at := range c.gone {
			expect(t, img, c.style+": hidden", at[0], at[1], clearPx, 2)
		}
	}
}

func TestACSSMaskIsPlacedOnTheFillBoxAndRepeated(t *testing.T) {
	const shape = `<rect x="30" y="30" width="40" height="40" fill="red" stroke="blue" stroke-width="10" style="`
	img := draw(t, shape+`mask: linear-gradient(to right, black 50%, transparent 50%)"/>`)
	expect(t, img, "the left of the fill box", 40, 50, redPx, 0)
	expect(t, img, "its right", 65, 50, clearPx, 0)
	expect(t, img, "repeated right of it, over the stroke", 72, 50, bluePx, 0)
	expect(t, img, "and left of it", 27, 50, clearPx, 0)
	img = draw(t, shape+`mask: linear-gradient(black, black) no-repeat"/>`)
	expect(t, img, "not repeated", 27, 50, clearPx, 0)
	img = draw(t, shape+`mask: linear-gradient(black, black) fill-box"/>`)
	expect(t, img, "cut to the fill box", 72, 50, clearPx, 0)
	img = draw(t, shape+`mask-image: radial-gradient(closest-side, black 99%, transparent); mask-origin: stroke-box"/>`)
	expect(t, img, "placed on the stroke box", 27, 50, bluePx, 0)
	img = draw(t, `<g style="mask: linear-gradient(black, transparent)"><rect width="100" height="100" fill="red"/></g>`)
	if px := img.RGBAAt(50, 50); px.A < 120 || px.A > 135 {
		t.Errorf("a group's box: alpha %d in the middle, want about half", px.A)
	}
}

func TestMasksThatAreNotDrawn(t *testing.T) {
	for style, want := range map[string][]string{
		"mask: circle(40%)":                                               nil,
		"mask: conic-gradient(red, blue)":                                 {"mask"},
		"mask: linear-gradient(red, blue), linear-gradient(red, blue)":    {"mask"},
		"mask: linear-gradient(red, blue) center / 50% 50%":               {"mask"},
		"mask-image: none":                                                nil,
		"mask: linear-gradient(red, blue) repeat-x stroke-box add":        nil,
		"mask-image: linear-gradient(red, blue); mask-composite: exclude": {"mask"},
	} {
		p := picture(t, `<rect width="10" height="10" fill="red" style="`+style+`"/>`)
		if got := p.Unsupported(); !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", style, got, want)
		}
	}
	img := draw(t, `<rect width="100" height="100" fill="red" style="mask: circle(40%)"/>`)
	expect(t, img, "a mask that is no image leaves the element", 5, 5, color.RGBA{255, 0, 0, 255}, 0)
}
