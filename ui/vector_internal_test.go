package ui

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
)

// shapesFile is a picture in the tests' own vector format: "SHAPES",
// then w=, h= and r= for what it asks for, no= for what it doesn't draw,
// most= for the longest side it draws, gated for one whose drawing waits
// to be let go, blank for one that can't be drawn; anything else can't be
// read.
func shapesFile(fields string) []byte { return []byte("SHAPES " + fields) }

// shapesPicture is a parsed shapesFile that notes the sizes it was drawn
// at.
type shapesPicture struct {
	w, h, r float64
	no      []string
	most    int
	blank   bool
	gate    chan struct{}
	mu      sync.Mutex
	drawn   [][2]int
	scale   float64
}

func (p *shapesPicture) Width() (float64, bool)  { return p.w, p.w > 0 }
func (p *shapesPicture) Height() (float64, bool) { return p.h, p.h > 0 }
func (p *shapesPicture) Ratio() (float64, bool)  { return p.r, p.r > 0 }
func (p *shapesPicture) Unsupported() []string   { return p.no }

func (p *shapesPicture) Rasterise(w, h int, scale float64) (*image.RGBA, error) {
	if p.gate != nil {
		<-p.gate
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drawn, p.scale = append(p.drawn, [2]int{w, h}), scale
	if p.blank {
		return nil, errors.New("shapes: nothing to draw")
	}
	if p.most > 0 && max(w, h) > p.most {
		k := float64(p.most) / float64(max(w, h))
		w, h = max(int(float64(w)*k), 1), max(int(float64(h)*k), 1)
	}
	return image.NewRGBA(image.Rect(0, 0, w, h)), nil
}

func (p *shapesPicture) sizes() [][2]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([][2]int(nil), p.drawn...)
}

//nolint:gochecknoglobals // the pictures the tests' format read, by their bytes
var (
	shapesMu   sync.Mutex
	shapesRead = map[string]*shapesPicture{}
)

// readShapes is the picture parsed from data, the last one for bytes
// read twice.
func readShapes(data []byte) *shapesPicture {
	shapesMu.Lock()
	defer shapesMu.Unlock()
	return shapesRead[string(data)]
}

//nolint:gochecknoinits // registers the tests' format as a package registers its own, on import
func init() {
	RegisterVectorFormat("image/x-shapes",
		func(name string, data []byte) bool {
			return strings.HasSuffix(name, ".shapes") || strings.HasPrefix(string(data), "SHAPES")
		},
		func(data []byte) (VectorImage, error) {
			p := &shapesPicture{}
			for _, f := range strings.Fields(strings.TrimPrefix(string(data), "SHAPES")) {
				k, v, _ := strings.Cut(f, "=")
				switch k {
				case "w":
					_, _ = fmt.Sscan(v, &p.w)
				case "h":
					_, _ = fmt.Sscan(v, &p.h)
				case "r":
					_, _ = fmt.Sscan(v, &p.r)
				case "most":
					_, _ = fmt.Sscan(v, &p.most)
				case "no":
					p.no = strings.Split(v, ",")
				case "blank":
					p.blank = true
				case "gated":
					p.gate = make(chan struct{})
				default:
					return nil, fmt.Errorf("shapes: unknown %q", f)
				}
			}
			shapesMu.Lock()
			defer shapesMu.Unlock()
			shapesRead[string(data)] = p
			return p, nil
		})
}

func shapesDoc(css string, files map[string][]byte, opts ...DocumentOption) *Document {
	fsys := fstest.MapFS{"page.css": {Data: []byte(css)}}
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: data}
	}
	return NewDocument(append([]DocumentOption{Stylesheet(fsys, "page.css")}, opts...)...)
}

// shown runs d in a window of 400×300 points and returns how to run more
// frames.
func shown(d *Document) (*platform.Window, func(int)) {
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(3)
	return win, w.RunFrames
}

