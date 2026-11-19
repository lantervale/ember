package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scrolledPage is a 400×300 viewport scrolled to y over an html of
// 400×1000 with a 10-point border, holding body, styled by edit, and a
// row 20 points high at 500.
func scrolledPage(y float32, edit func(body *style.Computed)) (*box.Tree, *box.Box) {
	row := el(0, 500, 380, 20, nil)
	body := el(10, 10, 380, 980, edit, row)
	body.Node = dom.NewElement(dom.Body)
	html := el(0, 0, 400, 1000, func(s *style.Computed) { s.Border = style.All[float32](10) }, body)
	html.Node = dom.NewElement(dom.HTML)
	port := style.Initial()
	port.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	v := &box.Box{
		Kind: box.Viewport, Node: html.Node, Style: &port,
		Size: geom.V2(400, 300), Content: geom.V2(400, 1000), Scroll: geom.V2(0, y),
	}
	v.Append(html)
	return tree(v), row
}

func bottomRight(s *style.Computed) {
	s.Background = blue
	s.BackgroundImage = style.Image{Source: "arrow"}
	s.BackgroundNoRepeat = true
	s.BackgroundPosition = style.BackgroundPosition{FromRight: true, FromBottom: true}
}

func paintPage(tr *box.Tree) *recorder {
	r := newRecorder()
	Paint(r, tr, Viewport(geom.Rect{W: 400, H: 300}), Images(func(string) *twod.Texture { return arrow }))
	return r
}

func TestACanvasPictureIsPlacedInTheRootsBoxAndScrollsWithThePage(t *testing.T) {
	tr, _ := scrolledPage(0, bottomRight)
	r := paintPage(tr)
	assert.Equal(t, geom.Rect{W: 400, H: 300}, fills(r)[0].Rect, "the colour covers the viewport")
	img := r.of("image")
	require.Len(t, img, 1)
	assert.Equal(t, geom.Rect{X: 380, Y: 984, W: 10, H: 6}, img[0].Rect,
		"the bottom right of html's padding box, far below the viewport")
	assert.Equal(t, geom.Rect{W: 400, H: 300}, img[0].Clip)

	tr, _ = scrolledPage(700, bottomRight)
	assert.Equal(t, geom.Rect{X: 380, Y: 284, W: 10, H: 6}, paintPage(tr).of("image")[0].Rect,
		"in view at the page's end")
}

func TestARepeatedCanvasPictureTilesOverTheViewportFromTheRootsBox(t *testing.T) {
	tr, _ := scrolledPage(3, func(s *style.Computed) { s.BackgroundImage = style.Image{Source: "arrow"} })
	img := paintPage(tr).of("image")
	require.NotEmpty(t, img)
	assert.Equal(t, geom.Rect{X: 0, Y: 7 - 6 - 6, W: 10, H: 6}, img[0].Rect,
		"tiles start a whole number of pictures before the padding box, at 10, 7")
	last := img[len(img)-1].Rect
	assert.GreaterOrEqual(t, last.X+last.W, float32(400))
	assert.GreaterOrEqual(t, last.Y+last.H, float32(300))
}

func TestACanvasGradientIsSizedByTheRootsBoxAndTilesOverTheViewport(t *testing.T) {
	tr, _ := scrolledPage(0, func(s *style.Computed) {
		s.BackgroundImage = style.Image{Gradient: &style.Gradient{Angle: 180, Stops: stops(red, blue)}}
		s.BackgroundSize = style.BackgroundSize{Width: style.Percent(50), Height: style.Px(100)}
	})
	bands := paintPage(tr).of("gradient")
	require.Len(t, bands, 16, "four across by four down: one before the padding box each way, cut by the viewport")
	assert.Equal(t, geom.Rect{X: -180, Y: -90, W: 190, H: 100}, bands[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 190, H: 100}, bands[5].Rect, "half of html's padding box wide")
	assert.Equal(t, geom.Rect{W: 400, H: 300}, bands[5].Clip)
}

func TestAViewportClipsAndScrollsWhatItHolds(t *testing.T) {
	tr, row := scrolledPage(400, bg(red))
	row.Style.Background = blue
	var drawn []op
	for _, o := range fills(paintPage(tr)) {
		if o.Color == blue {
			drawn = append(drawn, o)
		}
	}
	require.Len(t, drawn, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 110, W: 380, H: 20}, drawn[0].Rect)
	assert.Equal(t, geom.Rect{W: 400, H: 300}, drawn[0].Clip)
}

func TestAViewportHasAScrollersBars(t *testing.T) {
	tr, _ := scrolledPage(350, nil)
	l := BarsOf(tr.Root, geom.Vec2{})
	assert.True(t, l.V)
	assert.False(t, l.H)
	assert.Equal(t, geom.Rect{W: 400, H: 300}, l.View)
	assert.Equal(t, geom.V2(0, 350), l.Scroll)
	assert.InDelta(t, 150, l.VThumb.Center().Y, 1, "half way down the page")

	thumbs := 0
	for _, o := range paintPage(tr).of("round") {
		if o.Color == thumbColor {
			thumbs++
		}
	}
	assert.Equal(t, 1, thumbs)
}

func TestAViewportIsStyledByItsOwnStyleNotTheRootElements(t *testing.T) {
	tr, _ := scrolledPage(0, nil)
	shownRoot := style.Initial()
	dom.Data(tr.Root.Node).Shown = &shownRoot
	assert.Same(t, tr.Root.Style, shown(tr.Root))
	assert.True(t, BarsOf(tr.Root, geom.Vec2{}).V)
}

func TestHitTestingAViewportFindsThePageWhereItScrolledTo(t *testing.T) {
	tr, row := scrolledPage(400, nil)
	assert.Same(t, row, HitTest(tr, geom.V2(50, 115)))
	assert.Same(t, tr.Root.Children[0].Children[0], HitTest(tr, geom.V2(50, 50)), "body")

	short := el(0, 0, 400, 50, nil)
	short.Node = dom.NewElement(dom.HTML)
	port := style.Initial()
	port.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	v := &box.Box{Kind: box.Viewport, Node: short.Node, Style: &port, Size: geom.V2(400, 300), Content: geom.V2(400, 300)}
	v.Append(short)
	assert.Same(t, v, HitTest(tree(v), geom.V2(50, 200)), "below a short page, the viewport takes the pointer")
	assert.Nil(t, HitTest(tree(v), geom.V2(50, 400)))
}
