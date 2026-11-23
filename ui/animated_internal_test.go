package ui

import (
	"image"
	"image/color"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/animated"
)

// flipbook is a test format of animations: "FLIP" then a frame a byte,
// each a 4×2 picture of that grey shown for 100 ms, played twice.
//
//nolint:gochecknoinits // registers the test format once, as ui/web registers its own
func init() {
	animated.Register("FLIP", func(data []byte) (*animated.Picture, error) {
		c := animated.NewCanvas(4, 2, 2)
		for _, v := range data[4:] {
			if err := c.Add(100*time.Millisecond, func(img *image.NRGBA) (func(), error) {
				for i := 0; i < len(img.Pix); i += 4 {
					img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
				}
				return animated.Keep, nil
			}); err != nil {
				return nil, err
			}
		}
		return c.Picture(), nil
	})
}

func grey(img image.Image) uint8 {
	g, _ := color.GrayModel.Convert(img.At(0, 0)).(color.Gray)
	return g.Y
}

func TestAnAnimatedPictureMovesWithTheClockForEveryElementShowingIt(t *testing.T) {
	files := fstest.MapFS{"walk.flip": {Data: []byte("FLIP\x10\x20\x30")}}
	d := NewDocument(Images(files))
	a, b := Img(Attr("src", "walk.flip")), Div(Style("background-image: url(walk.flip); width: 8px; height: 8px"))
	d.Body().Append(a, b)
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, geom.V2(4, 2), geom.V2(a.Box().W, a.Box().H), "sized by its frames")
	tex := d.image("walk.flip")
	require.NotNil(t, tex)
	assert.Equal(t, uint8(0x10), grey(tex.Image))

	at := func(s float32) uint8 {
		d.clock = s
		d.replay.repaint = false
		d.animate()
		clear(d.drawnPictures)
		d.paintImage("walk.flip")
		return grey(tex.Image)
	}
	assert.Equal(t, uint8(0x10), at(5), "it begins on the first frame drawn")
	version := tex.Version()
	assert.Equal(t, uint8(0x20), at(5.15))
	assert.True(t, d.replay.repaint, "a new frame repaints")
	assert.Greater(t, tex.Version(), version, "and uploads again")
	assert.Equal(t, uint8(0x30), at(5.25))
	assert.Equal(t, uint8(0x10), at(5.31), "the second play")
	assert.Equal(t, uint8(0x30), at(5.6), "played twice, its last frame stays")
	assert.Equal(t, uint8(0x30), at(9))
	assert.False(t, d.replay.repaint)
	assert.Same(t, tex, d.image("walk.flip"), "one texture for the img and the background")
}

func TestAFetchedAnimationIsDecodedOffTheFrameAndPlays(t *testing.T) {
	a := &asked{pictures: map[string][]byte{"https://art.test/walk.flip": []byte("FLIP\x10\x20")}}
	d := NewDocument(FetchImages(a.fetch))
	img := Img(Attr("src", "https://art.test/walk.flip"))
	d.Body().Append(img)
	d.Layout(geom.V2(200, 100))
	d.fetching.Wait()
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, geom.V2(4, 2), geom.V2(img.Box().W, img.Box().H))
	d.clock = 0
	d.paintImage("https://art.test/walk.flip")
	d.animate()
	d.clock = 0.15
	d.animate()
	assert.Equal(t, uint8(0x20), grey(d.image("https://art.test/walk.flip").Image))
	assert.Empty(t, d.Warnings())
}

// shows has the Document's clock at s, and walk.flip drawn there when
// drawn, as a paint draws it or culls it out of view; it is the frame of
// walk.flip shown.
func shows(d *Document, s float32, drawn bool) uint8 {
	const url = "walk.flip"
	d.clock = s
	d.animate()
	clear(d.drawnPictures)
	if drawn {
		d.paintImage(url)
	}
	return grey(d.image(url).Image)
}

func TestAnAnimationBeginsWhenFirstDrawnAndMovesOnlyWhileDrawn(t *testing.T) {
	files := fstest.MapFS{"walk.flip": {Data: []byte("FLIP\x10\x20\x30\x40")}}
	d := NewDocument(Images(files))
	d.Body().Append(Img(Attr("src", "walk.flip")))
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, uint8(0x10), shows(d, 3, false), "out of view since it landed")
	assert.Equal(t, uint8(0x10), shows(d, 4, true), "it begins when first drawn")
	assert.Equal(t, uint8(0x20), shows(d, 4.15, true))
	assert.Equal(t, uint8(0x20), shows(d, 4.2, false), "out of view")
	assert.Equal(t, uint8(0x20), shows(d, 4.32, false), "it stays where it left")
	assert.Equal(t, uint8(0x20), shows(d, 4.33, true), "drawn again, once more as it left")
	assert.Equal(t, uint8(0x40), shows(d, 4.34, true), "then in step within its play")
	assert.Equal(t, uint8(0x40), shows(d, 4.35, false))
	assert.Equal(t, uint8(0x40), shows(d, 4.55, true))
	assert.Equal(t, uint8(0x10), shows(d, 4.56, true), "back past its play, the next begins")
	assert.Equal(t, uint8(0x10), shows(d, 4.6, true))
	assert.Equal(t, uint8(0x20), shows(d, 4.67, true))
	assert.Equal(t, uint8(0x40), shows(d, 4.99, true))
	assert.Equal(t, uint8(0x40), shows(d, 5.5, true), "played twice, its last frame stays")
}
