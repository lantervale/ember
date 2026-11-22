package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestASizeContainerIsSizedAsIfEmpty(t *testing.T) {
	filler := func() *dom.Node {
		return el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.Size = sizeOf(120, 30) })
	}
	for _, c := range []struct {
		name string
		box  func(*style.Computed)
		want geom.Vec2
	}{
		{"an inline-size flex item", func(st *style.Computed) { st.ContainerType = style.ContainerInlineSize },
			geom.V2(0, 30)},
		{"a size flex item", func(st *style.Computed) { st.ContainerType = style.ContainerSize }, geom.V2(0, 0)},
		{"padded, within its min-width", func(st *style.Computed) {
			st.ContainerType = style.ContainerSize
			st.Padding = style.All(style.Px(4))
			st.MinSize.Width = style.Px(20)
		}, geom.V2(20, 8)},
		{"sized by its own height", func(st *style.Computed) {
			st.ContainerType = style.ContainerSize
			st.Size.Height = style.Px(12)
		}, geom.V2(0, 12)},
		{"no container", func(*style.Computed) {}, geom.V2(120, 30)},
	} {
		t.Run(c.name, func(t *testing.T) {
			item := el(dom.Div, func(st *style.Computed) { st.Display = style.Block; c.box(st) }, filler())
			row := el(dom.Div, func(st *style.Computed) {
				st.Display = style.FlexDisplay
				st.AlignItems = style.AlignFlexStart
			}, item)
			tr := lay(el(dom.Div, block, row), 400)
			assert.Equal(t, c.want, tr.Of[item].Size)
		})
	}
}

func TestAnInlineSizeContainerInFlowIsAsWideAsItsRoom(t *testing.T) {
	inline := func(st *style.Computed) { st.Display = style.Block; st.ContainerType = style.ContainerInlineSize }
	item := el(dom.Div, inline,
		el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.Size = sizeOf(500, 30) }))
	tr := lay(el(dom.Div, block, item), 400)
	assert.Equal(t, geom.V2(400, 30), tr.Of[item].Size, "its content overflows it")
}

func sizeOf(w, h float32) style.Size[style.Length] {
	return style.Size[style.Length]{Width: style.Px(w), Height: style.Px(h)}
}
