package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func breaking(wb style.WordBreak, lb style.LineBreak, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		st.WordBreak, st.LineBreak = wb, lb
		for _, e := range edits {
			e(st)
		}
	}
}

func shrunk(st *style.Computed) {
	st.Display = style.InlineBlock
	st.Size.Width = style.Length{Kind: style.MinContent}
}

func TestBreakAllFillsEachLineBeforeBreakingAWord(t *testing.T) {
	plain := el(dom.P, width(50), text("HH HHHHH"))
	all := el(dom.P, breaking(style.BreakAll, style.LineBreakAuto, width(50)), text("HH HHHHH"))
	tr := lay(el(dom.Div, block, plain, all), 200)
	assert.Equal(t, []string{"HH", "HHHHH"}, lines(tr.Of[plain]))
	assert.Equal(t, []string{"HH HH", "HHH"}, lines(tr.Of[all]), "the long word is cut where the line ends")
}

func TestBreakAllAndAnywhereMakeMinContentOneGrapheme(t *testing.T) {
	all := el(dom.Div, breaking(style.BreakAll, style.LineBreakAuto, shrunk), text("HHH HH"))
	anywhere := el(dom.Div, breaking(style.WordBreakNormal, style.LineBreakAnywhere, shrunk), text("HHH HH"))
	word := el(dom.Div, breaking(style.WordBreakBreakWord, style.LineBreakAuto, shrunk), text("HHH HH"))
	keep := el(dom.Div, breaking(style.KeepAll, style.LineBreakAuto, shrunk), text("HHH HH"))
	tr := lay(el(dom.Div, block, el(dom.P, block, all), el(dom.P, block, anywhere), el(dom.P, block, word),
		el(dom.P, block, keep)), 200)
	assert.Equal(t, float32(10), tr.Of[all].Size.X)
	assert.Equal(t, float32(10), tr.Of[anywhere].Size.X)
	assert.Equal(t, float32(10), tr.Of[word].Size.X, "break-word is overflow-wrap: anywhere")
	assert.Equal(t, float32(30), tr.Of[keep].Size.X, "keep-all leaves words of letters whole")
}
