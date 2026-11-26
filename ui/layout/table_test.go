package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func cell(s string, edit ...func(*style.Computed)) *dom.Node {
	return el(dom.Td, func(st *style.Computed) {
		st.Padding = style.All(style.Px(1))
		st.VerticalAlign = style.AlignMiddle
		for _, e := range edit {
			e(st)
		}
	}, text(s))
}

// at is a box's top-left in the root's space.
func at(b *box.Box) geom.Vec2 { r := b.Rect(); return geom.Vec2{X: r.X, Y: r.Y} }

func row(cells ...*dom.Node) *dom.Node { return el(dom.Tr, nil, cells...) }

// table is a table with the user agent's 2 of border-spacing.
func table(children ...*dom.Node) *dom.Node {
	return el(dom.Table, func(st *style.Computed) { st.BorderSpacing = [2]float32{2, 2} }, children...)
}

func TestTableColumnsTakeTheirMaxContent(t *testing.T) {
	a, b := cell("HH"), cell("HHHH")
	c, d := cell("H"), cell("HH")
	r0, r1 := row(a, b), row(c, d)
	tb := table(el(dom.Tbody, nil, r0, r1))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 22 + 42 + 3*2, Y: 12*2 + 3*2},
		tr.Of[tb].Size, "columns, rows and 2 of spacing around them")
	assert.Equal(t, geom.Vec2{X: 2, Y: 2}, at(tr.Of[r0]).Sub(at(tr.Of[tb])))
	assert.Equal(t, geom.Vec2{X: 22 + 2 + 42, Y: 12}, tr.Of[r0].Size)
	assert.Equal(t, geom.Vec2{X: 2 + 22 + 2, Y: 2 + 12 + 2}, at(tr.Of[d]).Sub(at(tr.Of[tb])))
	assert.Equal(t, geom.Vec2{X: 42, Y: 12}, tr.Of[d].Size)
}

func TestATableShrinksBetweenMinAndMaxContent(t *testing.T) {
	a, b := cell("HH HH HH"), cell("H")
	tb := table(row(a, b))
	tr := lay(el(dom.Div, width(60), tb), 400)
	assert.Equal(t, float32(60), tr.Of[tb].Size.X)
	assert.Equal(t, float32(42), tr.Of[a].Size.X, "the 20 left above the min-contents go to the column that can grow")
	assert.Equal(t, float32(12), tr.Of[b].Size.X)
}

func TestTableCellsCentreTheirContent(t *testing.T) {
	tall := cell("H", func(st *style.Computed) { st.Size.Height = style.Px(30) })
	short := cell("H")
	tb := table(row(tall, short))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(30), tr.Of[short].Size.Y, "a row is as tall as its tallest cell")
	assert.Equal(t, float32(10), tr.Of[short].Children[0].Pos.Y, "1 of padding, then (28-10)/2")
}

func TestACellsVerticalAlignPlacesItsContent(t *testing.T) {
	tall := cell("H", func(st *style.Computed) { st.Size.Height = style.Px(30) })
	// On the baseline the text's first line stands on the row's, which the cell of a
	// block alone sets at its content box's bottom, 11 down: the text 2 lower.
	wants := []struct {
		at          style.VerticalAlign
		text, block float32
	}{
		{style.AlignTop, 1, 1}, {style.AlignMiddle, 10, 10}, {style.AlignBottom, 19, 19},
		{style.AlignBaselineText, 3, 1}, {style.AlignSub, 3, 1}, {style.AlignLength, 3, 1},
	}
	padded := func(st *style.Computed) { st.Padding = style.All(style.Px(1)) }
	for _, w := range wants {
		at := w.at
		words := cell("H", valign(at))
		blocks := el(dom.Td, valign(at, padded), el(dom.Div, func(st *style.Computed) {
			block(st)
			st.Size.Height = style.Px(10)
		}))
		tr := lay(el(dom.Div, block, table(row(tall, words, blocks))), 400)
		assert.Equal(t, w.text, tr.Of[words].Children[0].Pos.Y, "text, vertical-align %v", at)
		assert.Equal(t, w.block, tr.Of[blocks].Children[0].Pos.Y, "a block, vertical-align %v", at)
	}
	said := cell("H", valign(style.AlignTop), func(st *style.Computed) { st.AlignContent = style.AlignEnd })
	tr := lay(el(dom.Div, block, table(row(tall, said))), 400)
	assert.Equal(t, float32(19), tr.Of[said].Children[0].Pos.Y, "align-content says first")
}

