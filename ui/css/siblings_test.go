package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestSiblingsAndTheOtherSelectors(t *testing.T) {
	h := el(dom.H2, "h")
	p1, p2 := el(dom.P, "p1", "lead"), el(dom.P, "p2")
	b := el(dom.Button, "b")
	off := el(dom.Button, "off")
	off.SetState(dom.Disabled, true)
	link := el(dom.A, "link", "vector-icon", "mw-ui-icon-menu-progressive")
	link.SetAttr("href", "https://Example.org/a.PDF")
	link.SetAttr("lang", "fr-CA")
	link.SetAttr("rel", "nofollow noopener")
	span := el(dom.Span, "s")
	div := with(el(dom.Div, "d"), h, dom.NewText("text"), p1, p2, with(el(dom.Div, "in"), span), b, off, link)
	page(div)

	cases := []struct {
		src  string
		e    *dom.Node
		want bool
	}{
		{"h2 + p", p1, true},
		{"h2 + p", p2, false},
		{"h2 ~ p", p2, true},
		{"p ~ h2", h, false},
		{"p + p", p2, true},
		{"h2 + p + p", p2, true},
		{"h2 + .lead ~ div span", span, true},
		{"body h2 ~ button", b, true},
		{"#p2 + #in > span", span, true},
		{"#p1 + #in > span", span, false},
		{"h2 + h2", h, false},
		{":is(h2, p)", p1, true},
		{":is(h2, p)", b, false},
		{"div :where(.lead, #b)", b, true},
		{":is(blink, :nope, p)", p2, true},
		{":is(blink)", p2, false},
		{"p:not(:is(.lead, #x))", p2, true},
		{":is(h2 + p)", p1, true},
		{"button:enabled", b, true},
		{"button:enabled", off, false},
		{":enabled", p1, false},
		{"p:first-of-type", p1, true},
		{"p:first-of-type", p2, false},
		{"p:last-of-type", p2, true},
		{"h2:only-of-type", h, true},
		{"p:only-of-type", p1, false},
		{"span:only-child", span, true},
		{"p:only-child", p1, false},
		{"p:nth-of-type(2)", p2, true},
		{"p:nth-last-of-type(2)", p1, true},
		{":nth-last-child(1)", link, true},
		{":nth-last-child(2n)", off, true},
		{`[href^="https://"]`, link, true},
		{`[href^="http://"]`, link, false},
		{`[href$=".pdf"]`, link, false},
		{`[href$=".pdf" i]`, link, true},
		{`[href*="example" i]`, link, true},
		{`[href*=""]`, link, false},
		{`[class*='-progressive']`, link, true},
		{`[rel~="noopener"]`, link, true},
		{`[rel~="open"]`, link, false},
		{`[rel~="nofollow noopener"]`, link, false},
		{`[lang|="fr"]`, link, true},
		{`[lang|="f"]`, link, false},
		{`[LANG="FR-ca" i]`, link, true},
	}
	for _, c := range cases {
		if got := mustSelector(t, c.src).Matches(c.e); got != c.want {
			t.Errorf("%q on #%s: got %v", c.src, c.e.ID(), got)
		}
	}
}

// What a sibling combinator reads on an element restyles the elements
// after it, and theirs below; nothing else does.
func TestAChangeBesideRestylesTheElementsAfter(t *testing.T) {
	red := hex("#f00")
	build := func() (*dom.Node, *dom.Node, *dom.Node, *dom.Node, *dom.Node, *dom.Node) {
		before, first := el(dom.P, "before"), el(dom.Input, "first")
		a, b, deep := el(dom.P, "a"), el(dom.P, "b"), el(dom.Span, "deep")
		return page(with(el(dom.Div, ""), before, first, a, with(b, deep))), first, a, b, deep, before
	}
	sheets := []*Sheet{sheet(t, `p, span { color: #000 }
.on + p { color: #f00 }
input:checked ~ p span { color: #f00 }
input:not(:enabled) + p { color: #f00 }
[data-x] ~ :is(#b) { color: #f00 }`)}
	colors := func(nodes ...*dom.Node) []bool {
		var out []bool
		for _, n := range nodes {
			out = append(out, styleOf(n).Color == red)
		}
		return out
	}
	steps := []struct {
		name string
		do   func(first, a *dom.Node)
		want [4]bool
	}{
		{"a class", func(first, _ *dom.Node) { first.AddClass("on") }, [4]bool{true, false, false, false}},
		{"a state", func(first, _ *dom.Node) { first.SetState(dom.Checked, true) }, [4]bool{false, false, true, false}},
		{"disabled", func(first, _ *dom.Node) { first.SetState(dom.Disabled, true) }, [4]bool{true, false, false, false}},
		{"an attribute", func(first, _ *dom.Node) { first.SetAttr("data-x", "") }, [4]bool{false, true, false, false}},
		{"a child gone", func(first, a *dom.Node) {
			first.AddClass("on")
			a.Parent().Remove(a)
		}, [4]bool{false, true, false, false}},
	}
	for _, step := range steps {
		root, first, a, b, deep, before := build()
		Compute(root, sheets)
		if got := colors(a, b, deep, before); got[0] || got[1] || got[2] || got[3] {
			t.Fatalf("%s: red before it: %v", step.name, got)
		}
		step.do(first, a)
		Compute(root, sheets)
		got := colors(a, b, deep, before)
		if a.Parent() == nil {
			got[0] = false
		}
		if [4]bool(got) != step.want {
			t.Errorf("%s: a, b, b's span, the one before: %v, want %v", step.name, got, step.want)
		}
	}
}

func TestOnlyWhatIsReadBesideRestylesTheSiblings(t *testing.T) {
	first, a := el(dom.P, "first"), el(dom.P, "a")
	root := page(with(el(dom.Div, ""), first, a))
	tr := &Transitions{}
	sheets := []*Sheet{sheet(t, ".on + p { color: #f00 } .other { color: #00f }")}
	tr.Compute(root, sheets)
	was := tr.computed
	first.AddClass("other")
	if tr.Compute(root, sheets); tr.computed-was != 1 {
		t.Errorf("a class no sibling combinator reads restyled %d elements, want 1", tr.computed-was)
	}
	was = tr.computed
	first.AddClass("on")
	if tr.Compute(root, sheets); tr.computed-was != 2 {
		t.Errorf("a class read before + restyled %d elements, want 2", tr.computed-was)
	}
}