func TestARegisteredFormatIsRecognisedByNameAndByItsBytes(t *testing.T) {
	d := shapesDoc("", map[string][]byte{
		"named.shapes": []byte("w=40 h=20 r=2"),
		"sniffed.bin":  shapesFile("w=30 h=30 r=1"),
		"notes.txt":    []byte("SHAPE of things"),
	})
	size, ok := d.ImageSize("named.shapes")
	assert.True(t, ok, "by its name")
	assert.Equal(t, geom.V2(40, 20), size)
	size, ok = d.ImageSize("sniffed.bin")
	assert.True(t, ok, "by how its bytes start")
	assert.Equal(t, geom.V2(30, 30), size)
	assert.Empty(t, d.Warnings())

	_, ok = d.ImageSize("notes.txt")
	assert.False(t, ok, "any other file is a picture of pixels or none")
	assert.Nil(t, d.vector("notes.txt"))
	require.Len(t, d.Warnings(), 1)
	assert.Equal(t, "image/x-shapes", mediaType("named.shapes", nil), "export names it by its media type")
	assert.Equal(t, "image/png", mediaType("door.png", picture(t, 2, 2)))
}

func TestAnSVGIsNoPictureUntilItsFormatIsImported(t *testing.T) {
	const file = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16"/></svg>`
	d := shapesDoc("", map[string][]byte{"star.svg": []byte(file)})
	icon := Img(Attr("src", "star.svg"))
	d.Body().Append(icon)
	d.Layout(geom.V2(200, 100))
	assert.Nil(t, d.vector("star.svg"))
	assert.Nil(t, d.image("star.svg"))
	assert.Equal(t, geom.V2(16, 16), geom.V2(icon.Box().W, icon.Box().H), "a broken image's icon, as before")
	require.Len(t, d.Warnings(), 1)
	assert.Contains(t, d.Warnings()[0].Error(), "unknown format")
}

func TestWhatAVectorPictureAsksForIsWhatLayoutAndPaintSizeItBy(t *testing.T) {
	d := shapesDoc("", map[string][]byte{
		"both.shapes": shapesFile("w=40 h=20 r=2"), "shape.shapes": shapesFile("r=2"), "none.shapes": shapesFile(""),
	})
	for name, want := range map[string]box.Intrinsic{
		"both.shapes": {Width: 40, Height: 20, Ratio: 2}, "shape.shapes": {Ratio: 2}, "none.shapes": {},
	} {
		in, ok := d.ImageIntrinsic(name)
		assert.True(t, ok, name)
		assert.Equal(t, want, in, name)
		assert.Equal(t, want, d.vector(name).Intrinsic(), name)
	}
	size, _ := d.ImageSize("none.shapes")
	assert.Equal(t, geom.V2(300, 150), size, "CSS's size for a picture asking for none")
	size, _ = d.ImageSize("shape.shapes")
	assert.Equal(t, geom.V2(300, 150), size)

	both, shape, none := Img(Attr("src", "both.shapes")), Img(Attr("src", "shape.shapes")), Img(Attr("src", "none.shapes"))
	d.Body().Append(Div(Style("width: 200px"), both, shape, none))
	d.Layout(geom.V2(400, 600))
	assert.Equal(t, geom.V2(40, 20), geom.V2(both.Box().W, both.Box().H))
	assert.Equal(t, geom.V2(200, 100), geom.V2(shape.Box().W, shape.Box().H), "proportions alone: its block's width")
	assert.Equal(t, geom.V2(300, 150), geom.V2(none.Box().W, none.Box().H))
}

func TestAVectorPictureIsDrawnAtTheSizeItShowsTimesThePixelRatio(t *testing.T) {
	icon := shapesFile("w=40 h=20 r=2")
	d := shapesDoc(`div { width: 100px; height: 60px; background: url(bg.shapes) no-repeat; background-size: 50px auto; }`,
		map[string][]byte{"icon.shapes": icon, "bg.shapes": shapesFile("r=2")}, Zoom(2))
	d.Body().Append(Img(Attr("src", "icon.shapes"), Style("width: 30px")), Div())
	shown(d)
	assert.Equal(t, [][2]int{{60, 30}}, readShapes(icon).sizes(), "30×15 points, each two pixels")
	assert.InDelta(t, 2, readShapes(icon).scale, 1e-9)
	assert.Equal(t, [][2]int{{100, 50}}, readShapes(shapesFile("r=2")).sizes(), "the background's tile")
	assert.Empty(t, d.Warnings())
}

