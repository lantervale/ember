package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// multicol is a multi-column container 400 wide of count columns 20 apart: columns of
// 190 at 0 and 210 when there are two.
func multicol(count int, edit func(*style.Computed), children ...*dom.Node) *dom.Node {
	return el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Width = style.Px(400)
		st.ColumnCount = count
		st.Gap.Width = style.Px(20)
		if edit != nil {
			edit(st)
		}
	}, children...)
}

// para is a paragraph of n lines, each 10 high.
func para(n int, edit func(*style.Computed)) *dom.Node {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = "H"
	}
	return el(dom.P, func(st *style.Computed) {
		st.Display, st.WhiteSpace = style.Block, style.Pre
		if edit != nil {
			edit(st)
		}
	}, text(strings.Join(ls, "\n")))
}

func marginBottom(v float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Margin.Bottom = style.Px(v) }
}

func highBox(h float32) func(*style.Computed) {
	return func(st *style.Computed) { st.Display, st.Size.Height = style.Block, style.Px(h) }
}

// parts are b's parts in the root's space, the box itself when columns don't cut it.
func parts(b *box.Box) []geom.Rect {
	r := b.Rect()
	if !b.Broken {
		return []geom.Rect{r}
	}
	out := make([]geom.Rect, len(b.Fragments))
	for i, f := range b.Fragments {
		out[i] = geom.Rect{X: r.X + f.X, Y: r.Y + f.Y, W: f.W, H: f.H}
	}
	return out
}

func TestUsedColumnCountAndWidth(t *testing.T) {
	cases := []struct {
		count      int
		width      style.Length
		avail, gap float32
		wantCount  int
		wantWidth  float32
		why        string
	}{
		{2, style.Length{}, 400, 20, 2, 190, "a count shares the room less the gaps"},
		{0, style.Px(90), 400, 20, 3, 120, "a width gives as many columns as fit, widened to fill"},
		{3, style.Px(150), 400, 20, 2, 190, "a count is the most columns a width may give"},
		{3, style.Px(100), 400, 16, 3, 122.666664, "both, the count fitting"},
		{0, style.Px(500), 400, 20, 1, 400, "a width wider than the room is one column of the room"},
		{4, style.Length{}, 30, 20, 4, 0, "columns never go under zero"},
	}
	for _, c := range cases {
		st := style.Initial()
		st.ColumnCount, st.ColumnWidth = c.count, c.width
		n, w := usedColumns(&st, c.avail, c.gap)
		assert.Equal(t, c.wantCount, n, c.why)
		assert.InDelta(t, c.wantWidth, w, 0.001, c.why)
	}
}

func TestAColumnGapIsOneEmByDefault(t *testing.T) {
	p := para(4, nil)
	m := multicol(2, func(st *style.Computed) { st.Gap.Width = style.Length{} }, p)
	tr := lay(m, 600)
	assert.Equal(t, []geom.Rect{{W: 195, H: 20}, {X: 205, W: 195, H: 20}}, parts(tr.Of[p]), "the font is 10px")
}

func TestColumnsBalanceBetweenBlocks(t *testing.T) {
	a, b, c := para(3, marginBottom(5)), para(3, marginBottom(5)), para(5, marginBottom(5))
	m := multicol(2, nil, a, b, c)
	tr := lay(m, 600)
	assert.Equal(t, geom.Rect{W: 400, H: 65}, tr.Of[m].Rect(), "125 shared by two, grown by the 5 the second paragraph lacks")
	assert.Equal(t, geom.Rect{W: 190, H: 30}, tr.Of[a].Rect())
	assert.Equal(t, geom.Rect{Y: 35, W: 190, H: 30}, tr.Of[b].Rect())
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 50}, tr.Of[c].Rect())
	assert.Equal(t, &box.Columns{Count: 2, Width: 190, Gap: 20, Height: 65}, tr.Of[m].Columns)
}

func TestAMarginAtAColumnBreakIsDropped(t *testing.T) {
	margins := func(st *style.Computed) { st.Margin.Top, st.Margin.Bottom = style.Px(5), style.Px(5) }
	a, b := para(3, margins), para(3, margins)
	m := multicol(2, nil, a, b)
	tr := lay(m, 600)
	assert.Equal(t, float32(37.5), tr.Of[m].Size.Y, "the flow, its last margin too, shared by two")
	assert.Equal(t, geom.Rect{Y: 5, W: 190, H: 30}, tr.Of[a].Rect(), "the first column keeps the margin it starts with")
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 30}, tr.Of[b].Rect(), "the margins at the break are gone")
}

