package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// scrollingRoot is a root in a viewport that scrolls: overflow auto with
// a bar gutter points wide, scrolled down by scroll.
func scrollingRoot(gutter, scroll float32, children ...*dom.Node) *dom.Node {
	port := style.Initial()
	port.Display = style.Block
	port.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	port.ScrollbarWidth = gutter
	root := el(dom.HTML, func(st *style.Computed) {
		st.Display = style.Block
		st.Viewport = &port
	}, children...)
	dom.Data(root).Scroll = geom.V2(0, scroll)
	return root
}

func fixedAt(top, h float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Position = style.FixedPosition
		st.Inset.Top, st.Inset.Left = style.Px(top), style.Px(0)
		st.Size.Width = style.Px(50)
		st.Size.Height = style.Px(h)
	}
}

func TestAScrollingViewportHoldsTheRootAtItsOwnHeight(t *testing.T) {
	tall := blockOf(2000, nil)
	root := scrollingRoot(0, 0, tall)
	tr := lay(root, 400)
	v := tr.Root
	require.Equal(t, box.Viewport, v.Kind)
	assert.Same(t, root, v.Node, "the root keeps the viewport's scroll offset")
	assert.Equal(t, geom.V2(400, 600), v.Size)
	assert.Equal(t, geom.V2(400, 2000), v.Content)
	assert.Same(t, v, tr.Of[root].Parent)
	assert.Equal(t, geom.V2(400, 2000), tr.Of[root].Size, "html is as tall as the page, not the viewport")
}

func TestWithoutAScrollingViewportTheRootIsTheTreesRoot(t *testing.T) {
	root := el(dom.HTML, block, blockOf(2000, nil))
	tr := lay(root, 400)
	assert.Same(t, tr.Of[root], tr.Root)
	assert.Nil(t, tr.Root.Parent)
}

func TestAScrolledViewportMovesThePage(t *testing.T) {
	tall := blockOf(2000, nil)
	root := scrollingRoot(0, 300, tall)
	tr := lay(root, 400)
	assert.Equal(t, geom.V2(0, 300), tr.Root.Scroll)
	assert.Equal(t, geom.Rect{Y: -300, W: 400, H: 2000}, tr.Of[tall].Rect())
	assert.Equal(t, geom.Vec2{}, tr.Of[root].Scroll, "the offset is the viewport's, not html's")

	dom.Data(root).Scroll = geom.V2(0, 99999)
	assert.True(t, Refresh(tr, root))
	assert.Equal(t, geom.V2(0, 1400), tr.Root.Scroll, "clamped to the page's end")
	assert.Equal(t, geom.V2(0, 1400), dom.Data(root).Scroll)
}

func TestPercentHeightsResolveAgainstTheViewport(t *testing.T) {
	full := func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Height = style.Percent(100)
	}
	body := el(dom.Body, full, blockOf(1600, nil))
	root := scrollingRoot(0, 0, body)
	full(dom.Data(root).Style)
	tr := lay(root, 400)
	assert.Equal(t, float32(600), tr.Of[root].Size.Y)
	assert.Equal(t, float32(600), tr.Of[body].Size.Y)
	assert.Equal(t, float32(1600), tr.Root.Content.Y, "what overflows html still scrolls")
}

func TestAnAutoViewportMakesRoomForItsBarOnceThePageOverflows(t *testing.T) {
	page := blockOf(2000, nil)
	tr := lay(scrollingRoot(15, 0, page), 400)
	assert.Equal(t, geom.V2(15, 0), tr.Root.Scrollbar)
	assert.Equal(t, float32(385), tr.Of[page].Size.X)

	short := blockOf(100, nil)
	tr = lay(scrollingRoot(15, 0, short), 400)
	assert.Equal(t, geom.Vec2{}, tr.Root.Scrollbar)
	assert.Equal(t, float32(400), tr.Of[short].Size.X)
}

func TestARetainedViewportDropsItsBarOnceThePageFits(t *testing.T) {
	page := blockOf(2000, nil)
	root := scrollingRoot(15, 0, page)
	var kept Retained
	tr := Layout(root, geom.V2(400, 600), ahemFonts{}, Retain(&kept))
	require.Equal(t, float32(385), tr.Of[page].Size.X)
	first := tr.Root

	st := *dom.Data(page).Style
	st.Size.Height = style.Px(100)
	dom.Data(page).Style = &st
	dom.Data(page).Dirty |= dom.DirtyLayout
	dom.Data(root).Dirty |= dom.DirtyDescendant
	tr = Layout(root, geom.V2(400, 600), ahemFonts{}, Retain(&kept))
	assert.Equal(t, float32(400), tr.Of[page].Size.X)
	assert.Equal(t, geom.Vec2{}, tr.Root.Scrollbar)
	assert.Same(t, first, tr.Root, "the viewport's box is kept")
}

func TestAFixedBoxStaysPutInAScrolledViewport(t *testing.T) {
	inner := el(dom.Div, fixedAt(5, 10))
	bar := el(dom.Div, fixedAt(20, 30), inner)
	root := scrollingRoot(0, 300, blockOf(100, nil), bar, blockOf(2000, nil))
	tr := lay(root, 400)
	assert.Equal(t, geom.Rect{Y: 20, W: 50, H: 30}, tr.Of[bar].Rect())
	assert.Equal(t, geom.Rect{Y: 5, W: 50, H: 10}, tr.Of[inner].Rect(), "held once, by the fixed box around it")
	assert.Equal(t, float32(2100), tr.Root.Content.Y, "a fixed box adds nothing to what scrolls")

	dom.Data(root).Scroll = geom.V2(0, 700)
	Refresh(tr, root)
	assert.Equal(t, geom.Rect{Y: 20, W: 50, H: 30}, tr.Of[bar].Rect())
}

func TestAnAbsoluteBoxOfThePageAddsToWhatScrolls(t *testing.T) {
	low := el(dom.Div, func(st *style.Computed) {
		st.Position = style.Absolute
		st.Inset.Top, st.Inset.Left = style.Px(900), style.Px(0)
		st.Size = style.Size[style.Length]{Width: style.Px(50), Height: style.Px(50)}
	})
	tr := lay(scrollingRoot(0, 200, blockOf(100, nil), low), 400)
	assert.Equal(t, float32(950), tr.Root.Content.Y)
	assert.Equal(t, float32(700), tr.Of[low].Rect().Y, "it scrolls with the page")
}

func TestStickySticksToAScrolledViewport(t *testing.T) {
	header := blockOf(10, stickyTop(5))
	root := scrollingRoot(0, 300, blockOf(100, nil), header, blockOf(2000, nil))
	tr := lay(root, 400)
	assert.Equal(t, float32(5), tr.Of[header].Rect().Y)

	dom.Data(root).Scroll = geom.V2(0, 50)
	Refresh(tr, root)
	assert.Equal(t, float32(50), tr.Of[header].Rect().Y, "back in the flow, 100 down the page")
}

func TestAHiddenViewportStillHoldsItsOffset(t *testing.T) {
	root := scrollingRoot(0, 300, blockOf(2000, nil))
	dom.Data(root).Style.Viewport.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	tr := lay(root, 400)
	assert.Equal(t, geom.V2(0, 300), tr.Root.Scroll, "overflow: hidden scrolls by code, as an element's")
}
