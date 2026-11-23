package css //nolint:testpackage // reads computed styles off dom

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestSubstitutionFunctionsWaitForTheElement(t *testing.T) {
	s := sheet(t, `div { width: calc(1px * sibling-index()); z-index: sibling-count(); order: calc(2.5);
  height: attr(data-h px);
  transition: opacity 1s calc(sibling-index() * 50ms); --a: if(style(--b: 1): one; else: other); --b: 1; }
@function --twice(--v <number>) returns <number> { result: calc(var(--v) * 2); }
p { --t: --twice(3); --u: --twice(x); --r: random(0, 10); --sib: sibling-index(); }`)
	a, b := el(dom.Div, ""), el(dom.Div, "")
	b.SetAttr("data-h", "7")
	p := el(dom.P, "")
	root := page(a, b, p)
	Compute(root, []*Sheet{s})
	if w, z := styleOf(b).Size.Width.Px, styleOf(b).ZIndex; w != 2 || z != 3 {
		t.Errorf("the second div: width %v, z-index %d; want 2px, 3", w, z)
	}
	if tr := styleOf(b).Transitions; len(tr) != 1 || tr[0].Delay < 0.0999 || tr[0].Delay > 0.1001 {
		t.Errorf("the second div's transitions %v, want a delay of 100ms", tr)
	}
	if o := styleOf(a).Order; o != 3 {
		t.Errorf("order: calc(2.5) = %d, want 3", o)
	}
	if h := styleOf(b).Size.Height.Px; h != 7 {
		t.Errorf("height: attr(data-h px) = %v, want 7px", h)
	}
	if v := styleOf(a).Vars["--a"]; v != "one" {
		t.Errorf("--a = %q, want one", v)
	}
	vars := styleOf(p).Vars
	if vars["--t"] != "6" {
		t.Errorf("--twice(3) = %q, want 6", vars["--t"])
	}
	if _, ok := vars["--u"]; ok {
		t.Errorf("--twice(x) = %q, want the guaranteed-invalid value", vars["--u"])
	}
	if vars["--sib"] != "sibling-index()" || !strings.HasPrefix(vars["--r"], "random(") {
		t.Errorf("a custom property keeps math functions as written: %v", vars)
	}
}

func TestRandomKeepsItsValueOnTheElement(t *testing.T) {
	s := sheet(t, `div { width: random(0px, 100px); height: random(0px, 100px); }`)
	a, b := el(dom.Div, ""), el(dom.Div, "")
	root := page(a, b)
	Compute(root, []*Sheet{s})
	w, h := styleOf(a).Size.Width.Px, styleOf(a).Size.Height.Px
	if w < 0 || w > 100 || w == h || w == styleOf(b).Size.Width.Px {
		t.Errorf("random() gives %v and %v on one div, %v on the other: want values of their own",
			w, h, styleOf(b).Size.Width.Px)
	}
	a.SetAttr("data-x", "1")
	Compute(root, []*Sheet{s})
	if styleOf(a).Size.Width.Px != w {
		t.Error("random() keeps its value as the element is styled again")
	}
}

func TestAFunctionRuleWritesBack(t *testing.T) {
	src := `@function --f(--a <length>, --b: 1px) returns <length> { result: calc(var(--a) + var(--b)); }`
	s := sheet(t, src)
	if !strings.Contains(s.String(), "@function --f(--a <length>, --b: 1px) returns <length> {") {
		t.Errorf("the sheet writes back\n%s", s.String())
	}
	if _, warns := Parse("t.css", `@function f() { result: 1; }`); len(warns) != 1 {
		t.Errorf("a function without a dashed name: warnings %v, want one", warns)
	}
}