func TestAParagraphBreaksBetweenLines(t *testing.T) {
	p := para(7, marginBottom(5))
	m := multicol(2, nil, p)
	tr := lay(m, 600)
	b := tr.Of[p]
	assert.Equal(t, geom.Rect{W: 400, H: 40}, b.Rect(), "the box is its parts' union")
	assert.Equal(t, []geom.Rect{{W: 190, H: 40}, {X: 210, W: 190, H: 30}}, parts(b))
	want := []fragmentAt{{"H", 0, 0}, {"H", 0, 10}, {"H", 0, 20}, {"H", 0, 30}, {"H", 210, 0}, {"H", 210, 10}, {"H", 210, 20}}
	assert.Equal(t, want, fragments(b), "four lines, then three at the top of the second column")
}

func TestTextOfTheContainerItselfFillsColumns(t *testing.T) {
	m := multicol(2, func(st *style.Computed) { st.WhiteSpace = style.Pre }, text("H\nH\nH\nH"))
	tr := lay(m, 600)
	assert.Equal(t, float32(20), tr.Of[m].Size.Y)
	assert.Equal(t, []fragmentAt{{"H", 0, 0}, {"H", 0, 10}, {"H", 210, 0}, {"H", 210, 10}}, fragments(tr.Of[m]))
}

func TestABoxCutByAColumnFillsItToItsEnd(t *testing.T) {
	p := para(6, func(st *style.Computed) {
		st.Padding = style.All(style.Px(2))
		st.Border = style.Sides[float32]{Top: 1, Right: 1, Bottom: 1, Left: 1}
		st.Margin.Bottom = style.Px(5)
	})
	m := multicol(2, nil, p)
	tr := lay(m, 600)
	assert.Equal(t, float32(35.5), tr.Of[m].Size.Y, "71 shared by two")
	assert.Equal(t, []geom.Rect{{W: 190, H: 35.5}, {X: 210, W: 190, H: 33}}, parts(tr.Of[p]),
		"three lines under the top edges, to the column's end; three lines and the bottom edges")
}

func TestEmptyRoomIsCutAnywhere(t *testing.T) {
	d := el(dom.Div, highBox(130))
	m := multicol(2, nil, d)
	tr := lay(m, 600)
	assert.Equal(t, []geom.Rect{{W: 190, H: 65}, {X: 210, W: 190, H: 65}}, parts(tr.Of[d]))
	a, e, c := para(3, marginBottom(5)), el(dom.Div, highBox(25)), para(2, marginBottom(5))
	m = multicol(2, nil, a, e, c)
	tr = lay(m, 600)
	assert.Equal(t, float32(42.5), tr.Of[m].Size.Y)
	assert.Equal(t, []geom.Rect{{Y: 35, W: 190, H: 7.5}, {X: 210, W: 190, H: 17.5}}, parts(tr.Of[e]))
	assert.Equal(t, geom.Rect{X: 210, Y: 17.5, W: 190, H: 20}, tr.Of[c].Rect())
}

func TestOrphansAndWidows(t *testing.T) {
	heights := func(m *dom.Node, p *dom.Node) (float32, []float32) {
		tr := lay(m, 600)
		var hs []float32
		for _, r := range parts(tr.Of[p]) {
			hs = append(hs, r.H)
		}
		return tr.Of[m].Size.Y, hs
	}
	p := para(5, nil)
	h, hs := heights(multicol(2, nil, p), p)
	assert.Equal(t, float32(30), h)
	assert.Equal(t, []float32{30, 20}, hs, "three lines and two")

	p = para(3, nil)
	h, hs = heights(multicol(2, nil, p), p)
	assert.Equal(t, float32(20), h)
	assert.Equal(t, []float32{20, 10}, hs, "three lines can't leave two and take two: the break is made anyway")

	p = para(7, nil)
	h, hs = heights(multicol(3, nil, p), p)
	assert.Equal(t, float32(30), h)
	assert.Equal(t, []float32{30, 30, 20}, hs, "the second column gives a line up so the last takes two")

	p = para(5, nil)
	_, hs = heights(multicol(4, nil, p), p)
	assert.Equal(t, []float32{20, 20, 10}, hs, "a column never keeps fewer than orphans to feed widows")

	a, p := para(2, nil), para(3, nil)
	h, hs = heights(multicol(2, nil, a, p), p)
	assert.Equal(t, float32(30), h, "the paragraph goes to the next column whole")
	assert.Equal(t, []float32{30}, hs)

	a, p = para(1, nil), para(6, func(st *style.Computed) { st.Orphans = 4 })
	m := multicol(2, func(st *style.Computed) { st.Size.Height, st.ColumnFillAuto = style.Px(40), true }, a, p)
	_, hs = heights(m, p)
	assert.Equal(t, []float32{40, 20}, hs, "three lines would fit under the first, four orphans take them all along")
}

