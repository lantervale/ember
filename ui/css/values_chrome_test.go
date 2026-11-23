//go:build chrome

package css_test

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// colourFunctions are the colours of TestColourFunctionsComputeAsInChrome:
// every space, math in the channels, relative colours and color-mix().
//
//nolint:gochecknoglobals // a test's table
var colourFunctions = []string{
	"hsl(120 100% 25%)", "hsl(120deg 100% 25% / 0.5)", "hsla(240, 100%, 50%, 0.5)", "hsl(0.5turn 50 50)",
	"hsl(none 0% 50%)", "hsl(calc(60 * 2) 50% 50%)", "hsl(-120 60% 40%)", "hsl(30 150% 50%)",
	"hwb(200 20% 30%)", "hwb(90deg 10 10 / 50%)", "hwb(0 60% 60%)",
	"lab(50 40 -20)", "lab(70% 20% 30%)", "lab(30 none 10 / 0.75)",
	"lch(60 40 120)", "lch(40% 30% 300deg)", "lch(70 -10 40)",
	"oklab(0.6 0.1 -0.1)", "oklab(60% 20% 10%)", "oklch(0.7 0.1 200)",
	"color(srgb 0.2 0.4 0.6)", "color(srgb-linear 0.2 0.4 0.6)", "color(display-p3 0.3 0.5 0.7)",
	"color(a98-rgb 0.5 0.4 0.3)", "color(prophoto-rgb 0.5 0.4 0.3)", "color(rec2020 0.5 0.4 0.3)",
	"color(xyz 0.2 0.3 0.4)", "color(xyz-d50 0.2 0.3 0.4)", "color(xyz-d65 0.2 0.3 0.4)",
	"color(srgb 50% 20% 10% / 0.5)", "color(srgb 1.5 -0.5 0.5)",
	"rgb(calc(255 / 2) calc(10 + 20) 0)", "rgb(calc(100% / 3) 50% 0%)", "rgba(calc(10 * 10), 50, 50, calc(0.5 + 0.25))",
	"rgb(min(300, 120) max(10, 20) clamp(0, 400, 255))", "rgb(10 20 30 / calc(25% * 2))",
	"oklch(calc(0.5 + 0.1) calc(0.1 * 2) calc(90deg * 2))", "lab(calc(25 * 2) calc(-10 * 3) round(17, 5))",
	"rgb(from red r g b)", "rgb(from red calc(r / 2) g b)", "rgb(from #336699 b g r / 0.5)",
	"rgb(from rgb(10 20 30 / 0.4) r g b / alpha)", "rgb(from #fc0 r 50% none)",
	"hsl(from blue calc(h + 120) s l)", "hsl(from rgb(200 100 50) h s calc(l * 0.5))",
	"lab(from #369 l a b)", "lch(from orange l c calc(h + 180))", "oklch(from teal calc(l + 0.1) c h)",
	"oklab(from green l calc(a * -1) b)", "hwb(from #808080 h w b)", "hwb(from #3c6 h calc(w + 20) b)",
	"color(from red srgb calc(r * 0.5) g b)", "color(from #369 display-p3 r g b)", "color(from #369 xyz x y z)",
	"rgb(from hsl(120 50% 50%) r g b / calc(alpha / 2))", "hsl(from white h s l)",
	"color-mix(in srgb, red, blue)", "color-mix(in srgb, red 30%, blue)", "color-mix(in oklch, red, blue)",
	"color-mix(in oklch longer hue, red, blue)", "color-mix(in hsl, red 20%, blue 20%)", "color-mix(in lab, white, black)",
	"color-mix(in srgb-linear, red, lime)", "color-mix(in xyz, #369 25%, #fc0)", "color-mix(in oklab, red 40%, transparent)",
	"color-mix(in hsl shorter hue, hsl(350 50% 50%), hsl(10 50% 50%))", "color-mix(in lch decreasing hue, red, blue)",
	"color-mix(in lch increasing hue, red, blue)", "color-mix(in oklch, white, blue)", "color-mix(in hwb, red, blue)",
	"color-mix(in display-p3, red 75%, blue)", "color-mix(in srgb, red 10%, blue 30%)",
	"color-mix(in srgb, rgb(from red r g b) 50%, color-mix(in lab, blue, green))",
	"color-mix(in srgb, currentcolor 50%, white)", "rgb(from currentcolor r g calc(b / 2))",
	"color-mix(in srgb, light-dark(red, blue), white)", "color-mix(in srgb, red 0%, blue 0%)", "rgb(10% 20% 30%)",
	"rgba(10%, 20%, 30%, 40%)", "rgb(from Canvas r g b)",
}

// refusedColours are colours both refuse.
//
//nolint:gochecknoglobals // a test's table
var refusedColours = []string{
	"rgb(10, 20 30)", "hsl(120, 50, 50)", "hsl(120 50% 50% 0.5)", "rgb(from red r g)",
	"rgb(from red x g b)", "rgb(calc(1px) 0 0)", "color(foo 0 0 0)", "color-mix(in srgb, red)",
	"color-mix(in foo, red, blue)", "color-mix(in srgb longer hue, red, blue)",
	"color-mix(in srgb, red -10%, blue)", "lab(from red l a)", "hwb(10, 20%, 30%)", "rgb(from red r, g, b)",
}

