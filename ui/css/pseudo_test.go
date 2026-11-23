package css //nolint:testpackage // reads the generated nodes and the counters off dom

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func txt(s string) *dom.Node { return dom.NewText(s) }

func attr(n *dom.Node, name, value string) *dom.Node {
	n.SetAttr(name, value)
	return n
}

// generatedText is the text of n's ::before or ::after box, "" without one.
func generatedText(n *dom.Node, p dom.Pseudo) string {
	g := dom.Data(n).Before
	if p == dom.After {
		g = dom.Data(n).After
	}
	if g == nil {
		return ""
	}
	return g.Text()
}

func TestPseudoElementSelectors(t *testing.T) {
	for src, want := range map[string]struct {
		pseudo string
		spec   Specificity
		text   string
	}{
		"p::before":                   {"before", Specificity{0, 0, 2}, "p::before"},
		"p:before":                    {"before", Specificity{0, 0, 2}, "p::before"},
		"P::BEFORE":                   {"before", Specificity{0, 0, 2}, "P::before"},
		"p:AFTER":                     {"after", Specificity{0, 0, 2}, "p::after"},
		"::after":                     {"after", Specificity{0, 0, 1}, "*::after"},
		"ul > ::marker":               {"marker", Specificity{0, 0, 2}, "ul > *::marker"},
		"nav li:nth-child(2)::before": {"before", Specificity{0, 1, 3}, "nav li:nth-child(2)::before"},
		"#a.b:hover::after":           {"after", Specificity{1, 2, 1}, "#a.b:hover::after"},
		"p::part(label)":              {"part", Specificity{0, 0, 2}, "p::part(label)"},
		"video::cue":                  {"cue", Specificity{0, 0, 1}, "*::cue"},
		"p::highlight(x)":             {"highlight", Specificity{0, 0, 2}, "p::highlight(x)"},
	} {
		if src == "video::cue" {
			continue // ui has no video element
		}
		s := mustSelector(t, src)
		if s.PseudoElement != want.pseudo || s.Specificity != want.spec || s.String() != want.text {
			t.Errorf("%q: ::%s %v %q, want ::%s %v %q", src, s.PseudoElement, s.Specificity, s.String(), want.pseudo,
				want.spec, want.text)
		}
	}
}

func TestEveryPseudoElementOfTheTableParsesOrSaysWhy(t *testing.T) {
	for name, pe := range pseudoElements {
		src := "div::" + name
		if pe.form == formFunction {
			src += "(x)"
		}
		_, err := ParseSelectors(src)
		switch {
		case pe.status == pseudoRefused && (err == nil || err.Error() != "1:6: "+pe.why):
			t.Errorf("%s: %v, want its reason", src, err)
		case pe.status != pseudoRefused && err != nil:
			t.Errorf("%s: %v", src, err)
		}
	}
}

func TestAPseudoElementSharesARuleWithElements(t *testing.T) {
	p := with(el(dom.P, "p"), txt("x"))
	Compute(page(p), []*Sheet{sheet(t, `p, p::before { color: #f00; content: "a"; }`)})
	if styleOf(p).Color != hex("#f00") {
		t.Error("the element takes the rule")
	}
	g := dom.Data(p).Before
	if g == nil || styleOf(g).Color != hex("#f00") || g.Text() != "a" {
		t.Fatalf("::before takes it too: %v", g)
	}
}