func TestMonolithicBoxesAreNotCut(t *testing.T) {
	scroller := el(dom.Div, func(st *style.Computed) {
		highBox(35)(st)
		st.Overflow.Height = style.Hidden
	})
	a, c := para(2, nil), para(1, nil)
	m := multicol(2, nil, a, scroller, c)
	tr := lay(m, 600)
	assert.Equal(t, float32(45), tr.Of[m].Size.Y, "65 shared by two holds neither; the least more room does")
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 35}, tr.Of[scroller].Rect())
	assert.Equal(t, geom.Rect{X: 210, Y: 35, W: 190, H: 10}, tr.Of[c].Rect())

	avoid := para(4, func(st *style.Computed) { st.BreakInsideAvoid = true })
	a, c = para(2, nil), para(2, nil)
	m = multicol(2, nil, a, avoid, c)
	tr = lay(m, 600)
	assert.Equal(t, float32(60), tr.Of[m].Size.Y)
	assert.Equal(t, geom.Rect{Y: 20, W: 190, H: 40}, tr.Of[avoid].Rect(), "break-inside: avoid keeps the lines together")
	assert.False(t, tr.Of[avoid].Broken)
}

func TestForcedAndAvoidedBreaks(t *testing.T) {
	a := para(1, nil)
	b := para(3, func(st *style.Computed) { st.BreakBefore, st.Margin.Top = style.BreakColumn, style.Px(5) })
	c := para(1, nil)
	m := multicol(2, nil, a, b, c)
	tr := lay(m, 600)
	assert.Equal(t, float32(45), tr.Of[m].Size.Y)
	assert.Equal(t, geom.Rect{X: 210, Y: 5, W: 190, H: 30}, tr.Of[b].Rect(), "a forced break keeps the margin after it")
	assert.Equal(t, geom.Rect{X: 210, Y: 35, W: 190, H: 10}, tr.Of[c].Rect())

	h := para(1, func(st *style.Computed) { st.BreakAfter = style.BreakAvoid })
	a, c = para(4, nil), para(5, nil)
	m = multicol(2, nil, a, h, c)
	tr = lay(m, 600)
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 10}, tr.Of[h].Rect(), "break-after: avoid takes the heading with what follows")
}

func TestADefiniteHeightAndColumnFill(t *testing.T) {
	high := func(h float32, fill bool) func(*style.Computed) {
		return func(st *style.Computed) { st.Size.Height, st.ColumnFillAuto = style.Px(h), fill }
	}
	p := para(6, nil)
	m := multicol(2, high(50, false), p)
	tr := lay(m, 600)
	assert.Equal(t, float32(50), tr.Of[m].Size.Y)
	assert.Equal(t, []geom.Rect{{W: 190, H: 30}, {X: 210, W: 190, H: 30}}, parts(tr.Of[p]), "balanced under the height")

	p = para(6, nil)
	tr = lay(multicol(2, high(50, true), p), 600)
	assert.Equal(t, []geom.Rect{{W: 190, H: 50}, {X: 210, W: 190, H: 20}}, parts(tr.Of[p]),
		"column-fill: auto fills the first column, less a line for the widows")

	p = para(6, nil)
	m = multicol(2, func(st *style.Computed) { st.ColumnFillAuto = true }, p)
	tr = lay(m, 600)
	assert.Equal(t, geom.Rect{W: 190, H: 60}, tr.Of[p].Rect(), "column-fill: auto without a height is one column")
	assert.Equal(t, float32(60), tr.Of[m].Size.Y)

	p = para(6, nil)
	m = multicol(2, func(st *style.Computed) { st.MinSize.Height = style.Px(50) }, p)
	tr = lay(m, 600)
	assert.Equal(t, float32(50), tr.Of[m].Size.Y)
	assert.Equal(t, float32(30), tr.Of[m].Columns.Height, "a min-height leaves the columns balanced")
}

func TestColumnsPastTheCountOverflowAtTheEnd(t *testing.T) {
	for name, edit := range map[string]func(*style.Computed){
		"height":     func(st *style.Computed) { st.Size.Height = style.Px(25) },
		"max-height": func(st *style.Computed) { st.MaxSize.Height = style.Px(25) },
		"fill auto": func(st *style.Computed) {
			st.MaxSize.Height, st.ColumnFillAuto = style.Px(25), true
		},
	} {
		p := para(7, nil)
		m := multicol(2, edit, p)
		tr := lay(m, 600)
		assert.Equal(t, geom.Rect{W: 400, H: 25}, tr.Of[m].Rect(), name)
		assert.Equal(t, []geom.Rect{{W: 190, H: 25}, {X: 210, W: 190, H: 25}, {X: 420, W: 190, H: 25}, {X: 630, W: 190, H: 10}},
			parts(tr.Of[p]), name)
		assert.Equal(t, float32(820), tr.Of[m].Content.X, name+": the columns are the scrollable overflow")
	}
}

