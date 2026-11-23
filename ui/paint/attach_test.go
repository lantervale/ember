package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestTheCanvasPictureIsPlacedInTheRootsBox(t *testing.T) {
	root := el(24, 14, 300, 200, func(s *style.Computed) {
		stacked(s, style.BackgroundLayer{Image: style.Image{Source: "arrow"}, Repeat: style.NoRepeat,
			Position: style.TopLeft})
	})
	root.Border = box.Edges{Top: 2, Left: 3}
	r := newRecorder()
	Paint(r, tree(root), Viewport(geom.Rect{W: 400, H: 300}),
		Images(func(string) *twod.Texture { return arrow }))
	imgs := r.of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.V2(27, 16), geom.V2(imgs[0].Rect.X, imgs[0].Rect.Y), "in html's padding box, not the window")
}

func TestAFixedPictureUnderATransformIsPlacedInItsBox(t *testing.T) {
	b := withImage(100, 50, func(_ *style.Computed, l *style.BackgroundLayer) { l.Attachment = style.AttachFixed })
	turned := el(20, 30, 200, 100, func(s *style.Computed) {
		s.Transform = []style.TransformOp{{Kind: style.Translate}}
	}, b)
	imgs := pictures(turned).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.V2(30, 40), geom.V2(imgs[0].Rect.X, imgs[0].Rect.Y), "as if it scrolled")

	imgs = pictures(withImage(100, 50, func(_ *style.Computed, l *style.BackgroundLayer) {
		l.Attachment = style.AttachFixed
	})).of("image")
	require.Len(t, imgs, 1)
	assert.Equal(t, geom.V2(0, 0), geom.V2(imgs[0].Rect.X, imgs[0].Rect.Y), "in the viewport without one")
}
