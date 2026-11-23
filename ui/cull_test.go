package ui_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/cull"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bothWays has every paint and hit test until t ends done twice, walking
// every box and skipping those out of view, and t fail where the two
// differ: in what is drawn, its order, clips and transforms, or in the box
// hit.
func bothWays(t *testing.T) {
	t.Helper()
	t.Cleanup(cull.Set(cull.Check, func(what string) { t.Errorf("skipping the boxes out of view changed %s", what) }))
}

// Every construct that draws or is hit outside its box's place in the
// flow, between rows of filler, so a scroll cuts through each.
const cullSheet = `
html { scrollbar-width: none; }
body { margin: 0; font-size: 14px; color: #dddddd; background: #101418; }
.fill { height: 170px; border-bottom: 1px solid #444444; }
.fill:hover { background-color: #202830; }
.ring { height: 20px; outline: 6px solid #ff0000; outline-offset: 30px; }
.shadow { height: 20px; box-shadow: 0 -120px 0 10px #0000ff, 0 90px 40px #00ff00; }
.glow { text-shadow: 0 -100px 0 #ff0000, 0 80px 6px #0000ff; text-decoration: underline wavy #00ff00; }
.long { width: 50px; white-space: nowrap; background: #333333; }
.before::before { content: "before"; display: block; margin-top: -90px; outline: 2px solid #ff0000; }
.after { position: relative; }
.after::after { content: "after"; position: absolute; top: 60px; left: 200px; background: #884400; }
.shift { transform: translate(40px, -260px); background: #226622; height: 30px; }
.turn { transform: rotate(30deg) scale(1.5); background: #662222; width: 120px; height: 30px; }
.turn > div { height: 400px; width: 10px; background: #aa2222; }
.clipped-turn { transform: translateY(-200px); overflow: hidden; height: 50px; background: #333366; }
.held { position: relative; height: 40px; background: #1c2c3c; }
.rel { position: relative; top: -140px; left: 30px; height: 20px; background: #446688; }
.abs { position: absolute; top: 230px; left: 100px; width: 80px; height: 30px; background: #886644; }
.free { position: absolute; top: 1900px; right: 0; width: 60px; height: 60px; background: #448866; }
.fixed { position: fixed; right: 10px; bottom: 10px; width: 50px; height: 20px; background: #884488; }
.float { height: 0; }
.float > div { float: right; width: 100px; height: 260px; background: #555522; }
.neg { margin-top: -150px; margin-left: -30px; height: 40px; background: #225555; }
.spill { height: 10px; }
.spill > div { height: 300px; width: 600px; border: 2px dashed #777777; }
.sticky { position: sticky; top: 10px; height: 20px; background: #773333; }
.clip { overflow: hidden; height: 60px; border-radius: 12px; background: #222222; }
.clip > div { height: 300px; box-shadow: 0 0 20px #ffffff; }
.scroll { overflow: auto; height: 80px; background: #181818; }
.scroll > div { height: 500px; }
.across { overflow-x: hidden; overflow-y: visible; height: 30px; background: #303030; }
.across > div { height: 200px; width: 900px; background: #404040; }
.cols { column-count: 3; column-rule: 2px solid #ff0000; height: 90px; }
.faint { opacity: 0.5; background: #666666; height: 30px; }
.under { position: relative; z-index: -1; top: -50px; height: 60px; background: #443322; }
.over { position: relative; z-index: 3; top: 40px; height: 20px; background: #334422; }
.split { outline: 2px solid #ffff00; box-shadow: 0 30px 0 #00ffff; background: #333300; }
tr { background-color: #223344; }
td { padding: 4px; border: 1px solid #555555; }
dialog { background: #ffffff; color: #000000; }
dialog::backdrop { background: #00000080; }
.late { height: 30px; background: #999999; }
.slow { transition: transform 0.2s linear, box-shadow 0.2s linear; }
`

type cullPage struct {
	doc          *ui.Document
	w            *world.World
	in           *input.State
	inner, late  *ui.Node
	choice, note *ui.Node
	dialog       *ui.Node
}

