//go:build chrome

package css_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // the registered properties compared, a test table
var registeredNames = []string{"--len", "--lp", "--ang", "--t", "--col", "--int", "--num", "--pct", "--res",
	"--ids", "--lens", "--kw", "--star", "--tr", "--tf", "--str", "--bad", "--noinit", "--inh", "--ni", "--uni"}

func varsCompared(names []string) map[string]func(*style.Computed) string {
	props := map[string]func(*style.Computed) string{}
	for _, name := range names {
		props[name] = func(s *style.Computed) string { return s.Vars[name] }
	}
	return props
}

// TestRegisteredPropertiesComputeAsInChrome has @property rules of every
// syntax type register custom properties, and compares their computed
// values with Chrome's: typed, an invalid value unset, inherited only
// where the rule says so.
func TestRegisteredPropertiesComputeAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.AddSheet(registeredSheet)
	d.Body().Append(
		ui.Div(ui.Class("a"), ui.Div(ui.Class("b"), ui.Span(ui.Text("x")))),
		ui.Div(ui.Class("a", "c"), ui.Div(ui.Class("d"))),
	)
	matchChromeProps(t, d, varsCompared(registeredNames), 1)
}

// TestRegisterPropertyAsChromesRegisterProperty registers properties
// from Go, as CSS.registerProperty does, one over an @property rule of
// its name.
func TestRegisterPropertyAsChromesRegisterProperty(t *testing.T) {
	d := ui.NewDocument()
	d.AddSheet(`@property --w { syntax: "<color>"; inherits: true; initial-value: red; }
div { --w: 2em; --g: calc(1em * 2); font-size: 12px; }`)
	for _, def := range []css.PropertyDefinition{
		{Name: "--w", Syntax: "<length>", Initial: "1px"},
		{Name: "--g", Syntax: "<length> | <percentage>", Inherits: true, Initial: "50%"},
	} {
		if err := d.RegisterProperty(def); err != nil {
			t.Fatal(err)
		}
	}
	d.Body().Append(ui.Div(ui.Span(ui.Text("x"))))
	matchChromeProps(t, d, varsCompared([]string{"--w", "--g"}), 0)
}

// TestRegisteredPropertiesTransitionAsInChrome transitions registered
// properties of each type, by name: those that move interpolate, the
// rest flip halfway under allow-discrete.
func TestRegisteredPropertiesTransitionAsInChrome(t *testing.T) {
	names := []string{"--len", "--lp", "--ang", "--col", "--int", "--num", "--star", "--lens", "--kw", "--tr", "--ids",
		"--tf"}
	list := ""
	for i, n := range names {
		if i > 0 {
			list += ", "
		}
		list += n + " 10s linear"
	}
	src := registeredSheet + `.e { transition: ` + list + `; transition-behavior: allow-discrete; }
.f { transition: all 10s linear; }
.on { --len: 102px; --col: rgb(0, 0, 255); --int: 12; --ang: 180deg; --star: zzz; --lens: 11px, 12px, 13px;
  --kw: auto; --tr: translateX(100px) rotate(0deg); --lp: 10px; --num: 1; --ids: q; --tf: translate(0px, 0%); }
`
	for _, at := range []float32{2.5, 7.5} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			d := ui.NewDocument()
			d.AddSheet(src)
			d.Body().Append(ui.Div(ui.Class("a", "e")), ui.Div(ui.Class("a", "f")))
			tr := css.NewTransitions()
			tr.Compute(d.HTML(), d.Sheets())
			script := fmt.Sprintf(`els.forEach(e => getComputedStyle(e).color);
document.querySelectorAll(".a").forEach(e => e.classList.add("on"));
els.forEach(e => getComputedStyle(e).color);
document.getAnimations().forEach(a => { a.pause(); a.currentTime = %v; });`, at*1000)
			then := func() {
				d.HTML().Walk(func(n *dom.Node) bool {
					if n.HasClass("a") {
						n.AddClass("on")
					}
					return true
				})
				tr.Compute(d.HTML(), d.Sheets())
				tr.Step(at)
				tr.Compute(d.HTML(), d.Sheets())
			}
			matchChromeShown(t, d, varsCompared(names), 2, script, then,
				func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
		})
	}
}

