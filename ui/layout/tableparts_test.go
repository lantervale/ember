package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func part(p style.TablePart, edit ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display, st.TablePart = style.Block, p
		for _, e := range edit {
			e(st)
		}
	}
}

func asTable(st *style.Computed) { st.Display, st.Table = style.Block, true }

func TestAnyElementsAreATableByTheirDisplay(t *testing.T) {
	a, b := el(dom.Div, part(style.TableCell), text("HH")), el(dom.Span, part(style.TableCell), text("HHHH"))
	c, d := el(dom.Div, part(style.TableCell), text("H")), el(dom.Div, part(style.TableCell), text("HH"))
	r0, r1 := el(dom.Div, part(style.TableRow), a, b), el(dom.P, part(style.TableRow), c, d)
	tb := el(dom.Section, asTable, el(dom.Div, part(style.TableRowGroup), r0, r1))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 20 + 40, Y: 20}, tr.Of[tb].Size, "no spacing but a table's own")
	assert.Equal(t, geom.Vec2{X: 20, Y: 10}, at(tr.Of[d]).Sub(at(tr.Of[tb])))
	assert.Equal(t, geom.Vec2{X: 40, Y: 10}, tr.Of[d].Size)
	assert.Equal(t, geom.Vec2{X: 60, Y: 10}, tr.Of[r1].Size)
}

func TestCellsWithoutARowOrATableStandSideBySideInAnonymousOnes(t *testing.T) {
	a, b := el(dom.Div, part(style.TableCell), text("HH")), el(dom.Div, part(style.TableCell), text("HHH"))
	after := el(dom.Div, block, text("H"))
	root := el(dom.Div, block, a, text(" \n "), b, after)
	tr := lay(root, 400)
	assert.Equal(t, geom.Vec2{X: 0, Y: 0}, at(tr.Of[a]))
	assert.Equal(t, geom.Vec2{X: 20, Y: 0}, at(tr.Of[b]), "one row, the white space between them dropped")
	assert.Equal(t, geom.Vec2{X: 30, Y: 10}, tr.Of[b].Size)
	assert.Equal(t, geom.Vec2{X: 0, Y: 10}, at(tr.Of[after]), "the anonymous table is a block")
	table := tr.Of[a].Parent.Parent
	require.Equal(t, box.AnonymousBlock, table.Kind)
	assert.Equal(t, geom.Vec2{X: 50, Y: 10}, table.Size, "shrunk to its cells")
}

func TestARowWithoutATableIsWrappedInOne(t *testing.T) {
	a, b := el(dom.Div, part(style.TableCell), text("HH")), el(dom.Div, part(style.TableCell), text("H"))
	r0 := el(dom.Div, part(style.TableRow), a)
	r1 := el(dom.Div, part(style.TableRow), b)
	tr := lay(el(dom.Div, block, r0, r1), 400)
	assert.Equal(t, geom.Vec2{X: 0, Y: 10}, at(tr.Of[b]), "two rows of one table")
	assert.Equal(t, float32(20), tr.Of[b].Size.X, "in one column")
}

func TestWhatIsNoCellInARowIsWrappedInAnAnonymousCell(t *testing.T) {
	a := el(dom.Div, part(style.TableCell), text("HH"))
	loose, more := el(dom.Div, block, text("HHH")), el(dom.Span, nil, text("H"))
	r := el(dom.Div, part(style.TableRow), text("  "), a, loose, more, text(" "))
	tb := el(dom.Div, asTable, r)
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 50, Y: 20}, tr.Of[tb].Size, "two columns, the second its block and its line")
	assert.Equal(t, geom.Vec2{X: 20, Y: 0}, at(tr.Of[loose]))
	assert.Equal(t, geom.Vec2{X: 20, Y: 10}, at(tr.Of[more]), "the span in the same anonymous cell, below the block")
	assert.Equal(t, float32(20), tr.Of[a].Size.Y, "the cell as tall as its row")
}

