package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func tabbed(ws style.WhiteSpace, stop float32, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display, st.WhiteSpace, st.TabSize = style.Block, ws, style.TabSize{Px: stop, Length: true}
		for _, e := range edits {
			e(st)
		}
	}
}

func TestATabGoesToTheNextTabStop(t *testing.T) {
	p := el(dom.P, tabbed(style.Pre, 40), text("H\tHH\tH"))
	tr := lay(el(dom.Div, block, p), 300)
	run := tr.Of[p].Children[0]
	var xs []float32
	var texts []string
	for _, f := range run.Lines {
		xs, texts = append(xs, f.Rect.X), append(texts, f.Text)
	}
	assert.Equal(t, []string{"H", "HH", "H"}, texts, "the tabs are drawn as nothing")
	assert.Equal(t, []float32{0, 40, 80}, xs)
}

func TestATabStopIsMeasuredFromTheLinesStartBeforeItsIndent(t *testing.T) {
	p := el(dom.P, tabbed(style.Pre, 40, func(st *style.Computed) { st.TextIndent = style.Px(15) }), text("H\tH"))
	tr := lay(el(dom.Div, block, p), 300)
	fs := tr.Of[p].Children[0].Lines
	assert.Equal(t, float32(25), fs[1].Rect.X, "40 from the line's start, in a run that starts at the indent")
}

func TestATabThatEndsAWrappedLineHangs(t *testing.T) {
	p := el(dom.P, tabbed(style.PreWrap, 30, width(50)), text("HH\tHH\tH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HH", "HH", "H"}, lines(tr.Of[p]), "HH then a tab to 30 and HH would be 50")
	assert.LessOrEqual(t, tr.Of[p].Content.X, float32(50))
}

func TestACarriageReturnKeptIsNothing(t *testing.T) {
	p := el(dom.P, tabbed(style.Pre, 40), text("H\rH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HH"}, lines(tr.Of[p]), "one line, the carriage return neither shown nor a break")
	assert.Equal(t, float32(20), tr.Of[p].Children[0].Size.X)
}

func TestACarriageReturnKeptHoldsItsWordsTogether(t *testing.T) {
	p := el(dom.P, tabbed(style.PreWrap, 40, width(15)), text("H\rH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HH"}, lines(tr.Of[p]))
}

func TestTabsCollapseWhereSpacesDo(t *testing.T) {
	p := el(dom.P, tabbed(style.Normal, 40), text("H\tH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"H H"}, lines(tr.Of[p]))
}
