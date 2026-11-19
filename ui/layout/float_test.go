package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func float(side style.Float, w, h float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display, st.Float = style.Block, side
		st.Size.Width, st.Size.Height = style.Px(w), style.Px(h)
	}
}

// fragmentAt is a line fragment and where it starts in the border box asked about.
type fragmentAt struct {
	text string
	x, y float32
}

func fragments(b *box.Box) []fragmentAt {
	var out []fragmentAt
	var walk func(b *box.Box, at geom.Vec2)
	walk = func(b *box.Box, at geom.Vec2) {
		for _, f := range b.Lines {
			out = append(out, fragmentAt{f.Text, at.X + f.Rect.X, at.Y + f.Rect.Y})
		}
		for _, c := range b.Children {
			walk(c, at.Add(c.Pos))
		}
	}
	walk(b, geom.Vec2{})
	return out
}

func TestLinesGoRoundAFloatAndBelowIt(t *testing.T) {
	f := el(dom.Div, float(style.FloatLeft, 20, 15))
	p := el(dom.P, block, text("HH HH HHHH HHHH"))
	tr := lay(el(dom.Div, width(50), f, p), 200)
	assert.Equal(t, geom.Rect{W: 20, H: 15}, tr.Of[f].Rect())
	assert.Equal(t, geom.Rect{W: 50, H: 40}, tr.Of[p].Rect(), "the paragraph keeps its width, behind the float")
	assert.Equal(t, []fragmentAt{{"HH", 20, 0}, {"HH", 20, 10}, {"HHHH", 0, 20}, {"HHHH", 0, 30}}, fragments(tr.Of[p]),
		"two lines beside the float, the rest under it")
}

func TestAWordTooWideBesideAFloatGoesBelowIt(t *testing.T) {
	f := el(dom.Div, float(style.FloatRight, 30, 15))
	p := el(dom.P, block, text("HHHH H"))
	tr := lay(el(dom.Div, width(50), f, p), 200)
	assert.Equal(t, geom.Rect{X: 20, W: 30, H: 15}, tr.Of[f].Rect())
	assert.Equal(t, []fragmentAt{{"HHHH", 0, 15}, {"H", 0, 25}}, fragments(tr.Of[p]), "below the float, at the left edge")
	assert.Equal(t, float32(35), tr.Of[p].Size.Y)
}

func TestAFloatAmongTextStandsAtTheTopOfItsParagraph(t *testing.T) {
	f := el(dom.Span, float(style.FloatRight, 20, 10))
	p := el(dom.P, block, text("HH HH "), f, text("HH HH"))
	tr := lay(el(dom.Div, width(60), p), 200)
	assert.Equal(t, geom.Rect{X: 40, W: 20, H: 10}, tr.Of[f].Rect())
	assert.Equal(t, []fragmentAt{{"HH", 0, 0}, {"HH ", 0, 10}, {"HH", 30, 10}, {"HH", 0, 20}}, fragments(tr.Of[p]),
		"one word fits beside the float, and the text after it follows the text before on one line")
}

func TestClearPutsABlockBelowTheFloatsOfItsSide(t *testing.T) {
	l := el(dom.Div, float(style.FloatLeft, 10, 30))
	r := el(dom.Div, float(style.FloatRight, 10, 50))
	clears := func(c style.Clear) func(*style.Computed) {
		return func(st *style.Computed) { st.Display, st.Clear = style.Block, c }
	}
	a := el(dom.P, clears(style.ClearLeft), text("H"))
	b := el(dom.P, clears(style.ClearBoth), text("H"))
	tr := lay(el(dom.Div, width(60), l, r, a, b), 200)
	assert.Equal(t, float32(30), tr.Of[a].Pos.Y)
	assert.Equal(t, float32(50), tr.Of[b].Pos.Y)
}

func TestOnlyAFormattingContextRootGrowsRoundItsFloats(t *testing.T) {
	plain := el(dom.Div, block, el(dom.Div, float(style.FloatLeft, 10, 30)))
	root := el(dom.Div, func(st *style.Computed) { st.Display, st.FlowRoot = style.Block, true },
		el(dom.Div, float(style.FloatLeft, 10, 30)))
	after := el(dom.Div, block, text("H"))
	tr := lay(el(dom.Div, width(60), plain, root, after), 200)
	assert.Equal(t, float32(0), tr.Of[plain].Size.Y, "a plain block is as tall as its flow")
	assert.Equal(t, geom.Rect{X: 10, W: 50, H: 30}, tr.Of[root].Rect(), "flow-root holds its float, beside the one before")
	assert.Equal(t, float32(30), tr.Of[after].Pos.Y)
}

