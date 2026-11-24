package ui

import (
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
)

// arriving is a page whose pictures are fetched and land when the test
// says: held are those back from the fetcher, kept from the Document
// until land hands them over.
type arriving struct {
	t        testing.TB
	d        *Document
	size     geom.Vec2
	css      string
	build    func() []*Node
	pictures map[string][]byte
	held     map[string]fetchedImage
	shown    map[string]bool
}

// arrive lays out the page build makes under css, its pictures all asked
// for and none landed.
func arrive(t testing.TB, css string, pictures map[string][]byte, build func() []*Node) *arriving {
	t.Helper()
	a := &arriving{t: t, size: geom.V2(400, 300), css: css, build: build, pictures: pictures,
		held: map[string]fetchedImage{}, shown: map[string]bool{}}
	fetcher := &asked{pictures: pictures}
	a.d = NewDocument(CSS(css), FetchImages(fetcher.fetch))
	a.d.Body().Append(build()...)
	a.d.Layout(a.size)
	a.hold()
	return a
}

// hold takes what the fetcher brought back out of the Document's hands.
func (a *arriving) hold() {
	f := &a.d.fetching
	f.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, got := range f.landed {
		a.held[got.url] = got
	}
	f.landed = nil
}

// land has the pictures urls name land before the next layout, runs it,
// and reports how many layouts ran and how many boxes the last one wrote.
func (a *arriving) land(urls ...string) (int, int) {
	a.t.Helper()
	f := &a.d.fetching
	for _, url := range urls {
		got, ok := a.held[url]
		require.True(a.t, ok, "%s was not asked for", url)
		delete(a.held, url)
		a.shown[url] = true
		f.mu.Lock()
		f.landed = append(f.landed, got)
		f.mu.Unlock()
	}
	return a.layout()
}

func (a *arriving) layout() (int, int) {
	a.t.Helper()
	placed, tree := a.d.placed, a.d.tree
	a.d.Layout(a.size)
	a.hold()
	if a.d.placed == placed {
		assert.Same(a.t, tree, a.d.tree)
		return 0, 0
	}
	return (a.d.placed - placed) / len(a.d.tree.Of), len(a.d.tree.Written)
}

// fresh is the page laid out from nothing with the pictures that have
// landed in its files, a fetcher that never answers for the others.
func (a *arriving) fresh() *Document {
	files := fstest.MapFS{}
	for url := range a.shown {
		if b, ok := a.pictures[url]; ok {
			files[url] = &fstest.MapFile{Data: b}
		}
	}
	block := make(chan struct{})
	a.t.Cleanup(func() { close(block) })
	d := NewDocument(CSS(a.css), Images(files), FetchImages(func(string) ([]byte, error) {
		<-block
		return nil, assert.AnError
	}))
	d.Body().Append(a.build()...)
	d.Layout(a.size)
	return d
}

// matchesFresh checks the page's boxes are those a layout from nothing
// gives, to a hundredth of a point: a kept layout adds the same lengths
// up in another order.
func (a *arriving) matchesFresh(msg string) {
	a.t.Helper()
	want, got := boxes(a.fresh().tree.Root), boxes(a.d.tree.Root)
	for i := range min(len(want), len(got)) {
		w, g := want[i], got[i]
		same := w.what == g.what && len(w.at) == len(g.at)
		for j := 0; same && j < len(w.at); j++ {
			same = math.Abs(float64(w.at[j]-g.at[j])) < 0.01
		}
		if !same {
			assert.Fail(a.t, "the kept layout is not a fresh one's: "+msg,
				"box %d\nfresh: %s %v\nkept:  %s %v", i, w.what, w.at, g.what, g.at)
			return
		}
	}
	assert.Len(a.t, got, len(want), "the kept layout is not a fresh one's: "+msg)
}

// laidBox is a box written out: what it is and says, and its lengths.
type laidBox struct {
	what string
	at   []float32
}