func TestATableOfTextAloneShrinksRoundItsOneAnonymousCell(t *testing.T) {
	tb := el(dom.Div, asTable, text("HH HH"))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 50, Y: 10}, tr.Of[tb].Size)
	narrow := el(dom.Div, asTable, text("HH HH"))
	tr = lay(el(dom.Div, width(30), narrow), 400)
	assert.Equal(t, geom.Vec2{X: 30, Y: 20}, tr.Of[narrow].Size, "wrapped in the room it has")
}

func TestACellShownAsABlockIsWrappedWithItsNeighbours(t *testing.T) {
	a := cell("HH")
	b := el(dom.Td, part(style.NoTablePart), text("H"))
	c := el(dom.Td, part(style.NoTablePart), text("HHH"))
	tb := table(row(a, b, c))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 2 + 22 + 2, Y: 2}, at(tr.Of[b]).Sub(at(tr.Of[tb])), "in the second column")
	assert.Equal(t, geom.Vec2{X: 2 + 22 + 2, Y: 12}, at(tr.Of[c]).Sub(at(tr.Of[tb])),
		"below it, in the same anonymous cell")
	assert.Equal(t, float32(2+22+2+30+2), tr.Of[tb].Size.X)
}

func TestARowGroupInACellIsATableOfItsOwn(t *testing.T) {
	inner := el(dom.Div, part(style.TableCell), text("HH"))
	group := el(dom.Div, part(style.TableRowGroup), el(dom.Div, part(style.TableRow), inner))
	outer := el(dom.Div, part(style.TableCell), group)
	tb := el(dom.Div, asTable, el(dom.Div, part(style.TableRow), outer))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 20, Y: 10}, tr.Of[tb].Size)
	assert.Equal(t, geom.Vec2{X: 20, Y: 10}, tr.Of[inner].Size)
}

func TestAnInlineTableStandsInItsLine(t *testing.T) {
	c := el(dom.Span, part(style.TableCell), text("HH"))
	tb := el(dom.Span, func(st *style.Computed) { st.Display, st.Table = style.InlineBlock, true }, c)
	tr := lay(el(dom.Div, block, text("H"), tb, text("H")), 400)
	assert.Equal(t, geom.Vec2{X: 10, Y: 0}, at(tr.Of[tb]))
	assert.Equal(t, geom.Vec2{X: 20, Y: 10}, tr.Of[tb].Size)
}

func TestAnInlineTableOfEmptyBaselineCellsStandsOnTheirBottom(t *testing.T) {
	empty := el(dom.Td, func(st *style.Computed) { st.VerticalAlign = style.AlignBaselineText })
	tb := el(dom.Span, func(st *style.Computed) {
		st.Display, st.Table = style.InlineBlock, true
		st.Size.Height = style.Px(100)
	}, row(empty))
	tr := lay(el(dom.Div, block, text("H"), tb), 400)
	assert.Equal(t, float32(0), at(tr.Of[tb]).Y, "its baseline the cell's bottom, 100 down")
}

func TestCellsInAnInlineStandInAnAnonymousInlineTable(t *testing.T) {
	b, c := el(dom.Span, part(style.TableCell), text("H")), el(dom.Span, part(style.TableCell), text("HH"))
	d := el(dom.Span, nil, text("H"))
	span := el(dom.Span, nil, text("H "), b, text(" "), c, text(" "), d)
	tr := lay(el(dom.Div, block, span), 400)
	assert.Equal(t, geom.Vec2{X: 20, Y: 0}, at(tr.Of[b]), "after the text and its space, on its line")
	assert.Equal(t, geom.Vec2{X: 30, Y: 0}, at(tr.Of[c]), "beside it in one row, the space between them dropped")
	assert.Equal(t, geom.Vec2{X: 50, Y: 0}, at(tr.Of[d]), "the space after the table dropped too, as Chrome's")
	table := tr.Of[b].Parent.Parent
	require.Equal(t, box.AnonymousBlock, table.Kind)
	assert.Equal(t, geom.Vec2{X: 30, Y: 10}, table.Size)
}

