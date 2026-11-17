package svg_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	_ "github.com/lantervale/ember/ui/svg"
)

// bare is a picture with neither a size nor a view box.
const bare = `<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="60" fill="#2a7"/>` +
	`<circle cx="50" cy="30" r="20" fill="#fc0"/></svg>`

// samples are a sheet and pictures of each kind: sized.svg asks for
// 200×100, shaped.svg has a view box of 48×24 alone, plain.svg asks for
// 60×40 with no view box, bare.svg for nothing, and text.svg and
// clip.svg use what is not drawn.
func samples(t *testing.T, css string) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{"page.css": {Data: []byte(css)}, "bare.svg": {Data: []byte(bare)}}
	for name, file := range map[string]string{
		"sized.svg": "own-fill-rules.svg", "shaped.svg": "own-joins.svg", "plain.svg": "own-no-viewbox.svg",
		"text.svg": "own-text.svg", "clip.svg": "own-clip.svg",
	} {
		data, err := os.ReadFile("../../svg/testdata/" + file)
		require.NoError(t, err)
		files[name] = &fstest.MapFile{Data: data}
	}
	return files
}

func size(n *ui.Node) geom.Vec2 { return geom.V2(n.Box().W, n.Box().H) }

func TestAnSVGImgTakesTheSizeItsFileAsksFor(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(samples(t, ""), "page.css"))
	img := func(src string, args ...ui.Arg) *ui.Node {
		return ui.Img(append([]ui.Arg{ui.Attr("src", src)}, args...)...)
	}
	sized, shaped, plain, none := img("sized.svg"), img("shaped.svg"), img("plain.svg"), img("bare.svg")
	wide, high := img("shaped.svg", ui.Attr("width", "100")), img("bare.svg", ui.Style("height: 30px"))
	d.Body().Append(ui.Div(ui.Style("width: 400px"), sized, shaped, plain, none, wide, high))
	d.Layout(geom.V2(800, 2000))
	assert.Equal(t, geom.V2(200, 100), size(sized), "its width and height")
	assert.Equal(t, geom.V2(400, 200), size(shaped), "a view box alone: the width it is given, by its ratio")
	assert.Equal(t, geom.V2(60, 40), size(plain))
	assert.Equal(t, geom.V2(300, 150), size(none), "neither: CSS's 300 by 150")
	assert.Equal(t, geom.V2(100, 50), size(wide))
	assert.Equal(t, geom.V2(300, 30), size(high), "and its sides don't hold each other")
	assert.Empty(t, d.Warnings())
}

func TestWhatAnSVGUsesThatIsNotDrawnIsAWarningNamingTheFile(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(samples(t, `body { background: url(clip.svg); }`), "page.css"))
	for _, src := range []string{"text.svg", "text.svg", "sized.svg"} {
		d.Body().Append(ui.Img(ui.Attr("src", src)))
	}
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	ember.New().AddEntities(win).RunFrames(3)
	var texts []string
	for _, w := range d.Warnings() {
		texts = append(texts, w.Error())
	}
	assert.ElementsMatch(t, []string{
		"ui: image text.svg: image/svg+xml is not drawn: text",
		"ui: image clip.svg: image/svg+xml is not drawn: clip-path",
	}, texts)
}

func TestAFileThatIsNoSVGIsABrokenImage(t *testing.T) {
	files := fstest.MapFS{"page.css": {Data: nil}, "torn.svg": {Data: []byte("<svg")},
		"feed.xml": {Data: []byte(`<?xml version="1.0"?><rss version="2.0"/>`)}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	torn, feed := ui.Img(ui.Attr("src", "torn.svg")), ui.Img(ui.Attr("src", "feed.xml"))
	d.Body().Append(torn, feed)
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, geom.V2(16, 16), size(torn), "a broken image's icon")
	assert.Equal(t, geom.V2(16, 16), size(feed))
	require.Len(t, d.Warnings(), 2)
	assert.Contains(t, d.Warnings()[0].Error(), "ui: image torn.svg: svg: ")
	assert.Contains(t, d.Warnings()[1].Error(), "unknown format", "other XML is left to the pictures of pixels")
}