func TestTextIndentMovesTheFirstLineOnly(t *testing.T) {
	indent := func(l style.Length) func(*style.Computed) {
		return func(st *style.Computed) { st.Display, st.TextIndent = style.Block, l }
	}
	first := func(l style.Length, s string) (float32, []string) {
		p := el(dom.P, indent(l), text(s))
		tr := lay(el(dom.Div, width(50), p), 200)
		return fragments(tr.Of[p])[0].x, lines(tr.Of[p])
	}
	x, ls := first(style.Px(20), "HH HH HH HH")
	assert.Equal(t, float32(20), x)
	assert.Equal(t, []string{"HH", "HH HH", "HH"}, ls, "the first line has 30 points left")
	x, _ = first(style.Percent(20), "HH")
	assert.Equal(t, float32(10), x, "a percentage is of the block's width")
	x, ls = first(style.Px(-20), "HH HH HH HH")
	assert.Equal(t, float32(-20), x, "a negative indent hangs the line out")
	assert.Equal(t, []string{"HH HH", "HH HH"}, ls)
	_, ls = first(style.Px(-30), "HH HH HH HH")
	assert.Equal(t, []string{"HH HH HH", "HH"}, ls, "and gives it more room")
}

func TestTextIndentCountsInShrinkToFitWidths(t *testing.T) {
	s := el(dom.Span,
		func(st *style.Computed) { st.Display, st.TextIndent = style.InlineBlock, style.Px(15) }, text("HHH"))
	tr := lay(el(dom.Div, width(100), s), 200)
	assert.Equal(t, float32(45), tr.Of[s].Size.X)
}

func TestTextAfterABlockIsNotIndentedAgain(t *testing.T) {
	d := el(dom.Div, func(st *style.Computed) {
		st.Display, st.TextIndent, st.Size.Width = style.Block, style.Px(10), style.Px(100)
	}, text("HH"),
		el(dom.P, func(st *style.Computed) { st.Display, st.TextIndent = style.Block, style.Px(10) }, text("HH")),
		text("HH"))
	tr := lay(el(dom.Div, block, d), 200)
	var xs []float32
	for _, f := range fragments(tr.Of[d]) {
		xs = append(xs, f.x)
	}
	assert.Equal(t, []float32{10, 10, 0}, xs, "the div's first line, the paragraph's, then no indent")
}

func TestAKeptLayoutWithFloatsMatchesAFreshOne(t *testing.T) {
	build := func() (*dom.Node, *dom.Node, *dom.Node) {
		f := el(dom.Div, float(style.FloatLeft, 20, 25))
		p := el(dom.P, block, text("HH HH HHHH HHHH"))
		after := el(dom.P, block, text("HHH HHH HHH"))
		return el(dom.Div, block, el(dom.Div, width(60), f, p, after)), f, p
	}
	changes := map[string]func(f, p *dom.Node){
		"nothing":         func(*dom.Node, *dom.Node) {},
		"the float grows": func(f, _ *dom.Node) { restyle(f, func(st *style.Computed) { st.Size.Height = style.Px(60) }) },
		"the float moves": func(f, _ *dom.Node) { restyle(f, func(st *style.Computed) { st.Float = style.FloatRight }) },
		"the float stops floating": func(f, _ *dom.Node) {
			restyle(f, func(st *style.Computed) { st.Float = style.FloatNone })
		},
		"the float leaves":    func(f, _ *dom.Node) { f.Parent().Remove(f) },
		"the text changes":    func(_, p *dom.Node) { p.Children()[0].SetText("H H H H H H H H H H") },
		"a float in the text": func(_, p *dom.Node) { p.Append(el(dom.Span, float(style.FloatRight, 10, 40)), text(" HH")) },
		"the paragraph clears": func(_, p *dom.Node) {
			restyle(p, func(st *style.Computed) { st.Clear = style.ClearBoth })
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			root, f, p := build()
			var r Retained
			layKept(&r, root, 300)
			change(f, p)
			kept := boxes(layKept(&r, root, 300))
			assert.Equal(t, boxes(lay(root, 300)), kept)
			kept = boxes(layKept(&r, root, 300))
			assert.Equal(t, boxes(lay(root, 300)), kept, "and a frame later")
		})
	}
}
