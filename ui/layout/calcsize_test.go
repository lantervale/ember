package layout //nolint:testpackage // tests calc-size() heights on the layout tree

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func calcSized(scale, px float32, borderBox bool) func(*style.Computed) {
	return func(st *style.Computed) {
		block(st)
		st.Size.Height = style.Length{Kind: style.Auto, Sized: true, Scale: scale, Px: px}
		st.Padding = style.All(style.Px(5))
		st.BoxSizing = style.ContentBox
		if borderBox {
			st.BoxSizing = style.BorderBox
		}
	}
}

func TestCalcSizeWorksOnTheAutoHeight(t *testing.T) {
	half := el(dom.Div, calcSized(0.5, 0, false), text("HH"))
	more := el(dom.Div, calcSized(1, 20, false), text("HH"))
	bb := el(dom.Div, calcSized(0.5, 0, true), text("HH"))
	tr := lay(el(dom.Div, block, half, more, bb), 200)
	assert.InDelta(t, 15, tr.Of[half].Size.Y, 0.001, "half the content's 10, then the padding")
	assert.InDelta(t, 40, tr.Of[more].Size.Y, 0.001)
	assert.InDelta(t, 10, tr.Of[bb].Size.Y, 0.001, "half the border box's 20")
}
