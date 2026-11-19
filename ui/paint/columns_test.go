package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

// cutBlock is a block at (10, 10) that two columns 110 apart cut: 30 points
// high in the first, from 20 down it, and 20 at the top of the second,
// with a 2-point blue border on every side and a red background.
func cutBlock() *box.Box {
	b := el(10, 10, 210, 50, func(s *style.Computed) {
		s.Background = red
		s.Border = style.All[float32](2)
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(blue)
	})
	b.Fragments = []geom.Rect{{X: 0, Y: 20, W: 100, H: 30}, {X: 110, Y: 0, W: 100, H: 20}}
	b.Broken = true
	return b
}

// coloured are the rectangles r filled with c.
func coloured(r *recorder, c any) []geom.Rect {
	var out []geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == c {
			out = append(out, o.Rect)
		}
	}
	return out
}

func TestABlockCutByColumnsFillsEachPart(t *testing.T) {
	assert.Equal(t, []geom.Rect{{X: 10, Y: 30, W: 100, H: 30}, {X: 120, Y: 10, W: 100, H: 20}},
		coloured(paintOne(cutBlock()), red), "never the union box")
}

func TestABlockCutByColumnsIsSlicedAtTheBreak(t *testing.T) {
	assert.ElementsMatch(t, []geom.Rect{
		{X: 10, Y: 30, W: 100, H: 2}, {X: 10, Y: 30, W: 2, H: 30}, {X: 108, Y: 30, W: 2, H: 30},
		{X: 120, Y: 10, W: 2, H: 20}, {X: 218, Y: 10, W: 2, H: 20}, {X: 120, Y: 28, W: 100, H: 2},
	}, coloured(paintOne(cutBlock()), blue), "sides on every part, the top on the first, the bottom on the last")
}

func TestHitTestingACutBlockUsesItsParts(t *testing.T) {
	b := cutBlock()
	root := el(0, 0, 400, 300, nil, b)
	tr := tree(root)
	assert.Same(t, b, HitTest(tr, geom.V2(20, 40)))
	assert.Same(t, b, HitTest(tr, geom.V2(130, 20)))
	assert.Same(t, root, HitTest(tr, geom.V2(20, 20)), "above the first part")
	assert.Same(t, root, HitTest(tr, geom.V2(130, 40)), "below the last part")
}

// ruled is a multi-column container at (10, 10) with 5 of padding: count
// columns 80 wide, 20 apart and 60 high, a 4-point green rule between them.
func ruled(count int, edit func(*style.Computed)) *box.Box {
	b := el(10, 10, 290, 100, func(s *style.Computed) {
		s.Padding = style.All(style.Px(5))
		s.ColumnCount = 3
		s.ColumnRuleWidth, s.ColumnRuleStyle, s.ColumnRuleColor = 4, style.BorderSolid, green
		if edit != nil {
			edit(s)
		}
	})
	b.Padding = box.Edges{Top: 5, Right: 5, Bottom: 5, Left: 5}
	b.Columns = &box.Columns{Count: count, Width: 80, Gap: 20, Height: 60}
	return b
}

func TestColumnRulesStandInTheMiddleOfTheGaps(t *testing.T) {
	assert.Equal(t, []geom.Rect{{X: 103, Y: 15, W: 4, H: 60}, {X: 203, Y: 15, W: 4, H: 60}},
		coloured(paintOne(ruled(3, nil)), green), "as high as the columns, from the content box's top")
}

func TestARuleIsDrawnOnlyBetweenColumnsWithContent(t *testing.T) {
	assert.Equal(t, []geom.Rect{{X: 103, Y: 15, W: 4, H: 60}}, coloured(paintOne(ruled(2, nil)), green))
	assert.Empty(t, coloured(paintOne(ruled(1, nil)), green))
}

func TestARuleOfStyleNoneOrNoWidthDrawsNothing(t *testing.T) {
	none := ruled(3, func(s *style.Computed) { s.ColumnRuleStyle = style.BorderNone })
	assert.Empty(t, coloured(paintOne(none), green))
	thin := ruled(3, func(s *style.Computed) { s.ColumnRuleWidth = 0 })
	assert.Empty(t, coloured(paintOne(thin), green))
}

func TestRulesGoOnBetweenTheColumnsThatOverflow(t *testing.T) {
	assert.Len(t, coloured(paintOne(ruled(5, nil)), green), 4)
}
