package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withLayers is a box of w×h at (10, 10) with a red background under these
// layers, its pictures arrow.
func withLayers(w, h float32, ls ...style.BackgroundLayer) *box.Box {
	return el(10, 10, w, h, func(s *style.Computed) {
		s.Background = red
		stacked(s, ls...)
	})
}

func arrowLayer(edit func(l *style.BackgroundLayer)) style.BackgroundLayer {
	l := style.BackgroundLayer{Image: style.Image{Source: "arrow"}, Repeat: style.NoRepeat, Position: style.TopLeft}
	if edit != nil {
		edit(&l)
	}
	return l
}

func gradientLayer() style.BackgroundLayer {
	return style.BackgroundLayer{Image: style.Image{Gradient: &style.Gradient{Angle: 90, Stops: stops(red, blue)}},
		Position: style.TopLeft}
}

func TestTheFirstLayerIsPaintedLastOverTheColour(t *testing.T) {
	r := pictures(withLayers(120, 24, arrowLayer(nil), gradientLayer()))
	var kinds []string
	for _, o := range r.of("rect", "gradient", "image") {
		kinds = append(kinds, o.Kind)
	}
	assert.Equal(t, []string{"rect", "gradient", "image"}, kinds, "the colour, the last layer, then the first")
}

func TestEachLayerIsPlacedAndTiledByItsOwnLonghands(t *testing.T) {
	r := pictures(withLayers(120, 24,
		arrowLayer(func(l *style.BackgroundLayer) {
			l.Position = style.BackgroundPosition{X: style.Percent(100), Y: style.Percent(100)}
		}),
		arrowLayer(func(l *style.BackgroundLayer) { l.Repeat = repeatX }),
	))
	imgs := r.of("image")
	require.Len(t, imgs, 13, "twelve across for the last layer, then one at the bottom right")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 6}, imgs[0].Rect)
	assert.Equal(t, geom.Rect{X: 10 + 110, Y: 10, W: 10, H: 6}, imgs[11].Rect)
	assert.Equal(t, geom.Rect{X: 10 + 110, Y: 10 + 18, W: 10, H: 6}, imgs[12].Rect)
}

func TestTheColourShowsInTheLastLayersClipBox(t *testing.T) {
	b := withLayers(120, 40,
		arrowLayer(func(l *style.BackgroundLayer) { l.Clip = style.UnderContent }),
		arrowLayer(func(l *style.BackgroundLayer) { l.Clip = style.UnderPadding }),
	)
	b.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
	b.Padding = box.Edges{Top: 3, Right: 3, Bottom: 3, Left: 3}
	r := pictures(b)
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 116, H: 36}, r.of("rect")[0].Rect, "the padding box, the last layer's")
	imgs := r.of("image")
	require.Len(t, imgs, 2)
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 116, H: 36}, imgs[0].Clip)
	assert.Equal(t, geom.Rect{X: 15, Y: 15, W: 110, H: 30}, imgs[1].Clip)
}

func TestALayerOfNoneDrawsNothing(t *testing.T) {
	r := pictures(withLayers(120, 24, style.BackgroundLayer{}, arrowLayer(nil)))
	assert.Len(t, r.of("image"), 1)
	assert.Len(t, r.of("rect"), 1)
}

func TestSpaceSharesTheRoomLeftBetweenWholeTiles(t *testing.T) {
	spaced := arrowLayer(func(l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{X: style.TileSpace, Y: style.TileNoRepeat}
		l.Position = style.BackgroundPosition{X: style.Percent(100), Y: style.Percent(0)}
	})
	imgs := pictures(withLayers(45, 24, spaced)).of("image")
	require.Len(t, imgs, 4, "four of 10 fit in 45")
	for i, o := range imgs {
		assert.InDelta(t, 10+float32(i)*(10+5.0/3), o.Rect.X, 1e-4, "the first at the start, whatever the position")
		assert.Equal(t, float32(10), o.Rect.W, "never scaled")
	}
	assert.InDelta(t, 10+45, imgs[3].Rect.X+imgs[3].Rect.W, 1e-4, "the last at the end")

	lone := pictures(withLayers(15, 24, spaced)).of("image")
	require.Len(t, lone, 1, "one that fits once is placed by its position, as no-repeat")
	assert.Equal(t, geom.Rect{X: 10 + 15 - 10, Y: 10, W: 10, H: 6}, lone[0].Rect)
}

func TestSpaceBothWaysFillsThePaddingBox(t *testing.T) {
	imgs := pictures(withLayers(45, 20, arrowLayer(func(l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{X: style.TileSpace, Y: style.TileSpace}
	}))).of("image")
	require.Len(t, imgs, 4*3)
	last := imgs[len(imgs)-1].Rect
	assert.InDelta(t, 10+45, last.X+last.W, 1e-4)
	assert.InDelta(t, 10+20, last.Y+last.H, 1e-4)
}

func TestRoundScalesTheTileToAWholeNumberOfCopies(t *testing.T) {
	imgs := pictures(withLayers(45, 24, arrowLayer(func(l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{X: style.TileRound, Y: style.TileNoRepeat}
	}))).of("image")
	require.Len(t, imgs, 5, "4.5 copies round to 5")
	for i, o := range imgs {
		assert.InDelta(t, 10+float32(i)*9, o.Rect.X, 1e-4)
		assert.InDelta(t, 9, o.Rect.W, 1e-4)
		assert.InDelta(t, 5.4, o.Rect.H, 1e-4, "an auto height keeps the picture's proportions")
	}

	both := pictures(withLayers(45, 20, arrowLayer(func(l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{X: style.TileRound, Y: style.TileRound}
	}))).of("image")
	require.Len(t, both, 5*3)
	assert.InDelta(t, 9, both[0].Rect.W, 1e-4)
	assert.InDelta(t, 20.0/3, both[0].Rect.H, 1e-4, "each side rounded on its own")

	sized := pictures(withLayers(45, 24, arrowLayer(func(l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{X: style.TileRound, Y: style.TileNoRepeat}
		l.Size = style.BackgroundSize{Width: style.Px(10), Height: style.Px(8)}
	}))).of("image")
	assert.InDelta(t, 8, sized[0].Rect.H, 1e-4, "a given height is kept")
}

func TestLayersUnderOneClipShareABatch(t *testing.T) {
	w := world.New()
	win := platform.NewWindow(platform.Size(320, 200))
	w.AddEntities(win)
	w.AddPlugins(twod.Plugin{})
	w.Frame(1.0 / 60)
	c := twod.CanvasFor(w, win)
	require.NotNil(t, c)
	other := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	b := withLayers(120, 24, arrowLayer(nil), gradientLayer(), arrowLayer(func(l *style.BackgroundLayer) {
		l.Image.Source = "other"
		l.Repeat = style.BackgroundRepeat{}
	}))
	Paint(c, tree(el(0, 0, 320, 200, nil, b)), Images(func(name string) *twod.Texture {
		if name == "other" {
			return other
		}
		return arrow
	}))
	l := draw.FrontFor(w, win)
	require.NotNil(t, l)
	require.Len(t, l.Batches, 1, "the colour, a tiled picture, a gradient and a picture over them")
	assert.Equal(t, 2, l.Batches[0].ImageCount)
}
