package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestTheDisplaysOfATablesPartsMakeAnyElementOne(t *testing.T) {
	want := map[string]style.TablePart{
		"table-row": style.TableRow, "table-cell": style.TableCell, "table-row-group": style.TableRowGroup,
		"table-header-group": style.TableHeaderGroup, "table-footer-group": style.TableFooterGroup,
		"table-column": style.TableColumn, "table-column-group": style.TableColumnGroup,
		"table-caption": style.TableCaption,
	}
	for display, part := range want {
		d := el(dom.Div, "d")
		Compute(page(d), []*Sheet{sheet(t, "#d { display: "+display+" }")})
		assert.Equal(t, part, styleOf(d).TablePart, display)
		assert.Equal(t, style.Block, styleOf(d).Display, display)
		assert.False(t, styleOf(d).Table, display)
	}
	table, inline := el(dom.Div, "table"), el(dom.Span, "inline")
	Compute(page(table, inline), []*Sheet{sheet(t, "#table { display: table } #inline { display: inline-table }")})
	assert.True(t, styleOf(table).Table)
	assert.Equal(t, style.NoTablePart, styleOf(table).TablePart)
	assert.Equal(t, style.InlineBlock, styleOf(inline).Display)
}

func TestTableElementsAreTheirPartsUnlessASheetSaysOtherwise(t *testing.T) {
	td, th, block := el(dom.Td, "td"), el(dom.Th, "th"), el(dom.Td, "block")
	tr := with(el(dom.Tr, "tr"), td, th, block)
	head, body, foot := el(dom.Thead, "head"), with(el(dom.Tbody, "body"), tr), el(dom.Tfoot, "foot")
	caption := el(dom.Caption, "caption")
	table := with(el(dom.Table, "table"), caption, head, body, foot)
	Compute(page(table), []*Sheet{sheet(t, "#block { display: block }")})
	assert.True(t, styleOf(table).Table)
	assert.Equal(t, style.Block, styleOf(table).Display)
	for n, part := range map[*dom.Node]style.TablePart{
		td: style.TableCell, th: style.TableCell, tr: style.TableRow, head: style.TableHeaderGroup,
		body: style.TableRowGroup, foot: style.TableFooterGroup, caption: style.TableCaption, block: style.NoTablePart,
	} {
		assert.Equal(t, part, styleOf(n).TablePart, describe(n))
	}
}

func TestBorderCollapseInheritsAndHiddenIsANoneBorderThatIsKnown(t *testing.T) {
	td := el(dom.Td, "td")
	table := with(el(dom.Table, "table"), with(el(dom.Tbody, "b"), with(el(dom.Tr, "r"), td)))
	plain := el(dom.Table, "plain")
	Compute(page(table, plain), []*Sheet{sheet(t, `#table { border-collapse: collapse }
#td { border: 4px hidden red; border-left-style: solid }`)})
	assert.True(t, styleOf(table).BorderCollapse)
	assert.True(t, styleOf(td).BorderCollapse, "inherited")
	assert.False(t, styleOf(plain).BorderCollapse)
	st := styleOf(td)
	assert.Equal(t, style.Sides[bool]{Top: true, Right: true, Bottom: true}, st.BorderHidden)
	assert.Equal(t, style.Sides[float32]{Left: 4}, st.Border, "hidden is none where borders are apart")
	assert.Equal(t, style.BorderSolid, st.BorderStyle.Left)
}

func TestATablePartThatFloatsOrLeavesTheFlowOrIsAFlexItemIsABlock(t *testing.T) {
	float, abs, item, cell := el(dom.Div, "float"), el(dom.Div, "abs"), el(dom.Div, "item"), el(dom.Div, "cell")
	flex := with(el(dom.Div, "flex"), item)
	Compute(page(float, abs, flex, cell), []*Sheet{sheet(t, `div { display: table-cell } #flex { display: flex }
#float { float: left } #abs { position: absolute }`)})
	for _, n := range []*dom.Node{float, abs, item} {
		assert.Equal(t, style.NoTablePart, styleOf(n).TablePart, describe(n))
		assert.Equal(t, style.Block, styleOf(n).Display, describe(n))
	}
	assert.Equal(t, style.TableCell, styleOf(cell).TablePart)
}

func TestEmptyCellsInheritsAndCollapseIsAVisibilityThatHides(t *testing.T) {
	td := el(dom.Td, "td")
	row := with(el(dom.Tr, "r"), td)
	table := with(el(dom.Table, "table"), with(el(dom.Tbody, "b"), row))
	Compute(page(table), []*Sheet{sheet(t, `#table { empty-cells: hide } #r { visibility: collapse }`)})
	assert.True(t, styleOf(table).EmptyCellsHide)
	assert.True(t, styleOf(td).EmptyCellsHide, "inherited")
	for _, n := range []*dom.Node{row, td} {
		st := styleOf(n)
		assert.False(t, st.Visible, describe(n))
		assert.True(t, st.Collapse, describe(n))
	}
	td = el(dom.Td, "td")
	table = with(el(dom.Table, "table"), with(el(dom.Tbody, "b"), with(el(dom.Tr, "r"), td)))
	Compute(page(table), []*Sheet{sheet(t, `#table { empty-cells: hide } #td { empty-cells: show; visibility: hidden }`)})
	assert.False(t, styleOf(td).EmptyCellsHide)
	assert.False(t, styleOf(td).Visible)
	assert.False(t, styleOf(td).Collapse, "hidden is no collapse")
}
