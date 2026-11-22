//go:build chrome

package css_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// matchMovedAt styles the body build makes under src in ui and in Chrome
// at scale device pixels per point, adds the class on to every element
// whose class begins with t, and compares props once every animation and
// transition is at the time at.
func matchMovedAt(t *testing.T, src string, build func(body *dom.Node), props map[string]func(*style.Computed) string,
	at, scale float32) {
	t.Helper()
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"m.css": {Data: []byte(src)}}, "m.css"))
	build(d.Body())
	tr := css.NewTransitions()
	tr.SetMedia(css.Media{Width: 800, Height: 600, Resolution: scale})
	tr.Compute(d.HTML(), d.Sheets())
	script := fmt.Sprintf(`els.forEach(e => getComputedStyle(e).fontSize);
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
		for left := at; left > 0; left -= frame {
			tr.Step(min(left, frame))
			tr.Compute(d.HTML(), d.Sheets())
		}
	}
	matchChromeScaled(t, d, scale, props, 2, script, then, func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
}

// frame is the time a frame of matchMovedAt moves by.
const frame = 0.05

func divs(classes ...string) func(*dom.Node) {
	return func(body *dom.Node) {
		for _, c := range classes {
			body.Append(ui.Div(ui.Class(c)))
		}
	}
}

//nolint:gochecknoglobals // a test table
var widthProps = map[string]func(*style.Computed) string{
	"border-top-width":   compared["border-top-width"],
	"border-left-width":  func(s *style.Computed) string { return px(s.Border.Left) },
	"border-right-width": func(s *style.Computed) string { return px(s.Border.Right) },
	"outline-width":      compared["outline-width"],
	"outline-offset":     compared["outline-offset"],
	"column-rule-width":  func(s *style.Computed) string { return px(s.ColumnRuleWidth) },
	"border-spacing":     func(s *style.Computed) string { return moved["border-spacing"](s) },
}

// TestWidthsSnapToDevicePixelsAsChromes compares border, outline and
// column-rule widths, outline-offset and border-spacing, given and
// moving, on displays of one and two pixels a point.
func TestWidthsSnapToDevicePixelsAsChromes(t *testing.T) {
	src := `div { border: 4.7px solid; border-left-width: 0.2px; border-right-width: 1px; outline: 0.3px solid; outline-offset: -3.7px;
  column-rule: 2.3px solid; border-spacing: 3.7px 0.3px; display: table }
.t0 { border-top-width: 1px; outline-width: 0px; outline-offset: 0px; border-spacing: 0px; column-rule-width: 1px;
  transition: all 1s linear }
.t0.on { border-top-width: 11px; outline-width: 10px; outline-offset: -10px; border-spacing: 10px 20px;
  column-rule-width: 6px }
.s1 { outline-offset: 3.7px; border-spacing: 3.2px }`
	for _, scale := range []float32{1, 1.5, 2} {
		for _, at := range []float32{0.1, 0.35, 0.65} {
			t.Run(fmt.Sprint(scale, "x/", at), func(t *testing.T) {
				matchMovedAt(t, src, divs("s0", "s1", "t0"), widthProps, at, scale)
			})
		}
	}
}

// TestFontWeightKeepsQuartersAsChromes moves font-weight on an eased
// transition, whose values fall between whole numbers.
func TestFontWeightKeepsQuartersAsChromes(t *testing.T) {
	src := `.t0 { font-weight: 400; transition: font-weight 1s ease } .t0.on { font-weight: 700 }
@keyframes w { from { font-weight: 100 } to { font-weight: 900 } }
.k0 { animation: w 1s ease-in }`
	props := map[string]func(*style.Computed) string{"font-weight": compared["font-weight"]}
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, divs("t0", "k0"), props, at, 1)
		})
	}
}

//nolint:gochecknoglobals // a test table
var discreteProps = map[string]func(*style.Computed) string{
	"display":    compared["display"],
	"position":   compared["position"],
	"text-align": compared["text-align"],
	"float": func(s *style.Computed) string {
		return []string{"none", "left", "right"}[s.Float]
	},
	"flex-basis": everything["flex-basis"],
	"opacity":    compared["opacity"],
}

// TestDiscreteMovesMatchChrome runs display and other discrete properties
// through keyframes and through transitions, with and without
// transition-behavior: allow-discrete: none stays out of an interval
// while the other end shows a box, every other value flips halfway.
func TestDiscreteMovesMatchChrome(t *testing.T) {
	src := `@keyframes k0 { from { display: none } to { display: block } }
@keyframes k1 { from { display: inline } to { display: flex } }
@keyframes k2 { from { float: left; text-align: left; position: static } to { float: right; text-align: right;
  position: relative } }
@keyframes k3 { 0% { display: none } 50% { display: grid } 100% { display: none } }
.k0 { animation: k0 1s linear } .k1 { animation: k1 1s linear } .k2 { animation: k2 1s linear }
.k3 { animation: k3 1s linear }
.t0 { display: block; opacity: 1; transition: all 1s linear allow-discrete } .t0.on { display: none; opacity: 0 }
.t1 { display: inline; transition: display 1s linear; transition-behavior: allow-discrete }
.t1.on { display: flex }
.t2 { float: left; text-align: left; position: static; transition: all 1s linear allow-discrete }
.t2.on { float: right; text-align: right; position: relative }
.t3 { float: left; text-align: left; transition: all 1s linear } .t3.on { float: right; text-align: right }
.t4 { flex-basis: auto; transition: flex-basis 1s linear allow-discrete } .t4.on { flex-basis: 100px }
.t5 { flex-basis: auto; transition: flex-basis 1s linear } .t5.on { flex-basis: 100px }
.t6 { display: none; transition: display 1s linear allow-discrete } .t6.on { display: block }`
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, divs("k0", "k1", "k2", "k3", "t0", "t1", "t2", "t3", "t4", "t5", "t6"),
				discreteProps, at, 1)
		})
	}
}

// TestStartingStyleMatchesChrome starts transitions from @starting-style:
// an element's first style, and one that leaves display: none, which
// without it starts none.
func TestStartingStyleMatchesChrome(t *testing.T) {
	src := `.t0 { display: none; opacity: 1; transition: opacity 1s linear, display 1s linear allow-discrete }
.t0.on { display: block }
@starting-style { .t0.on { opacity: 0 } }
.t1 { display: none; opacity: 0.2; transition: opacity 1s linear } .t1.on { display: block; opacity: 1 }
.s0 { opacity: 1; transition: opacity 1s linear }
@starting-style { .s0 { opacity: 0.2 } }
.t2 { opacity: 1; transition: opacity 1s linear } .t2.on { opacity: 0.5 }
@starting-style { .t2 { opacity: 0 } .t2.on { opacity: 0.1 } }`
	props := map[string]func(*style.Computed) string{"display": compared["display"], "opacity": compared["opacity"]}
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, divs("t0", "t1", "s0", "t2"), props, at, 1)
		})
	}
}

// TestAnimationCompositionMatchesChrome runs one @keyframes replacing,
// adding to and accumulating on each element's own values, then two
// animations, the second adding to what the first gives, and a keyframe
// of its own composition.
func TestAnimationCompositionMatchesChrome(t *testing.T) {
	src := `@keyframes w { from { width: 10px; text-indent: 10%; opacity: 0.2; color: rgb(100, 0, 0);
  transform: translateX(10px) scale(2) rotate(10deg); box-shadow: 1px 1px 1px rgb(200, 0, 0); filter: blur(1px) }
  to { width: 20px; text-indent: 20px; opacity: 0.4; color: rgb(0, 0, 100);
  transform: translateX(20px) scale(3) rotate(20deg); box-shadow: 3px 3px 3px rgb(0, 0, 200); filter: blur(3px) } }
@keyframes v { to { width: 50px; transform: rotate(90deg) } }
@keyframes z { from { width: 1px; transform: translateY(4px) } to { width: 2px; transform: translateY(8px) } }
@keyframes q { from { width: 10px; opacity: 0.1; animation-composition: add } to { width: 20px; opacity: 0.2 } }
div { width: 100px; text-indent: 5px; opacity: 0.5; color: rgb(50, 50, 50);
  transform: translateX(5px) scale(2) rotate(5deg); box-shadow: 2px 2px 2px green; filter: blur(2px) }
.r { animation: w 1s linear }
.a { animation: w 1s linear; animation-composition: add }
.c { animation: w 1s linear; animation-composition: accumulate }
.v { animation: v 1s linear; animation-composition: add }
.m { animation: w 1s linear, z 1s linear; animation-composition: replace, add }
.q { animation: q 1s linear }`
	props := map[string]func(*style.Computed) string{
		"width": everything["width"], "text-indent": moved["text-indent"],
		"opacity": compared["opacity"], "color": compared["color"],
		"transform":  func(s *style.Computed) string { return matrixCSS(s.Transform) },
		"box-shadow": everything["box-shadow"], "filter": func(s *style.Computed) string { return filtersCSS(s.Filters) },
	}
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, divs("r", "a", "c", "v", "m", "q"), props, at, 1)
		})
	}
}

// TestOwnTransitionOfAnInheritedValueMatchesChrome has a child run its
// own, slower transition of the font size its parent transitions, both
// restyled at once: it runs on to the parent's end value, whatever the
// parent shows meanwhile, where a child without one shows the parent's.
func TestOwnTransitionOfAnInheritedValueMatchesChrome(t *testing.T) {
	src := `.tp { font-size: 10px; transition: font-size 1s linear } .tp.on { font-size: 20px }
.tk { transition: font-size 2s linear } .tk.on { color: blue }`
	build := func(body *dom.Node) {
		body.Append(ui.Div(ui.ID("parent"), ui.Class("tp"), ui.Div(ui.ID("own"), ui.Class("tk")),
			ui.Div(ui.ID("inherits"), ui.Class("f"))))
	}
	props := map[string]func(*style.Computed) string{"font-size": compared["font-size"]}
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, build, props, at, 1)
		})
	}
}
