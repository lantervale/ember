package animated_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"

	"github.com/lantervale/ember/ui/internal/animated"
	"github.com/lantervale/ember/ui/internal/animated/animatedtest"
)

func picture(plays int, delays ...time.Duration) *animated.Picture {
	p := &animated.Picture{Plays: plays, Delays: delays}
	for range delays {
		p.Frames = append(p.Frames, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	}
	return p
}

func TestAFrameOfTenMillisecondsOrLessShowsForAHundred(t *testing.T) {
	p := picture(0, 0, 10*time.Millisecond, 11*time.Millisecond, time.Second)
	assert.Equal(t, 100*time.Millisecond, p.Duration(0))
	assert.Equal(t, 100*time.Millisecond, p.Duration(1))
	assert.Equal(t, 11*time.Millisecond, p.Duration(2))
	assert.Equal(t, time.Second, p.Duration(3))
}

func TestAnAnimationPlaysItsTimesThenStaysOnItsLastFrame(t *testing.T) {
	ms := time.Millisecond
	forever := picture(0, 200*ms, 0, 300*ms)
	for _, c := range []struct {
		at   time.Duration
		want int
	}{{0, 0}, {199 * ms, 0}, {200 * ms, 1}, {299 * ms, 1}, {300 * ms, 2}, {599 * ms, 2}, {600 * ms, 0}, {6250 * ms, 1}} {
		i, more := forever.At(c.at)
		assert.Equal(t, c.want, i, "at %v", c.at)
		assert.True(t, more)
	}
	twice := picture(2, 200*ms, 300*ms)
	i, more := twice.At(600 * ms)
	assert.Equal(t, 0, i, "the second play")
	assert.True(t, more)
	i, more = twice.At(999 * ms)
	assert.Equal(t, 1, i)
	assert.True(t, more)
	i, more = twice.At(1000 * ms)
	assert.Equal(t, 1, i, "the last frame stays")
	assert.False(t, more)
}

func frame(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) }

func TestAWebPFrameOutsideItsCanvasOrItsFileIsAnError(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	_, err := animated.DecodeWebP(animatedtest.WebP(8, 8, 0,
		animatedtest.Frame{At: image.Rect(0, 0, 8, 8), Color: red, Delay: 10},
		animatedtest.Frame{At: image.Rect(4, 4, 10, 8), Color: red, Delay: 10}), frame)
	require.Error(t, err)

	good := animatedtest.WebP(8, 8, 0,
		animatedtest.Frame{At: image.Rect(0, 0, 8, 8), Color: red, Delay: 10},
		animatedtest.Frame{At: image.Rect(2, 2, 4, 4), Color: red, Delay: 10})
	for n := range len(good) {
		if pic, cutErr := animated.DecodeWebP(good[:n], frame); cutErr == nil && pic != nil {
			assert.Less(t, len(pic.Frames), 2, "cut at %d", n)
		}
	}
	huge := bytes.Clone(good)
	binary.LittleEndian.PutUint32(huge[24:], 0xffffffff)
	_, err = animated.DecodeWebP(huge, frame)
	require.Error(t, err, "a canvas over the most pixels")
}

func TestAnAnimationOverTheMostPixelsKeepsTheFramesBeforeIt(t *testing.T) {
	c := animated.NewCanvas(4096, 4096, 0)
	shown := 0
	for c.Show(time.Second) {
		shown++
	}
	assert.Equal(t, animated.MaxPixels/(4096*4096), shown)
	assert.Len(t, c.Picture().Frames, shown)
}

func FuzzDecodeWebP(f *testing.F) {
	red := color.NRGBA{255, 0, 0, 255}
	f.Add(animatedtest.WebP(8, 8, 3,
		animatedtest.Frame{At: image.Rect(0, 0, 8, 8), Color: red, Delay: 10},
		animatedtest.Frame{At: image.Rect(2, 2, 6, 8), Color: color.NRGBA{0, 0, 255, 99}, Delay: 0, Dispose: true},
		animatedtest.Frame{At: image.Rect(0, 0, 2, 2), Color: red, NoBlend: true}))
	f.Add(animatedtest.Still(3, 3, red))
	f.Fuzz(func(t *testing.T, data []byte) {
		pic, err := animated.DecodeWebP(data, frame)
		if err != nil || pic == nil {
			return
		}
		require.Len(t, pic.Delays, len(pic.Frames))
		size := pic.Frames[0].Rect
		for _, f := range pic.Frames {
			require.Equal(t, size, f.Rect)
		}
		require.LessOrEqual(t, len(pic.Frames)*size.Dx()*size.Dy(), animated.MaxPixels)
	})
}