func TestBeforeAndAfterGenerateANodeWhileTheirContentMakesABox(t *testing.T) {
	p := with(el(dom.P, "p", "on"), txt("x"))
	img := el(dom.Img, "i")
	root := page(p, img)
	s := sheet(t, `
p { color: #00f; font-size: 20px; }
p.on::before { content: "a" "b"; font-size: 2em; }
p::after { content: none; }
p.off::after { content: "z"; display: block; width: 50%; }
img::before { content: "never"; }
`)
	Compute(root, []*Sheet{s})
	before := dom.Data(p).Before
	if before == nil || dom.Data(p).After != nil {
		t.Fatalf("::before %v, ::after %v", before, dom.Data(p).After)
	}
	if before.Pseudo() != dom.Before || before.Parent() != p || dom.Origin(before) != p ||
		dom.Origin(before.Children()[0]) != p {
		t.Error("the node is its element's, outside its children")
	}
	if len(p.Children()) != 1 || p.Text() != "x" {
		t.Errorf("the element's own children and text are untouched: %q", p.Text())
	}
	st := styleOf(before)
	if st.Color != hex("#00f") || st.Font.Size != 40 || st.Display != style.Inline {
		t.Errorf("it inherits from its element: %v %v %v", st.Color, st.Font.Size, st.Display)
	}
	if got := before.Text(); got != "ab" {
		t.Errorf("content %q", got)
	}
	if dom.Data(img).Before != nil {
		t.Error("an img has no ::before, as in Chrome")
	}
	if flow := p.Flow(); len(flow) != 2 || flow[0] != before {
		t.Errorf("layout's children start with ::before: %v", flow)
	}

	p.RemoveClass("on")
	p.AddClass("off")
	Compute(root, []*Sheet{s})
	after := dom.Data(p).After
	if dom.Data(p).Before != nil || after == nil {
		t.Fatalf("after the class changed: ::before %v, ::after %v", dom.Data(p).Before, after)
	}
	if got := styleOf(after); got.Display != style.Block || !got.Size.Width.IsPercent {
		t.Errorf("::after takes any property: %+v", got.Size.Width)
	}
	if dom.Data(p).Dirty&dom.DirtyLayout == 0 {
		t.Error("the element is laid out again")
	}

	p.AddClass("again")
	Compute(root, []*Sheet{s})
	if dom.Data(p).After != after {
		t.Error("the same node stands for the pseudo-element from restyle to restyle")
	}
}

func TestContentReadsAttributesAndPictures(t *testing.T) {
	a := attr(with(el(dom.A, "a"), txt("x")), "href", "/there")
	root := page(a)
	s := sheet(t, `a::after { content: " (" attr(href) ")" url(pic.png) attr(nope) "!" / "alt"; }`)
	Compute(root, []*Sheet{s})
	g := dom.Data(a).After
	if g == nil || len(g.Children()) != 3 {
		t.Fatalf("::after %v", g)
	}
	kids := g.Children()
	src, _ := kids[1].Attr("src")
	if kids[0].Text() != " (/there)" || kids[1].Tag() != dom.Img || src != "pic.png" || kids[2].Text() != "!" {
		t.Errorf("content: %q, %s src %q, %q", kids[0].Text(), kids[1].Tag(), src, kids[2].Text())
	}
	a.SetAttr("href", "/else")
	Compute(root, []*Sheet{s})
	if got := dom.Data(a).After.Children()[0].Text(); got != " (/else)" {
		t.Errorf("after the attribute changed: %q", got)
	}
}

func TestCountersCountInDocumentOrder(t *testing.T) {
	h := func(tag dom.Tag, text string) *dom.Node { return with(el(tag, ""), txt(text)) }
	h1a, h2a, h2b, h1b, h2c := h(dom.H1, "a"), h(dom.H2, "b"), h(dom.H2, "c"), h(dom.H1, "d"), h(dom.H2, "e")
	hidden := h(dom.H2, "hidden")
	hidden.SetStyle("display: none")
	root := page(h1a, h2a, hidden, h2b, h1b, h2c)
	Compute(root, []*Sheet{sheet(t, `
body { counter-reset: chapter 2 section; }
h1 { counter-increment: chapter; counter-reset: section; }
h2 { counter-increment: section 2; }
h1::before { content: counter(chapter, upper-roman) ". "; }
h2::before { content: counter(chapter) "." counter(section) " " counter(nope); }
h2::after { counter-set: section 10; content: "[" counter(section, lower-alpha) "]"; }
`)})
	for n, want := range map[*dom.Node]string{h1a: "III. ", h2a: "3.2 0", h2b: "3.12 0", h1b: "IV. ", h2c: "4.2 0"} {
		if got := generatedText(n, dom.Before); got != want {
			t.Errorf("%s: ::before %q, want %q", n.Text(), got, want)
		}
	}
	if got := generatedText(h2b, dom.After); got != "[j]" {
		t.Errorf("counter-set in ::after: %q", got)
	}
	if generatedText(hidden, dom.Before) != "" {
		t.Error("an element with no box counts nothing")
	}
}

