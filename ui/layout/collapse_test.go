package layout //nolint:testpackage // white-box: lays out styled dom trees by hand

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bordered(w float32, s style.BorderStyle, c color.Color) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Border, st.BorderStyle, st.BorderColor = style.All(w), style.All(s), style.All(c)
	}
}

func collapsing(edit ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.BorderCollapse = true
		for _, e := range edit {
			e(st)
		}
	}
}

func collapsed(edit func(*style.Computed), children ...*dom.Node) *dom.Node {
	return el(dom.Table, func(st *style.Computed) {
		st.BorderSpacing, st.Padding, st.BorderCollapse = [2]float32{2, 2}, style.All(style.Px(7)), true
		if edit != nil {
			edit(st)
		}
	}, children...)
}

func TestTheBorderThatWinsAnEdge(t *testing.T) {
	red, blue := color.RGB(1, 0, 0), color.RGB(0, 0, 1)
	solid := func(w float32, c color.Color) shared { return shared{style: style.BorderSolid, width: w, color: c} }
	assert.Equal(t, solid(3, red), resolved(solid(1, blue), solid(3, red)), "the wider")
	assert.Equal(t, shared{hidden: true}, resolved(solid(9, blue), shared{hidden: true, width: 4}), "hidden over all")
	assert.Equal(t, solid(2, blue), resolved(solid(2, blue), solid(2, red)),
		"as good, the first offered: the nearer the cell")
	order := []style.BorderStyle{
		style.BorderDouble, style.BorderSolid, style.BorderDashed, style.BorderDotted, style.BorderRidge, style.BorderOutset,
		style.BorderGroove, style.BorderInset,
	}
	for i, s := range order[1:] {
		better, worse := shared{style: order[i], width: 2}, shared{style: s, width: 2}
		assert.Equal(t, better, resolved(worse, better), "%v over %v", order[i], s)
	}
}

func TestCollapsedBordersAreSharedHalfEach(t *testing.T) {
	black := color.RGB(0, 0, 0)
	edit := collapsing(bordered(4, style.BorderSolid, black))
	a, b, c, d := cell("H", edit), cell("H", edit), cell("H", edit), cell("H", edit)
	tb := collapsed(nil, row(a, b), row(c, d))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, geom.V2(2+2*16+2, 2+2*16+2), tr.Of[tb].Size,
		"no spacing, no padding, and half of the outer borders")
	assert.Equal(t, box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}, tr.Of[tb].Border)
	assert.Equal(t, geom.Rect{X: 2, Y: 2, W: 16, H: 16}, tr.Of[a].Rect(), "each cell from the middle of its borders")
	assert.Equal(t, geom.Rect{X: 18, Y: 18, W: 16, H: 16}, tr.Of[d].Rect())
	assert.Equal(t, box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}, tr.Of[d].Border)
	require.Len(t, tr.Of[tb].Collapsed, 12, "the edges of the grid, for the table to draw")
	assert.Equal(t, box.Edge{Rect: geom.Rect{X: 0, Y: 4, W: 4, H: 12}, Style: style.BorderSolid, Color: black},
		tr.Of[tb].Collapsed[0], "over the middle of the edge, stopping where the border across goes through")
}

func TestADoubleEdgeGoesThroughABorderAsWideAcrossIt(t *testing.T) {
	black := color.RGB(0, 0, 0)
	edit := collapsing(bordered(4, style.BorderDouble, black))
	a, b := cell("H", edit), cell("H", edit)
	tb := collapsed(nil, row(a, b))
	tr := lay(el(dom.Div, block, tb), 400)
	require.NotEmpty(t, tr.Of[tb].Collapsed)
	assert.Equal(t, box.Edge{Rect: geom.Rect{X: 0, Y: 0, W: 4, H: 20}, Style: style.BorderDouble, Color: black},
		tr.Of[tb].Collapsed[0], "its two lines cross the edge across, as Chrome draws them")
}

func TestAWiderBorderWinsItsEdgesAndWidensTheTable(t *testing.T) {
	thin := collapsing(bordered(2, style.BorderSolid, color.RGB(0, 0, 0)))
	wide := collapsing(bordered(6, style.BorderSolid, color.RGB(1, 0, 0)))
	a, b := cell("H", thin), cell("H", wide)
	tb := collapsed(bordered(2, style.BorderDashed, color.RGB(0, 0, 1)), row(a, b))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, box.Edges{Top: 1, Right: 3, Bottom: 1, Left: 1}, tr.Of[a].Border)
	assert.Equal(t, box.Edges{Top: 3, Right: 3, Bottom: 3, Left: 3}, tr.Of[b].Border)
	assert.Equal(t, box.Edges{Top: 3, Right: 3, Bottom: 3, Left: 1}, tr.Of[tb].Border, "the widest half along each side")
	assert.Equal(t, float32(1+1+12+3+3+12+3+3), tr.Of[tb].Size.X)
	left := tr.Of[tb].Collapsed[0]
	assert.Equal(t, style.BorderSolid, left.Style, "the cell's solid over the table's dashed, as wide")
}

func TestAHiddenBorderLeavesItsEdgeBare(t *testing.T) {
	black := color.RGB(0, 0, 0)
	plain := collapsing(bordered(4, style.BorderSolid, black))
	hidden := collapsing(bordered(0, style.BorderNone, black), func(st *style.Computed) {
		st.BorderHidden = style.Sides[bool]{Top: true, Right: true, Bottom: true, Left: true}
	})
	a, b := cell("H", plain), cell("H", hidden)
	tb := collapsed(nil, row(a, b))
	tr := lay(el(dom.Div, block, tb), 400)
	assert.Equal(t, box.Edges{Top: 2, Bottom: 2, Left: 2}, tr.Of[a].Border)
	assert.Equal(t, box.Edges{}, tr.Of[b].Border)
	assert.Len(t, tr.Of[tb].Collapsed, 3)
}

func TestATableThatStopsCollapsingGivesItsCellsTheirBordersBack(t *testing.T) {
	edit := collapsing(bordered(4, style.BorderSolid, color.RGB(0, 0, 0)))
	a, b := cell("H", edit), cell("H", edit)
	tb := collapsed(nil, row(a, b))
	root := el(dom.Div, block, tb)
	kept := &Retained{}
	tr := layKept(kept, root, 400)
	assert.Equal(t, float32(2+2*16+2), tr.Of[tb].Size.X)
	restyle(b, func(st *style.Computed) { st.Border = style.All[float32](8) })
	tr = layKept(kept, root, 400)
	assert.Equal(t, box.Edges{Top: 2, Right: 4, Bottom: 2, Left: 2}, tr.Of[a].Border,
		"a cell kept takes its neighbour's wider border")
	restyle(tb, func(st *style.Computed) { st.BorderCollapse = false })
	tr = layKept(kept, root, 400)
	assert.Equal(t, box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}, tr.Of[a].Border)
	assert.Equal(t, float32(7+2+20+2+28+2+7), tr.Of[tb].Size.X)
	assert.Nil(t, tr.Of[tb].Collapsed)
}
