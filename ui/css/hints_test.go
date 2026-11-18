package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func attrs(n *dom.Node, pairs ...string) *dom.Node {
	for i := 0; i < len(pairs); i += 2 {
		n.SetAttr(pairs[i], pairs[i+1])
	}
	return n
}

func TestATablesAttributesStyleItAndItsCells(t *testing.T) {
	wide := attrs(el(dom.Td, "wide"),
		"width", "25%", "height", "40", "nowrap", "", "align", "center", "bgcolor", "#ff0000")
	plain := attrs(el(dom.Td, "plain"), "width", "0", "align", "right", "valign", "bottom", "bgcolor", "fff")
	head := attrs(el(dom.Th, "head"), "align", "left", "bgcolor", "red; width: 9px")
	row := attrs(with(el(dom.Tr, "row"), wide, plain, head), "valign", "top", "align", "justify", "bgcolor", "blue")
	table := attrs(with(el(dom.Table, "t"), with(dom.NewElement(dom.Tbody), row)),
		"width", "300", "cellpadding", "6", "cellspacing", "0", "bgcolor", "#00ff00", "align", "center")
	Compute(page(table), nil)

	ts := styleOf(table)
	if ts.Size.Width != style.Px(300) || ts.BorderSpacing != [2]float32{} || ts.Background != hex("#0f0") {
		t.Errorf("table: %+v %v %v", ts.Size, ts.BorderSpacing, ts.Background)
	}
	if ts.Margin.Left.Kind != style.Auto || ts.Margin.Right.Kind != style.Auto {
		t.Errorf("a table aligned center has auto margins: %+v", ts.Margin)
	}
	if rs := styleOf(row); rs.Background != hex("#00f") || rs.TextAlign != style.Justify {
		t.Errorf("row: %v %v", rs.Background, rs.TextAlign)
	}
	ws := styleOf(wide)
	if ws.Size.Width != pct(25) || ws.Size.Height != style.Px(40) || ws.WhiteSpace != style.NoWrapSpace {
		t.Errorf("wide: %+v %v", ws.Size, ws.WhiteSpace)
	}
	if ws.TextAlign != style.BlockCenter || ws.Background != hex("#f00") || ws.Padding.Left != style.Px(6) {
		t.Errorf("wide: %v %v %+v", ws.TextAlign, ws.Background, ws.Padding)
	}
	if ws.AlignContent != style.AlignStart {
		t.Errorf("a cell takes its row's valign: %v", ws.AlignContent)
	}
	ps := styleOf(plain)
	if ps.Size.Width.Kind != style.Auto || ps.TextAlign != style.BlockRight || ps.AlignContent != style.AlignEnd {
		t.Errorf("plain, a width of 0 saying nothing and its own valign: %+v %v %v", ps.Size, ps.TextAlign, ps.AlignContent)
	}
	if ps.Background != hex("#fff") {
		t.Errorf("hexadecimal digits without their #: %v", ps.Background)
	}
	if hs := styleOf(head); hs.TextAlign != style.BlockLeft || hs.Background.A != 0 || hs.Size.Width.Kind != style.Auto {
		t.Errorf("head, a colour that is none left out whole: %v %v %+v", hs.TextAlign, hs.Background, hs.Size)
	}
}

func TestAnyRuleWinsOverAPresentationalAttribute(t *testing.T) {
	cell := attrs(el(dom.Td, "c"), "width", "100", "align", "center", "nowrap", "")
	table := attrs(with(el(dom.Table, "t"), with(dom.NewElement(dom.Tbody), with(dom.NewElement(dom.Tr), cell))),
		"cellpadding", "6")
	Compute(page(table), []*Sheet{sheet(t, `td { width: 40px; text-align: left; white-space: normal; padding: 2px }`)})
	s := styleOf(cell)
	if s.Size.Width != style.Px(40) || s.TextAlign != style.Left || s.WhiteSpace != style.Normal ||
		s.Padding.Top != style.Px(2) {
		t.Errorf("%+v %v %v %+v", s.Size, s.TextAlign, s.WhiteSpace, s.Padding)
	}
}

func TestACellFollowsItsTablesCellpaddingAndItsRowsValignWhenTheyChange(t *testing.T) {
	cell := el(dom.Td, "c")
	row := with(dom.NewElement(dom.Tr), cell)
	table := with(el(dom.Table, "t"), with(dom.NewElement(dom.Tbody), row))
	root := page(table)
	Compute(root, nil)
	if s := styleOf(cell); s.Padding.Top != style.Px(1) || s.AlignContent != style.AlignNormal {
		t.Fatalf("before: %+v %v", s.Padding, s.AlignContent)
	}
	table.SetAttr("cellpadding", "8")
	row.SetAttr("valign", "top")
	Compute(root, nil)
	if s := styleOf(cell); s.Padding.Top != style.Px(8) || s.AlignContent != style.AlignStart {
		t.Errorf("after: %+v %v", s.Padding, s.AlignContent)
	}
}

func TestAlignOnABlockAndBodysColours(t *testing.T) {
	div, p, h, span := el(dom.Div, "d"), el(dom.P, "p"), el(dom.H2, "h"), el(dom.Span, "s")
	attrs(div, "align", "center")
	attrs(p, "align", "RIGHT")
	attrs(h, "align", "justify")
	attrs(span, "align", "center")
	root := page(div, p, h, span)
	body := root.Children()[0]
	attrs(body, "bgcolor", "#102030", "text", "white")
	Compute(root, nil)
	if styleOf(div).TextAlign != style.BlockCenter || styleOf(p).TextAlign != style.BlockRight ||
		styleOf(h).TextAlign != style.Justify {
		t.Errorf("%v %v %v", styleOf(div).TextAlign, styleOf(p).TextAlign, styleOf(h).TextAlign)
	}
	if styleOf(span).TextAlign != style.Start {
		t.Errorf("a span takes no align: %v", styleOf(span).TextAlign)
	}
	if s := styleOf(body); s.Background != hex("#102030") || s.Color != hex("#fff") {
		t.Errorf("body: %v %v", s.Background, s.Color)
	}
}

func TestAHiddenInputHasNoBoxAndCenterPlacesBlocks(t *testing.T) {
	in, shown := el(dom.Input, "h"), el(dom.Input, "s")
	in.SetAttr("type", "hidden")
	center := el(dom.Center, "c")
	Compute(page(in, shown, center), nil)
	if styleOf(in).Display != style.DisplayNone || styleOf(shown).Display == style.DisplayNone {
		t.Errorf("%v %v", styleOf(in).Display, styleOf(shown).Display)
	}
	if styleOf(center).TextAlign != style.BlockCenter {
		t.Errorf("center: %v", styleOf(center).TextAlign)
	}
}
