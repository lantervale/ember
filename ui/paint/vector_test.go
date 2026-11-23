package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// icon is a picture of shapes that notes the sizes it was drawn at.
type icon struct {
	asks   box.Intrinsic
	drawn  [][2]int
	scales []float32
	none   bool
}

func (i *icon) Intrinsic() box.Intrinsic { return i.asks }

func (i *icon) Texture(w, h int, scale float32) *twod.Texture {
	i.drawn = append(i.drawn, [2]int{w, h})
	i.scales = append(i.scales, scale)
	if i.none {
		return nil
	}
	return twod.NewTexture(image.NewRGBA(image.Rect(0, 0, w, h)))
}

// shapesOf paints b once, i its picture of shapes: not also the two times
// EMBER_UI_CULL=check paints to compare, which would ask i for its texture
// three times.
func shapesOf(i *icon, b *box.Box, opts ...Option) *recorder {
	defer cull.Set(cull.On, nil)()
	found := func(name string) Vector {
		if name == "icon.svg" {
			return i
		}
		return nil
	}
	return paintOne(b, append([]Option{Images(func(string) *twod.Texture { return arrow }), Vectors(found)}, opts...)...)
}

func withIcon(w, h float32, edit func(s *style.Computed, l *style.BackgroundLayer)) *box.Box {
	return withImage(w, h, func(s *style.Computed, l *style.BackgroundLayer) {
		l.Image = style.Image{Source: "icon.svg"}
		if edit != nil {
			edit(s, l)
		}
	})
}

//nolint:gochecknoglobals // the tests' three kinds of picture
var (
	asksBoth    = box.Intrinsic{Width: 40, Height: 20, Ratio: 2}
	asksShape   = box.Intrinsic{Ratio: 2}
	asksNothing = box.Intrinsic{}
)

func TestABackgroundOfShapesIsSizedByWhatItAsksFor(t *testing.T) {
	auto, area := style.Length{}, geom.V2(120, 30)
	cover, contain := style.BackgroundSize{Scale: style.ScaleCover}, style.BackgroundSize{Scale: style.ScaleContain}
	for _, c := range []struct {
		why  string
		size style.BackgroundSize
		asks box.Intrinsic
		want geom.Vec2
	}{
		{"auto: the size it asks for", sized(auto, auto), asksBoth, geom.V2(40, 20)},
		{"a width keeps its proportions", sized(style.Px(60), auto), asksBoth, geom.V2(60, 30)},
		{"cover", cover, asksBoth, geom.V2(120, 60)},
		{"contain", contain, asksBoth, geom.V2(60, 30)},

		{"proportions alone: auto is contain", sized(auto, auto), asksShape, geom.V2(60, 30)},

		{"a width gives the height", sized(style.Percent(50), auto), asksShape, geom.V2(60, 30)},
		{"a height the width", sized(auto, style.Px(10)), asksShape, geom.V2(20, 10)},
		{"both stretch it", sized(style.Px(10), style.Px(10)), asksShape, geom.V2(10, 10)},
		{"cover", cover, asksShape, geom.V2(120, 60)},
		{"contain", contain, asksShape, geom.V2(60, 30)},

		{"nothing asked: auto is the whole box", sized(auto, auto), asksNothing, area},
		{"a width leaves the box's height", sized(style.Px(50), auto), asksNothing, geom.V2(50, 30)},
		{"a height the box's width", sized(auto, style.Percent(50)), asksNothing, geom.V2(120, 15)},
		{"cover is the box", cover, asksNothing, area},
		{"and contain", contain, asksNothing, area},

		{"a width alone asked: the box's height", sized(auto, auto), box.Intrinsic{Width: 80}, geom.V2(80, 30)},
		{"a height alone: the box's width", sized(auto, auto), box.Intrinsic{Height: 12}, geom.V2(120, 12)},
		{"its height holds beside a width given", sized(style.Px(50), auto), box.Intrinsic{Height: 12}, geom.V2(50, 12)},
	} {
		assert.Equal(t, c.want, tileSize(c.size, c.asks, area, 1), c.why)
	}
	assert.Equal(t, geom.V2(20, 10), tileSize(sized(auto, auto), asksShape, geom.V2(20, 120), 1), "contain in a tall box")
}