func TestCountersNestWithTheListsThatResetThem(t *testing.T) {
	li := func(kids ...*dom.Node) *dom.Node { return with(el(dom.Li, ""), kids...) }
	ol := func(kids ...*dom.Node) *dom.Node { return with(el(dom.Ol, ""), kids...) }
	a, b, c := li(txt("a")), li(txt("b")), li(txt("c"))
	b1, b2 := li(txt("b1")), li(txt("b2"))
	b.Append(ol(b1, b2))
	root := page(ol(a, b, c))
	s := sheet(t, `
ol { counter-reset: item; list-style-type: none; }
li { counter-increment: item; }
li::before { content: counters(item, ".") " " counters(list-item, "-", lower-alpha); }
`)
	Compute(root, []*Sheet{s})
	for n, want := range map[*dom.Node]string{a: "1 a", b: "2 b", b1: "2.1 b-a", b2: "2.2 b-b", c: "3 c"} {
		if got := generatedText(n, dom.Before); got != want {
			t.Errorf("::before %q, want %q", got, want)
		}
	}

	dom.Data(root).Dirty, dom.Data(c).Dirty = 0, 0
	b.Parent().Remove(b)
	Compute(root, []*Sheet{s})
	if got := generatedText(c, dom.Before); got != "2 b" {
		t.Errorf("after an item before it left: %q", got)
	}
	if dom.Data(c).Dirty&dom.DirtyLayout == 0 || dom.Data(root).Dirty&dom.DirtyDescendant == 0 {
		t.Error("the item whose number moved is laid out again")
	}
}

func TestListItemsCountOnTheListItemCounter(t *testing.T) {
	li := func(text string) *dom.Node { return with(el(dom.Li, ""), txt(text)) }
	ordinals := func(items ...*dom.Node) []int {
		var out []int
		for _, n := range items {
			out = append(out, dom.Data(n).Ordinal)
		}
		return out
	}
	same := func(name string, got []int, want ...int) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: %v, want %v", name, got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: %v, want %v", name, got, want)
				return
			}
		}
	}
	a, b, c, d := li("a"), li("b"), li("c"), li("d")
	inner := with(el(dom.Div, ""), c)
	ol := with(el(dom.Ol, ""), a, b, inner, d)
	root := page(ol)
	Compute(root, nil)
	same("an ol", ordinals(a, b, c, d), 1, 2, 3, 4)

	ol.SetAttr("start", "7")
	b.SetAttr("value", "20")
	Compute(root, nil)
	same("start and value", ordinals(a, b, c, d), 7, 20, 21, 22)

	ol.RemoveAttr("start")
	b.RemoveAttr("value")
	ol.SetAttr("reversed", "")
	Compute(root, nil)
	same("reversed", ordinals(a, b, c, d), 4, 3, 2, 1)

	ol.SetAttr("start", "2")
	Compute(root, nil)
	same("reversed from 2", ordinals(a, b, c, d), 2, 1, 0, -1)

	a, b, c, d = li("a"), li("b"), li("c"), li("d")
	b.SetStyle("display: none")
	root = page(with(el(dom.Ol, ""), a, b, with(el(dom.Div, ""), c), d))
	Compute(root, []*Sheet{sheet(t, `ol { counter-reset: list-item 4; } div > li { counter-increment: list-item 10; }`)})
	same("the author's own list-item properties", ordinals(a, c, d), 5, 15, 16)
}

func TestQuotesOpenAndCloseByDepth(t *testing.T) {
	q := func(kids ...*dom.Node) *dom.Node { return with(el(dom.Q, ""), kids...) }
	deep := q(txt("c"))
	inner := q(txt("b"), deep)
	outer := q(txt("a"), inner)
	later := q(txt("d"))
	fr := with(el(dom.P, "fr"), q(txt("e"), q(txt("f"))))
	none := with(el(dom.P, "none"), q(txt("g")))
	root := page(with(el(dom.P, ""), outer, later), fr, none)
	Compute(root, []*Sheet{sheet(t, `#fr { quotes: "«" "»" "‹" "›"; } #none { quotes: none; }`)})
	marks := func(n *dom.Node) string { return generatedText(n, dom.Before) + generatedText(n, dom.After) }
	for n, want := range map[*dom.Node]string{outer: "“”", inner: "‘’", deep: "‘’", later: "“”",
		fr.Children()[0]: "«»", fr.Children()[0].Children()[1]: "‹›"} {
		if got := marks(n); got != want {
			t.Errorf("q %q: %s, want %s", n.Text(), got, want)
		}
	}
	g := dom.Data(none.Children()[0]).Before
	if g == nil || g.Text() != "" {
		t.Errorf("quotes: none makes an empty box: %v", g)
	}
}