// boxes writes a box tree out, depth first: each box's kind, element and
// text, with its place, size, edges, overflow and lines.
func boxes(root *box.Box) []laidBox {
	var out []laidBox
	var walk func(b *box.Box, depth int)
	walk = func(b *box.Box, depth int) {
		l := laidBox{what: fmt.Sprint(depth, " ", b.Kind)}
		if b.Node != nil {
			l.what += " " + b.Node.Tag().String()
		}
		rect := func(r geom.Rect) { l.at = append(l.at, r.X, r.Y, r.W, r.H) }
		edges := func(e box.Edges) { l.at = append(l.at, e.Top, e.Right, e.Bottom, e.Left) }
		rect(geom.Rect{X: b.Pos.X, Y: b.Pos.Y, W: b.Size.X, H: b.Size.Y})
		rect(geom.Rect{X: b.Content.X, Y: b.Content.Y, W: b.Scrollbar.X, H: b.Scrollbar.Y})
		edges(b.Margin)
		edges(b.Border)
		edges(b.Padding)
		edges(b.Wrap)
		l.at = append(l.at, b.Baseline)
		for _, f := range b.Fragments {
			rect(f)
		}
		for _, line := range b.Lines {
			l.what += fmt.Sprintf(" %q", line.Text)
			rect(line.Rect)
			l.at = append(l.at, line.Baseline)
		}
		out = append(out, l)
		for _, c := range b.Children {
			walk(c, depth+1)
		}
	}
	walk(root, 0)
	return out
}

func img(src string, args ...Arg) *Node { return Img(append([]Arg{Attr("src", src)}, args...)...) }

// shownIn is an img of a.png under a sheet: attrs are its attributes,
// each a name and its value.
type shownIn struct {
	css   string
	attrs []string
}

func (s shownIn) img() *Node {
	var args []Arg
	for i := 0; i < len(s.attrs); i += 2 {
		args = append(args, Attr(s.attrs[i], s.attrs[i+1]))
	}
	return img("a.png", args...)
}

func TestAPictureLandingInABoxOfItsOwnSizeLaysNothingOut(t *testing.T) {
	cases := map[string]shownIn{
		"width and height attributes":  {attrs: []string{"width", "40", "height", "30"}},
		"a width and a height in CSS":  {css: `img { width: 40px; height: 30px }`},
		"a width and an aspect-ratio":  {css: `img { width: 40px; aspect-ratio: 2 / 1 }`},
		"a height and an aspect-ratio": {css: `img { height: 40px; aspect-ratio: 2 / 1; min-width: 10px }`},
		"alt text shown meanwhile":     {css: `img { width: 40px; height: 30px }`, attrs: []string{"alt", "a door ajar"}},
		"a block with borders": {
			css: `img { display: block; width: 4em; height: 3em; border: 2px solid; padding: 1px }`},
		"an absolute box":     {css: `img { position: absolute; left: 5px; width: 40px; height: 30px }`},
		"a float":             {css: `img { float: right; width: 40px; height: 30px }`},
		"object-fit in a box": {css: `img { width: 40px; height: 30px; object-fit: contain }`},
		"min and max around both": {
			css: `img { width: 40px; height: 30px; max-width: 100%; min-width: 10px; min-height: 10px }`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := arrive(t, c.css, map[string][]byte{"a.png": picture(t, 120, 80)}, func() []*Node {
				return []*Node{Div(Text("before "), c.img(), Text(" after")), P(Text("below"))}
			})
			a.d.replay.repaint = false
			lays, _ := a.land("a.png")
			assert.Zero(t, lays, "nothing is laid out")
			assert.True(t, a.d.replay.repaint, "the picture is painted")
			a.matchesFresh("after it landed")
			require.NotNil(t, a.d.image("a.png"))

			a.size.X += 40
			a.layout()
			a.matchesFresh("in another viewport")
		})
	}
}

func TestAPictureNoBoxIsSizedByLaysNothingOut(t *testing.T) {
	cases := map[string]string{
		"a background":       `div { background-image: url(a.png) }`,
		"a sized background": `div { background: url(a.png) no-repeat center / auto }`,
		"a mask":             `div { mask-image: url(a.png) }`,
		"a border image":     `div { border: 8px solid; border-image: url(a.png) 4 }`,
	}
	for name, css := range cases {
		t.Run(name, func(t *testing.T) {
			a := arrive(t, css, map[string][]byte{"a.png": picture(t, 120, 80)}, func() []*Node {
				return []*Node{Div(Text("a box")), P(Text("below"))}
			})
			assert.Nil(t, a.d.image("a.png"), "paint asks for it")
			a.hold()
			a.d.replay.repaint = false
			lays, _ := a.land("a.png")
			assert.Zero(t, lays, "nothing is laid out")
			assert.True(t, a.d.replay.repaint, "the picture is painted")
			a.matchesFresh("after it landed")
		})
	}
}

