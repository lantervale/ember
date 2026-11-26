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

func TestABlockCutsTheInlineHoldingIt(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	span := el(dom.Span, nil, text("HH "), inner, text(" H"))
	root := el(dom.Div, width(100), span, text(" HH"))
	tr := lay(root, 200)
	assert.Equal(t, geom.Rect{Y: 10, W: 100, H: 10}, tr.Of[inner].Rect(), "a block of its container's width, on its own")
	assert.Same(t, tr.Of[root], tr.Of[inner].Parent)
	assert.Equal(t, []fragmentAt{{"HH", 0, 0}, {"HHH", 0, 10}, {"H", 0, 20}, {" HH", 10, 20}}, fragments(tr.Of[root]),
		"the text before the block above it, the text after below")
	assert.Equal(t, geom.Rect{W: 100, H: 30}, tr.Of[span].Rect(), "the inline's box covers its parts and the block")
	assert.Equal(t, float32(30), tr.Of[root].Size.Y)
}

func TestABlockAloneInAnInlineLeavesNoLines(t *testing.T) {
	item := el(dom.Li, block, text("HH"))
	list := el(dom.Ul, block, item)
	nav := el(dom.Nav, nil, text(" "), list, text(" "))
	root := el(dom.Div, width(100), nav)
	tr := lay(root, 200)
	assert.Equal(t, geom.Rect{W: 100, H: 10}, tr.Of[list].Rect())
	assert.Equal(t, geom.Rect{W: 100, H: 10}, tr.Of[nav].Rect())
	assert.Equal(t, float32(10), tr.Of[root].Size.Y, "the empty parts round the block take no line")
	require.Len(t, tr.Of[root].Children, 1)
}

func TestAnEmptyInlineWithRoomOnItsSidesKeepsALine(t *testing.T) {
	framed := func(edit func(st *style.Computed)) float32 {
		root := el(dom.Div, width(100), el(dom.Span, edit), text(" "))
		return lay(root, 200).Of[root].Size.Y
	}
	assert.Equal(t, float32(10), framed(func(st *style.Computed) { st.Border.Left = 3 }), "a border on its start")
	assert.Equal(t, float32(10), framed(func(st *style.Computed) { st.Padding.Right = style.Px(2) }), "padding on its end")
	assert.Zero(t, framed(func(st *style.Computed) { st.Border.Top, st.Padding.Bottom = 3, style.Px(2) }),
		"room above and below keeps none")
	assert.Zero(t, framed(nil))

	inner := el(dom.Div, block, text("H"))
	cut := el(dom.Span, func(st *style.Computed) { st.Dir, st.Padding.Right = style.RTL, style.Px(2) }, inner)
	root := el(dom.Div, width(100), cut)
	tr := lay(root, 200)
	assert.Equal(t, float32(10), tr.Of[inner].Rect().Y, "right to left, its right is its start: a line before the block")
	assert.Equal(t, float32(20), tr.Of[root].Size.Y, "and none after it")
}

func TestAFramedPartWithoutTextKeepsALine(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	span := el(dom.Span, func(st *style.Computed) {
		st.Padding.Left, st.Padding.Right = style.Px(5), style.Px(7)
	}, inner)
	root := el(dom.Div, width(100), span)
	tr := lay(root, 200)
	assert.Equal(t, geom.Rect{Y: 10, W: 100, H: 10}, tr.Of[inner].Rect(), "below the line of the start's padding")
	assert.Equal(t, float32(30), tr.Of[root].Size.Y, "and above the line of the end's")
	assert.Equal(t, geom.Rect{W: 100, H: 30}, tr.Of[span].Rect())
}