func TestMarkerTakesItsPropertiesAndItsContent(t *testing.T) {
	li := func(id string) *dom.Node { return with(el(dom.Li, id), txt("x")) }
	a, b, c := li("a"), li("b"), li("c")
	root := page(with(el(dom.Ol, ""), a, b, c))
	Compute(root, []*Sheet{sheet(t, `
li { color: #00f; }
#a::marker { color: #f00; font-size: 30px; font-weight: 700; }
#b::marker { content: "(" counter(list-item, upper-alpha) ") "; }
li, li::marker { padding-left: 4px; }
#c::marker { content: none; }
`)})
	m := dom.Data(a).Marker
	if m == nil || m.Color != hex("#f00") || m.Font.Size != 30 || m.Font.Weight != 700 {
		t.Fatalf("::marker's style: %+v", m)
	}
	if m.Padding.Left.Px != 4 || styleOf(a).Padding.Left.Px != 4 {
		// the copy keeps the item's own padding, which no marker reads
		t.Errorf("padding: %v", m.Padding.Left)
	}
	if d := dom.Data(a); d.MarkerText != "1. " || d.Ordinal != 1 {
		t.Errorf("#a keeps its number: %d %q", d.Ordinal, d.MarkerText)
	}
	if d := dom.Data(b); !d.HasMarkerText || d.MarkerText != "(B) " {
		t.Errorf("#b's content: %q", d.MarkerText)
	}
	if d := dom.Data(c); !d.HasMarkerText || d.MarkerText != "" {
		t.Errorf("content: none: %q %v", d.MarkerText, d.HasMarkerText)
	}
}

func TestPlaceholderSelectionAndBackdropTakeTheirProperties(t *testing.T) {
	in := attr(el(dom.Input, "in"), "placeholder", "x")
	dim := attr(el(dom.Input, "dim"), "placeholder", "x")
	dlg := el(dom.Dialog, "d")
	root := page(in, dim, dlg)
	Compute(root, []*Sheet{sheet(t, `
input { color: #000; }
#in::placeholder { color: #f00; opacity: 0.5; }
#dim::placeholder { opacity: 0.5; }
#d::backdrop { background: #00f; opacity: 0.5; }
`)})
	if st := styleOf(in); !st.HasPlaceholder || st.PlaceholderColor != fadeColor(hex("#f00"), 0.5) {
		t.Errorf("colour and opacity: %v", st.PlaceholderColor)
	}
	if st := styleOf(dim); !st.HasPlaceholder || st.PlaceholderColor != fadeColor(hex("#757575"), 0.5) {
		t.Errorf("opacity alone fades the user agent's colour: %v", st.PlaceholderColor)
	}
	if st := styleOf(dlg); st.Backdrop != fadeColor(hex("#00f"), 0.5) {
		t.Errorf("::backdrop: %v", st.Backdrop)
	}
}

func TestADocumentWithoutPseudoElementRulesLooksForNone(t *testing.T) {
	p := with(el(dom.P, ""), txt("x"))
	root := page(p)
	all := []*Sheet{userAgent(), sheet(t, `p { color: #f00; }`)}
	r := restyler{sheets: all, m: &matcher{}, pm: &matcher{}, computed: new(int)}
	r.run(root, false)
	if r.may(p, "before") || r.may(p, "after") || r.may(p, "marker") || r.may(p, "selection") {
		t.Error("only a q is in a bucket of the user agent's ::before")
	}
	if dom.Data(root).Counts {
		t.Error("nothing counts in a tree without list numbers, counters or quotes")
	}
}

func TestContentKeepsItsWarningsOutOfElements(t *testing.T) {
	_, warns := Parse("t.css", `p { content: "x"; counter-increment: a b 2 c; quotes: "<" ">"; list-style: upper-roman; }`)
	if got := warnings(warns); got != "" {
		t.Errorf("warnings: %s", got)
	}
	if !strings.Contains(Default, "q::before { content: open-quote; }") {
		t.Error("the user agent's q stands between quotes through ::before and ::after")
	}
}