func TestAPositionedBoxInColumns(t *testing.T) {
	badge := el(dom.Div, func(st *style.Computed) {
		absolute(st)
		st.Inset.Top, st.Inset.Right = style.Px(3), style.Px(4)
	})
	static := el(dom.Div, func(st *style.Computed) {
		absolute(st)
		st.Inset.Top = style.Length{}
	})
	moved := para(3, func(st *style.Computed) {
		st.Position, st.Inset.Top, st.Inset.Left = style.Relative, style.Px(-5), style.Px(5)
	})
	m := multicol(2, func(st *style.Computed) { st.Position = style.Relative }, badge, para(3, nil), static, moved)
	tr := lay(m, 600)
	assert.Equal(t, geom.Rect{X: 386, Y: 3, W: 10, H: 10}, tr.Of[badge].Rect(), "the container is its containing block, whole")
	assert.Equal(t, geom.Rect{Y: 30, W: 10, H: 10}, tr.Of[static].Rect(),
		"a box with no inset between blocks stays under the block before it, as Chrome's")
	assert.Equal(t, geom.Rect{X: 215, Y: -5, W: 190, H: 30}, tr.Of[moved].Rect(), "position: relative moves it from its column")
}

func TestAFloatStaysWithTheLinesBesideIt(t *testing.T) {
	f := el(dom.Div, float(style.FloatLeft, 50, 25))
	p := para(6, nil)
	m := multicol(2, nil, f, p)
	tr := lay(m, 600)
	assert.Equal(t, geom.Rect{W: 50, H: 25}, tr.Of[f].Rect())
	assert.Equal(t, []geom.Rect{{W: 190, H: 30}, {X: 210, W: 190, H: 30}}, parts(tr.Of[p]))
	assert.Equal(t, fragmentAt{"H", 50, 0}, fragments(tr.Of[p])[0], "the lines beside the float start after it")
}

func TestANestedMultiColumnContainerIsOneBox(t *testing.T) {
	inner := multicol(2, func(st *style.Computed) { st.Size.Width = style.Length{} }, para(4, nil))
	a, c := para(2, nil), para(2, nil)
	m := multicol(2, nil, a, inner, c)
	tr := lay(m, 600)
	assert.Equal(t, geom.Rect{Y: 20, W: 190, H: 20}, tr.Of[inner].Rect())
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 20}, tr.Of[c].Rect())
	assert.Equal(t, 2, tr.Of[inner].Columns.Count)
}

func TestAMultiColumnContainerShrinksToItsColumns(t *testing.T) {
	a, b := el(dom.P, block, text("HHHH HH")), el(dom.P, block, text("HH"))
	m := el(dom.Div, func(st *style.Computed) {
		st.Display, st.Float = style.Block, style.FloatLeft
		st.ColumnCount, st.Gap.Width = 2, style.Px(20)
	}, a, b)
	tr := lay(el(dom.Div, width(500), m), 600)
	assert.Equal(t, geom.Rect{W: 160, H: 10}, tr.Of[m].Rect(), "two columns as wide as the widest line, and the gap")
	assert.Equal(t, geom.Rect{X: 90, W: 70, H: 10}, tr.Of[b].Rect())
}

func TestARetainedLayoutMovesKeptTextToItsNewColumn(t *testing.T) {
	single := func(st *style.Computed) { st.Orphans, st.Widows = 1, 1 }
	c, a, b := para(4, nil), para(2, nil), para(2, single)
	m := multicol(2, nil, c, a, b)
	root := el(dom.Div, block, m)
	var keep Retained
	tr := Layout(root, geom.V2(600, 600), ahemFonts{}, Retain(&keep))
	assert.Equal(t, geom.Rect{X: 210, Y: 20, W: 190, H: 20}, tr.Of[b].Rect())
	assert.Equal(t, []fragmentAt{{"H", 0, 0}, {"H", 0, 10}}, fragments(tr.Of[b]))
	m.Remove(c)
	tr = Layout(root, geom.V2(600, 600), ahemFonts{}, Retain(&keep))
	assert.Equal(t, geom.Rect{W: 190, H: 20}, tr.Of[a].Rect())
	assert.Equal(t, geom.Rect{X: 210, W: 190, H: 20}, tr.Of[b].Rect(), "a kept paragraph follows the columns")
	c = para(1, nil)
	m.Append(c)
	tr = Layout(root, geom.V2(600, 600), ahemFonts{}, Retain(&keep))
	assert.Equal(t, []geom.Rect{{Y: 20, W: 190, H: 10}, {X: 210, W: 190, H: 10}}, parts(tr.Of[b]),
		"a kept paragraph is cut where the columns now break")
	assert.Equal(t, []fragmentAt{{"H", 0, 20}, {"H", 210, 0}}, fragments(tr.Of[b]))
}
