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
// 60×40 with no view box, squashed.svg for 50×25 over a square view box,
// bare.svg for nothing, and effect.svg and object.svg use what is not
// drawn.
func samples(t *testing.T, css string) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{
		"page.css": {Data: []byte(css)}, "bare.svg": {Data: []byte(bare)},
		"effect.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">` +
			`<rect width="24" height="24" vector-effect="non-rotation"/></svg>`)},
		"object.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><foreignObject/></svg>`)},
		"squashed.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="25" ` +
			`viewBox="0 0 100 100" preserveAspectRatio="none"><rect width="100" height="100"/></svg>`)},
	}
	for name, file := range map[string]string{
		"sized.svg": "own-fill-rules.svg", "shaped.svg": "own-joins.svg", "plain.svg": "own-no-viewbox.svg",
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

func TestAViewBoxAloneTakesTheRoomOfAFlexRowAndOfAnAbsoluteBox(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(samples(t, "body { margin: 0; width: 500px; }"), "page.css"))
	img := func(src string, args ...ui.Arg) *ui.Node {
		return ui.Img(append([]ui.Arg{ui.Attr("src", src)}, args...)...)
	}
	alone := img("shaped.svg")
	pair := []*ui.Node{img("shaped.svg"), img("shaped.svg")}
	free, between, high := img("shaped.svg", ui.Style("position: absolute")),
		img("shaped.svg", ui.Style("position: absolute; left: 10px; right: 30px")),
		img("sized.svg", ui.Style("position: absolute; top: 0; bottom: 100px"))
	d.Body().Append(
		ui.Div(ui.Style("display: flex"), alone),
		ui.Div(ui.Style("display: flex; width: 300px"), pair[0], pair[1]),
		ui.Div(ui.Style("position: relative; width: 240px; height: 300px"), free, between, high))
	d.Layout(geom.V2(800, 900))
	assert.Equal(t, geom.V2(500, 250), size(alone), "the row's width")
	assert.Equal(t, geom.V2(300, 150), size(pair[0]), "no narrower than the row, whatever the items beside")
	assert.Equal(t, float32(300), pair[1].Box().X)
	assert.Equal(t, geom.V2(240, 120), size(free), "its containing block's width")
	assert.Equal(t, geom.V2(200, 100), size(between), "between its insets")
	assert.Equal(t, geom.V2(200, 100), size(high), "a picture of its own size keeps it between insets")
}

func TestWhatAnSVGUsesThatIsNotDrawnIsAWarningNamingTheFile(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(samples(t, `body { background: url(object.svg); }`), "page.css"))
	for _, src := range []string{"effect.svg", "effect.svg", "sized.svg"} {
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
		"ui: image effect.svg: image/svg+xml is not drawn: vector-effect",
		"ui: image object.svg: image/svg+xml is not drawn: foreignObject",
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
