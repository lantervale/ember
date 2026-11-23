//go:build chrome

package ui_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"slices"
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
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
	_ "github.com/lantervale/ember/ui/web" // reads animated GIFs
	"github.com/lantervale/ember/world"
)

// flags are the colours of the frames of the GIFs these tests play.
//
//nolint:gochecknoglobals // the frames' colours, read by every test
var flags = color.Palette{
	color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 0, 255},
}

// flagGIF is a 40×40 GIF playing for ever the colours of flags, each
// for delay hundredths of a second.
func flagGIF(t *testing.T, delay int) []byte {
	t.Helper()
	g := &gif.GIF{Config: image.Config{ColorModel: flags, Width: 40, Height: 40}}
	for i := range flags {
		p := image.NewPaletted(image.Rect(0, 0, 40, 40), flags)
		for j := range p.Pix {
			p.Pix[j] = uint8(i)
		}
		g.Image, g.Delay = append(g.Image, p), append(g.Delay, delay)
	}
	var b bytes.Buffer
	require.NoError(t, gif.EncodeAll(&b, g))
	return b.Bytes()
}

// flagAt is the frame of flags Chrome shows at x, y of a shot, -1 for
// none.
func flagAt(shot image.Image, x, y int) int {
	r, g, b, _ := shot.At(x, y).RGBA()
	for i, c := range flags {
		cr, cg, cb, _ := c.RGBA()
		if r>>8 == cr>>8 && g>>8 == cg>>8 && b>>8 == cb>>8 {
			return i
		}
	}
	return -1
}

// framesWithin are the frames a picture of n frames, each delay seconds,
// playing from began, shows between from and to, give or take the frame
// of Chrome's it is first drawn on.
func framesWithin(began, from, to, delay float64, n int) []int {
	var out []int
	for i := max(int(math.Floor((from-0.05-began)/delay)), 0); i <= int(math.Floor((to-began)/delay)); i++ {
		out = append(out, i%n)
	}
	return out
}

// runUntil runs w's frames until its real time reaches s seconds.
func runUntil(w *world.World, s float64) {
	for float64(w.Time().RealElapsed) < s {
		w.RunFrames(1)
	}
}

func TestAnAnimationOutOfViewStandsAndComesBackAsChromes(t *testing.T) {
	files := map[string][]byte{"top.gif": flagGIF(t, 40), "below.gif": flagGIF(t, 40)}
	build := func(opts ...ui.DocumentOption) *ui.Document {
		d := ui.NewDocument(append(opts, ui.ScrollingViewport(), ui.CSS(`
			body { margin: 0; height: 10000px; position: relative; }
			img { position: absolute; left: 0; width: 40px; height: 40px; }`))...)
		d.Body().Append(ui.Img(ui.Attr("src", "top.gif"), ui.Style("top: 0")),
			ui.Img(ui.Attr("src", "below.gif"), ui.Style("top: 3000px")))
		return d
	}
	// Each step scrolls to its offset s seconds after the page loaded,
	// then reads the frame at the top-left: the picture below begins when
	// first drawn, long after it landed; the top one, out of view across
	// its play's end, begins the next play when drawn again, and out of
	// view within a play comes back in step. Chrome's frame was drawn
	// somewhere between the asking for the shot and its coming back, a
	// while on a loaded machine, so ui is to show it somewhere in that
	// span, scrolled when Chrome was, past the one frame a picture drawn
	// again shows as it left.
	steps := []struct {
		at     float64
		scroll float32
		url    string
	}{{1.0, 3000, "below.gif"}, {2.4, 0, "top.gif"}, {2.7, 3000, ""}, {3.4, 0, "top.gif"}}
	srv, _ := chrome.Serve(t, build(), files, 0)
	s := chrome.Open(t, srv.URL, geom.V2(400, 300), 1)
	var loaded float64
	s.Eval(`performance.getEntriesByType("navigation")[0].loadEventEnd / 1000`, &loaded)
	var want []int
	var scrolled, shotFrom, shotTo []float64
	sinceLoad := func() float64 {
		var now float64
		s.Eval(`performance.now() / 1000`, &now)
		return now - loaded
	}
	for _, step := range steps {
		var now float64
		s.Eval(fmt.Sprintf(`new Promise(r => setTimeout(r, Math.max(0, %f * 1000 - performance.now())))
			.then(() => (scrollTo(0, %g), performance.now() / 1000))`, step.at+loaded, step.scroll), &now)
		scrolled = append(scrolled, now-loaded)
		if step.url == "" {
			continue
		}
		shotFrom = append(shotFrom, sinceLoad())
		want = append(want, flagAt(s.Shot(), 20, 20))
		shotTo = append(shotTo, sinceLoad())
	}
	require.Len(t, want, 3)
	for i, w := range want {
		require.NotEqual(t, -1, w, "Chrome's frame at step %d", i)
	}
	// Chrome's own timing: below.gif plays from when it is first drawn,
	// top.gif drawn again after its play's end plays again from then, and
	// comes back in step with that play.
	for i, began := range []float64{scrolled[0], scrolled[1], scrolled[1]} {
		assert.Contains(t, framesWithin(began, shotFrom[i], shotTo[i], 0.4, len(flags)), want[i],
			"Chrome's frame at step %d, shot between %gs and %gs", i, shotFrom[i], shotTo[i])
	}

	mapFS := fstest.MapFS{}
	for name, data := range files {
		mapFS[name] = &fstest.MapFile{Data: data}
	}
	d := build(ui.Images(mapFS))
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	k := 0
	for i, step := range steps {
		runUntil(w, scrolled[i])
		ui.ScrollTo(d, d.HTML(), geom.V2(0, step.scroll))
		if step.url == "" {
			continue
		}
		w.RunFrames(1)
		runUntil(w, shotFrom[k])
		var shown []int
		for {
			w.RunFrames(1)
			shown = append(shown, ui.ShownFrame(d, step.url))
			if float64(w.Time().RealElapsed) >= shotTo[k] {
				break
			}
		}
		assert.Contains(t, shown, want[k], "%s between %gs and %gs", step.url, shotFrom[k], shotTo[k])
		k++
	}
}