func TestAVectorPictureIsDrawnAgainWhenItsSizeChangesAndOnceItSettles(t *testing.T) {
	icon := shapesFile("w=40 h=20 r=2 most=4096")
	d := shapesDoc("", map[string][]byte{"icon.shapes": icon})
	img := Img(Attr("src", "icon.shapes"), Style("width: 80px"))
	d.Body().Append(img)
	_, frames := shown(d)
	pic := readShapes(icon)
	assert.Equal(t, [][2]int{{80, 40}}, pic.sizes())

	for _, w := range []int{90, 100, 110, 120} {
		img.SetAttr("style", fmt.Sprintf("width: %dpx", w))
		frames(1)
	}
	assert.Equal(t, [][2]int{{80, 40}}, pic.sizes(), "a size changing every frame stretches what is drawn")
	frames(1)
	assert.Equal(t, [][2]int{{80, 40}, {120, 60}}, pic.sizes(), "drawn once it holds for a frame")
	frames(5)
	assert.Len(t, pic.sizes(), 2, "and kept")

	img.SetAttr("style", "width: 80px")
	frames(3)
	assert.Len(t, pic.sizes(), 2, "a size still kept is not drawn again")
}

func TestAVectorPictureKeepsAFewSizesAndStretchesTheNearestPastThem(t *testing.T) {
	icon := shapesFile("w=10 h=10 r=1")
	d := shapesDoc("", map[string][]byte{"icon.shapes": icon})
	d.Layout(geom.V2(100, 100))
	v, ok := d.vectors["icon.shapes"]
	if !ok {
		d.vector("icon.shapes")
		v = d.vectors["icon.shapes"]
	}
	require.NotNil(t, v)
	pic := readShapes(icon)

	d.painted = 1
	var first *twod.Texture
	for i := range vectorRasters {
		tex := v.Texture(10+i, 10+i, 1)
		require.NotNil(t, tex)
		assert.Equal(t, float32(10+i), tex.Size().X)
		if i == 0 {
			first = tex
		}
	}
	assert.Len(t, v.rasters, vectorRasters)
	over := v.Texture(100, 100, 1)
	assert.Len(t, pic.sizes(), vectorRasters, "every size kept is showing: nothing more is drawn")
	assert.Equal(t, float32(10+vectorRasters-1), over.Size().X, "the nearest size is stretched")
	assert.False(t, d.unsettled, "and no paint is asked for")

	d.painted = 2
	assert.Same(t, first, v.Texture(10, 10, 1), "a size kept is the same texture")
	fresh := v.Texture(100, 100, 1)
	assert.Equal(t, float32(100), fresh.Size().X, "a size no longer shown makes room")
	assert.Len(t, v.rasters, vectorRasters)
	for _, r := range v.rasters {
		assert.NotEqual(t, 11, r.w, "the one unseen the longest went")
	}

	for i := range 200 {
		d.painted++
		v.Texture(20+i, 20+i, 1)
	}
	assert.LessOrEqual(t, len(v.rasters), vectorRasters, "a size animating for long keeps no more")
}

func TestAVectorPicturePastTheLargestTextureIsDrawnSmallerAndStretched(t *testing.T) {
	at := vectorFit(40000, 100, 2)
	assert.Equal(t, drawnAt{vectorSide, 20, 2 * vectorSide / 40000.0}, at,
		"the longest side a texture takes, its shape kept, its scale as much smaller")
	at = vectorFit(8000, 8000, 1)
	assert.LessOrEqual(t, at.w*at.h, vectorPixels)
	assert.Equal(t, at.w, at.h)
	assert.Equal(t, drawnAt{1, 5, 1}, vectorFit(0, 5, 0), "never under a pixel")

	small := shapesFile("w=10 h=10 r=1 most=64")
	d := shapesDoc("", map[string][]byte{"small.shapes": small})
	d.Body().Append(Img(Attr("src", "small.shapes"), Style("width: 200px; height: 200px")))
	shown(d)
	assert.Equal(t, [][2]int{{200, 200}}, readShapes(small).sizes())
	v := d.vectors["small.shapes"]
	require.Len(t, v.rasters, 1)
	assert.Equal(t, geom.V2(64, 64), v.rasters[0].tex.Size(), "what the format drew, kept for the size asked")
	assert.Empty(t, d.Warnings())
}