func newCullPage(t *testing.T, zoom float32) *cullPage {
	t.Helper()
	p := &cullPage{doc: ui.NewDocument(ui.ScrollingViewport(), ui.Zoom(zoom), ui.CSS(cullSheet))}
	fill := func() *ui.Node { return ui.Div(ui.Class("fill"), ui.Text("filler")) }
	words := strings.Repeat("words that wrap over several lines ", 12)
	p.inner = ui.Div(ui.Class("scroll"), ui.Div(ui.Text(words),
		ui.Div(ui.Class("scroll"), ui.Div(ui.Text(words))), ui.Div(ui.Class("abs"), ui.Text("abs in a scroller"))))
	p.choice = ui.Select(ui.Option(ui.Text("one")), ui.Option(ui.Text("two")), ui.Option(ui.Text("three")))
	p.note = ui.Input(ui.Value("typed"))
	p.dialog = ui.Dialog(ui.Text("a modal dialog"), ui.Button(ui.Text("Close")))
	p.late = ui.Div(ui.Class("late"), ui.Text("late"))
	orb := ui.Canvas(func(c *twod.Canvas) {
		c.Circle(geom.V2(30, -150), 40, color.Hex("#ff8800"))
		c.Rect(geom.Rect{X: -20, Y: 10, W: 400, H: 300}, color.Hex("#0088ff"))
	}, ui.Style("display: block; width: 60px; height: 40px"))
	rows := []*ui.Node{
		ui.Div(ui.Class("ring"), ui.Text("outline")),
		ui.Div(ui.Class("shadow"), ui.Text("box shadows")),
		ui.Div(ui.Class("glow"), ui.Text("text shadows and a wavy line")),
		ui.Div(ui.Class("long"), ui.Text("text that does not wrap and runs out of its box to the right")),
		ui.Ul(ui.Li(ui.Text("one")), ui.Li(ui.Text("two"), ui.Ol(ui.Li(ui.Text("nested"))))),
		ui.Details(ui.Summary(ui.Text("summary")), ui.Text("details")),
		ui.Div(ui.Class("before"), ui.Text("generated before")),
		ui.Div(ui.Class("after"), ui.Text("generated after, placed")),
		ui.Div(ui.Class("shift"), ui.Text("translated up")),
		ui.Div(ui.Class("turn"), ui.Text("turned"), ui.Div()),
		ui.Div(ui.Class("clipped-turn"), ui.Div(ui.Style("height: 400px"), ui.Text("translated and clipped"))),
		ui.Div(ui.Class("held"), ui.Div(ui.Class("rel"), ui.Text("relative")), ui.Div(ui.Class("abs"), ui.Text("absolute"))),
		ui.Div(ui.Class("free"), ui.Text("free")),
		ui.Div(ui.Class("fixed"), ui.Text("fixed")),
		ui.Div(ui.Class("float"), ui.Div(ui.Text("a float taller than its parent"))),
		ui.Div(ui.Class("neg"), ui.Text("negative margins")),
		ui.Div(ui.Class("spill"), ui.Div(ui.Text("a child overflowing a visible box"))),
		ui.Div(ui.Class("sticky"), ui.Text("sticky")),
		ui.Div(ui.Class("clip"), ui.Div(ui.Text("clipped"), ui.Div(ui.Class("abs"), ui.Text("abs under a clip")))),
		p.inner,
		ui.Div(ui.Class("across"), ui.Div(ui.Text("cut across only"))),
		ui.Div(ui.Class("cols"), ui.P(ui.Text(words)), ui.P(ui.Text(words))),
		ui.Div(ui.Class("faint"), ui.Text("half opaque")),
		ui.Div(ui.Class("under"), ui.Text("under the flow")),
		ui.Div(ui.Class("over"), ui.Text("over the flow")),
		ui.P(ui.Text("a paragraph with "), ui.Span(ui.Class("split"), ui.Text(words)), ui.Text(" a split inline")),
		ui.Table(ui.Tr(ui.Td(ui.Text("a")), ui.Td(ui.Text("b"))), ui.Tr(ui.Td(ui.Text("c")), ui.Td(ui.Text("d")))),
		ui.Div(p.choice, p.note, ui.Progress(), ui.Input(ui.Type("checkbox"), ui.Checked(true))),
		orb,
	}
	for _, r := range rows {
		p.doc.Body().Append(fill(), r)
	}
	p.doc.Body().Append(fill(), fill(), p.late, p.dialog)
	win := platform.NewWindow(platform.Size(480, 360))
	win.Add(p.doc)
	p.w = ember.New().AddEntities(win)
	p.in = win.Input()
	p.w.RunFrames(2)
	return p
}

// height is how far the page scrolls.
func (p *cullPage) height() float32 { return ui.ScrollSize(p.doc, p.doc.HTML()).Y }

