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
