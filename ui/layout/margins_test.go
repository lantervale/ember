package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestMarginsCollapsingThroughEveryChildLeaveTheTopOnce(t *testing.T) {
	empty := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Margin = style.Sides[style.Length]{Top: style.Px(16), Bottom: style.Px(16)}
	})
	framed := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Border.Bottom = 10
	}, empty)
	root := el(dom.Div, func(st *style.Computed) { st.Display, st.FlowRoot = style.Block, true }, framed)
	tr := lay(root, 200)
	assert.Equal(t, float32(16), tr.Of[framed].Rect().Y, "the empty block's margins out of its top")
	assert.Equal(t, float32(10), tr.Of[framed].Rect().H, "its border alone below them, as Chrome")
	assert.Equal(t, float32(26), tr.Of[root].Rect().H)
}

func TestTheRootsAutoMarginsShareTheViewportsWidth(t *testing.T) {
	root := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Width = style.Px(100)
		st.Margin.Left, st.Margin.Right = style.Length{Kind: style.Auto}, style.Length{Kind: style.Auto}
	})
	tr := lay(root, 300)
	assert.Equal(t, float32(100), tr.Of[root].Rect().X, "centred, as Chrome centres html")
}
