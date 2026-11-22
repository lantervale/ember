//go:build chrome

package css_test

import (
	"strconv"
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/style"
)

// TestMathFunctionsComputeAsInChrome compares the values math functions
// compute to in each type: lengths, numbers, angles in transforms, and
// the percentages of a line-height, which compute against the font.
func TestMathFunctionsComputeAsInChrome(t *testing.T) {
	values := []struct{ width, opacity, lineHeight, transform, grow string }{
		{"round(13px, 5px)", "sign(-3)", "min(150%, 40px)", "rotate(atan2(1em, 10px))", "round(up, 1.2, 0.5)"},
		{"round(up, 1em, 3px)", "abs(-0.5)", "max(120%, 10px)", "rotate(calc(45deg + 0.5turn))", "abs(-2)"},
		{"calc(-1 * round(down, -13px, 5px))", "sin(30deg)", "clamp(10px, 200%, 50px)", "rotate(asin(0.5))", "pow(2, 3)"},
		{"round(to-zero, 13px, 5px)", "calc(1px / 4px)", "round(150%, 4px)", "rotate(min(10deg, 1rad))", "sqrt(9)"},
		{"mod(-7px, 5px)", "calc(1em / 40px)", "calc(1em / 10px)", "rotate(calc(10deg * sign(1px)))", "log(8, 2)"},
		{"calc(-1 * rem(-7px, 5px))", "calc(infinity)", "calc(2 * sign(50% - 1px))", "rotate(calc(1rad * pi))", "exp(0)"},
		{"abs(-12px)", "calc(-infinity)", "abs(-80%)", "rotate(acos(-1))", "hypot(3, 4)"},
		{"hypot(3px, 4px)", "calc(NaN)", "calc(1.5 * e)", "rotate(atan(1))", "clamp(1, 5, none)"},
		{"clamp(none, 50px, 20px)", "round(up, 0.33, 0.25)", "max(1, 2)", "skew(atan2(1, 2), calc(0.1turn / 2))", "1"},
		{"calc(10px / 1px * 3px)", "calc(50% * 2)", "calc(150% * 2)", "rotate(calc(30deg * 2 / 3))", "mod(7, 3)"},
		{"calc(1px * pow(2, 7))", "calc(1 / 0)", "calc(10px * abs(50% - 10px) / 1px)", "scale(calc(1px / 2px))", "rem(7, 3)"},
		{"calc(20px * pi)", "calc(1 - 1 / 0)", "round(nearest, 25%, 3px)", "scale(abs(-2), sign(-1))", "round(2.5)"},
	}
	css := "body { font-size: 10px; } div { height: 4px; }\n"
	d := ui.NewDocument()
	for i, v := range values {
		class := "m" + strconv.Itoa(i)
		css += "." + class + " { width: " + v.width + "; opacity: " + v.opacity + "; line-height: " + v.lineHeight +
			"; transform: " + v.transform + "; flex-grow: " + v.grow + "; }\n"
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	d.AddSheet(css)
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"width":       func(s *style.Computed) string { return lengthCSS(s.Size.Width) },
		"opacity":     func(s *style.Computed) string { return num(s.Opacity) },
		"line-height": compared["line-height"],
		"transform":   func(s *style.Computed) string { return matrixCSS(s.Transform) },
		"flex-grow":   func(s *style.Computed) string { return num(s.Grow) },
	}, 2)
}

// TestMathFunctionsInShapesComputeAsInChrome compares basic shapes whose
// math is known at compute time, written as Chrome writes them.
func TestMathFunctionsInShapesComputeAsInChrome(t *testing.T) {
	shapes := []string{"circle(round(13px, 5px))", "inset(abs(-4px) mod(-7px, 5px) hypot(3px, 4px) 1em)",
		"circle(calc(1em * sign(-1px) * -2) at calc(20px * sin(30deg)) 30%)", "ellipse(clamp(none, 50px, 20px) 2em)",
		"polygon(calc(10px / 1px * 3px) 0, 50% round(up, 1em, 3px), 0 calc(20px * cos(0.5turn) + 30px))",
		"inset(10px round calc(2px * pow(2, 3)))"}
	sheet := "body { font-size: 10px; } div { width: 100px; height: 50px; }\n"
	d := ui.NewDocument()
	for i, s := range shapes {
		sheet += ".s" + strconv.Itoa(i) + " { clip-path: " + s + "; }\n"
		d.Body().Append(ui.Div(ui.Class("s" + strconv.Itoa(i))))
	}
	d.AddSheet(sheet)
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"clip-path": func(s *style.Computed) string { return css.ClipPathCSS(s.ClipPath) },
	}, 2)
}
