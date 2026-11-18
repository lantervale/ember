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
		s.MaskImage = style.Image{Source: "arrow"}
		if edit != nil {
			edit(s)
		}
	})
}

func TestAMaskImageIsTheShapeOfTheBackgroundColour(t *testing.T) {
	r := pictures(masked(20, 20, func(s *style.Computed) {
		s.MaskNoRepeat = true
		s.MaskPosition = style.BackgroundPosition{X: style.Percent(50), Y: style.Percent(50)}
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
		s.MaskSize = style.BackgroundSize{Width: style.Px(20), Height: style.Px(12)}
	})
	b.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
	imgs := pictures(b).of("image")
	require.Len(t, imgs, 2, "two across the border box")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 20, H: 12}, imgs[0].Rect)
	assert.Equal(t, geom.Rect{X: 30, Y: 10, W: 20, H: 12}, imgs[1].Rect)
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