func TestAPictureLandingInABoxItSizesLaysOutItsPathAlone(t *testing.T) {
	cases := map[string]shownIn{
		"no size":                  {},
		"a width alone":            {css: `img { width: 60px }`},
		"a height attribute alone": {attrs: []string{"height", "20"}},
		"max-width alone":          {css: `img { max-width: 100%; display: block } .path { width: 90px }`},
		"a min-width alone":        {css: `img { width: 40px; height: 30px; min-width: 50px }`},
		"alt text shown meanwhile": {attrs: []string{"alt", "a door standing ajar"}},
		"an empty alt":             {attrs: []string{"alt", ""}},
		"an absolute box":          {css: `img { position: absolute; right: 0 }`},
		"a flex item":              {css: `.path { display: flex }`},
		"a grid item":              {css: `.path { display: grid; grid-template-columns: auto 1fr }`},
		"an inline-block's":        {css: `.path { display: inline-block }`},
		"a vertical flex item":     {css: `.path { display: flex; writing-mode: vertical-lr }`},
		"a vertical flex item of a set width": {
			css: `.path { display: flex; writing-mode: vertical-lr; width: 100px; align-items: start } img { width: 40px }`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := arrive(t, c.css, map[string][]byte{"a.png": picture(t, 120, 80)}, func() []*Node {
				page := []*Node{Div(Class("path"), Text("before "), c.img(), Text(" after"))}
				for i := range 200 {
					page = append(page, Div(P(Text(fmt.Sprint("paragraph ", i))), P(Span(Text("and its second")))))
				}
				return page
			})
			total := len(a.d.tree.Of)
			require.Greater(t, total, 1000)
			lays, written := a.land("a.png")
			assert.Equal(t, 1, lays, "one layout")
			assert.Less(t, written, 20, "of the picture's path, not the page's %d boxes", total)
			a.matchesFresh("after it landed")
			lays, _ = a.layout()
			assert.Zero(t, lays, "and no other")
		})
	}
}

// The pages of WPT's css/css-flexbox/flex-aspect-ratio-img-vert-lr.html and
// flexbox-vert-lr-with-img.html: a picture alone in a vertical flex container.
func TestAPictureLandingInAVerticalFlexItemSizesIt(t *testing.T) {
	cases := map[string]struct {
		css  string
		w, h int
		abs  bool
		want geom.Vec2
	}{
		"its width set": {css: `.flex { position: relative; display: flex; writing-mode: vertical-lr; width: 100px;
align-items: start } img { width: 40px } .abs { position: absolute; left: 40px; width: 60px; height: 100px }`,
			w: 20, h: 50, abs: true, want: geom.V2(40, 100)},
		"no size": {css: `.flex { writing-mode: vertical-lr; display: flex }`, w: 100, h: 100, want: geom.V2(100, 100)},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var pic *Node
			a := arrive(t, c.css, map[string][]byte{"a.png": picture(t, c.w, c.h)}, func() []*Node {
				pic = img("a.png")
				flex := Div(Class("flex"), Text("\n    "), pic, Text("\n    "))
				if c.abs {
					flex.Append(Div(Class("abs")))
				}
				return []*Node{P(Text("Test passes if there is a filled green square.")), flex}
			})
			lays, _ := a.land("a.png")
			assert.Equal(t, 1, lays)
			assert.Equal(t, c.want, a.d.tree.Of[pic].Size)
			a.matchesFresh("after it landed")
		})
	}
}

func TestPicturesLandingInOneFrameAreOneLayout(t *testing.T) {
	pictures := map[string][]byte{}
	for i := range 12 {
		pictures[fmt.Sprint(i, ".png")] = picture(t, 20+i, 10+i)
	}
	a := arrive(t, `.sized { width: 30px; height: 20px }`, pictures, func() []*Node {
		var page []*Node
		for i := range 12 {
			page = append(page, P(Text("one "), img(fmt.Sprint(i, ".png"), classIf(i%2 == 0, "sized"))))
		}
		return page
	})
	lays, _ := a.land("0.png", "2.png", "4.png", "6.png")
	assert.Zero(t, lays, "sized pictures")
	a.matchesFresh("sized pictures")
	lays, _ = a.land("1.png", "3.png", "8.png", "5.png", "7.png")
	assert.Equal(t, 1, lays, "pictures of their own size, and one sized")
	a.matchesFresh("pictures of their own size")
	lays, _ = a.land("9.png", "10.png", "11.png")
	assert.Equal(t, 1, lays)
	a.matchesFresh("the last ones")
}