func TestWhiteSpaceAloneInARowIsNoCellWhateverItsWhiteSpace(t *testing.T) {
	pre := func(st *style.Computed) { st.WhiteSpace = style.Pre }
	a, b := el(dom.Div, part(style.TableCell), text("HH")), el(dom.Div, part(style.TableCell), text("H"))
	r := el(dom.Div, part(style.TableRow, pre), text("\n  "), a, text("\t "), b, text(" "))
	tb := el(dom.Div, func(st *style.Computed) { asTable(st); pre(st) }, r)
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 30, Y: 10}, tr.Of[tb].Size, "two cells, no others")
	assert.Equal(t, geom.Vec2{X: 20, Y: 0}, at(tr.Of[b]))
}

func baselineCell(s string, edit ...func(*style.Computed)) *dom.Node {
	return el(dom.Td, func(st *style.Computed) {
		st.VerticalAlign = style.AlignBaselineText
		for _, e := range edit {
			e(st)
		}
	}, text(s))
}

func TestBaselineCellsOfARowPutTheirFirstLinesOnOneBaseline(t *testing.T) {
	padded := baselineCell("H", func(st *style.Computed) { st.Padding.Top = style.Px(7) })
	plain := baselineCell("H H")
	big := baselineCell("H", func(st *style.Computed) { st.Font.Size = 20 })
	top := baselineCell("H", func(st *style.Computed) { st.VerticalAlign = style.AlignTop })
	tb := el(dom.Table, nil, row(padded, plain, big, top))
	tr := lay(el(dom.Div, width(200), tb), 400)
	line := func(c *dom.Node) float32 { return at(tr.Of[c.Children()[0]]).Y }
	assert.Equal(t, float32(20), tr.Of[padded].Size.Y, "the row as tall as its tallest cell from the baseline")
	assert.Equal(t, float32(0), line(big), "the tallest ascent on top: Ahem's is 0.8 of 20")
	assert.Equal(t, float32(8), line(plain), "its baseline 8 down, on the big cell's 16")
	assert.Equal(t, float32(8), line(padded), "the padded cell's content pushed no further than its padding")
	assert.Equal(t, float32(0), line(top), "top is no baseline")
	assert.Equal(t, float32(20), tr.Of[plain].Size.Y, "every cell fills the row")
}

func TestABaselineCellHoldingBlocksIsShiftedWhole(t *testing.T) {
	blocks := el(dom.Td, func(st *style.Computed) { st.VerticalAlign = style.AlignBaselineText },
		el(dom.Div, block, text("H")), el(dom.Div, block, text("H")))
	big := baselineCell("H", func(st *style.Computed) { st.Font.Size = 20 })
	tb := el(dom.Table, nil, row(blocks, big))
	tr := lay(el(dom.Div, width(200), tb), 400)
	first, second := blocks.Children()[0], blocks.Children()[1]
	assert.Equal(t, float32(8), at(tr.Of[first]).Y)
	assert.Equal(t, float32(18), at(tr.Of[second]).Y)
	assert.Equal(t, float32(28), tr.Of[tb].Size.Y)
}

func TestAnInlineTablesBaselineIsItsFirstRows(t *testing.T) {
	c := el(dom.Span, part(style.TableCell, func(st *style.Computed) { st.Padding = style.All(style.Px(5)) }), text("H"))
	tb := el(dom.Span, func(st *style.Computed) { st.Display, st.Table = style.InlineBlock, true },
		el(dom.Span, part(style.TableRow), c), el(dom.Span, part(style.TableRow),
			el(dom.Span, part(style.TableCell), text("H"))))
	before := el(dom.Span, nil, text("H"))
	tr := lay(el(dom.Div, block, before, tb), 400)
	assert.Equal(t, float32(5), at(tr.Of[before]).Y, "the text on the first row's baseline")
	assert.Equal(t, float32(0), at(tr.Of[tb]).Y)
}

