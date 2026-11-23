package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	stdcolor "image/color"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func masked(w, h float32, edit func(s *style.Computed)) *box.Box {
	return el(10, 10, w, h, func(s *style.Computed) {
		s.Background = color.RGB(0, 0, 1)
		s.MaskImages = []style.Image{{Source: "arrow"}}
		if edit != nil {
			edit(s)
		}
	})
}

func TestAMaskImageIsTheShapeOfTheBackgroundColour(t *testing.T) {
	r := pictures(masked(20, 20, func(s *style.Computed) {
		s.MaskRepeats = []style.BackgroundRepeat{style.NoRepeat}
		s.MaskPositions = []style.BackgroundPosition{{X: style.Percent(50), Y: style.Percent(50)}}
	}))
	assert.Empty(t, r.of("rect"), "the box is not filled")
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 15, Y: 17, W: 10, H: 6}, imgs[0].Rect, "the picture at its size, centred")
	assert.Equal(t, color.RGB(0, 0, 1), imgs[0].Color, "in the background colour")
	assert.NotSame(t, arrow, imgs[0].Tex, "its silhouette, not its colours")
}

func TestAMaskImageIsSizedAndTiledInTheBorderBox(t *testing.T) {
	b := masked(40, 12, func(s *style.Computed) {
		s.MaskSizes = []style.BackgroundSize{{Width: style.Px(20), Height: style.Px(12)}}
	})
	imgs := pictures(b).of("image")
	require.Len(t, imgs, 2, "two across the border box")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 20, H: 12}, imgs[0].Rect)
	assert.Equal(t, geom.Rect{X: 30, Y: 10, W: 20, H: 12}, imgs[1].Rect)

	b = masked(40, 12, func(s *style.Computed) {
		s.MaskSizes = []style.BackgroundSize{{Width: style.Px(20), Height: style.Px(12)}}
		s.MaskOrigins = []style.BackgroundBox{style.UnderContent}
		s.MaskRepeats = []style.BackgroundRepeat{style.NoRepeat}
	})
	b.Padding = box.Edges{Top: 1, Left: 3}
	imgs = pictures(b).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.Rect{X: 13, Y: 11, W: 20, H: 12}, imgs[0].Rect, "placed in the box mask-origin names")
}

func TestAMaskWithNoColourOrNoPicturePaintsNothing(t *testing.T) {
	colourless := masked(20, 20, func(s *style.Computed) { s.Background = color.Color{} })
	assert.Empty(t, pictures(colourless).of("image"))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, masked(20, 20, nil))), Images(func(string) *twod.Texture { return nil }))
	assert.Empty(t, r.of("image"), "a picture not there yet")
	assert.Empty(t, r.of("rect"), "and no square in its place")
}

func TestASilhouetteKeepsTheAlphaAlone(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, stdcolor.NRGBA{A: 0xff})
	src.SetNRGBA(1, 0, stdcolor.NRGBA{R: 0x80, G: 0x10, A: 0x40})
	tex := twod.NewTexture(src)
	got := silhouette(tex)
	assert.Equal(t, stdcolor.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, got.Image.At(0, 0))
	assert.Equal(t, stdcolor.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x40}, got.Image.At(1, 0))
	assert.Same(t, got, silhouette(tex), "made once")
	tex.Touch()
	assert.NotSame(t, got, silhouette(tex), "made again for a new picture")
}

// maskedPage paints a masked element holding a red child, its mask's
// layers set by edit, on a recorder of pictures.
func maskedPage(edit func(s *style.Computed)) *picturing {
	r := newPicturing()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, func(s *style.Computed) {
		s.Background = blue
		s.MaskImages = []style.Image{{Source: "arrow"}}
		s.MaskRepeats = []style.BackgroundRepeat{style.NoRepeat}
		if edit != nil {
			edit(s)
		}
	}, el(10, 10, 20, 20, bg(red))))), Images(func(string) *twod.Texture { return arrow }))
	return r
}

func TestAMaskedElementAndItsContentAreOnePictureCutByItsMasksPicture(t *testing.T) {
	r := maskedPage(nil)
	assert.Equal(t, "begin, rect, rect, end 0, begin, push 10, pop, end 0, cut 2, show(), pop", r.steps(),
		"the element and its child, then the mask's picture, the first cut by the second")
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Same(t, arrow, imgs[0].Tex, "the picture in its own colours")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 10, H: 6}, imgs[0].Rect, "placed in the border box")
}