func TestALazyPictureIsAskedForNearTheViewportAsChromes(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 10, 10))))
	files := map[string][]byte{}
	lazyImg := func(name string, x, y int) *ui.Node {
		files[name] = b.Bytes()
		return ui.Img(ui.Attr("src", name), ui.Attr("loading", "lazy"),
			ui.Style(fmt.Sprintf("left: %dpx; top: %dpx", x, y)))
	}
	build := func(opts ...ui.DocumentOption) *ui.Document {
		d := ui.NewDocument(append(opts, ui.ScrollingViewport(), ui.CSS(`
			body { margin: 0; height: 10000px; position: relative; }
			img { position: absolute; width: 10px; height: 10px; }
			.list { position: absolute; left: 100px; top: 100px; width: 200px; height: 100px; overflow: auto; }
			.room { width: 3000px; height: 3000px; }`))...)
		var top []*ui.Node
		for _, y := range []int{0, 500, 1400, 1540, 1560, 1700, 2540, 2560, 3000} {
			top = append(top, lazyImg(fmt.Sprintf("y%d.png", y), 0, y))
		}
		d.Body().Append(top...)
		d.Body().Append(ui.Div(ui.Class("list"), ui.Div(ui.Class("room")),
			lazyImg("in-1340.png", 0, 1340), lazyImg("in-1360.png", 0, 1360),
			lazyImg("in-x800.png", 800, 0), lazyImg("in-x830.png", 830, 0)))
		return d
	}
	size := geom.V2(400, 300)
	srv, requests := chrome.Serve(t, build(), files, 0)
	s := chrome.Open(t, srv.URL, size, 1)
	settle := func() []string {
		s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(r, 300))))`, nil)
		got := slices.Clone(requests())
		for i, p := range got {
			got[i] = strings.TrimPrefix(p, "/")
		}
		slices.Sort(got)
		return slices.Compact(got)
	}
	atLoad := settle()
	s.Eval(`scrollTo(0, 1000)`, nil)
	scrolled := settle()
	require.NotEmpty(t, atLoad)
	require.Greater(t, len(scrolled), len(atLoad), "Chrome waited for some until scrolled")

	var mu sync.Mutex
	var asked []string
	d := build(ui.FetchImages(func(url string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, url)
		return files[url], nil
	}))
	got := func() []string {
		d.Layout(size)
		ui.LandImages(d, size)
		mu.Lock()
		defer mu.Unlock()
		out := slices.Clone(asked)
		slices.Sort(out)
		return slices.Compact(out)
	}
	assert.Equal(t, atLoad, got(), "at load")
	ui.ScrollTo(d, d.HTML(), geom.V2(0, 1000))
	assert.Equal(t, scrolled, got(), "scrolled down")
}

func TestAnImgChoosesAgainWhenChromesDoes(t *testing.T) {
	files := map[string][]byte{}
	pic := func(name string, w int) string {
		var b bytes.Buffer
		require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, w/2))))
		files[name] = b.Bytes()
		return name
	}
	build := func(opts ...ui.DocumentOption) *ui.Document {
		d := ui.NewDocument(opts...)
		d.Body().Append(
			ui.Img(ui.ID("x"), ui.Attr("srcset", pic("x1.png", 100)+" 1x, "+pic("x2.png", 200)+" 2x, "+
				pic("x4.png", 400)+" 4x")),
			ui.Img(ui.ID("w"), ui.Attr("sizes", "25vw"), ui.Attr("srcset", pic("w100.png", 100)+" 100w, "+
				pic("w200.png", 200)+" 200w, "+pic("w400.png", 400)+" 400w")),
			ui.Picture(ui.Source(ui.Attr("media", "(min-resolution: 2dppx)"), ui.Attr("srcset", pic("m2.png", 200))),
				ui.Img(ui.ID("m"), ui.Attr("src", pic("m1.png", 100)))),
			ui.Picture(ui.Source(ui.Attr("media", "(prefers-color-scheme: dark)"), ui.Attr("srcset", pic("dark.png", 100))),
				ui.Img(ui.ID("d"), ui.Attr("src", pic("light.png", 100)))),
		)
		return d
	}
	steps := []struct {
		name  string
		size  geom.Vec2
		scale float32
		dark  bool
	}{
		{"opened", geom.V2(400, 300), 1, false},
		{"denser", geom.V2(400, 300), 2, false},
		{"dark", geom.V2(400, 300), 2, true},
		{"wider", geom.V2(800, 300), 2, true},
	}
	srv, _ := chrome.Serve(t, build(), files, 0)
	s := chrome.Open(t, srv.URL, steps[0].size, steps[0].scale)
	want := make([]map[string]string, len(steps))
	for i, step := range steps {
		s.Resize(step.size, step.scale)
		scheme := "light"
		if step.dark {
			scheme = "dark"
		}
		s.Media("prefers-color-scheme", scheme)
		s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(r, 300))))
			.then(() => Object.fromEntries([...document.images].map(e => [e.id, e.currentSrc.replace(/.*\//, "")])))`,
			&want[i])
	}
	assert.Equal(t, "x1.png", want[1]["x"], "Chrome keeps its choice when the density alone changes")
	assert.Equal(t, "x2.png", want[3]["x"], "and chooses again when the viewport resizes")

	d := build(ui.FetchImages(func(url string) ([]byte, error) { return files[url], nil }))
	for i, step := range steps {
		fonts.Of(d).SetScale(step.scale)
		d.SetColorScheme(platform.AppearanceLight)
		if step.dark {
			d.SetColorScheme(platform.AppearanceDark)
		}
		ui.LandImages(d, step.size)
		got := map[string]string{}
		d.HTML().Walk(func(n *ui.Node) bool {
			if n.Tag().String() == "img" {
				got[n.ID()], _ = ui.PictureOf(n)
			}
			return true
		})
		assert.Equal(t, want[i], got, step.name)
	}
}

