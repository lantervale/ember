package css //nolint:testpackage // uses the unexported test helpers (sheet, page) and reads computed styles off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestVerticalAlignIsAKeywordALengthOrAPercentage(t *testing.T) {
	type want struct {
		at    style.VerticalAlign
		shift style.Length
	}
	wants := map[string]want{
		"baseline": {at: style.AlignBaselineText}, "sub": {at: style.AlignSub}, "super": {at: style.AlignSuper},
		"text-top": {at: style.AlignTextTop}, "text-bottom": {at: style.AlignTextBottom},
		"middle": {at: style.AlignMiddle}, "top": {at: style.AlignTop}, "bottom": {at: style.AlignBottom},
		"7px":    {at: style.AlignLength, shift: style.Length{Kind: style.Fixed, Px: 7}},
		"-0.5em": {at: style.AlignLength, shift: style.Length{Kind: style.Fixed, Px: -10}},
		"50%":    {at: style.AlignLength, shift: style.Length{Kind: style.Fixed, Percent: 50, IsPercent: true}},
		"calc(25% + 2px)": {
			at: style.AlignLength, shift: style.Length{Kind: style.Fixed, Px: 2, Percent: 25, IsPercent: true},
		},
		"0":       {at: style.AlignLength, shift: style.Length{Kind: style.Fixed}},
		"inherit": {at: style.AlignMiddle},
	}
	for value, w := range wants {
		kid := el(dom.Span, "kid")
		parent := with(el(dom.P, "parent"), kid)
		Compute(page(parent), []*Sheet{sheet(t, `p { font-size: 20px; vertical-align: middle; }
span { vertical-align: `+value+`; }`)})
		got := styleOf(kid)
		if got.VerticalAlign != w.at || got.VerticalShift != w.shift {
			t.Errorf("%s: %v %+v, want %v %+v", value, got.VerticalAlign, got.VerticalShift, w.at, w.shift)
		}
	}
}

func TestVerticalAlignIsNotInheritedButACellTakesItsRows(t *testing.T) {
	span := el(dom.Span, "s")
	cell, head, lone := el(dom.Td, "cell"), el(dom.Th, "head"), el(dom.Td, "lone")
	top := with(el(dom.Tr, "top"), cell)
	group := with(el(dom.Thead, "group"), with(el(dom.Tr, "in"), head))
	table := with(el(dom.Table, "t"), with(dom.NewElement(dom.Tbody), top), group, with(el(dom.Tr, "bare"), lone))
	Compute(page(with(el(dom.P, "p"), span), table), []*Sheet{sheet(t, `
p { vertical-align: top; }
#top { vertical-align: top; }
#group { vertical-align: bottom; }
`)})
	if got := styleOf(span).VerticalAlign; got != style.AlignBaselineText {
		t.Errorf("a span under a top-aligned block: %v", got)
	}
	for n, want := range map[*dom.Node]style.VerticalAlign{
		cell: style.AlignTop, head: style.AlignBottom, lone: style.AlignMiddle,
	} {
		if got := styleOf(n).VerticalAlign; got != want {
			t.Errorf("%v: %v, want %v", n.Tag(), got, want)
		}
	}
}

func TestACellFollowsItsRowsVerticalAlignAsItChanges(t *testing.T) {
	cell := el(dom.Td, "cell")
	row := with(el(dom.Tr, "row"), cell)
	root := page(with(el(dom.Table, "t"), with(dom.NewElement(dom.Tbody), row)))
	sheets := []*Sheet{sheet(t, `.up { vertical-align: top; }`)}
	Compute(root, sheets)
	if got := styleOf(cell).VerticalAlign; got != style.AlignMiddle {
		t.Fatalf("before: %v", got)
	}
	row.SetAttr("class", "up")
	Compute(root, sheets)
	if got := styleOf(cell).VerticalAlign; got != style.AlignTop {
		t.Errorf("after: %v", got)
	}
}
