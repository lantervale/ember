package ui

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
)

// sized are pictures named by their width, each half as tall.
func sized(t testing.TB, widths ...int) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	for _, w := range widths {
		files[pictureName(w)] = &fstest.MapFile{Data: picture(t, w, w/2)}
	}
	return files
}

func pictureName(w int) string { return fmt.Sprintf("p%d.png", w) }

func shownAs(t testing.TB, n *Node) (string, geom.Vec2) {
	t.Helper()
	src, _ := PictureOf(n)
	return src, geom.V2(n.Box().W, n.Box().H)
}

func TestAnImgShowsTheCandidateOfItsSrcsetForTheScreensDensity(t *testing.T) {
	img := Img(Attr("srcset", "p100.png 1x, p200.png 2x, p400.png 4x"))
	d := NewDocument(Images(sized(t, 100, 200, 400)))
	d.Body().Append(img)

	d.Layout(geom.V2(800, 600))
	src, size := shownAs(t, img)
	assert.Equal(t, "p100.png", src)
	assert.Equal(t, geom.V2(100, 50), size)

	d.fonts.SetScale(2)
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p200.png", src, "chosen again on a denser screen")
	assert.Equal(t, geom.V2(100, 50), size, "a 2x picture shows at half its pixels")

	d.fonts.SetScale(3)
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p400.png", src)
	assert.Equal(t, geom.V2(100, 50), size)
}

func TestAPictureOfShapesShowsAtTheSizeItAsksForOverItsDensity(t *testing.T) {
	img := Img(Attr("srcset", "star.shapes 2x"))
	d := NewDocument(Images(fstest.MapFS{"star.shapes": {Data: []byte("SHAPES w=80 h=40")}}))
	d.Body().Append(img)
	d.Layout(geom.V2(800, 600))
	assert.Equal(t, geom.V2(40, 20), geom.V2(img.Box().W, img.Box().H), "as Chrome shows an SVG at 2x")
}

func TestWidthDescriptorsAreChosenForTheSlotSizesGivesInTheViewport(t *testing.T) {
	img := Img(Attr("srcset", "p100.png 100w, p200.png 200w, p400.png 400w"),
		Attr("sizes", "(max-width: 500px) 100px, 50vw"))
	d := NewDocument(Images(sized(t, 100, 200, 400)))
	d.Body().Append(img)

	d.Layout(geom.V2(400, 300))
	src, size := shownAs(t, img)
	assert.Equal(t, "p100.png", src)
	assert.Equal(t, geom.V2(100, 50), size, "a picture as wide as its slot")

	d.Layout(geom.V2(600, 300))
	src, size = shownAs(t, img)
	assert.Equal(t, "p400.png", src, "the slot is 300 wide: the smallest picture that fills it")
	assert.Equal(t, geom.V2(300, 150), size, "shown at the slot's width")

	d.Layout(geom.V2(400, 300))
	src, size = shownAs(t, img)
	assert.Equal(t, "p400.png", src, "a denser picture it has wins, as one in Chrome's cache")
	assert.Equal(t, geom.V2(100, 50), size)
}

func TestAPictureShowsTheFirstSourceWhoseMediaHoldsAndWhoseTypeUIReads(t *testing.T) {
	img := Img(Attr("src", "p100.png"))
	d := NewDocument(Images(sized(t, 100, 200, 300, 400)))
	pic := Picture(
		Source(Attr("srcset", "p400.png"), Attr("media", "(max-width: 10px)")),
		Source(Attr("srcset", "p300.png"), Attr("type", "image/avif")),
		Source(Attr("srcset", "")),
		Source(Attr("src", "p300.png")),
		Source(Attr("srcset", "p200.png 2x"), Attr("type", "IMAGE/PNG; q=1"), Attr("media", "screen")),
		img,
		Source(Attr("srcset", "p300.png")),
	)
	d.Body().Append(pic)
	d.Layout(geom.V2(800, 600))
	src, size := shownAs(t, img)
	assert.Equal(t, "p200.png", src)
	assert.Equal(t, geom.V2(100, 50), size)

	pic.Children()[4].SetAttr("type", "image/x-unknown")
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p100.png", src, "no source left: the img's src")
	assert.Equal(t, geom.V2(100, 50), size)

	pic.Children()[0].SetAttr("media", "(min-width: 10px)")
	d.Layout(geom.V2(800, 600))
	src, _ = shownAs(t, img)
	assert.Equal(t, "p400.png", src)
}

