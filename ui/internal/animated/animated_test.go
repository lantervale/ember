package animated_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
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

func TestAPlayShowsEachFrameForItsDuration(t *testing.T) {
	ms := time.Millisecond
	p := picture(0, 200*ms, 0, 300*ms)
	assert.Equal(t, 600*ms, p.Whole())
	for _, c := range []struct {
		at   time.Duration
		want int
	}{{0, 0}, {199 * ms, 0}, {200 * ms, 1}, {299 * ms, 1}, {300 * ms, 2}, {599 * ms, 2}} {
		assert.Equal(t, c.want, p.Within(c.at), "at %v", c.at)
	}
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

// grey is a step drawing a w×w square of grey v at the canvas's
// top-left over what is there, cleared once shown when cleared.
func grey(w int, v uint8, cleared bool) animated.Step {
	return func(img *image.NRGBA) (func(), error) {
		at := image.Rect(0, 0, w, w)
		draw.Draw(img, at, image.NewUniform(color.NRGBA{v, v, v, 255}), image.Point{}, draw.Src)
		if cleared {
			return func() { animated.Clear(img, at) }, nil
		}
		return nil, nil
	}
}

func TestTheFramesPastTheMostPixelsAreDrawnAsTheyShow(t *testing.T) {
	const side = 4096
	held := animated.MaxPixels / (side * side)
	c := animated.NewCanvas(side, side, 0)
	for i := range held + 3 {
		require.NoError(t, c.Add(time.Second, grey(side-i, uint8(i+1), i == held+1)))
	}
	p := c.Picture()
	assert.Len(t, p.Frames, held, "held within the most pixels")
	assert.Equal(t, held+3, p.Len())
	value := func(f *image.NRGBA, x int) uint8 { return f.NRGBAAt(x, 0).R }
	last := p.Frame(held + 2)
	assert.Equal(t, uint8(held+3), value(last, 0))
	assert.Equal(t, uint8(0), value(last, side-held-2), "the frame before cleared")
	assert.Equal(t, uint8(held+1), value(last, side-held-1), "over what the frames before left")
	assert.Equal(t, uint8(1), value(last, side-1))
	assert.Same(t, p.Frames[0], p.Frame(0))
	again := p.Frame(held + 1)
	assert.Equal(t, uint8(held+2), value(again, 0), "drawn again from the frames held")
	assert.Equal(t, uint8(held+1), value(again, side-held-1))
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
		require.LessOrEqual(t, len(pic.Frames), pic.Len())
		size := pic.Frames[0].Rect
		for _, f := range pic.Frames {
			require.Equal(t, size, f.Rect)
		}
		require.LessOrEqual(t, len(pic.Frames)*size.Dx()*size.Dy(), animated.MaxPixels)
	})
}