func TestMaskLayersJoinAsTheirCompositingSays(t *testing.T) {
	two := func(c style.MaskComposite) func(s *style.Computed) {
		return func(s *style.Computed) {
			s.MaskImages = []style.Image{{Source: "arrow"}, {Source: "arrow"}}
			s.MaskComposites = []style.MaskComposite{c}
		}
	}
	const content = "begin, rect, rect, end 0, "
	for c, want := range []string{ // add, subtract, intersect and exclude
		"begin, push 10, pop, push 10, pop, end 0, cut 2",
		"begin, begin, push 10, pop, end 0, cut out 2, push 10, pop, pop, end 0, cut 3",
		"begin, begin, push 10, pop, end 0, cut 2, push 10, pop, pop, end 0, cut 3",
		"begin, begin, push 10, pop, end 0, begin, push 10, pop, end 0, cut out 2, show(), pop, " +
			"cut out 3, sum 2, pop, end 0, cut 4",
	} {
		assert.Equal(t, content+want+", show(), pop", maskedPage(two(style.MaskComposite(c))).steps(), "composite %d", c)
	}

	r := maskedPage(func(s *style.Computed) {
		s.MaskImages = []style.Image{{Source: "arrow"}, {}}
		s.MaskComposites = []style.MaskComposite{style.CompositeIntersect}
	})
	assert.Equal(t, content+"begin, begin, end 0, cut 2, push 10, pop, pop, end 0, cut 3, show(), pop", r.steps(),
		"cut by a layer of no picture, nothing is left")
	r = maskedPage(func(s *style.Computed) {
		s.MaskImages = []style.Image{{}, {Source: "arrow"}}
		s.MaskComposites = []style.MaskComposite{style.CompositeIntersect}
	})
	assert.Equal(t, content+"begin, push 10, pop, end 0, cut 2, show(), pop", r.steps(),
		"a layer of no picture is left out")
	r = maskedPage(func(s *style.Computed) { s.MaskComposites = []style.MaskComposite{style.CompositeSubtract} })
	assert.Equal(t, content+"begin, push 10, pop, end 0, cut 2, show(), pop", r.steps(),
		"the last layer's compositing has nothing to join")
}

func TestALuminanceMaskIsItsLayersPictureThroughAnEffect(t *testing.T) {
	r := maskedPage(func(s *style.Computed) { s.MaskModes = []style.MaskMode{style.MaskLuminance} })
	assert.Equal(t, "begin, rect, rect, end 0, begin, begin, push 10, pop, end 0, show(luminance), end 0, cut 3, "+
		"show(), pop", r.steps())
}

func TestATranslucentMaskedElementIsFadedAsOnePicture(t *testing.T) {
	r := maskedPage(func(s *style.Computed) { s.Opacity = 0.5 })
	assert.Equal(t, "begin, rect, rect, end 0, begin, push 10, pop, end 0, cut 2, show(matrix), pop", r.steps())
	for _, o := range r.of("rect") {
		assert.InDelta(t, 1, o.Color.A, 1e-6, "drawn opaque in the picture")
	}
	a, _ := r.effects[0].Uniform("uA")
	assert.Equal(t, geom.Vec4{W: 0.5}, a, "and shown at half")
}

func TestAMaskedElementIsCutByItsClipPathInItsPicture(t *testing.T) {
	r := maskedPage(func(s *style.Computed) { s.ClipPath = circleOf(40) })
	assert.Equal(t, "begin, mask 20, rect, rect, pop, end 0, begin, push 10, pop, end 0, cut 2, show(), pop",
		r.steps())
}

func TestASurfaceWithoutPicturesDrawsAMaskedElementUnmasked(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, el(10, 10, 100, 100, func(s *style.Computed) {
		s.Background = blue
		s.MaskImages = []style.Image{{Gradient: &style.Gradient{}}}
	}))))
	assert.Len(t, r.of("rect"), 1)
}

func TestAMaskedElementIsHitInItsBorderBoxWhateverItsMask(t *testing.T) {
	child := el(-20, -20, 140, 140, bg(red))
	masker := el(50, 50, 100, 100, func(s *style.Computed) {
		s.MaskImages = []style.Image{{Gradient: &style.Gradient{}}}
		s.MaskClips = []style.BackgroundBox{style.NoClip}
	}, child)
	tr := tree(el(0, 0, 400, 300, nil, masker))
	assert.Same(t, child, HitTest(tr, geom.V2(55, 55)), "inside, though the mask may be clear")
	assert.NotSame(t, child, HitTest(tr, geom.V2(40, 40)), "outside, though no-clip")
}

func TestMasksStartAStackingContext(t *testing.T) {
	st := style.Initial()
	st.MaskImages = []style.Image{{Source: "a"}}
	assert.True(t, startsContext(&st))
	st.MaskImages = []style.Image{{}}
	assert.False(t, startsContext(&st), "mask-image: none masks nothing")
}
