package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func generatedNode(origin *dom.Node, p dom.Pseudo, edit func(*style.Computed), s string) *dom.Node {
	g := dom.NewGenerated(origin, p)
	st := style.Initial()
	st.Font = style.Font{Family: []string{"ahem"}, Size: 10, Weight: 400}
	st.LineHeight = style.LineHeight{Factor: 1}
	if edit != nil {
		edit(&st)
	}
	dom.Data(g).Style = &st
	if s != "" {
		g.SetGenerated([]*dom.Node{text(s)})
	}
	return g
}

func TestGeneratedNodesAreLaidOutBeforeAndAfterTheChildren(t *testing.T) {
	p := el(dom.P, block, text("HH"))
	before := generatedNode(p, dom.Before, nil, "HHH")
	after := generatedNode(p, dom.After, block, "H")
	dom.Data(p).Before, dom.Data(p).After = before, after
	root := el(dom.Div, width(100), p)
	tr := lay(root, 200)

	require.NotNil(t, tr.Of[before])
	require.NotNil(t, tr.Of[after])
	assert.Equal(t, geom.Rect{W: 30, H: 10}, tr.Of[before].Rect(), "inline, at the start of the first line")
	assert.Equal(t, geom.Rect{Y: 10, W: 100, H: 10}, tr.Of[after].Rect(), "a block, below the line")
	assert.Equal(t, float32(20), tr.Of[p].Size.Y)
	assert.Same(t, tr.Of[p], tr.Of[before].Parent.Parent, "inside its element's box, in the line's anonymous block")

	dom.Data(p).After = nil
	dom.Data(p).Dirty |= dom.DirtyLayout
	tr = lay(root, 200)
	assert.Equal(t, float32(10), tr.Of[p].Size.Y, "a box that went takes its room with it")
}

func TestAnEmptyPositionedGeneratedNodeIsABoxOfItsSize(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { st.Display, st.Position = style.Block, style.Relative }, text("HH"))
	photo := generatedNode(p, dom.Before, func(st *style.Computed) {
		st.Display, st.Position = style.Block, style.Absolute
		st.Inset.Left, st.Inset.Top = style.Percent(-60), style.Px(0)
		st.Size.Width, st.Size.Height = style.Percent(60), style.Px(40)
	}, "")
	dom.Data(p).Before = photo
	tr := lay(el(dom.Div, func(st *style.Computed) {
		st.Display, st.Size.Width, st.Margin.Left = style.Block, style.Px(100), style.Px(100)
	}, p), 300)
	assert.Equal(t, geom.Rect{X: 40, W: 60, H: 40}, tr.Of[photo].Rect())
	assert.Equal(t, float32(10), tr.Of[p].Size.Y, "out of the flow")
}
