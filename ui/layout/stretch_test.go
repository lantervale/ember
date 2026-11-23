package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestStretchFillsTheContainingBlockLessTheMargins(t *testing.T) {
	stretched := func(st *style.Computed) {
		st.Size = style.Size[style.Length]{Width: style.Length{Kind: style.Stretch},
			Height: style.Length{Kind: style.Stretch}}
		st.Margin = style.Sides[style.Length]{Left: style.Px(10), Right: style.Px(5), Top: style.Px(3),
			Bottom: style.Px(2)}
	}
	ib := el(dom.Span, func(st *style.Computed) { stretched(st); st.Display = style.InlineBlock }, text("HH"))
	b := el(dom.Div, func(st *style.Computed) { stretched(st); st.Display = style.Block })
	tr := lay(el(dom.Div, func(st *style.Computed) { block(st); width(100)(st); st.Size.Height = style.Px(50) },
		el(dom.Div, block, ib), b), 200)
	assert.Equal(t, float32(85), tr.Of[ib].Size.X, "an inline block as wide as its line less its margins")
	assert.Equal(t, float32(85), tr.Of[b].Size.X)
	assert.Equal(t, float32(45), tr.Of[b].Size.Y, "a block as tall as its containing block less its margins")
}

func TestStretchLimitsAndFlexBasesFillTheContainingBlockLessTheMargins(t *testing.T) {
	margins := func(st *style.Computed) {
		st.Margin = style.Sides[style.Length]{Left: style.Px(10), Right: style.Px(5)}
	}
	least := el(dom.Div, func(st *style.Computed) {
		block(st)
		margins(st)
		width(20)(st)
		st.MinSize.Width = style.Length{Kind: style.Stretch}
	})
	most := el(dom.Div, func(st *style.Computed) {
		block(st)
		margins(st)
		width(900)(st)
		st.MaxSize.Width = style.Length{Kind: style.Stretch}
	})
	item := el(dom.Div, func(st *style.Computed) {
		block(st)
		margins(st)
		st.Basis = style.Length{Kind: style.Stretch}
	})
	tr := lay(el(dom.Div, func(st *style.Computed) { block(st); width(100)(st) }, least, most,
		el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay; width(100)(st) }, item)), 200)
	assert.Equal(t, float32(85), tr.Of[least].Size.X, "min-width: stretch")
	assert.Equal(t, float32(85), tr.Of[most].Size.X, "max-width: stretch")
	assert.Equal(t, float32(85), tr.Of[item].Size.X, "flex-basis: stretch")
}