func TestPositionRelativeMovesAnInlineAndTheBlocksItHolds(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	span := el(dom.Span, func(st *style.Computed) {
		st.Position = style.Relative
		st.Inset.Left, st.Inset.Top = style.Px(20), style.Px(5)
	}, text("HH"), inner, text("H"))
	root := el(dom.Div, width(100), span)
	tr := lay(root, 200)
	assert.Equal(t, geom.Rect{X: 20, Y: 15, W: 100, H: 10}, tr.Of[inner].Rect(), "the block moves with it")
	assert.Equal(t, geom.Rect{X: 20, Y: 5, W: 100, H: 30}, tr.Of[span].Rect())
	assert.Equal(t, float32(30), tr.Of[root].Size.Y, "and leaves its room where it was")
}

func TestABlockInNestedInlinesCutsThemAll(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	em := el(dom.Em, nil, text("H"), inner, text("HH"))
	span := el(dom.Span, nil, text("HH "), em, text(" H"))
	root := el(dom.Div, width(100), span)
	tr := lay(root, 200)
	assert.Equal(t, []fragmentAt{{"HH ", 0, 0}, {"H", 30, 0}, {"HHH", 0, 10}, {"HH", 0, 20}, {" H", 20, 20}},
		fragments(tr.Of[root]))
	assert.Equal(t, geom.Rect{W: 100, H: 30}, tr.Of[span].Rect())
	assert.Equal(t, geom.Rect{W: 100, H: 30}, tr.Of[em].Rect())
}

func TestACutInlineKeepsItsStartEdgeBeforeTheBlockAndItsEndEdgeAfter(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	span := el(dom.Span, func(st *style.Computed) {
		st.Padding.Left, st.Padding.Right = style.Px(5), style.Px(7)
	}, text("HH"), inner, text("H"))
	root := el(dom.Div, width(100), span, text("H"))
	tr := lay(root, 200)
	assert.Equal(t, []fragmentAt{{"HH", 5, 0}, {"HHH", 0, 10}, {"H", 0, 20}, {"H", 17, 20}}, fragments(tr.Of[root]),
		"the left padding before the first part, the right one after the last")
	var parts []geom.Rect
	for _, anon := range tr.Of[root].Children {
		for _, c := range anon.Children {
			if c.Kind == box.Element && c.Node == span {
				parts = append(parts, c.Rect())
			}
		}
	}
	assert.Equal(t, []geom.Rect{{W: 25, H: 10}, {Y: 20, W: 17, H: 10}}, parts)
}

func TestABlockInAnInlineClearsTheFloatsBesideIt(t *testing.T) {
	f := el(dom.Div, float(style.FloatLeft, 60, 30))
	foot := el(dom.Footer, func(st *style.Computed) { st.Display, st.Clear = style.Block, style.ClearBoth }, text("HH"))
	main := el(dom.Div, nil, f, foot)
	root := el(dom.Div, width(100), main)
	tr := lay(root, 200)
	assert.Equal(t, geom.Rect{W: 60, H: 30}, tr.Of[f].Rect())
	assert.Equal(t, geom.Rect{Y: 30, W: 100, H: 10}, tr.Of[foot].Rect(), "below the float, as wide as the container")
}

func TestAKeptLayoutOfACutInlineMatchesAFreshOne(t *testing.T) {
	inner := el(dom.Div, block, text("HHH"))
	span := el(dom.Span, nil, text("HH"), inner, text("H"))
	root := el(dom.Div, width(100), span)
	var r Retained
	layKept(&r, root, 200)
	assert.Equal(t, boxes(lay(root, 200)), boxes(layKept(&r, root, 200)))
	inner.Children()[0].SetText("HH HH HH HH HH")
	tr := layKept(&r, root, 60)
	assert.Equal(t, boxes(lay(root, 60)), boxes(tr))
	assert.Equal(t, geom.Rect{W: 100, H: 40}, tr.Of[span].Rect())
	root.Remove(span)
	tr = layKept(&r, root, 60)
	assert.Nil(t, tr.Of[span], "an inline gone leaves no box")
	assert.Nil(t, tr.Of[inner])
}
