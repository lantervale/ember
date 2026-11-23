package web_test

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/animated"
	"github.com/lantervale/ember/ui/internal/animated/animatedtest"
)

//nolint:gochecknoglobals // the palette of the GIFs these tests write
var gifPalette = color.Palette{
	color.RGBA{}, color.RGBA{200, 40, 40, 255}, color.RGBA{40, 160, 60, 255}, color.RGBA{40, 80, 200, 255},
}

// gifFrame is a frame of r in colour index c, its every third pixel
// clear, so what is under it shows through.
func gifFrame(r image.Rectangle, c uint8) *image.Paletted {
	p := image.NewPaletted(r, gifPalette)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if (x+y)%3 != 0 {
				p.SetColorIndex(x, y, c)
			}
		}
	}
	return p
}

func encodeGIF(t *testing.T, loops int, disposal []byte, delays []int, frames ...*image.Paletted) []byte {
	t.Helper()
	return encodeGIFOf(t, 8, loops, disposal, delays, frames...)
}

func encodeGIFOf(t *testing.T, size, loops int, disposal []byte, delays []int, frames ...*image.Paletted) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, gif.EncodeAll(&b, &gif.GIF{
		Image: frames, Delay: delays, Disposal: disposal, LoopCount: loops,
		Config: image.Config{ColorModel: gifPalette, Width: size, Height: size},
	}))
	return b.Bytes()
}

// pastGIF is a GIF of an 8×8 picture its frames reach past: written on
// a canvas of 16×16, its size then said to be 8×8.
func pastGIF(t *testing.T, loops int, disposal []byte, delays []int, frames ...*image.Paletted) []byte {
	t.Helper()
	data := encodeGIFOf(t, 16, loops, disposal, delays, frames...)
	data[6], data[7], data[8], data[9] = 8, 0, 8, 0
	return data
}

// animatedGIFs are GIFs of every disposal and loop count, by name.
func animatedGIFs(t *testing.T) map[string][]byte {
	t.Helper()
	a, b := gifFrame(image.Rect(0, 0, 8, 8), 1), gifFrame(image.Rect(2, 1, 7, 6), 2)
	c := gifFrame(image.Rect(4, 3, 8, 8), 3)
	none, back, prev := byte(gif.DisposalNone), byte(gif.DisposalBackground), byte(gif.DisposalPrevious)
	delays := []int{5, 0, 1, 2}
	return map[string][]byte{
		"left":                 encodeGIF(t, 0, []byte{none, none, none}, delays[:3], a, b, c),
		"cleared":              encodeGIF(t, 0, []byte{none, back, none}, delays[:3], a, b, c),
		"put-back":             encodeGIF(t, 0, []byte{none, prev, none}, delays[:3], a, b, c),
		"first-put-back":       encodeGIF(t, 0, []byte{prev, back, prev, none}, delays, a, b, c, a),
		"first-cleared":        encodeGIF(t, 0, []byte{back, 0, back}, delays[:3], b, c, a),
		"once":                 encodeGIF(t, -1, nil, delays[:2], a, b),
		"loop-count-1":         encodeGIF(t, 1, nil, delays[:2], a, b),
		"loop-count-3":         encodeGIF(t, 3, nil, delays[:2], a, b),
		"whole-frames":         encodeGIF(t, 0, nil, delays[:2], a, gifFrame(image.Rect(0, 0, 8, 8), 2)),
		"unspecified-disposal": encodeGIF(t, 0, []byte{0, 0}, delays[:2], b, c),
		"past-the-edge":        pastGIF(t, 0, nil, delays[:2], a, gifFrame(image.Rect(4, 3, 11, 10), 2)),
		"first-past-the-edge": pastGIF(t, 0, []byte{back, none}, delays[:2],
			gifFrame(image.Rect(0, 0, 12, 10), 1), b),
		"first-moved-past-the-edge": pastGIF(t, 0, nil, delays[:2], gifFrame(image.Rect(2, 3, 12, 9), 1), b),
		"one-frame-past-the-edge":   pastGIF(t, -1, nil, delays[:1], gifFrame(image.Rect(2, 3, 12, 9), 3)),
		"outside":                   pastGIF(t, 0, nil, delays[:3], a, gifFrame(image.Rect(9, 9, 12, 12), 2), c),
	}
}

func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{r, g, b, a} }