func TestABackgroundOfShapesIsDrawnAtTheSizeItShowsInDevicePixels(t *testing.T) {
	i := &icon{asks: asksShape}
	b := withIcon(120, 30, func(_ *style.Computed, l *style.BackgroundLayer) {
		l.Size = sized(style.Px(50), style.Length{})
	})
	imgs := shapesOf(i, b, Scale(2)).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 50, H: 25}, imgs[0].Rect)
	assert.Equal(t, [][2]int{{100, 50}}, i.drawn, "50×25 points on a 2x screen")
	assert.Equal(t, []float32{2}, i.scales, "which a length the file gives in pixels is drawn by")
	assert.Equal(t, geom.V2(100, 50), imgs[0].Tex.Size(), "stretched over its box whatever its pixels")

	i = &icon{asks: asksShape}
	shapesOf(i, b)
	assert.Equal(t, [][2]int{{50, 25}}, i.drawn, "a pixel a point where no scale is known")
	assert.Equal(t, []float32{1}, i.scales)
}

func TestARepeatedBackgroundOfShapesIsDrawnOnceForAllItsTiles(t *testing.T) {
	i := &icon{asks: asksBoth}
	imgs := shapesOf(i, withIcon(100, 40, func(_ *style.Computed, l *style.BackgroundLayer) {
		l.Repeat = style.BackgroundRepeat{}
	})).of("image")
	require.Len(t, imgs, 6, "three across, two down")
	assert.Equal(t, [][2]int{{40, 20}}, i.drawn, "one tile")
	for _, o := range imgs {
		assert.Same(t, imgs[0].Tex, o.Tex)
	}
}

func TestABackgroundOfShapesNotDrawnYetShowsNothing(t *testing.T) {
	i := &icon{asks: asksBoth, none: true}
	r := shapesOf(i, withIcon(100, 40, nil))
	assert.Empty(t, r.of("image"))
	assert.Empty(t, r.of("push"), "and clips nothing")
}

func TestAnImgOfShapesIsDrawnAtItsContentBox(t *testing.T) {
	for _, asks := range []box.Intrinsic{asksBoth, asksShape, asksNothing} {
		i := &icon{asks: asks}
		imgs := shapesOf(i, ctl(dom.Img, 100, 50, "src", "icon.svg"), Scale(1.5)).of("image")
		require.Len(t, imgs, 1)
		assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, imgs[0].Rect, "object-fit: fill, whatever it asks for")
		assert.Equal(t, [][2]int{{150, 75}}, i.drawn)
	}
}

func TestAnImgOfShapesFollowsItsObjectFit(t *testing.T) {
	fit := func(asks box.Intrinsic, f style.ObjectFit) geom.Rect {
		b := ctl(dom.Img, 100, 80, "src", "icon.svg")
		b.Style.ObjectFit = f
		b.Style.ObjectPosition = style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
		imgs := shapesOf(&icon{asks: asks}, b).of("image")
		require.Len(t, imgs, 1)
		return imgs[0].Rect
	}
	assert.Equal(t, geom.Rect{X: 40, Y: 40, W: 40, H: 20}, fit(asksBoth, style.FitNone), "the size it asks for")
	assert.Equal(t, geom.Rect{X: 10, Y: 25, W: 100, H: 50}, fit(asksBoth, style.FitContain))
	assert.Equal(t, geom.Rect{X: 10, Y: 25, W: 100, H: 50}, fit(asksShape, style.FitNone),
		"proportions alone: the largest of them its box holds")
	assert.Equal(t, geom.Rect{X: -20, Y: 10, W: 160, H: 80}, fit(asksShape, style.FitCover))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 80}, fit(asksNothing, style.FitNone), "nothing asked: its box")
}

func TestABorderImageOfShapesIsDrawnAtTheSizeItAsksFor(t *testing.T) {
	i := &icon{asks: asksBoth}
	b := el(10, 10, 100, 40, func(s *style.Computed) {
		s.BorderImage = style.Image{Source: "icon.svg"}
		s.Border = style.Sides[float32]{Top: 4, Right: 4, Bottom: 4, Left: 4}
		solid := style.BorderSolid
		s.BorderStyle = style.Sides[style.BorderStyle]{Top: solid, Right: solid, Bottom: solid, Left: solid}
	})
	b.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
	shapesOf(i, b, Scale(2))
	assert.Equal(t, [][2]int{{80, 40}}, i.drawn, "at the screen's density, its slices scaled to match")
}

func TestAPictureOfPixelsIsUntouchedByVectors(t *testing.T) {
	i := &icon{asks: asksBoth}
	imgs := shapesOf(i, withImage(120, 24, nil)).of("image")
	require.Len(t, imgs, 1)
	assert.Same(t, arrow, imgs[0].Tex)
	assert.Empty(t, i.drawn)
}