func TestAPictureOfHighFetchPriorityIsAskedForFirstAsChromes(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 10, 10))))
	files := map[string][]byte{}
	build := func(opts ...ui.DocumentOption) *ui.Document {
		d := ui.NewDocument(opts...)
		for i := range 22 {
			name := fmt.Sprintf("p%d.png", i)
			files[name] = b.Bytes()
			img := ui.Img(ui.Attr("src", name))
			if i >= 20 {
				img.SetAttr("fetchpriority", "high")
			}
			d.Body().Append(img)
		}
		return d
	}
	srv, requests := chrome.Serve(t, build(), files, 300*time.Millisecond)
	chrome.Open(t, srv.URL, geom.V2(400, 300), 1)
	asked := requests()
	require.Len(t, asked, 22)
	first := asked[:12]
	assert.Contains(t, first, "/p20.png", "in the first two rounds of six Chrome asks for, not the last")
	assert.Contains(t, first, "/p21.png", "in the first two rounds of six Chrome asks for, not the last")

	release := make(chan struct{})
	var mu sync.Mutex
	var got []string
	d := build(ui.FetchImages(func(url string) ([]byte, error) {
		mu.Lock()
		got = append(got, url)
		mu.Unlock()
		<-release
		return files[url], nil
	}))
	d.Layout(geom.V2(400, 300))
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == 6
	}, 10*time.Second, time.Millisecond)
	mu.Lock()
	assert.Contains(t, got, "p20.png")
	assert.Contains(t, got, "p21.png")
	mu.Unlock()
	close(release)
	ui.LandImages(d, geom.V2(400, 300))
}
