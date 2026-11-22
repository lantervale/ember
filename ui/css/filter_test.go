package css //nolint:testpackage // reads computed and shown styles off dom with the package's test helpers

import (
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestFilterFunctionsAreReadInOrder(t *testing.T) {
	s := gridOf(t, "color: #0f0; filter: blur(2px) brightness(150%) contrast(0.5) grayscale() hue-rotate(0.5turn) "+
		"invert(2) opacity(40%) saturate(3) sepia(1) drop-shadow(#f00 2px 3px 4px) drop-shadow(1px -1px)")
	want := []style.Filter{
		{Kind: style.Blur, Amount: 2},
		{Kind: style.Brightness, Amount: 1.5},
		{Kind: style.Contrast, Amount: 0.5},
		{Kind: style.Grayscale, Amount: 1},
		{Kind: style.HueRotate, Amount: 180},
		{Kind: style.Invert, Amount: 1},
		{Kind: style.OpacityFilter, Amount: 0.4},
		{Kind: style.Saturate, Amount: 3},
		{Kind: style.Sepia, Amount: 1},
		{Kind: style.DropShadow, Shadow: style.Shadow{X: 2, Y: 3, Blur: 4, Color: hex("#f00")}},
		{Kind: style.DropShadow, Shadow: style.Shadow{X: 1, Y: -1, Color: hex("#0f0"), CurrentColor: true}},
	}
	if len(s.Filters) != len(want) {
		t.Fatalf("filters %+v", s.Filters)
	}
	for i := range want {
		if got := s.Filters[i]; got.Kind != want[i].Kind || !nearf(got.Amount, want[i].Amount) ||
			got.Shadow != want[i].Shadow {
			t.Errorf("filter %d: %+v, want %+v", i, got, want[i])
		}
	}
	if none := gridOf(t, "filter: blur(1px); filter: none"); none.Filters != nil {
		t.Errorf("none: %+v", none.Filters)
	}
	if bare := gridOf(t, "filter: blur() hue-rotate() brightness()"); !slices.Equal(bare.Filters, []style.Filter{
		{Kind: style.Blur}, {Kind: style.HueRotate}, {Kind: style.Brightness, Amount: 1},
	}) {
		t.Errorf("without arguments, each its default: %+v", bare.Filters)
	}
	math := gridOf(t, "font-size: 10px; filter: blur(calc(1em + 2px)) brightness(clamp(0.5, 3, 1.5)) "+
		"hue-rotate(calc(45deg * 2)) grayscale(min(30%, 50%)) drop-shadow(calc(1em / 2) max(1px, 3px))")
	if f := math.Filters; len(f) != 5 || !nearf(f[0].Amount, 12) || !nearf(f[1].Amount, 1.5) ||
		!nearf(f[2].Amount, 90) || !nearf(f[3].Amount, 0.3) || !nearf(f[4].Shadow.X, 5) || !nearf(f[4].Shadow.Y, 3) {
		t.Errorf("math functions: %+v", f)
	}
}

func TestADropShadowInCurrentColorFollowsTheElementsColour(t *testing.T) {
	p, k := el(dom.Div, "p"), el(dom.Span, "k")
	p.Append(k)
	Compute(page(p), []*Sheet{sheet(t,
		"#p { color: #f00; filter: drop-shadow(0 1px) } #k { color: #00f; filter: inherit }")})
	if f := styleOf(p).Filters; len(f) != 1 || f[0].Shadow.Color != hex("#f00") {
		t.Errorf("p: %+v", f)
	}
	if f := styleOf(k).Filters; len(f) != 1 || f[0].Shadow.Color != hex("#00f") {
		t.Errorf("inherited as currentColor: %+v", f)
	}
}

func TestWhatFilterDoesNotReadIsRefused(t *testing.T) {
	for _, decl := range []string{
		"filter: url(#f)",
		"filter: blur(1px) url(f.svg#a)",
		"filter: blur(-1px)",
		"filter: blur(10%)",
		"filter: brightness(-1)",
		"filter: hue-rotate(10px)",
		"filter: drop-shadow(1px)",
		"filter: drop-shadow(1px 2px 3px 4px)",
		"filter: drop-shadow(1px 2px -3px)",
		"filter: drop-shadow(1px red 2px)",
		"filter: glow(1)",
		"filter: blur(1px) none",
		"filter: saturate(1, 2)",
		"filter: brightness(2px)",
		"filter: blur(calc(1px + 1))",
	} {
		_, warns := Parse("t.css", "div { "+decl+" }")
		if len(warns) != 1 {
			t.Errorf("%s: warnings %v, want one", decl, warns)
		}
	}
	_, warns := Parse("t.css", "div { filter: url(#f) }")
	if len(warns) != 1 || !strings.Contains(warns[0].Error(), "url() filters are not read") {
		t.Errorf("an SVG filter: %v", warns)
	}
	if _, warns = Parse("t.css", "div { transition: filter 1s }"); len(warns) != 0 {
		t.Errorf("filter transitions: %v", warns)
	}
}

func filterPage(t *testing.T, css string) (*Transitions, *dom.Node, *dom.Node, []*Sheet) {
	t.Helper()
	tr, root, b, sheets := transitionPage(t, "button { transition: filter 1s linear } "+css)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	return tr, root, b, sheets
}

func TestFilterTransitionsRunFunctionByFunction(t *testing.T) {
	_, _, b, _ := filterPage(t, `button { filter: blur(2px) drop-shadow(0 0 0 #000) }
.on { filter: blur(6px) drop-shadow(4px 2px 8px rgba(0, 0, 0, 0)) }`)
	f := shown(b).Filters
	if len(f) != 2 || !nearf(f[0].Amount, 4) || !nearf(f[1].Shadow.X, 2) || !nearf(f[1].Shadow.Blur, 4) ||
		!nearf(f[1].Shadow.Color.A, 0.5) {
		t.Errorf("same functions: %+v", f)
	}

	_, _, b, _ = filterPage(t, `.on { filter: brightness(2) grayscale(1) }`)
	f = shown(b).Filters
	if len(f) != 2 || !nearf(f[0].Amount, 1.5) || !nearf(f[1].Amount, 0.5) {
		t.Errorf("none to a list, from each function's neutral value: %+v", f)
	}

	_, _, b, _ = filterPage(t, `button { filter: opacity(0) } .on { filter: opacity(1) hue-rotate(90deg) }`)
	f = shown(b).Filters
	if len(f) != 2 || !nearf(f[0].Amount, 0.5) || !nearf(f[1].Amount, 45) {
		t.Errorf("a list that starts with the other: %+v", f)
	}

	tr, _, b, _ := filterPage(t, `button { filter: blur(2px) } .on { filter: sepia(1) }`)
	if f = shown(b).Filters; len(f) != 1 || f[0].Kind != style.Sepia {
		t.Errorf("other functions switch halfway: %+v", f)
	}
	tr.Step(0.1)
	if f = shown(b).Filters; len(f) != 1 || f[0].Kind != style.Sepia {
		t.Errorf("and stay: %+v", f)
	}
}

func TestKeyframesAnimateFilter(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes f { from { filter: blur(0) } to { filter: blur(10px) } }
div { animation: f 1s linear }`)
	tr.Step(0.25)
	if f := shown(d).Filters; len(f) != 1 || !nearf(f[0].Amount, 2.5) {
		t.Errorf("a quarter in: %+v", f)
	}
}