func TestColumnsStandOverTheirTracksAndSizeThem(t *testing.T) {
	first := el(dom.Col, func(st *style.Computed) { st.Size.Width = style.Px(50) })
	pair := el(dom.Col, nil)
	pair.SetAttr("span", "2")
	group := el(dom.Colgroup, nil, pair)
	a, b, c := cell("H"), cell("HH"), cell("HHH")
	tb := table(first, group, row(a, b, c), row(cell("H"), cell("H"), cell("H")))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(50), tr.Of[a].Size.X, "the column's width is its cells'")
	assert.Equal(t, geom.Rect{X: 2, Y: 2, W: 50, H: 26}, tr.Of[first].Rect(), "from the first row to the last")
	assert.Equal(t, geom.Rect{X: 54, Y: 2, W: 22 + 2 + 32, H: 26}, tr.Of[pair].Rect(),
		"two tracks and the spacing between")
	assert.Equal(t, tr.Of[pair].Rect(), tr.Of[group].Rect(), "a group round its columns")
	assert.Same(t, tr.Of[tb].Children[0], tr.Of[first].Parent, "under the rows, which are drawn after")
}

func TestColumnsWithoutCellsAreColumnsOfTheTable(t *testing.T) {
	wide := el(dom.Col, func(st *style.Computed) { st.Size.Width = style.Px(40) })
	tb := table(el(dom.Col, nil), wide, row(cell("H")))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(2+12+2+40+2), tr.Of[tb].Size.X)
}

func TestAHeightGivenToATableCollapsingARowCountsFromBeforeTheCollapse(t *testing.T) {
	gone := func(st *style.Computed) { st.Collapse = true }
	for name, container := range map[string]func(*style.Computed){
		"a flex column": func(st *style.Computed) { st.Display, st.Direction = style.FlexDisplay, style.Column },
		"a flex row":    func(st *style.Computed) { st.Display, st.AlignItems = style.FlexDisplay, style.AlignFlexStart },
		"a grid":        func(st *style.Computed) { st.Display = style.GridDisplay },
	} {
		t.Run(name, func(t *testing.T) {
			tb := table(row(cell("H")), el(dom.Tr, gone, cell("H")), row(cell("H")))
			tr := lay(el(dom.Div, block, el(dom.Div, container, tb)), 400)
			assert.Equal(t, float32(2+12+2+12+2), tr.Of[tb].Size.Y, "the collapsed row and a spacing out, once")
		})
	}
}

func TestAStretchedTableNarrowsByItsCollapsedColumnButInAFlexRow(t *testing.T) {
	gone := func(st *style.Computed) { st.Collapse = true }
	for name, c := range map[string]struct {
		container func(*style.Computed)
		narrows   bool
	}{
		"a grid": {func(st *style.Computed) { st.Display = style.GridDisplay }, true},
		"a flex column": {func(st *style.Computed) {
			st.Display, st.Direction = style.FlexDisplay, style.Column
		}, true},
		"a flex row": {func(st *style.Computed) { st.Display = style.FlexDisplay }, false},
	} {
		t.Run(name, func(t *testing.T) {
			middle := cell("HH")
			shown := table(row(cell("H"), middle, cell("H")))
			collapsed := table(el(dom.Col, nil), el(dom.Col, gone), row(cell("H"), cell("HH"), cell("H")))
			wrap := func(tb *dom.Node) *dom.Node {
				return el(dom.Div, func(st *style.Computed) {
					c.container(st)
					st.Size.Width = style.Px(200)
				}, tb)
			}
			tr := lay(el(dom.Div, block, wrap(shown), wrap(collapsed)), 400)
			full := tr.Of[shown].Size.X
			if !c.narrows {
				assert.Equal(t, full, tr.Of[collapsed].Size.X, "as wide as its container made it")
				return
			}
			assert.Equal(t, float32(200), full)
			assert.Equal(t, full-tr.Of[middle].Size.X-2, tr.Of[collapsed].Size.X,
				"the column and a spacing taken out of the stretched width")
		})
	}
}

func TestACollapsedRowGroupTakesItsAnonymousRowsWithIt(t *testing.T) {
	gone := el(dom.Tbody, func(st *style.Computed) { st.Collapse = true }, el(dom.Div, block, text("H")))
	kept := el(dom.Tbody, nil, row(cell("H")))
	tb := el(dom.Table, nil, gone, kept)
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(12), tr.Of[tb].Size.Y, "the block's anonymous row out with its group")
}
