package gpu //nolint:testpackage // tests the unexported cache of what the GPU holds

import (
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// swept sweeps k until it has dropped n textures, for a few seconds at
// most: a picture nothing holds goes at a collection, some time after.
func swept(k *kept[render.Texture], n int) int {
	dropped := 0
	deadline := time.Now().Add(5 * time.Second)
	for dropped < n && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		k.sweep(func(gpuTexture) { dropped++ })
	}
	return dropped
}

func TestTheGPUHoldsAPictureOnlyWhileItLives(t *testing.T) {
	k := newKept[render.Texture]()
	held := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	k.set(&held.Texture, gpuTexture{version: 1})
	func() {
		for range 50 {
			tex := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 64, 64)))
			k.set(&tex.Texture, gpuTexture{version: 1})
			k.set(&tex.Texture, gpuTexture{version: 2})
			got, ok := k.get(&tex.Texture)
			require.True(t, ok)
			require.Equal(t, uint64(2), got.version)
		}
	}()
	require.Equal(t, 51, k.len())

	assert.Equal(t, 50, swept(k, 50), "each picture nothing holds any more is dropped once")
	assert.Equal(t, 1, k.len())
	got, ok := k.get(&held.Texture)
	assert.True(t, ok, "a picture still held stays")
	assert.Equal(t, uint64(1), got.version)
	runtime.KeepAlive(held)
}

func TestAPictureTheGPUNeverHeldIsNotThere(t *testing.T) {
	k := newKept[render.Texture]()
	_, ok := k.get(&render.Texture{})
	assert.False(t, ok)
	assert.False(t, k.sweep(func(gpuTexture) { t.Error("nothing to drop") }))
}