// animatedWebPs are WebPs blending and not, disposing and not, at
// places and loop counts, by name.
func animatedWebPs() map[string][]byte {
	type f = animatedtest.Frame
	whole := image.Rect(0, 0, 8, 8)
	red, half, quarter := rgba(200, 40, 40, 255), rgba(40, 160, 60, 128), rgba(40, 80, 200, 64)
	return map[string][]byte{
		"blended": animatedtest.WebP(8, 8, 0,
			f{At: whole, Color: red, Delay: 50}, f{At: image.Rect(2, 2, 7, 6), Color: half, Delay: 0},
			f{At: image.Rect(4, 0, 8, 8), Color: quarter, Delay: 10}),
		"replaced": animatedtest.WebP(8, 8, 0,
			f{At: whole, Color: red, Delay: 50}, f{At: image.Rect(2, 2, 7, 6), Color: half, Delay: 20, NoBlend: true},
			f{At: image.Rect(4, 0, 8, 8), Color: quarter, Delay: 11}),
		"disposed": animatedtest.WebP(8, 8, 0,
			f{At: image.Rect(0, 0, 6, 6), Color: red, Delay: 30, Dispose: true},
			f{At: image.Rect(2, 2, 8, 8), Color: half, Delay: 30, Dispose: true},
			f{At: image.Rect(0, 4, 4, 8), Color: quarter, Delay: 30}),
		"clear-over": animatedtest.WebP(8, 8, 0,
			f{At: whole, Color: half, Delay: 30}, f{At: image.Rect(2, 2, 6, 6), Color: rgba(0, 0, 0, 0), Delay: 30}),
		"plays-once": animatedtest.WebP(8, 8, 1, f{At: whole, Color: red, Delay: 30}, f{At: whole, Color: half, Delay: 30}),
		"plays-4":    animatedtest.WebP(8, 8, 4, f{At: whole, Color: red, Delay: 30}, f{At: whole, Color: half, Delay: 30}),
	}
}

func TestAnAnimatedGIFDrawsEachFrameOverWhatItsDisposalLeft(t *testing.T) {
	pic, err := animated.Decode(animatedGIFs(t)["put-back"])
	require.NoError(t, err)
	require.Len(t, pic.Frames, 3)
	assert.Equal(t, uint8(0), pic.Frames[0].NRGBAAt(0, 0).A, "a clear pixel of the first frame shows nothing")
	assert.Equal(t, color.NRGBA{200, 40, 40, 255}, pic.Frames[1].NRGBAAt(1, 0), "the first frame under the second")
	assert.Equal(t, color.NRGBA{200, 40, 40, 255}, pic.Frames[2].NRGBAAt(3, 2),
		"the second put back to what was under it")
	assert.Equal(t, []time.Duration{50 * time.Millisecond, 0, 10 * time.Millisecond}, pic.Delays)
	assert.Equal(t, 0, pic.Plays)

	once, err := animated.Decode(animatedGIFs(t)["once"])
	require.NoError(t, err)
	assert.Equal(t, 1, once.Plays, "no loop count")
	three, err := animated.Decode(animatedGIFs(t)["loop-count-3"])
	require.NoError(t, err)
	assert.Equal(t, 4, three.Plays, "a loop count of 3 plays four times")
}

func TestAStillGIFOrWebPIsNoAnimation(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, gif.Encode(&b, gifFrame(image.Rect(0, 0, 4, 4), 1), nil))
	pic, err := animated.Decode(b.Bytes())
	require.NoError(t, err)
	assert.Nil(t, pic)
	pic, err = animated.Decode(animatedtest.Still(4, 4, rgba(1, 2, 3, 255)))
	require.NoError(t, err)
	assert.Nil(t, pic)
}

func TestAnAnimatedWebPBlendsDisposesAndLoopsAsItSays(t *testing.T) {
	pic, err := animated.Decode(animatedWebPs()["disposed"])
	require.NoError(t, err)
	require.Len(t, pic.Frames, 3)
	assert.Equal(t, uint8(0), pic.Frames[1].NRGBAAt(0, 0).A, "the first frame cleared")
	assert.Equal(t, uint8(0), pic.Frames[2].NRGBAAt(7, 7).A, "the second cleared")
	assert.Equal(t, 0, pic.Plays)

	blended, err := animated.Decode(animatedWebPs()["blended"])
	require.NoError(t, err)
	got := blended.Frames[1].NRGBAAt(3, 3)
	assert.Equal(t, color.NRGBA{120, 100, 50, 255}, got, "half green over red")
	replaced, err := animated.Decode(animatedWebPs()["replaced"])
	require.NoError(t, err)
	assert.Equal(t, rgba(40, 160, 60, 128), replaced.Frames[1].NRGBAAt(3, 3))

	once, err := animated.Decode(animatedWebPs()["plays-once"])
	require.NoError(t, err)
	assert.Equal(t, 1, once.Plays, "a loop count of 1 plays once")
}

func TestAGIFFramePastThePictureIsCutAndAFirstOneGrowsIt(t *testing.T) {
	gifs := animatedGIFs(t)
	cut, err := animated.Decode(gifs["past-the-edge"])
	require.NoError(t, err)
	require.Len(t, cut.Frames, 2)
	assert.Equal(t, image.Rect(0, 0, 8, 8), cut.Frames[1].Rect)
	assert.Equal(t, color.NRGBA{40, 160, 60, 255}, cut.Frames[1].NRGBAAt(7, 7), "drawn up to the edge")

	grown, err := animated.Decode(gifs["first-moved-past-the-edge"])
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 12, 9), grown.Frames[0].Rect, "as far as the first frame reaches")

	still, err := animated.Decode(gifs["one-frame-past-the-edge"])
	require.NoError(t, err)
	require.NotNil(t, still, "the image package refuses it")
	assert.False(t, still.Moves())
	assert.Equal(t, image.Rect(0, 0, 12, 9), still.Frames[0].Rect)
}