// TestColourFunctionsComputeAsInChrome has Chrome write the computed
// colour of each of colourFunctions into a canvas, eight bits a channel
// in sRGB, and compares ui's colour with it; and asks Chrome whether it
// reads each of refusedColours.
func TestColourFunctionsComputeAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS("body { color: rgb(0 128 255) }"))
	for _, c := range colourFunctions {
		d.Body().Append(ui.Div(ui.Style("color: " + c)))
	}
	refused, _ := json.Marshal(refusedColours)
	raw, err := chrome.Eval(d, geom.V2(400, 300), `
const cv = document.createElement("canvas");
cv.width = cv.height = 1;
const cx = cv.getContext("2d", {willReadFrequently: true});
const px = c => {
	cx.clearRect(0, 0, 1, 1);
	cx.fillStyle = c;
	cx.fillRect(0, 0, 1, 1);
	return Array.from(cx.getImageData(0, 0, 1, 1).data);
};
return {
	Colours: [...document.body.children].filter(e => e.tagName == "DIV").map(e => px(getComputedStyle(e).color)),
	Refused: `+string(refused)+`.map(c => CSS.supports("color", c)),
};`)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Colours [][4]float64
		Refused []bool
	}
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, w := range d.Warnings() {
		t.Errorf("warning: %v", w)
	}
	d.Layout(geom.V2(400, 300))
	i := 0
	for _, n := range d.Body().Children() {
		if n.IsText() {
			continue
		}
		c := dom.Data(n).Style.Color
		s := c.SRGB()
		ours := [4]float64{float64(s[0]) * 255, float64(s[1]) * 255, float64(s[2]) * 255, float64(c.A) * 255}
		want := got.Colours[i]
		// The canvas keeps premultiplied bytes: a faint colour's channels
		// come back rounded to a step of 255/alpha.
		tol := 1.5 * 255 / max(want[3], 1)
		for k := range 4 {
			if k < 3 && ours[3] < 1 || math.Abs(ours[k]-want[k]) <= tol {
				continue
			}
			t.Errorf("%s: ember %v, Chrome %v", colourFunctions[i], round(ours), want)
			break
		}
		i++
	}
	for k, c := range refusedColours {
		_, warns := css.Parse("t.css", "p { color: "+c+" }")
		if got.Refused[k] || len(warns) == 0 {
			t.Errorf("%s: Chrome reads it %v, ui warns %v", c, got.Refused[k], warns)
		}
	}
}

func round(v [4]float64) [4]string {
	var out [4]string
	for i, x := range v {
		out[i] = strconv.FormatFloat(x, 'f', 1, 64)
	}
	return out
}

// TestMathFunctionsInTimesComputeAsInChrome compares durations and delays
// written with math functions, in the longhands and the shorthands, a
// negative duration clamped to 0.
func TestMathFunctionsInTimesComputeAsInChrome(t *testing.T) {
	times := []string{
		"transition: opacity calc(1s + 100ms) calc(-1 * 250ms)",
		"transition: opacity max(2s, 300ms * 10), color calc(1s / 4)",
		"transition: color 1s; transition-delay: clamp(0s, 5s, 2s)",
		"transition: opacity calc(0.5s - 1s)",
		"animation: spin calc(2 * 1.5s) linear calc(100ms * 3)",
		"animation-name: spin; animation-duration: round(1.26s, 0.1s); animation-delay: abs(-2s)",
		"animation: spin calc(1s - 2s)",
	}
	sheet := "@keyframes spin { to { opacity: 0 } }\n"
	d := ui.NewDocument()
	for i, v := range times {
		sheet += ".t" + strconv.Itoa(i) + " { " + v + " }\n"
		d.Body().Append(ui.Div(ui.Class("t" + strconv.Itoa(i))))
	}
	d.AddSheet(sheet)
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"transition-duration": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return seconds(t.Duration) })
		},
		"transition-delay": func(s *style.Computed) string {
			return transitionsCSS(s.Transitions, func(t style.Transition) string { return seconds(t.Delay) })
		},
		"animation-duration": func(s *style.Computed) string {
			return animationsCSS(s.Animations, func(a style.Animation) string { return seconds(a.Duration) })
		},
		"animation-delay": func(s *style.Computed) string {
			return animationsCSS(s.Animations, func(a style.Animation) string { return seconds(a.Delay) })
		},
	}, 2)
}

// TestPercentagesInSignInALineHeightComputeAsInChrome compares a
// line-height whose sign() reads a percentage: Chrome reads it as a
// percentage of 100 pixels there, not of the font size as elsewhere in
// the line-height.
func TestPercentagesInSignInALineHeightComputeAsInChrome(t *testing.T) {
	values := []string{"calc(20px + 10px * sign(1% - 0.5px))", "calc(20px + 10px * sign(1% - 1.5px))",
		"calc(20px + 10px * sign(10% - 9px))", "calc(20px * sign(50% - 49px) + 150%)",
		"calc(2 * sign(5% - 6px) + 3)", "calc(20px + 10px * sign(abs(-1%) - 0.5px))",
		"calc(10px * sign(1% - 1px) + 10px)", "max(150%, 10px * sign(2% - 1px))"}
	sheet := "body { font-size: 10px; } div { height: 4px; }\n"
	d := ui.NewDocument()
	for i, v := range values {
		sheet += ".l" + strconv.Itoa(i) + " { line-height: " + v + " }\n"
		d.Body().Append(ui.Div(ui.Class("l" + strconv.Itoa(i))))
	}
	d.AddSheet(sheet)
	matchChromeProps(t, d, map[string]func(*style.Computed) string{"line-height": compared["line-height"]}, 2)
}

func animationsCSS(as []style.Animation, f func(style.Animation) string) string {
	if len(as) == 0 {
		return "0s"
	}
	parts := make([]string, len(as))
	for i, a := range as {
		parts[i] = f(a)
	}
	return strings.Join(parts, ", ")
}
