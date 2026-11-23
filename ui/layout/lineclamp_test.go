package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func clamped(lines int, keeps bool) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display, st.FlowRoot = style.Block, true
		width(60)(st)
		if keeps {
			st.WebkitBox, st.BoxVertical, st.WebkitLineClamp = true, true, lines
			return
		}
		st.LineClamp = lines
	}
}

// textsOf is the text of each line n draws, laid out in root.
func textsOf(root, n *dom.Node) []string {
	tr := lay(root, 200)
	var out []string
	for _, k := range tr.Of[n].Children {
		for _, f := range k.Lines {
			out = append(out, f.Text)
		}
	}
	return out
}

func TestALineClampEndsItsLastLineInAnEllipsis(t *testing.T) {
	d := el(dom.Div, clamped(2, true), text("HH HH HH HH HH HH"))
	tr := lay(el(dom.Div, block, d), 200)
	assert.Equal(t, float32(20), tr.Of[d].Size.Y, "two lines of 10")
	frags := tr.Of[d].Children[0].Lines
	require.Len(t, frags, 3, "-webkit-line-clamp draws the lines after its own")
	assert.Equal(t, "HH HH…", frags[1].Text, "the line fits with its ellipsis")
	assert.True(t, frags[1].Ellipsis)
	assert.Equal(t, "HH HH", frags[2].Text)
}

func TestALineClampCutsItsLastLineForTheEllipsis(t *testing.T) {
	d := el(dom.Div, clamped(1, false), text("HHHHHH HH"))
	assert.Equal(t, []string{"HHHHH…"}, textsOf(el(dom.Div, block, d), d),
		"six glyphs and the ellipsis are wider than 60; line-clamp draws nothing after its lines")
}

func TestALineClampEndingWithItsTextHasNoEllipsis(t *testing.T) {
	d := el(dom.Div, clamped(2, false), text("HH HH HH HH"))
	tr := lay(el(dom.Div, block, d), 200)
	assert.Equal(t, []string{"HH HH", "HH HH"}, textsOf(el(dom.Div, block, d), d))
	assert.Equal(t, float32(20), tr.Of[d].Size.Y)
}

func TestALineClampCountsTheLinesOfTheBlocksInIt(t *testing.T) {
	first := el(dom.P, block, text("HH HH HH"))
	second := el(dom.P, func(st *style.Computed) { st.Display = style.Block; st.Padding.Bottom = style.Px(4) },
		text("HH HH HH"))
	after := el(dom.P, block, text("HH"))
	d := el(dom.Div, clamped(3, true), first, second, after)
	tr := lay(el(dom.Div, block, d), 200)
	assert.Equal(t, float32(34), tr.Of[d].Size.Y, "two lines, one, and the padding below it")
	assert.Equal(t, float32(14), tr.Of[second].Size.Y, "the block holding the clamp ends below its line")
	assert.Equal(t, float32(34), tr.Of[after].Pos.Y, "a block after it stands after it")
	assert.Equal(t, "HH HH…", tr.Of[second].Children[0].Lines[0].Text, "more follows the clamp")
}

func TestLineClampDiscardsTheBlocksAfterItsLastLine(t *testing.T) {
	kept := el(dom.P, block, text("HH HH HH"))
	after := el(dom.P, block, text("HH"))
	d := el(dom.Div, clamped(2, false), kept, after)
	tr := lay(el(dom.Div, block, d), 200)
	assert.False(t, tr.Of[kept].Discarded)
	assert.True(t, tr.Of[after].Discarded, "laid out, never drawn")
	assert.Equal(t, float32(20), tr.Of[after].Pos.Y)

	kept, after = el(dom.P, block, text("HH HH HH")), el(dom.P, block, text("HH"))
	d = el(dom.Div, clamped(2, true), kept, after)
	tr = lay(el(dom.Div, block, d), 200)
	assert.False(t, tr.Of[after].Discarded, "-webkit-line-clamp draws it")
}

func TestLineClampHidesTheLinesOfAnInlinePastItsLast(t *testing.T) {
	s := el(dom.Span, nil, text("HH HH HH HH HH HH"))
	d := el(dom.Div, clamped(2, false), s)
	tr := lay(el(dom.Div, block, d), 200)
	assert.Len(t, tr.Of[s].Fragments, 3, "laid out on every line, as Chrome's client rects")
	assert.Equal(t, 1, tr.Of[s].Hidden)

	s = el(dom.Span, nil, text("HH HH HH HH HH HH"))
	tr = lay(el(dom.Div, block, el(dom.Div, clamped(2, true), s)), 200)
	assert.Zero(t, tr.Of[s].Hidden, "-webkit-line-clamp draws it")
}

func TestLineClampDiscardsTheFloatsAfterItsLastLine(t *testing.T) {
	float := func(st *style.Computed) { st.Display, st.Float, st.Size.Width = style.Block, style.FloatLeft, style.Px(10) }
	before, after := el(dom.Div, float, text("H")), el(dom.Div, float, text("H"))
	d := el(dom.Div, clamped(1, false), before, el(dom.P, block, text("HH HH HH")), after)
	tr := lay(el(dom.Div, block, d), 200)
	assert.False(t, tr.Of[before].Discarded)
	assert.True(t, tr.Of[after].Discarded)
}

func TestLineClampAutoShowsTheLinesItsHeightHolds(t *testing.T) {
	auto := func(edit func(*style.Computed)) func(*style.Computed) {
		return func(st *style.Computed) {
			width(60)(st)
			st.ClampAuto = true
			edit(st)
		}
	}
	d := el(dom.Div, auto(func(st *style.Computed) { st.MaxSize.Height = style.Px(25) }), text("HH HH HH HH HH HH"))
	tr := lay(el(dom.Div, block, d), 200)
	assert.Equal(t, float32(20), tr.Of[d].Size.Y, "the two lines 25 holds")
	assert.Equal(t, []string{"HH HH", "HH HH…"}, lines(tr.Of[d]))

	p := el(dom.P, block, text("HH HH HH HH"))
	d = el(dom.Div, auto(func(st *style.Computed) { st.Size.Height = style.Px(15) }), p, el(dom.P, block, text("HH")))
	tr = lay(el(dom.Div, block, d), 200)
	assert.Equal(t, float32(15), tr.Of[d].Size.Y)
	assert.Equal(t, float32(10), tr.Of[p].Size.Y, "the block ends below the one line 15 holds")

	d = el(dom.Div, auto(func(*style.Computed) {}), text("HH HH HH HH HH HH"))
	assert.Equal(t, float32(30), lay(el(dom.Div, block, d), 200).Of[d].Size.Y, "nothing limits it")
}

func TestLinesHeldAreTheMostWhoseHeightFits(t *testing.T) {
	laid := func(lines int) (float32, int) { return float32(min(lines, 7)*10 + 4), 7 }
	assert.Equal(t, 3, linesHeld(35, laid))
	assert.Equal(t, 2, linesHeld(33, laid))
	assert.Equal(t, 0, linesHeld(80, laid), "all fit")
	assert.Equal(t, 0, linesHeld(none, laid))
	assert.Equal(t, 1, linesHeld(5, laid), "not even one")
}