func classIf(on bool, class string) Arg {
	if on {
		return Class(class)
	}
	return Class()
}

func TestAPictureThatFailsLaysNothingOut(t *testing.T) {
	a := arrive(t, "", map[string][]byte{"notes.png": []byte("no picture")}, func() []*Node {
		return []*Node{P(img("gone.png", Attr("alt", "gone")), img("notes.png"))}
	})
	lays, _ := a.land("gone.png", "notes.png")
	assert.Zero(t, lays)
	assert.Len(t, a.d.Warnings(), 2)
	a.matchesFresh("broken pictures")
}

func TestAPictureOfShapesLandingLaysOutOnlyWhatItSizes(t *testing.T) {
	pictures := map[string][]byte{
		"sized.shapes": shapesFile("w=48 h=24"),
		"own.shapes":   shapesFile("w=50 h=20"),
		"ratio.shapes": shapesFile("r=2"),
		"fills.shapes": shapesFile("r=4"),
	}
	a := arrive(t, `.box { width: 30px; height: 20px }`, pictures, func() []*Node {
		return []*Node{
			P(Text("in a box "), img("sized.shapes", Class("box"))),
			P(Text("its own "), img("own.shapes")),
			P(Text("a ratio "), img("ratio.shapes", Attr("width", "40"))),
			P(Text("filling, in a box "), img("fills.shapes", Class("box"))),
		}
	})
	lays, _ := a.land("sized.shapes")
	assert.Zero(t, lays, "a box of its own size")
	a.matchesFresh("in a box")
	lays, _ = a.land("own.shapes")
	assert.Equal(t, 1, lays, "the size the picture asks for")
	a.matchesFresh("its own size")
	lays, _ = a.land("ratio.shapes")
	assert.Equal(t, 1, lays, "a height from its proportions")
	a.matchesFresh("a ratio")
	a.land("fills.shapes")
	a.matchesFresh("a picture that fills")
}

func TestAnotherPictureFromAScriptLaysOutItsPath(t *testing.T) {
	pictures := map[string][]byte{"a.png": picture(t, 120, 80), "b.png": picture(t, 60, 90)}
	var door *Node
	src := "a.png"
	a := arrive(t, "", pictures, func() []*Node {
		door = img(src)
		return []*Node{P(Text("before "), door, Text(" after")), P(Text("below"))}
	})
	a.land("a.png")
	live := door
	live.SetAttr("src", "b.png")
	src = "b.png"
	lays, _ := a.layout()
	assert.Equal(t, 1, lays, "it shows as one without a picture")
	a.matchesFresh("b.png on its way")
	lays, _ = a.land("b.png")
	assert.Equal(t, 1, lays)
	a.matchesFresh("b.png landed")
	assert.InDelta(t, 60, live.Box().W, 0.01)

	live.SetAttr("src", "a.png")
	src = "a.png"
	lays, _ = a.layout()
	assert.Equal(t, 1, lays, "a picture it already has")
	a.matchesFresh("a.png again")
	assert.InDelta(t, 120, live.Box().W, 0.01)
}

func TestAPictureInGeneratedContentLaysOutItsPath(t *testing.T) {
	a := arrive(t, `p::before { content: url(a.png) } p.sized::before { content: url(b.png) }`,
		map[string][]byte{"a.png": picture(t, 24, 18), "b.png": picture(t, 12, 12)}, func() []*Node {
			return []*Node{P(Text("a paragraph")), P(Class("sized"), Text("another")), P(Text("the last"))}
		})
	lays, _ := a.land("b.png")
	assert.Equal(t, 1, lays)
	a.matchesFresh("one landed")
	lays, _ = a.land("a.png")
	assert.Equal(t, 1, lays)
	a.matchesFresh("both landed")
}

