package css //nolint:testpackage // reads registrations and computed styles off dom

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAnInvalidPropertyRuleIsDropped(t *testing.T) {
	for src, why := range map[string]string{
		`@property --a { syntax: "<length>"; inherits: false; }`:                               "needs an initial value",
		`@property --a { syntax: "<length>"; initial-value: 1px; }`:                            "needs a syntax and inherits",
		`@property --a { inherits: true; initial-value: 1px; }`:                                "needs a syntax and inherits",
		`@property --a { syntax: "<length>"; inherits: false; initial-value: 1em; }`:           "depends on the element",
		`@property --a { syntax: "<length>"; inherits: false; initial-value: red; }`:           "is no <length>",
		`@property --a { syntax: "<lenth>"; inherits: false; initial-value: 1px; }`:            "is no syntax type",
		`@property --a { syntax: "<transform-list>+"; inherits: false; initial-value: none; }`: "no multiplier",
		`@property a { syntax: "*"; inherits: false; }`:                                        "no custom property name",
		`@property --a { syntax: "inherit"; inherits: false; initial-value: inherit; }`:        "no syntax component",
	} {
		s, warns := Parse("t.css", src)
		if len(s.properties) != 0 || len(warns) != 1 || !strings.Contains(warns[0].Msg, why) {
			t.Errorf("%s: %d rules kept, warnings %v; want one saying %q", src, len(s.properties), warns, why)
		}
	}
}

func TestTheLastPropertyRuleOfANameWins(t *testing.T) {
	s := sheet(t, `@property --a { syntax: "<length>"; inherits: false; initial-value: 1px; }
@property --a { syntax: "<number>"; inherits: true; initial-value: 2; }`)
	g := newRegistry([]*Sheet{s}, nil)
	if r := g.get("--a"); r == nil || r.initial != "2" || !r.inherits {
		t.Fatalf("--a registered as %+v, want the second rule", r)
	}
	if !strings.Contains(s.String(), `@property --a { syntax: "<number>"; inherits: true; initial-value: 2; }`) {
		t.Errorf("the sheet writes back\n%s", s.String())
	}
}

func TestARegisteredPropertyComputesAsItsType(t *testing.T) {
	s := sheet(t, `@property --w { syntax: "<length> | auto"; inherits: false; initial-value: auto; }
@property --c { syntax: "<color>+"; inherits: true; initial-value: red; }
body { font-size: 10px; --w: calc(2em + 1px); --c: blue lime; }
div { font-size: 20px; --w: 1.5em; width: var(--w); }
span { --w: nonsense; }`)
	d := el(dom.Div, "")
	sp := el(dom.Span, "")
	root := page(d, sp)
	Compute(root, []*Sheet{s})
	body := root.Children()[0]
	for n, want := range map[*dom.Node]map[string]string{
		body: {"--w": "21px", "--c": "rgb(0, 0, 255) rgb(0, 255, 0)"},
		d:    {"--w": "30px", "--c": "rgb(0, 0, 255) rgb(0, 255, 0)"},
		sp:   {"--w": "auto"},
	} {
		for name, v := range want {
			if got := styleOf(n).Vars[name]; got != v {
				t.Errorf("%s %s = %q, want %q", n.Tag(), name, got, v)
			}
		}
	}
	if w := styleOf(d).Size.Width; w.Px != 30 {
		t.Errorf("width: var(--w) = %v, want 30px", w)
	}
}

func TestRegisteredValuesInterpolate(t *testing.T) {
	for _, c := range []struct {
		syntax, a, b, want string
	}{
		{"<length>", "0px", "100px", "25px"},
		{"<length-percentage>", "calc(50% + 20px)", "10px", "calc(37.5% + 17.5px)"},
		{"<integer>", "0", "10", "3"},
		{"<color>", "rgb(255, 0, 0)", "rgb(0, 0, 255)", "rgb(191, 0, 64)"},
		{"<length>#", "1px, 2px", "5px, 6px", "2px, 3px"},
		{"<length>#", "1px, 2px", "5px", "1px, 2px"},
		{"<transform-list>", "translateX(20px) rotate(180deg)", "translateX(100px) rotate(0deg)",
			"translateX(40px) rotate(135deg)"},
		{"<custom-ident>", "a", "b", "a"},
		{"auto | <length>", "auto", "4px", "auto"},
	} {
		r, err := register(PropertyDefinition{Name: "--x", Syntax: c.syntax, Initial: c.a})
		if err != nil {
			t.Fatal(err)
		}
		if got := r.interpolate(c.a, c.b, 0.25); got != c.want {
			t.Errorf("%s from %s to %s at 25%% = %q, want %q", c.syntax, c.a, c.b, got, c.want)
		}
	}
}

func TestRegisterPropertyRefusesTwiceAndInvalid(t *testing.T) {
	tr := NewTransitions()
	def := PropertyDefinition{Name: "--a", Syntax: "<number>", Initial: "1"}
	if err := tr.RegisterProperty(def); err != nil {
		t.Fatal(err)
	}
	if tr.RegisterProperty(def) == nil {
		t.Error("a second registration of --a is an error")
	}
	if tr.RegisterProperty(PropertyDefinition{Name: "--b", Syntax: "<number>", Initial: "x"}) == nil {
		t.Error("an initial value the syntax refuses is an error")
	}
}