func TestBorderSpacingSetsTheRoomBetweenCells(t *testing.T) {
	a, b := cell("HH"), cell("HHHH")
	c, d := cell("H"), cell("HH")
	r0, r1 := row(a, b), row(c, d)
	tb := el(dom.Table, func(st *style.Computed) { st.BorderSpacing = [2]float32{4, 0} }, el(dom.Tbody, nil, r0, r1))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.Vec2{X: 22 + 42 + 3*4, Y: 12 * 2}, tr.Of[tb].Size, "4 across, none down")
	assert.Equal(t, geom.Vec2{X: 4, Y: 0}, at(tr.Of[r0]).Sub(at(tr.Of[tb])))
	assert.Equal(t, geom.Vec2{X: 4 + 22 + 4, Y: 12}, at(tr.Of[d]).Sub(at(tr.Of[tb])))
}

func TestATableKeepsToItsMinAndMaxSizes(t *testing.T) {
	sized := func(edit func(st *style.Computed)) *dom.Node {
		return el(dom.Table, func(st *style.Computed) {
			st.TableLayout = style.TableFixed
			edit(st)
		}, row(el(dom.Td, nil)))
	}
	narrowed := sized(func(st *style.Computed) { st.Size.Width, st.MaxSize.Width = style.Px(300), style.Px(100) })
	widened := sized(func(st *style.Computed) { st.MinSize.Width = style.Px(100) })
	tall := sized(func(st *style.Computed) { st.MinSize.Height = style.Px(50) })
	held := sized(func(st *style.Computed) { st.Size.Height, st.MaxSize.Height = style.Px(300), style.Px(40) })
	tr := lay(el(dom.Div, block, narrowed, widened, tall, held), 400)
	assert.Equal(t, float32(100), tr.Of[narrowed].Size.X, "max-width over width")
	assert.Equal(t, float32(100), tr.Of[widened].Size.X, "min-width over an auto width")
	assert.Equal(t, float32(50), tr.Of[tall].Size.Y, "min-height, its row grown to it")
	assert.Equal(t, float32(40), tr.Of[held].Size.Y, "max-height over height")
}

func TestACollapsedTableIsItsWidthAndHalfItsOuterBorders(t *testing.T) {
	collapsed := func(group, own float32) *dom.Node {
		return el(dom.Table, func(st *style.Computed) {
			st.BorderCollapse, st.BoxSizing, st.Size.Width = true, style.ContentBox, style.Px(100)
			st.Border = style.Sides[float32]{Top: own, Right: own, Bottom: own, Left: own}
			st.BorderStyle = style.Sides[style.BorderStyle]{Top: style.BorderSolid, Right: style.BorderSolid,
				Bottom: style.BorderSolid, Left: style.BorderSolid}
		}, el(dom.Tbody, func(st *style.Computed) {
			st.Border = style.Sides[float32]{Top: group, Right: group, Bottom: group, Left: group}
			st.BorderStyle = style.Sides[style.BorderStyle]{Top: style.BorderSolid, Right: style.BorderSolid,
				Bottom: style.BorderSolid, Left: style.BorderSolid}
		}, row(cell(""), cell(""))))
	}
	byGroup, byTable := collapsed(4, 0), collapsed(0, 6)
	tr := lay(el(dom.Div, block, byGroup, byTable), 400)
	assert.Equal(t, float32(104), tr.Of[byGroup].Size.X, "a row group's border, half outside the content")
	assert.Equal(t, float32(106), tr.Of[byTable].Size.X, "the table's own, halved")
}

func TestAColumnKeepsToItsMinAndMaxWidths(t *testing.T) {
	a, b := cell(""), cell("")
	tb := el(dom.Table, func(st *style.Computed) { st.TableLayout = style.TableFixed },
		el(dom.Col, func(st *style.Computed) { st.Size.Width, st.MaxSize.Width = style.Px(300), style.Px(60) }),
		el(dom.Col, func(st *style.Computed) { st.MinSize.Width = style.Px(40) }),
		row(a, b))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(60), tr.Of[a].Size.X)
	assert.Equal(t, float32(40), tr.Of[b].Size.X, "a min-width alone sizes it")
}

func TestAColumnWithAWidthTakesNoExtra(t *testing.T) {
	name, size := cell("HH"), cell("H", func(st *style.Computed) { st.Size.Width = style.Px(30) })
	tb := el(dom.Table, func(st *style.Computed) { st.Size.Width = style.Px(200) }, row(name, size))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, float32(30), tr.Of[size].Size.X, "the sized column keeps its width")
	assert.Equal(t, float32(170), tr.Of[name].Size.X, "the rest goes to the auto column")
}