func TestAPictureShownTwiceLaysOutOnlyTheBoxItSizes(t *testing.T) {
	a := arrive(t, `.sized { width: 30px; height: 20px } div { background: url(a.png) }`,
		map[string][]byte{"a.png": picture(t, 120, 80)}, func() []*Node {
			return []*Node{
				Div(P(Text("sized "), img("a.png", Class("sized")))),
				Div(P(Text("its own "), img("a.png"))),
				Div(P(Text("none"))),
			}
		})
	lays, written := a.land("a.png")
	assert.Equal(t, 1, lays)
	assert.Less(t, written, 12)
	a.matchesFresh("landed")
}

// mixedSheet styles the pages mixedPage makes: sized and unsized imgs
// among floats, tables, flex and grid items, columns and scrollers.
const mixedSheet = `
	body { width: 380px }
	.w { width: 40px } .h { height: 24px } .wh { width: 36px; height: 28px }
	.max { max-width: 100% } .ratio { width: 30px; aspect-ratio: 3 / 2 } .pct { width: 20% }
	.min { width: 30px; height: 20px; min-width: 44px } .block { display: block } .fit { object-fit: cover }
	.left { float: left; margin: 2px } .right { float: right }
	.flex { display: flex; gap: 4px } .wrap { display: flex; flex-wrap: wrap; width: 150px }
	.col { display: flex; flex-direction: column; align-items: flex-start }
	.grid { display: grid; grid-template-columns: auto 1fr auto; gap: 3px }
	.narrow { width: 70px } .abs { position: relative } .abs img { position: absolute; top: 2px; right: 2px }
	.cols { column-count: 2 } .ib { display: inline-block; border: 1px solid } .ib img { float: none }
	td { border: 1px solid; padding: 2px } .scroll { overflow: scroll; height: 60px }
	.bg { background: url(bg.png); border: 4px solid; border-image: url(edge.png) 2 }
	.gen::before { content: url(gen.png) }`

// mixedPage makes the same random page from a seed each time it is
// built: pictures are the files of its imgs, but those that fail, and
// count how many imgs it names.
type mixedPage struct {
	t        *testing.T
	seed     uint64
	pictures map[string][]byte
	count    int
	r        *rand.Rand
}

func (m *mixedPage) pic() *Node {
	classes := []string{"", "", "", "w", "h", "wh", "wh", "wh", "max", "ratio", "pct", "min", "block", "wh fit",
		"left", "right", "wh left", "w right", "max block"}
	r := m.r
	url := fmt.Sprint("p", m.count, ".png")
	m.count++
	fails, w, h := r.IntN(8) == 0, 8+r.IntN(120), 6+r.IntN(90)
	if _, made := m.pictures[url]; !made && !fails {
		m.pictures[url] = picture(m.t, w, h)
	}
	args := []Arg{Class(strings.Fields(classes[r.IntN(len(classes))])...)}
	switch r.IntN(6) {
	case 0:
		args = append(args, Attr("width", fmt.Sprint(10+r.IntN(60))), Attr("height", fmt.Sprint(10+r.IntN(40))))
	case 1:
		args = append(args, Attr("width", fmt.Sprint(10+r.IntN(60))))
	case 2:
		args = append(args, Attr("alt", "what it shows"))
	case 3:
		args = append(args, Attr("alt", ""))
	}
	if r.IntN(10) == 0 && m.count > 1 {
		url = fmt.Sprint("p", r.IntN(m.count-1), ".png")
	}
	return img(url, args...)
}

func (m *mixedPage) build() []*Node {
	m.r, m.count = rand.New(rand.NewPCG(m.seed, 7)), 0
	pic := m.pic
	words := func() Arg { return Text(strings.Repeat("some words ", 1+m.r.IntN(6))) }
	var page []*Node
	for range 14 {
		switch m.r.IntN(10) {
		case 0:
			page = append(page, P(words(), pic(), words(), pic(), words()))
		case 1:
			page = append(page, Div(Class("flex"), pic(), Div(words()), pic()))
		case 2:
			page = append(page, Div(Class("grid"), pic(), P(words()), pic(), pic(), Div(words())))
		case 3:
			page = append(page, Table(Tr(Td(pic()), Td(words())), Tr(Td(words(), pic()), Td(pic()))))
		case 4:
			page = append(page, Div(Class("narrow"), pic(), words()), P(words()))
		case 5:
			page = append(page, Div(Class("abs bg"), words(), pic()), Div(Class("wrap"), pic(), pic(), pic()))
		case 6:
			page = append(page, Div(Class("cols"), words(), pic(), words()), Div(Class("col"), pic(), pic()))
		case 7:
			page = append(page, P(Class("gen"), Span(Class("ib"), pic(), words())),
				Div(Class("scroll"), pic(), words(), pic()))
		case 8:
			page = append(page, Ul(Li(pic(), words()), Li(A(Attr("href", "#"), pic()))))
		default:
			page = append(page, P(words()), Div(Class("bg"), words()))
		}
	}
	return page
}