// TestCustomPropertiesAnimateAsInChrome runs keyframes of registered
// and unregistered custom properties: the registered interpolate as
// their type, the others flip halfway, and what reads them follows.
func TestCustomPropertiesAnimateAsInChrome(t *testing.T) {
	src := registeredSheet + `
@keyframes k { from { --len: 0px; --col: red; --star: a; --u: 1px; --int: 0; --tr: rotate(0deg); }
  to { --len: 100px; --col: blue; --star: b; --u: 2px; --int: 10; --tr: rotate(90deg) } }
.g { animation: k 10s linear; width: var(--u); height: var(--len); }
.g span { --u: inherit; display: block; width: var(--u); height: 3px; }
`
	props := varsCompared([]string{"--len", "--col", "--star", "--u", "--int", "--tr"})
	props["width"] = everything["width"]
	props["height"] = everything["height"]
	for _, at := range []float32{2.5, 7.5} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			d := ui.NewDocument()
			d.AddSheet(src)
			d.Body().Append(ui.Div(ui.Class("g"), ui.Span(ui.Text("x"))))
			tr := css.NewTransitions()
			tr.Compute(d.HTML(), d.Sheets())
			script := fmt.Sprintf(`document.getAnimations().forEach(a => { a.pause(); a.currentTime = %v; });`, at*1000)
			then := func() {
				tr.Step(at)
				tr.Compute(d.HTML(), d.Sheets())
			}
			matchChromeShown(t, d, props, 2, script, then,
				func(n *dom.Node) *style.Computed { return dom.Data(n).Shown })
		})
	}
}

const registeredSheet = `
@property --len { syntax: "<length>"; inherits: false; initial-value: 0px; }
@property --lp { syntax: "<length-percentage>"; inherits: true; initial-value: 0px; }
@property --ang { syntax: "<angle>"; inherits: false; initial-value: 0deg; }
@property --t { syntax: "<time>"; inherits: false; initial-value: 0s; }
@property --col { syntax: "<color>"; inherits: false; initial-value: red; }
@property --int { syntax: "<integer>"; inherits: false; initial-value: 0; }
@property --num { syntax: "<number>"; inherits: false; initial-value: 0; }
@property --pct { syntax: "<percentage>"; inherits: false; initial-value: 0%; }
@property --res { syntax: "<resolution>"; inherits: false; initial-value: 1dppx; }
@property --ids { syntax: "<custom-ident>+"; inherits: false; initial-value: a; }
@property --lens { syntax: "<length>#"; inherits: false; initial-value: 1px; }
@property --kw { syntax: "auto | none | <length>"; inherits: false; initial-value: auto; }
@property --star { syntax: "*"; inherits: false; }
@property --tr { syntax: "<transform-list>"; inherits: false; initial-value: scale(1); }
@property --tf { syntax: "<transform-function>"; inherits: false; initial-value: scale(1); }
@property --str { syntax: "<string>"; inherits: false; initial-value: "x"; }
@property --inh { syntax: "<length>"; inherits: true; initial-value: 5px; }
@property --ni { syntax: "<length>"; inherits: false; initial-value: 5px; }
body { font-size: 10px; --inh: 7px; --ni: 9px; --bad: 3px; }
.a { font-size: 20px; color: blue;
  --len: calc(1em + 2px); --lp: calc(50% + 1em); --ang: 0.25turn; --t: 500ms; --col: currentcolor;
  --int: calc(1.5 + 1); --num: calc(1 / 3); --pct: calc(10% * 2); --res: 96dpi; --ids: foo bar;
  --lens: 1em, 2px, 3px; --kw: none; --star: a b; --tr: translateX(1em) rotate(0.5turn);
  --tf: translate(1em, 10%); --str: 'y'; }
.a { --len: red; }
.a { --kw: 2em; --pct: 3px; }
.b { --ni: inherit; --inh: initial; --len: var(--missing); --col: rgb(0 128 0 / 50%); --num: var(--int); }
.b span { width: var(--lens); }
.c { --lp: 5%; --lens: var(--len), 4px; --kw: calc(2px * 3); }
.d { --uni: var(--lp); }
`
