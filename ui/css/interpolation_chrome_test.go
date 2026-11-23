//go:build chrome

package css_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// interpolated are the properties checked moving between two values, each
// group on an element of its own, with the value before and after.
//
//nolint:gochecknoglobals // a test table
var interpolated = [][][3]string{
	{
		{"font-size", "10px", "20px"}, {"font-weight", "400", "700"}, {"letter-spacing", "normal", "10px"},
		{"word-spacing", "0px", "50%"}, {"line-height", "1", "3"}, {"text-indent", "0px", "50%"},
		{"vertical-align", "10px", "-6px"},
		{"text-shadow", "1px 2px 3px rgb(200, 0, 0)", "4px 5px 6px rgb(0, 0, 200), 1px 1px 1px green"},
	},
	{
		{"border-top-width", "1px", "11px"}, {"outline-width", "0px", "10px"}, {"outline-offset", "0px", "-10px"},
		{"border-top-left-radius", "0px", "10px"}, {"max-width", "100px", "50%"}, {"min-height", "0px", "40px"},
		{"flex-basis", "10px", "100px"}, {"flex-grow", "0", "3"}, {"order", "0", "-3"}, {"z-index", "0", "3"},
	},
	{
		{"column-count", "1", "5"}, {"column-width", "10px", "30px"}, {"column-gap", "0px", "10px"},
		{"row-gap", "10px", "0px"}, {"column-rule-width", "1px", "5px"}, {"column-rule-color", "red", "blue"},
		{"grid-template-columns", "100px 1fr", "200px 3fr"}, {"orphans", "1", "5"}, {"widows", "5", "2"},
	},
	{
		{"background-position", "right 10px top", "0px 0px"}, {"background-size", "10px auto", "30px auto"},
		{"mask-position", "0px 0px", "10px 20px"}, {"mask-size", "10px 10px", "20px 30px"},
		{"object-position", "0px 0px", "right 10px bottom 20%"}, {"border-spacing", "0px", "10px 20px"},
		{"aspect-ratio", "1 / 1", "4 / 1"},
	},
	{
		{"width", "100px", "200px"}, {"height", "40px", "10px"}, {"margin-top", "0px", "20px"},
		{"padding-left", "4px", "24px"}, {"top", "0px", "-20px"}, {"left", "10px", "30px"},
		{"transform-origin", "0px 0px", "10px 20px"}, {"flex-shrink", "1", "0"},
		{"border-bottom-right-radius", "20px", "0px"},
	},
	{
		{"caret-color", "blue", "red"}, {"accent-color", "blue", "red"}, {"text-decoration-color", "red", "blue"},
		{"text-decoration-thickness", "1px", "4px"}, {"border-image-slice", "10", "30"},
		{"border-image-width", "1", "3"}, {"visibility", "hidden", "visible"},
	},
	{
		{"-webkit-text-stroke-width", "1px", "5px"}, {"-webkit-text-stroke-color", "red", "blue"},
	},
}

