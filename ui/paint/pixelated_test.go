package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPixelatedCopyKeepsItsSourcesView(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	assert.True(t, pixelated(twod.NewTexture(img, twod.Encoded())).Encoded(), "a UI picture stays encoded")
	assert.False(t, pixelated(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 4, 4)))).Encoded(),
		"a game's sprite stays linear")
}

// TestAPixelatedPictureGivesTiesToTheUpperLeftTexel: at a half-pixel
// texel size (a 16-texel picture over 72 pixels), a pixel's centre falls
// on the line between two texels; Chrome gives that pixel the upper left
// one, so a pixelated img and background are sampled a sliver of a texel
// up and left, and a smooth one as it is.
func TestAPixelatedPictureGivesTiesToTheUpperLeftTexel(t *testing.T) {
	size := arrow.Size()
	nudged := geom.Rect{X: -tieBias / size.X, Y: -tieBias / size.Y, W: 1, H: 1}

	img := ctl(dom.Img, 100, 50, "src", "arrow")
	img.Style.ImageRendering = style.RenderPixelated
	imgs := pictures(img).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, nudged, imgs[0].UV, "a pixelated img")

	bg := pictures(withImage(40, 40, func(s *style.Computed, _ *style.BackgroundLayer) {
		s.ImageRendering = style.RenderPixelated
	})).of("image")
	require.Len(t, bg, 1)
	assert.Equal(t, nudged, bg[0].UV, "a pixelated background")

	tiled := pictures(withImage(40, 40, func(s *style.Computed, l *style.BackgroundLayer) {
		s.ImageRendering, l.Repeat = style.RenderPixelated, style.BackgroundRepeat{}
	})).of("image")
	require.NotEmpty(t, tiled)
	for _, tile := range tiled {
		assert.Equal(t, nudged, tile.UV, "each copy of a tiled pixelated background")
	}

	smooth := pictures(ctl(dom.Img, 100, 50, "src", "arrow")).of("image")
	require.Len(t, smooth, 1)
	assert.Equal(t, whole, smooth[0].UV, "a smooth img is sampled as it is")

	assert.Less(t, float32(tieBias), float32(0.01), "far less than a texel: only ties move")
}