// TestPicturesLandingInAnyOrderLeaveAFreshLayout lands the pictures of
// random pages a few a frame, in a random order, some failing, and holds
// the kept layout against a fresh one each frame, and in a narrower
// viewport every fifth.
func TestPicturesLandingInAnyOrderLeaveAFreshLayout(t *testing.T) {
	for seed := range uint64(24) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			m := &mixedPage{t: t, seed: seed, pictures: map[string][]byte{
				"bg.png": picture(t, 6, 6), "edge.png": picture(t, 6, 6), "gen.png": picture(t, 14, 10)}}
			a := arrive(t, mixedSheet, m.pictures, m.build)
			a.d.image("bg.png")
			a.d.image("edge.png")
			a.hold()
			waiting := slices.Sorted(maps.Keys(a.held))
			order := rand.New(rand.NewPCG(seed, 11))
			order.Shuffle(len(waiting), func(i, j int) { waiting[i], waiting[j] = waiting[j], waiting[i] })
			for frame := 0; len(waiting) > 0 && !t.Failed(); frame++ {
				n := min(1+order.IntN(3), len(waiting))
				lays, _ := a.land(waiting[:n]...)
				assert.LessOrEqual(t, lays, 1)
				a.matchesFresh(fmt.Sprint("frame ", frame, " landing ", waiting[:n]))
				waiting = waiting[n:]
				if frame%5 == 4 {
					a.size.X -= 30
					a.layout()
					a.matchesFresh(fmt.Sprint("frame ", frame, " in a narrower viewport"))
				}
			}
		})
	}
}

// BenchmarkAPictureLandsInAPageOfThousandsOfElements is one frame in which
// a picture lands in a box of its own size, among sixty in a page of
// 7,000 elements, and with unsized the same for a picture that sizes its
// box.
func BenchmarkAPictureLandsInAPageOfThousandsOfElements(b *testing.B) {
	for _, c := range []struct{ name, css string }{
		{"sized", `img { width: 220px; height: 140px }`},
		{"unsized", ``},
	} {
		b.Run(c.name, func(b *testing.B) {
			const sections, each = 600, 10
			pictures := map[string][]byte{}
			var urls []string
			for i := range sections / each {
				urls = append(urls, fmt.Sprint("p", i, ".png"))
				pictures[urls[i]] = picture(b, 220, 140)
			}
			build := func() []*Node {
				var page []*Node
				for i := range sections {
					s := Section(H2(Text(fmt.Sprint("Section ", i))),
						P(Text(strings.Repeat("words of a paragraph, ", 12)), A(Attr("href", "#"), Text("a link")), Text(" and more.")),
						Ul(Li(Text("one")), Li(Text("two")), Li(Span(Text("three")))),
						Table(Tr(Td(Text("a")), Td(Text("b"))), Tr(Td(Text("c")), Td(Text("d")))))
					if i%each == 0 {
						s.Append(Figure(img(urls[i/each]), Figcaption(Text("what it shows"))))
					}
					page = append(page, s)
				}
				return page
			}
			var a *arriving
			lays, landed := 0, len(urls)
			b.ResetTimer()
			for range b.N {
				if landed == len(urls) {
					b.StopTimer()
					a = arrive(b, c.css, pictures, build)
					landed = 0
					b.StartTimer()
				}
				n, _ := a.land(urls[landed])
				lays += n
				landed++
			}
			b.ReportMetric(float64(lays)/float64(b.N), "layouts/op")
			b.ReportMetric(float64(len(a.d.tree.Of)), "boxes")
		})
	}
}