func TestAVectorPictureThatCannotBeReadOrDrawnIsABrokenImageWithAWarning(t *testing.T) {
	d := shapesDoc("", map[string][]byte{"bad.shapes": shapesFile("nonsense"), "blank.shapes": shapesFile("blank")})
	bad, blank := Img(Attr("src", "bad.shapes")), Img(Attr("src", "blank.shapes"), Style("width: 20px; height: 20px"))
	d.Body().Append(bad, blank)
	_, frames := shown(d)
	assert.Nil(t, d.vector("bad.shapes"))
	assert.Equal(t, geom.V2(16, 16), geom.V2(bad.Box().W, bad.Box().H), "a broken image's icon")
	frames(5)
	assert.Len(t, readShapes(shapesFile("blank")).sizes(), 1, "one that can't be drawn is tried once")
	require.Len(t, d.Warnings(), 2)
	texts := d.Warnings()[0].Error() + "\n" + d.Warnings()[1].Error()
	assert.Contains(t, texts, `ui: image bad.shapes: shapes: unknown "nonsense"`)
	assert.Contains(t, texts, "ui: image blank.shapes: shapes: nothing to draw")
}

func TestWhatAVectorPictureDoesNotDrawIsAWarningOncePerPicture(t *testing.T) {
	d := shapesDoc(`body { background: url(label.shapes); }`,
		map[string][]byte{"label.shapes": shapesFile("w=20 h=20 r=1 no=filter,text")})
	d.Body().Append(Img(Attr("src", "label.shapes")), Img(Attr("src", "label.shapes")))
	_, frames := shown(d)
	frames(3)
	require.Len(t, d.Warnings(), 1)
	assert.EqualError(t, d.Warnings()[0], "ui: image label.shapes: image/x-shapes is not drawn: filter, text")
}

func TestAFetchedVectorPictureIsReadAndDrawnOffTheFrame(t *testing.T) {
	icon := shapesFile("w=40 h=20 r=2 no=mask gated")
	a := &asked{pictures: map[string][]byte{"https://art.test/icon": icon}}
	d := NewDocument(FetchImages(a.fetch))
	img := Img(Attr("src", "https://art.test/icon"))
	d.Body().Append(img)
	_, frames := shown(d)
	drawing := func() bool {
		v := d.vectors["https://art.test/icon"]
		return v != nil && len(v.drawing) > 0
	}
	for !drawing() {
		time.Sleep(time.Millisecond)
		frames(1)
	}
	frames(3)
	assert.Equal(t, geom.V2(40, 20), geom.V2(img.Box().W, img.Box().H), "laid out again once it is read")
	v := d.vectors["https://art.test/icon"]
	require.NotNil(t, v)
	assert.Empty(t, v.rasters, "the frames that ask for it don't wait for it")

	close(readShapes(icon).gate)
	d.fetching.Wait()
	assert.Empty(t, v.rasters, "drawn, it waits for the next layout")
	frames(1)
	require.Len(t, v.rasters, 1)
	assert.True(t, d.replay.recorded)
	frames(3)
	d.fetching.Wait()
	assert.Equal(t, [][2]int{{40, 20}}, readShapes(icon).sizes(), "once")
	require.Len(t, d.Warnings(), 1)
	assert.Contains(t, d.Warnings()[0].Error(), "https://art.test/icon: image/x-shapes is not drawn: mask")
	assert.Equal(t, []string{"https://art.test/icon"}, a.urls)
}
