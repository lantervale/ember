package paint //nolint:testpackage // white-box: reads the caches of what paint makes of a texture

import (
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
)

func TestWhatPaintMakesOfAPictureGoesWithIt(t *testing.T) {
	kept := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	shape, crisp := silhouette(kept), pixelated(kept)
	shapesBefore, sharpBefore := shapes.len(), sharp.len()
	func() {
		page := make([]*twod.Texture, 40)
		for i := range page {
			page[i] = twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 64, 64)))
			silhouette(page[i])
			pixelated(page[i])
		}
		assert.Equal(t, shapesBefore+40, shapes.len())
		assert.Equal(t, sharpBefore+40, sharp.len())
		runtime.KeepAlive(page)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for (shapes.len() > shapesBefore || sharp.len() > sharpBefore) && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, shapesBefore, shapes.len(), "a silhouette is kept only while its picture is")
	assert.Equal(t, sharpBefore, sharp.len(), "a pixelated copy is kept only while its picture is")
	assert.Same(t, shape, silhouette(kept), "a picture still held keeps its silhouette")
	assert.Same(t, crisp, pixelated(kept))
}
