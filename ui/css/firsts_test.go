package css //nolint:testpackage // reads the first letters' nodes off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAFirstLetterTakesThePunctuationRoundIt(t *testing.T) {
	for s, want := range map[string]string{
		"Hello":           "H",
		"  \n Hello":      "  \n H",
		`"Well," he said`: `"W`,
		"(a) first":       "(a)",
		"«Ça» va":         "«Ç",
		"7 lanterns":      "7",
		"éte":            "é",
		"... ":            "",
		"( x":             "",
		"   ":             "",
	} {
		if got := s[:letterEnd(s)]; got != want {
			t.Errorf("%q: %q, want %q", s, got, want)
		}
	}
}

func TestAFirstLetterIsFoundWhereChromeLooksForIt(t *testing.T) {
	letterOf := func(n *dom.Node) string {
		g := dom.Data(n).FirstLetter
		if g == nil || dom.Data(g).Letter == nil {
			return ""
		}
		return g.Text() + " in " + dom.Data(g).Letter.Text()
	}
	inline := with(el(dom.P, "", "a"), with(el(dom.B, ""), txt("Bold")), txt(" rest"))
	before := with(el(dom.P, "", "a", "told"), txt("upon"))
	nested := with(el(dom.Div, "", "a"), txt("  "), with(el(dom.P, ""), txt("inner")))
	floated := with(el(dom.P, "", "a"), with(el(dom.Span, "", "float"), txt("Aside")), txt("main"))
	picture := with(el(dom.P, "", "a"), el(dom.Img, ""), txt("after"))
	root := page(inline, before, nested, floated, picture)
	s := sheet(t, `.a::first-letter { color: #f00; } .told::before { content: "Once "; } .float { float: left; }`)
	Compute(root, []*Sheet{s})
	for n, want := range map[*dom.Node]string{inline: "B in Bold", before: "O in Once ", nested: "i in inner",
		floated: "m in main", picture: ""} {
		if got := letterOf(n); got != want {
			t.Errorf("%s: %q, want %q", describe(n), got, want)
		}
	}
	text := inline.Children()[0].Children()[0]
	if dom.Data(text).Letter == nil || dom.Data(text).LetterEnd != 1 {
		t.Fatalf("the text knows its letter is taken: %v %d", dom.Data(text).Letter, dom.Data(text).LetterEnd)
	}
	Compute(root, []*Sheet{sheet(t, `p { color: #00f; }`)})
	if dom.Data(inline).FirstLetter != nil || dom.Data(text).Letter != nil {
		t.Error("with no rule left, the first letter goes and gives its text back")
	}
}

func TestAFirstLineStylesTheInlineElementsOnIt(t *testing.T) {
	span := with(el(dom.Span, "", "half"), txt("small"))
	p := with(el(dom.P, ""), txt("Lead "), span)
	inner := with(el(dom.P, ""), txt("inner"))
	div := with(el(dom.Div, "", "outer"), inner)
	root := page(p, div)
	Compute(root, []*Sheet{sheet(t, `
p::first-line { font-size: 20px; color: #f00; }
.half { font-size: 0.5em; }
.outer::first-line { font-size: 30px; background-color: #0f0; }`)})
	if fl := dom.Data(p).FirstLine; fl == nil || fl.Font.Size != 20 || fl.Color != hex("#f00") {
		t.Fatalf("p's first line: %v", fl)
	}
	if s := dom.Data(span).FirstLine; s == nil || s.Font.Size != 10 || s.Color != hex("#f00") {
		t.Errorf("the span inherits from the first line there: %v", s)
	}
	if styleOf(span).Font.Size != 8 {
		t.Errorf("and keeps its own style elsewhere: %v", styleOf(span).Font.Size)
	}
	if s := dom.Data(inner).FirstLine; s == nil || s.Font.Size != 20 {
		t.Errorf("a first block takes its parent's first line, under its own: %v", s)
	}
	if dom.Data(div).LineBack == nil {
		t.Error("a first line's background has a box of its own")
	}
}