func TestAnImgNoLongerChoosingShowsItsSrc(t *testing.T) {
	img := Img(Attr("src", "p100.png"), Attr("srcset", "p200.png 2x"))
	d := NewDocument(Images(sized(t, 100, 200)))
	d.fonts.SetScale(2)
	d.Body().Append(img)
	d.Layout(geom.V2(800, 600))
	src, size := shownAs(t, img)
	require.Equal(t, "p200.png", src)
	assert.Equal(t, geom.V2(100, 50), size)

	img.RemoveAttr("srcset")
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p100.png", src)
	assert.Equal(t, geom.V2(100, 50), size)
}

func TestAnImgKeepsItsPictureUntilTheOneChosenInItsPlaceLands(t *testing.T) {
	release := make(chan struct{})
	pictures := map[string][]byte{"p100.png": picture(t, 100, 50), "p200.png": picture(t, 200, 100)}
	d := NewDocument(FetchImages(func(url string) ([]byte, error) {
		if url == "p200.png" {
			<-release
		}
		return pictures[url], nil
	}))
	img := Img(Attr("srcset", "p100.png 1x, p200.png 2x"))
	d.Body().Append(img)
	d.Layout(geom.V2(800, 600))
	d.fetching.Wait()
	d.Layout(geom.V2(800, 600))
	src, size := shownAs(t, img)
	require.Equal(t, "p100.png", src)
	require.Equal(t, geom.V2(100, 50), size)

	d.fonts.SetScale(2)
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p100.png", src, "the denser picture is on its way")
	assert.Equal(t, geom.V2(100, 50), size)

	close(release)
	d.fetching.Wait()
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, img)
	assert.Equal(t, "p200.png", src, "shown once it landed")
	assert.Equal(t, geom.V2(100, 50), size)
}

func TestAutoSizesChooseForTheWidthTheImgIsLaidOutAt(t *testing.T) {
	set := Attr("srcset", "p100.png 100w, p200.png 200w, p400.png 400w")
	lazy := Img(Attr("loading", "lazy"), set, Attr("sizes", "auto, 400px"), Style("width: 80px"))
	d := NewDocument(Images(sized(t, 100, 200, 400)))
	d.Body().Append(lazy)
	d.Layout(geom.V2(800, 600))
	src, size := shownAs(t, lazy)
	assert.Equal(t, "p100.png", src)
	assert.Equal(t, geom.V2(80, 150), size, "sized as Chrome's sheet sizes it: contain-intrinsic-size 300px 150px")

	lazy.SetStyle("width: 150px")
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, lazy)
	assert.Equal(t, "p200.png", src, "chosen again at its new width")
	assert.Equal(t, geom.V2(150, 150), size)

	wide := Img(Attr("loading", "lazy"), set, Attr("sizes", "AUTO"))
	eager := Img(set, Attr("sizes", "auto, 100px"), Style("width: 50px"))
	d = NewDocument(Images(sized(t, 100, 200, 400)))
	d.Body().Append(wide, eager)
	d.Layout(geom.V2(800, 600))
	src, size = shownAs(t, wide)
	assert.Equal(t, "p400.png", src, "as wide as Chrome's sheet makes it")
	assert.Equal(t, geom.V2(300, 150), size)
	src, size = shownAs(t, eager)
	assert.Equal(t, "p100.png", src, "auto is only for lazy pictures: the next entry")
	assert.Equal(t, geom.V2(50, 150), size)
}

func TestASourcesTypeIsOneUIReads(t *testing.T) {
	for typ, want := range map[string]bool{
		"": true, "image/png": true, "Image/PNG;x=1": true, "image/apng": true,
		"image/avif": false, "image/x-foo": false, "text/html": false, "image/svg+xml": false, "image/x-shapes": true,
	} {
		assert.Equal(t, want, readsType(typ), typ)
	}
	assert.True(t, readsType(" image/png ; x=1"), "spaces round it")
}
