package ui_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// widePage is a 400×300 window whose Document's viewport scrolls under a
// root of a vertical writing mode: forty columns 30 points wide, the
// page 1,200 wide, and a fixed badge.
type widePage struct {
	doc   *ui.Document
	w     *world.World
	in    *input.State
	cols  []*ui.Node
	badge *ui.Node
}

func newWidePage(t *testing.T, sheet string) *widePage {
	t.Helper()
	p := &widePage{doc: ui.NewDocument(ui.ScrollingViewport(), ui.OverlayScrollbars(), ui.CSS(`body { margin: 0; }
.col { width: 30px; }
#badge { position: fixed; left: 10px; bottom: 10px; width: 20px; height: 20px; }
`+sheet))}
	p.badge = ui.Div(ui.ID("badge"))
	p.doc.Body().Append(p.badge)
	for i := range 40 {
		c := ui.Div(ui.Class("col"), ui.Text(fmt.Sprint(i)))
		p.cols = append(p.cols, c)
		p.doc.Body().Append(c)
	}
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(p.doc)
	p.w = ember.New().AddEntities(win)
	p.in = win.Input()
	p.in.SetMouse(geom.V2(200, 150))
	p.w.RunFrames(2)
	return p
}

func (p *widePage) key(k input.Key) {
	p.in.SetKey(k, true)
	p.w.RunFrames(1)
	p.in.SetKey(k, false)
	p.w.RunFrames(settle)
}

func (p *widePage) scroll() geom.Vec2 { return p.doc.Scroll(p.doc.HTML()) }

func TestAVerticalRLRootStartsAtTheViewportsRightAndScrollsLeft(t *testing.T) {
	p := newWidePage(t, "html { writing-mode: vertical-rl; }")
	assert.Equal(t, geom.V2(1200, 300), ui.ScrollSize(p.doc, p.doc.HTML()))
	assert.Equal(t, geom.Rect{X: -800, Y: 0, W: 1200, H: 300}, p.doc.HTML().Box(), "against the right edge")
	assert.Equal(t, geom.Rect{X: 370, Y: 0, W: 30, H: 300}, p.cols[0].Box())

	p.doc.SetScroll(p.doc.HTML(), geom.V2(-300, 0))
	p.w.RunFrames(2)
	assert.Equal(t, geom.V2(-300, 0), p.scroll())
	assert.InDelta(t, -500, p.doc.HTML().Box().X, 0.01)
	assert.Equal(t, geom.Rect{X: 10, Y: 270, W: 20, H: 20}, p.badge.Box(), "the fixed badge stays put")

	p.doc.SetScroll(p.doc.HTML(), geom.V2(200, 0))
	p.w.RunFrames(2)
	assert.Equal(t, geom.V2(0, 0), p.scroll(), "nothing to scroll right of the start")
	p.doc.SetScroll(p.doc.HTML(), geom.V2(-1e6, 0))
	p.w.RunFrames(2)
	assert.Equal(t, geom.V2(-800, 0), p.scroll())
}

func TestAVerticalLRRootScrollsRight(t *testing.T) {
	p := newWidePage(t, "html { writing-mode: vertical-lr; }")
	assert.Equal(t, geom.Rect{X: 0, Y: 0, W: 1200, H: 300}, p.doc.HTML().Box())
	p.doc.SetScroll(p.doc.HTML(), geom.V2(-300, 0))
	p.w.RunFrames(2)
	assert.Equal(t, geom.V2(0, 0), p.scroll())
	p.doc.SetScroll(p.doc.HTML(), geom.V2(1e6, 0))
	p.w.RunFrames(2)
	assert.Equal(t, geom.V2(800, 0), p.scroll())
}

func TestTheRootTakesItsBodysWritingModeAndDirection(t *testing.T) {
	p := newWidePage(t, "body { writing-mode: vertical-rl; }")
	assert.Equal(t, geom.Rect{X: -800, Y: 0, W: 1200, H: 300}, p.doc.HTML().Box())

	p = newWidePage(t, "html { writing-mode: vertical-rl; } body { writing-mode: horizontal-tb; } .col { height: 30px; }")
	assert.Equal(t, float32(400), p.doc.HTML().Box().W, "html's lines run across, as body's")
	assert.Equal(t, geom.V2(400, 1200), ui.ScrollSize(p.doc, p.doc.HTML()))
}

func TestPagingKeysScrollAVerticalRootAlongItsBlocks(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	p := newWidePage(t, "html { writing-mode: vertical-rl; }")
	p.key(input.KeyPageDown)
	assert.Equal(t, geom.V2(-360, 0), p.scroll(), "a page left: the viewport less 40")
	p.key(input.KeyPageUp)
	assert.Equal(t, geom.V2(0, 0), p.scroll())
	p.key(input.KeyEnd)
	assert.Equal(t, geom.V2(-800, 0), p.scroll(), "to the left end")
	p.key(input.KeyHome)
	assert.Equal(t, geom.V2(0, 0), p.scroll())
	p.key(input.KeySpace)
	assert.Equal(t, geom.V2(-360, 0), p.scroll())
	p.key(input.KeyRight)
	assert.Equal(t, geom.V2(-320, 0), p.scroll(), "the arrows scroll as they point")
}

func TestOnMacPageDownPagesDownWhateverTheWritingMode(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	p := newWidePage(t, "html { writing-mode: vertical-rl; }")
	p.key(input.KeyPageDown)
	assert.Equal(t, geom.V2(0, 0), p.scroll(), "nothing below to page to")
	p.key(input.KeySpace)
	assert.Equal(t, geom.V2(-360, 0), p.scroll(), "Space pages along the blocks")
	p.key(input.KeyEnd)
	assert.Equal(t, geom.V2(-800, 0), p.scroll())
}

func TestScrollToStartPutsAColumnAtAVerticalRootsBlockStart(t *testing.T) {
	p := newWidePage(t, "html { writing-mode: vertical-rl; }")
	p.doc.ScrollToStart(p.cols[20])
	p.w.RunFrames(settle)
	assert.Equal(t, geom.V2(-600, 0), p.scroll())
	assert.Equal(t, float32(370), p.cols[20].Box().X, "its right edge at the viewport's")

	p.doc.SetScroll(p.doc.HTML(), geom.V2(0, 0))
	p.w.RunFrames(2)
	p.doc.ScrollIntoView(p.cols[20])
	p.w.RunFrames(settle)
	assert.Equal(t, geom.V2(-230, 0), p.scroll(), "by the least: its left edge at the viewport's")
	require.Equal(t, float32(0), p.cols[20].Box().X)
}
