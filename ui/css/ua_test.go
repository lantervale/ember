package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestInsetAndOutsetBordersAndADialogsBackdrop(t *testing.T) {
	in, out := el(dom.Div, "in"), el(dom.Div, "out")
	d := el(dom.Dialog, "d")
	root := page(in, out, d)
	Compute(root, []*Sheet{sheet(t, `#in { border: 1px inset gray } #out { border-style: outset } `+
		`#d::backdrop { background-color: rgba(0, 0, 0, 0.5) }`)})
	if got := styleOf(in).BorderStyle.Top; got != style.BorderInset {
		t.Errorf("inset: %v", got)
	}
	if got := styleOf(out).BorderStyle.Left; got != style.BorderOutset {
		t.Errorf("outset: %v", got)
	}
	if got := styleOf(d).Backdrop; got != color.RGBA(0, 0, 0, 0.5) {
		t.Errorf("backdrop: %v", got)
	}
	if got := styleOf(in).Backdrop; got != (color.Color{}) {
		t.Errorf("only a dialog has a backdrop: %v", got)
	}
}

func TestAClosedDetailsShowsOnlyItsSummaryWhateverTheirDisplay(t *testing.T) {
	sum, sum2 := el(dom.Summary, "s"), el(dom.Summary, "s2")
	body := el(dom.Div, "", "body")
	d := el(dom.Details, "d")
	d.Append(sum, body, sum2)
	root := page(d)
	sheets := []*Sheet{sheet(t, `.body { display: flex }`)}
	Compute(root, sheets)
	if styleOf(sum).Display == style.DisplayNone {
		t.Error("the first summary shows")
	}
	if styleOf(body).Display != style.DisplayNone || styleOf(sum2).Display != style.DisplayNone {
		t.Errorf("closed: body %v, second summary %v", styleOf(body).Display, styleOf(sum2).Display)
	}
	d.SetAttr("open", "")
	Compute(root, sheets)
	if styleOf(body).Display != style.FlexDisplay {
		t.Errorf("open, the body takes its display: %v", styleOf(body).Display)
	}
}

func TestControlsDoNotTakeTheirParagraphsIndent(t *testing.T) {
	p, in, b, s := el(dom.P, "p"), el(dom.Input, "in"), el(dom.Button, "b"), el(dom.Span, "s")
	Compute(page(with(p, in, b, s)), []*Sheet{sheet(t, `#p { text-indent: 30px }`)})
	if got := styleOf(s).TextIndent; got != style.Px(30) {
		t.Errorf("a span inherits it: %+v", got)
	}
	for name, e := range map[string]*dom.Node{"input": in, "button": b} {
		if got, _ := styleOf(e).TextIndent.Resolve(100); got != 0 {
			t.Errorf("%s: text-indent %v, want 0 as in Chrome", name, got)
		}
	}
}

func TestTheTextElementsStartAsChromes(t *testing.T) {
	link, plain, abbr, bare := el(dom.A, "link"), el(dom.A, "plain"), el(dom.Abbr, "abbr"), el(dom.Abbr, "bare")
	link.SetAttr("href", "/vaults")
	abbr.SetAttr("title", "Hit points")
	u, del, mark := el(dom.U, "u"), el(dom.Del, "del"), el(dom.Mark, "mark")
	small, big := el(dom.Small, "small"), el(dom.Big, "big")
	li, nested, numbered := el(dom.Li, "li"), el(dom.Li, "nested"), el(dom.Li, "numbered")
	inner := el(dom.Ul, "inner")
	inner.Append(nested)
	li.Append(inner)
	ul, ol := el(dom.Ul, "ul"), el(dom.Ol, "ol")
	ul.Append(li)
	ol.Append(numbered)
	Compute(page(link, plain, abbr, bare, u, del, mark, small, big, ul, ol), nil)

	if st := styleOf(link); st.Color != color.Hex("#0000ee") || st.TextDecoration.Line != style.Underline {
		t.Errorf("a link is blue and underlined: %v %v", st.Color, st.TextDecoration.Line)
	}
	if st := styleOf(plain); st.Color != color.Hex("#000") || st.TextDecoration.Line != 0 {
		t.Errorf("an a without an href is plain text: %v %v", st.Color, st.TextDecoration.Line)
	}
	if d := styleOf(abbr).TextDecoration; d.Line != style.Underline || d.Style != style.DecorationDotted {
		t.Errorf("an abbr with a title has a dotted underline: %v", d)
	}
	if d := styleOf(bare).TextDecoration; d.Line != 0 {
		t.Errorf("an abbr without a title has no line: %v", d)
	}
	if styleOf(u).TextDecoration.Line != style.Underline || styleOf(del).TextDecoration.Line != style.LineThrough {
		t.Error("u is underlined and del struck through")
	}
	if got := styleOf(mark).Background; got != color.Hex("#ff0") {
		t.Errorf("mark is on yellow: %v", got)
	}
	if s, b := styleOf(small).Font.Size, styleOf(big).Font.Size; s >= 16 || b <= 16 {
		t.Errorf("small is %v and big %v beside 16", s, b)
	}
	for n, want := range map[*dom.Node]style.ListStyle{
		li: style.ListDisc, nested: style.ListCircle, numbered: style.ListDecimal,
	} {
		if st := styleOf(n); !st.ListItem || st.ListStyle != want {
			t.Errorf("#%s: list item %v, marker %v, want %v", n.ID(), st.ListItem, st.ListStyle, want)
		}
	}
}