// look scrolls the viewport to y, draws, and asks for the element at each
// point of a grid over the window, the pointer moving over the page too.
func (p *cullPage) look(y float32) {
	ui.ScrollTo(p.doc, p.doc.HTML(), geom.V2(0, y))
	p.w.RunFrames(1)
	for gy := float32(-10); gy < 380; gy += 37 {
		for gx := float32(-10); gx < 500; gx += 41 {
			p.doc.ElementAt(geom.V2(gx, gy))
		}
		p.in.SetMouse(geom.V2(gy+60, gy))
		p.w.RunFrames(1)
	}
}

func (p *cullPage) lookEverywhere() {
	for y := float32(0); y < p.height(); y += 263.5 {
		p.look(y)
	}
	p.look(p.height())
}

func TestCullingChangesNothingAPageDrawsOrHits(t *testing.T) {
	bothWays(t)
	for _, zoom := range []float32{1, 1.5, 0.75} {
		p := newCullPage(t, zoom)
		require.Greater(t, p.height(), float32(5000), "a page many screens high")
		p.lookEverywhere()

		ui.ScrollTo(p.doc, p.inner, geom.V2(0, 210))
		p.lookEverywhere()

		p.doc.Focus(p.note)
		p.lookEverywhere()

		p.doc.ScrollIntoView(p.choice)
		p.w.RunFrames(2)
		area, _ := p.doc.BoxOf(p.choice)
		at := area.Center()
		p.in.SetMouse(at)
		p.w.RunFrames(1)
		p.in.SetMouseButton(input.MouseLeft, true)
		p.w.RunFrames(1)
		p.in.SetMouseButton(input.MouseLeft, false)
		p.w.RunFrames(2)
		p.lookEverywhere()

		p.doc.ShowModal(p.dialog)
		p.lookEverywhere()
	}
}

// A control's parts, a range's track and thumb, a progress's or a meter's
// bars, are drawn as boxes of their own but hit as the control.
func TestAControlsPartsAreHitAsTheControl(t *testing.T) {
	bothWays(t)
	d := ui.NewDocument(ui.CSS(`
body { margin: 0; }
input, progress { appearance: none; display: block; width: 120px; }
input::-webkit-slider-runnable-track { background: #99cc66; }
input::-webkit-slider-thumb { appearance: none; width: 10px; height: 24px; background: #003366; }
progress { height: 14px; }
progress::-webkit-progress-bar { background: #dddddd; padding: 2px; }
progress::-webkit-progress-value { background: #22aa77; }
`))
	slider := ui.Input(ui.Type("range"), ui.Value("100"))
	bar := ui.Progress(ui.Value("0.6"))
	d.Body().Append(slider, bar)
	d.Layout(geom.V2(400, 300))
	for _, n := range []*ui.Node{slider, bar} {
		area, ok := d.BoxOf(n)
		require.True(t, ok)
		for x := area.X + 1; x < area.X+area.W; x += 7 {
			assert.Same(t, n, d.ElementAt(geom.V2(x, area.Y+area.H/2)))
		}
	}
}

func TestAStyleChangeThatNeedsNoLayoutMovesWhatIsCulled(t *testing.T) {
	bothWays(t)
	p := newCullPage(t, 1)
	p.look(0)
	up := p.late.Box().Y - 100
	require.Greater(t, up, float32(2000), "far below the viewport")
	laid := ui.Placed(p.doc)

	p.late.SetAttr("style", fmt.Sprintf("box-shadow: 0 %gpx 0 4px #ffffff", -up))
	p.look(0)
	p.late.SetAttr("style", fmt.Sprintf("transform: translateY(%gpx)", -up))
	p.look(0)
	assert.Same(t, p.late, p.doc.ElementAt(geom.V2(200, 110)), "hit where its transform shows it")
	p.late.SetAttr("style", "")
	p.look(0)
	assert.NotSame(t, p.late, p.doc.ElementAt(geom.V2(200, 110)), "and no longer once it is back")
	assert.Equal(t, laid, ui.Placed(p.doc), "none of which asked for a layout")

	p.late.SetAttr("class", "late slow")
	p.w.RunFrames(2)
	p.late.SetAttr("style", fmt.Sprintf("box-shadow: 0 %gpx 0 4px #ffffff", -up))
	for range 3 {
		p.look(0)
	}
	p.late.SetAttr("style", fmt.Sprintf("transform: translateY(%gpx)", -up))
	for range 3 {
		p.look(0)
	}
	assert.Same(t, p.late, p.doc.ElementAt(geom.V2(200, 110)), "where its transition brought it")
}