// moved is how to write each of interpolated as getComputedStyle does.
//
//nolint:gochecknoglobals // a test table
var moved = func() map[string]func(*style.Computed) string {
	m := map[string]func(*style.Computed) string{
		"text-indent": func(s *style.Computed) string {
			if s.TextIndent.Kind == style.Auto {
				return "0px"
			}
			return lengthCSS(s.TextIndent)
		},
		"vertical-align": func(s *style.Computed) string {
			if s.VerticalAlign != style.AlignLength {
				return "baseline"
			}
			return lengthCSS(s.VerticalShift)
		},
		"column-count": func(s *style.Computed) string {
			if s.ColumnCount == 0 {
				return "auto"
			}
			return strconv.Itoa(s.ColumnCount)
		},
		"column-width":      func(s *style.Computed) string { return lengthCSS(s.ColumnWidth) },
		"column-rule-width": func(s *style.Computed) string { return px(s.ColumnRuleWidth) },
		"column-rule-color": func(s *style.Computed) string { return rgb(s.ColumnRuleColor) },
		"background-position": func(s *style.Computed) string {
			p := s.BackgroundLayer(0).Position
			return positionCSS(p.X, p.FromRight) + " " + positionCSS(p.Y, p.FromBottom)
		},
		"background-size": func(s *style.Computed) string { return sizeCSS(s.BackgroundLayer(0).Size) },
		"mask-position": func(s *style.Computed) string {
			p := s.MaskLayer(0).Position
			return positionCSS(p.X, p.FromRight) + " " + positionCSS(p.Y, p.FromBottom)
		},
		"mask-size": func(s *style.Computed) string { return sizeCSS(s.MaskLayer(0).Size) },
		"border-spacing": func(s *style.Computed) string {
			if s.BorderSpacing[0] == s.BorderSpacing[1] {
				return px(s.BorderSpacing[0])
			}
			return px(s.BorderSpacing[0]) + " " + px(s.BorderSpacing[1])
		},
		"text-decoration-color":     func(s *style.Computed) string { return rgb(s.TextDecoration.Color) },
		"text-decoration-thickness": func(s *style.Computed) string { return lengthCSS(s.TextDecoration.Thickness) },
		"border-image-width": func(s *style.Computed) string {
			return strconv.FormatFloat(float64(s.BorderImageWidth.Top.V), 'f', -1, 32)
		},
		"orphans": func(s *style.Computed) string { return strconv.Itoa(s.Orphans) },
		"widows":  func(s *style.Computed) string { return strconv.Itoa(s.Widows) },
	}
	for _, group := range interpolated {
		for _, p := range group {
			if m[p[0]] == nil {
				m[p[0]] = everything[p[0]]
			}
		}
	}
	return m
}()

func declarations(group [][3]string, end int) string {
	var b strings.Builder
	for _, p := range group {
		fmt.Fprintf(&b, "%s: %s; ", p[0], p[end])
	}
	return b.String()
}

// TestInterpolationMatchesChrome moves every property ui interpolates by a
// keyframes animation and by a transition, and compares them with
// Chrome's paused at four times.
func TestInterpolationMatchesChrome(t *testing.T) {
	var src strings.Builder
	src.WriteString("div { font-size: 10px; border: 1px solid; outline: 1px solid; column-rule: 1px solid; " +
		"text-decoration: underline; position: relative; width: 100px; height: 40px; top: 0; left: 0; " +
		"transform-origin: 0 0 }\n")
	for i, group := range interpolated {
		fmt.Fprintf(&src, "@keyframes k%d { from { %s} to { %s} }\n.k%d { animation: k%d 1s linear }\n",
			i, declarations(group, 1), declarations(group, 2), i, i)
		fmt.Fprintf(&src, ".t%d { %stransition: all 1s linear }\n.t%d.on { %s}\n",
			i, declarations(group, 1), i, declarations(group, 2))
	}
	props := map[string]func(*style.Computed) string{}
	for _, group := range interpolated {
		for _, p := range group {
			props[p[0]] = moved[p[0]]
		}
	}
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"i.css": {Data: []byte(src.String())}}, "i.css"))
			for i := range interpolated {
				d.Body().Append(ui.Div(ui.Class(fmt.Sprintf("k%d", i))), ui.Div(ui.Class(fmt.Sprintf("t%d", i))))
			}
			tr := css.NewTransitions()
			tr.Compute(d.HTML(), d.Sheets())
			script := holdAtStart + fmt.Sprintf(`els.forEach(e => getComputedStyle(e).fontSize);
document.querySelectorAll("[class^=t]").forEach(e => e.classList.add("on"));
els.forEach(e => getComputedStyle(e).fontSize);
document.getAnimations().forEach(a => { a.pause(); a.currentTime = %v; });`, at*1000)
			then := func() {
				d.HTML().Walk(func(n *dom.Node) bool {
					if strings.HasPrefix(strings.Join(n.Classes(), " "), "t") {
						n.AddClass("on")
					}
					return true
				})
				tr.Compute(d.HTML(), d.Sheets())
				tr.Step(at)
				tr.Compute(d.HTML(), d.Sheets())
			}
			matchChromeShown(t, d, props, 2, script, then, func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
		})
	}
}
