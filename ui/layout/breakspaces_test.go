package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func breakSpaces(w float32, edits ...func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		width(w)(st)
		st.WhiteSpace = style.BreakSpaces
		for _, e := range edits {
			e(st)
		}
	}
}

func TestBreakSpacesTakeRoomAtALinesEnd(t *testing.T) {
	p := el(dom.P, breakSpaces(50), text("HH   HH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HH   ", "HH"}, lines(tr.Of[p]))
	assert.Equal(t, float32(50), tr.Of[p].Children[0].Size.X, "the spaces are the line's")
}

func TestBreakSpacesBreakAfterEachSpace(t *testing.T) {
	p := el(dom.P, breakSpaces(0), text("HH  HH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HH ", " ", "HH"}, lines(tr.Of[p]), "the first space stays with its word")
}

func TestBreakSpacesKeepTheFirstSpaceWithItsWord(t *testing.T) {
	p := el(dom.P, breakSpaces(30), text("HHH HH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HHH ", "HH"}, lines(tr.Of[p]))
}

func TestBreakSpacesStartALineWithTheSpacesThatWrap(t *testing.T) {
	p := el(dom.P, breakSpaces(40), text("HHH  HH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HHH ", " HH"}, lines(tr.Of[p]))
}

func TestBreakSpacesBreakingAnywhere(t *testing.T) {
	p := el(dom.P, breakSpaces(30, func(st *style.Computed) { st.LineBreak = style.LineBreakAnywhere }),
		text("HHH  HHHH"))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, []string{"HHH", "  H", "HHH"}, lines(tr.Of[p]))
}

func TestBreakSpacesAreInTheContentSizes(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		st.Display, st.WhiteSpace = style.InlineBlock, style.BreakSpaces
	}, text("HH   "))
	tr := lay(el(dom.Div, block, p), 300)
	assert.Equal(t, float32(50), tr.Of[p].Size.X)

	p = el(dom.P, func(st *style.Computed) {
		st.Display, st.WhiteSpace, st.Size.Width = style.Block, style.BreakSpaces, style.Length{Kind: style.MinContent}
	}, text("H  HHH  "))
	tr = lay(el(dom.Div, block, p), 300)
	assert.Equal(t, float32(40), tr.Of[p].Size.X, "HHH and the space after it")
}