// randomTree is a tree of boxes with styles drawn from what moves a box,
// cuts it or draws outside it, those that scroll added to scrollers.
func randomTree(r *rand.Rand, depth int, scrollers *[]*ui.Node) *ui.Node {
	pick := func(of ...string) string { return of[r.IntN(len(of))] }
	px := func(lo, hi int) string { return fmt.Sprintf("%dpx", lo+r.IntN(hi-lo+1)) }
	var css []string
	add := func(chance float64, decl string) {
		if r.Float64() < chance {
			css = append(css, decl)
		}
	}
	add(0.7, "height: "+px(0, 260))
	add(0.3, "width: "+px(20, 700))
	add(0.3, "margin: "+px(-120, 60)+" "+px(-40, 40))
	add(0.3, "background: "+pick("#803030", "#308030", "#303080", "#808030"))
	add(0.2, "position: "+pick("relative", "absolute", "fixed", "sticky"))
	add(0.3, "top: "+px(-300, 300))
	add(0.2, "left: "+px(-100, 300))
	add(0.15, "z-index: "+pick("-1", "0", "2"))
	scrolls := r.Float64() < 0.15
	if scrolls {
		css = append(css, "overflow: "+pick("auto", "scroll"))
	}
	add(0.1, "overflow: "+pick("hidden", "visible"))
	add(0.08, "overflow-x: "+pick("hidden", "auto"))
	add(0.12, "transform: "+pick("translate(30px, -200px)", "rotate(20deg)", "scale(1.4)", "translateY(150px)"))
	add(0.12, "float: "+pick("left", "right"))
	add(0.15, "box-shadow: 0 "+px(-200, 200)+" "+px(0, 30)+" #ffffff")
	add(0.15, "outline: "+px(1, 8)+" solid #ff00ff; outline-offset: "+px(-4, 60))
	add(0.1, "text-shadow: 0 "+px(-150, 150)+" 0 #00ffff")
	add(0.1, "opacity: "+pick("0", "0.5"))
	add(0.1, "display: "+pick("inline-block", "flex", "inline", "grid"))
	add(0.05, "column-count: 2")
	add(0.05, "visibility: hidden")
	add(0.05, "pointer-events: none")
	args := []ui.Arg{ui.Style(strings.Join(css, "; "))}
	if r.Float64() < 0.5 {
		args = append(args, ui.Text(pick("text", "a longer line of text that may wrap where its box is narrow", "x")))
	}
	if depth > 0 {
		for range r.IntN(5) {
			args = append(args, randomTree(r, depth-1, scrollers))
		}
	}
	n := ui.Div(args...)
	if scrolls {
		*scrollers = append(*scrollers, n)
	}
	return n
}

func TestCullingChangesNothingRandomPagesDrawOrHit(t *testing.T) {
	bothWays(t)
	for seed := range uint64(40) {
		r := rand.New(rand.NewPCG(seed, 7))
		doc := ui.NewDocument(ui.ScrollingViewport(), ui.Zoom([]float32{1, 1.25, 2}[seed%3]),
			ui.CSS("body { margin: 0; color: #ffffff; }"))
		var scrollers []*ui.Node
		for range 12 {
			doc.Body().Append(randomTree(r, 4, &scrollers))
		}
		win := platform.NewWindow(platform.Size(400, 300))
		win.Add(doc)
		w := ember.New().AddEntities(win)
		w.RunFrames(2)
		high := ui.ScrollSize(doc, doc.HTML()).Y
		for range 12 {
			ui.ScrollTo(doc, doc.HTML(), geom.V2(0, r.Float32()*high))
			for range min(3, len(scrollers)) {
				if n := scrollers[r.IntN(len(scrollers))]; n.Box() != (geom.Rect{}) {
					ui.ScrollTo(doc, n, geom.V2(r.Float32()*200, r.Float32()*400))
				}
			}
			w.RunFrames(1)
			for range 60 {
				doc.ElementAt(geom.V2(r.Float32()*420-10, r.Float32()*320-10))
			}
			win.Input().SetMouse(geom.V2(r.Float32()*400, r.Float32()*300))
			w.RunFrames(1)
		}
		if t.Failed() {
			t.Fatalf("seed %d", seed)
		}
	}
}
