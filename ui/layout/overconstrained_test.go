package layout //nolint:testpackage // tests the layout tree

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAnOverConstrainedBlockInRightToLeftGivesWayOnTheLeft(t *testing.T) {
	rtl := func(w float32, margin style.Length) func(*style.Computed) {
		return func(st *style.Computed) {
			st.Dir = style.RTL
			st.Size.Width = style.Px(w)
			st.Margin.Right = margin
		}
	}
	inner := blockOf(10, rtl(40, style.Px(-100)))
	narrow := blockOf(10, rtl(0, style.Px(50)), inner)
	wide := blockOf(10, rtl(100, style.Px(150)))
	root := blockOf(20, rtl(0, style.Px(0)), narrow, wide)
	tr := lay(root, 400)
	assert.InDelta(t, -50, tr.Of[narrow].Rect().X, 0.01, "its right margin kept with no room left at all")
	assert.InDelta(t, 10, tr.Of[inner].Rect().X, 0.01, "a negative right margin past its container's right")
	assert.InDelta(t, -250, tr.Of[wide].Rect().X, 0.01)
}

func TestRightWinsOverLeftOnARelativeBoxInRightToLeft(t *testing.T) {
	moved := blockOf(10, func(st *style.Computed) {
		st.Dir, st.Position = style.RTL, style.Relative
		st.Size.Width = style.Px(50)
		st.Inset.Left, st.Inset.Right = style.Px(30), style.Px(20)
	})
	ltr := blockOf(10, func(st *style.Computed) {
		st.Position = style.Relative
		st.Size.Width = style.Px(50)
		st.Inset.Left, st.Inset.Right = style.Px(30), style.Px(20)
	})
	root := blockOf(20, func(st *style.Computed) { st.Dir, st.Size.Width = style.RTL, style.Px(100) }, moved)
	tr := lay(el(dom.Div, nil, root, blockOf(10, nil, ltr)), 400)
	assert.InDelta(t, 30, tr.Of[moved].Rect().X, 0.01, "at the right, 20 left of it")
	assert.InDelta(t, 30, tr.Of[ltr].Rect().X, 0.01, "left wins in left-to-right")
}
