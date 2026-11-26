package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestHitTestingFindsTheTopmostBoxInPaintOrder(t *testing.T) {
	under := el(0, 0, 50, 50, nil)
	over := el(0, 0, 50, 50, func(s *style.Computed) { s.Position, s.ZIndex, s.ZAuto = style.Relative, 1, false })
	flow := el(0, 0, 50, 50, nil)
	root := el(0, 0, 200, 200, nil, under, over, flow)
	tr := tree(root)
	assert.Same(t, over, HitTest(tr, geom.V2(10, 10)), "z 1 paints over the later flow box")
	assert.Same(t, root, HitTest(tr, geom.V2(100, 100)), "transparent boxes still take the pointer")
	assert.Nil(t, HitTest(tr, geom.V2(300, 10)))
}

func TestHitTestingSkipsTextRunsAndHiddenBoxes(t *testing.T) {
	p := el(0, 0, 100, 20, nil)
	p.Append(text(0, 0, "hello", p.Style))
	gone := el(0, 0, 100, 20, func(s *style.Computed) { s.Visible = false })
	root := el(0, 0, 200, 200, nil, p, gone)
	assert.Same(t, p, HitTest(tree(root), geom.V2(5, 5)), "the text's element, never its run, nor a hidden box")
}

func TestHitTestingPassesThroughPointerEventsNone(t *testing.T) {
	under := el(0, 0, 100, 100, nil)
	back := el(0, 0, 20, 20, nil)
	over := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Position, s.ZIndex, s.ZAuto, s.PointerNone = style.Relative, 1, false, true
	}, back)
	root := el(0, 0, 200, 200, nil, under, over)
	tr := tree(root)
	assert.Same(t, under, HitTest(tr, geom.V2(50, 50)), "what is under a pointer-events: none box takes the pointer")
	assert.Same(t, back, HitTest(tr, geom.V2(10, 10)), "a child with auto takes it back")
}

func TestHitTestingHonoursClipsScrollAndTransforms(t *testing.T) {
	item := el(0, 0, 100, 300, nil)
	sc := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, item)
	sc.Scroll = geom.V2(0, 50)
	turned := el(200, 0, 40, 20, func(s *style.Computed) {
		s.Transform = []style.TransformOp{{Kind: style.Rotate, A: [6]float32{geom.Pi / 2}}}
	})
	root := el(0, 0, 400, 400, nil, sc, turned)
	tr := tree(root)
	assert.Same(t, item, HitTest(tr, geom.V2(50, 90)))
	assert.Same(t, root, HitTest(tr, geom.V2(50, 150)), "clipped away below the scroller")
	assert.Same(t, turned, HitTest(tr, geom.V2(220, 25)), "turned upright about its centre")
	assert.Same(t, root, HitTest(tr, geom.V2(201, 2)), "its old corner is empty now")
}

func TestHitTestingAZeroOpacityBoxStillTakesThePointer(t *testing.T) {
	ghost := el(0, 0, 50, 50, func(s *style.Computed) { s.Opacity = 0 })
	root := el(0, 0, 100, 100, nil, ghost)
	assert.Same(t, ghost, HitTest(&box.Tree{Root: root}, geom.V2(5, 5)), "as in CSS: opacity 0 is still there")
}

func TestAHitTestReadsTheShownStyleOfRowsOutOfView(t *testing.T) {
	var rows []*box.Box
	for i := range 40 {
		rows = append(rows, el(0, float32(i)*20, 100, 20, bg(red)))
	}
	moved := rows[35]
	moved.Node = dom.NewElement(dom.Div)
	st := *moved.Style
	st.Transform = []style.TransformOp{{Kind: style.Translate, X: style.Px(0), Y: style.Px(-700)}}
	dom.Data(moved.Node).Shown = &st
	list := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, rows...)
	assert.Same(t, moved, HitTest(tree(el(0, 0, 400, 300, nil, list)), geom.V2(50, 10)),
		"the row a transform set at run time brought into view, though its layout style has none")
}

func TestHitTestingTakesABoxsRightAndBottomEdgesAsOutsideAsChromeDoes(t *testing.T) {
	cell := el(100, 10, 40, 40, nil)
	below := el(100, 50, 40, 40, nil)
	root := el(0, 0, 200, 200, nil, cell, below)
	tr := tree(root)
	assert.Same(t, cell, HitTest(tr, geom.V2(100, 10)), "its top-left corner is in")
	assert.Same(t, below, HitTest(tr, geom.V2(120, 50)), "its bottom edge is the next box's top")
	assert.Same(t, root, HitTest(tr, geom.V2(140, 30)), "its right edge is out")
	clipped := el(0, 0, 100, 100, nil)
	sc := el(0, 0, 50, 50, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, clipped)
	outer := el(0, 0, 200, 200, nil, sc)
	assert.Same(t, outer, HitTest(tree(outer), geom.V2(50, 20)), "a clip's right edge is out too")
	assert.Same(t, clipped, HitTest(tree(outer), geom.V2(49, 20)))
}

func TestAPointHitsThroughATransformByItsPixelSquare(t *testing.T) {
	r := geom.Rect{X: 0, Y: 0, W: 40, H: 20}
	turned := geom.Transform2DFrom(geom.Vec2{}, geom.Pi/6, 0, geom.V2(1, 1))
	back := turned.Inverse()
	inside := turned.Xform(geom.V2(20, 10))
	assert.True(t, reaches(r, back, inside), "a point inside")
	before := turned.Xform(geom.V2(-0.5, 10))
	assert.True(t, reaches(r, back, before), "half a pixel before the left edge, the square reaches in")
	assert.False(t, reaches(r, back, turned.Xform(geom.V2(43, 10))), "three pixels past the right edge")
	assert.True(t, shifted(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(3, -2)}))
	assert.False(t, shifted(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(0.5, 0)}))
}

func TestATablesClipLeavesItsCaptionsOut(t *testing.T) {
	caption := el(0, 0, 100, 10, func(s *style.Computed) { s.TablePart = style.TableCaption })
	above := el(0, 10, 100, 40, nil)
	table := el(0, 0, 100, 60, func(s *style.Computed) {
		s.Table = true
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, caption, above)
	table.Wrap = box.Edges{Top: 20}
	root := el(0, 0, 200, 200, nil, table)
	assert.Same(t, caption, HitTest(tree(root), geom.V2(50, 5)), "the caption stands outside the table's clip")
	assert.Same(t, root, HitTest(tree(root), geom.V2(50, 15)), "what overflows the grid into the caption's room is cut")
	assert.Same(t, above, HitTest(tree(root), geom.V2(50, 25)))
}
