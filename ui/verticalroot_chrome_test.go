//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

const verticalRootPage = `html { margin: 0 30px 0 0; border: 4px solid #335; padding: 6px; }
#head { position: sticky; right: 10px; width: 20px; }
#badge { position: fixed; right: 30px; bottom: 10px; width: 50px; height: 20px; }
#note { position: absolute; top: 20px; left: -1500px; width: 80px; height: 40px; }
#inner { position: fixed; top: 5px; left: 5px; width: 10px; height: 10px; }
#list { overflow: auto; width: 100px; height: 200px; }
.row { width: 30px; }
`

// verticalRootDoc is chromeDoc's page turned: its blocks run along the
// x axis from the side writing-mode starts them on, some 1,400 points
// wide, its viewport scrolled to x.
func verticalRootDoc(x float32, sheet string, opts ...ui.DocumentOption) *ui.Document {
	d := ui.NewDocument(append([]ui.DocumentOption{ui.ScrollingViewport(), ui.CSS(verticalRootPage + sheet)},
		opts...)...)
	list := ui.Div(ui.ID("list"), ui.Div(ui.Style("width: 400px")))
	d.Body().Append(ui.P(ui.Text("Before the head")), ui.Div(ui.ID("head"), ui.Text("Head")),
		ui.Div(ui.ID("badge"), ui.Div(ui.ID("inner"))), ui.Div(ui.ID("note")), list)
	for i := range 40 {
		d.Body().Append(ui.Div(ui.Class("row"), ui.Text(fmt.Sprint("row ", i))))
	}
	dom.Data(d.HTML()).Scroll = geom.V2(x, 0)
	return d
}

const (
	rlRoot = "html { writing-mode: vertical-rl; }"
	lrRoot = "html { writing-mode: vertical-lr; }"
)

func TestAVerticalRootsViewportMatchesChrome(t *testing.T) {
	for name, c := range map[string]struct {
		x     float32
		sheet string
		opts  []ui.DocumentOption
	}{
		"vertical-rl at its start":             {0, rlRoot, nil},
		"vertical-rl scrolled":                 {-700, rlRoot, nil},
		"vertical-rl past its end":             {-1e6, rlRoot, nil},
		"vertical-rl before its start":         {300, rlRoot, nil},
		"vertical-lr at its start":             {0, lrRoot, nil},
		"vertical-lr scrolled":                 {700, lrRoot, nil},
		"vertical-lr past its end":             {1e6, lrRoot, nil},
		"vertical-rl, overlay bars":            {-700, rlRoot, []ui.DocumentOption{ui.OverlayScrollbars()}},
		"vertical-rl, a page that fits":        {0, rlRoot + ".row, #note { display: none; }", nil},
		"vertical-rl, html's overflow hidden":  {-300, rlRoot + "html { overflow: hidden; }", nil},
		"vertical-rl, a stable gutter":         {-700, rlRoot + "html { scrollbar-gutter: stable; }", nil},
		"body's vertical-rl":                   {-700, "body { writing-mode: vertical-rl; }", nil},
		"body's vertical-lr under html's rl":   {700, rlRoot + "body { writing-mode: vertical-lr; }", nil},
		"body's horizontal-tb under html's rl": {0, rlRoot + "body { writing-mode: horizontal-tb; }", nil},
		"vertical-rl, right to left":           {-700, rlRoot + "html { direction: rtl; } #list { display: none; }", nil},
		"vertical-rl, tall lines":              {-700, rlRoot + "#head { height: 900px; }", nil},
	} {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, verticalRootDoc(c.x, c.sheet, c.opts...), geom.V2(640, 400))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestAVerticalRootScrollsAsFarAsChromes(t *testing.T) {
	for name, c := range map[string]struct {
		x     float32
		sheet string
	}{
		"vertical-rl":             {-1e6, rlRoot},
		"vertical-lr":             {1e6, lrRoot},
		"body's vertical-rl":      {-1e6, "body { writing-mode: vertical-rl; }"},
		"vertical-rl, tall lines": {-1e6, rlRoot + "#head { height: 900px; }"},
	} {
		t.Run(name, func(t *testing.T) {
			d := verticalRootDoc(c.x, c.sheet)
			out, err := chrome.Eval(d, geom.V2(640, 400), `const e = document.documentElement;
				return [e.scrollWidth, e.scrollHeight, e.clientWidth, e.clientHeight, scrollX, scrollY];`)
			require.NoError(t, err)
			var want [6]float32
			require.NoError(t, json.Unmarshal(out, &want))
			d.Layout(geom.V2(640, 400))
			got := ui.ScrollSize(d, d.HTML())
			assert.InDelta(t, want[0], got.X, 1, "scrollWidth (Chrome %v)", want)
			assert.InDelta(t, want[1], got.Y, 1, "scrollHeight (Chrome %v)", want)
			assert.InDelta(t, want[2], ui.ClientSize(d, d.HTML()).X, 1, "clientWidth (Chrome %v)", want)
			assert.InDelta(t, want[4], d.Scroll(d.HTML()).X, 1, "scrollX (Chrome %v)", want)
			assert.InDelta(t, want[5], d.Scroll(d.HTML()).Y, 1, "scrollY (Chrome %v)", want)
		})
	}
}

// TestAVerticalRootScrollsByKeysAndWheelAsChromes pages, ends and turns
// the wheel over a vertical root's viewport in both browsers, with
// nothing focused, and compares where each step lands: Space moves
// along the root's blocks, and Home and End on macOS; the page keys,
// Home and End elsewhere, the arrows and the wheel as they point.
func TestAVerticalRootScrollsByKeysAndWheelAsChromes(t *testing.T) {
	type step struct {
		key   string
		wheel geom.Vec2
	}
	steps := []step{{key: "PageDown"}, {key: "PageDown"}, {key: "PageUp"}, {key: "End"}, {key: "Home"},
		{key: " "}, {key: "Home"}, {key: " "}, {key: "ArrowLeft"}, {key: "ArrowLeft"}, {key: "ArrowRight"},
		{key: "ArrowDown"},
		{wheel: geom.V2(0, 100)}, {wheel: geom.V2(-100, 0)}, {wheel: geom.V2(60, 0)}}
	at := geom.V2(300, 330)
	for name, sheet := range map[string]string{"vertical-rl": rlRoot, "vertical-lr": lrRoot,
		"body's vertical-rl": "body { writing-mode: vertical-rl; }"} {
		t.Run(name, func(t *testing.T) {
			doc := verticalRootDoc(0, sheet+"#list { display: none; }")
			srv, _ := chrome.Serve(t, doc, nil, 0)
			win := platform.NewWindow(platform.Size(640, 400))
			win.Add(doc)
			w := ember.New().AddEntities(win)
			in := win.Input()
			in.SetMouse(at)
			w.RunFrames(3)
			s := chrome.Open(t, srv.URL, geom.V2(640, 400), 1)
			s.Move(at.X, at.Y, 0, false)
			for _, st := range steps {
				if st.key != "" {
					in.SetKey(keysOf[st.key], true)
					w.RunFrames(1)
					in.SetKey(keysOf[st.key], false)
					s.PageKey(st.key)
				} else {
					in.AddPreciseWheel(st.wheel.Scale(1.0 / 10))
					s.Wheel(at, st.wheel)
				}
				w.RunFrames(60)
				time.Sleep(700 * time.Millisecond)
				var want [2]float32
				s.Eval(`[scrollX, scrollY]`, &want)
				got := doc.Scroll(doc.HTML())
				what := st.key
				if what == "" {
					what = fmt.Sprint("wheel ", st.wheel)
				}
				assert.InDelta(t, want[0], got.X, 1, "%q: scrollX", what)
				assert.InDelta(t, want[1], got.Y, 1, "%q: scrollY", what)
			}
		})
	}
}

// TestAVerticalRootScrollsByTheDocumentAsChromes scrolls a vertical
// root's viewport as decision 0111's calls do, and Chrome's scrollTo
// and scrollIntoView, and compares where each lands and where the rows
// stand then: ScrollIntoView by the least that shows a row, ScrollToStart
// a row's block start to the viewport's.
func TestAVerticalRootScrollsByTheDocumentAsChromes(t *testing.T) {
	type step struct {
		call string
		row  int
		at   geom.Vec2
	}
	steps := []step{{call: "to", at: geom.V2(-500, 0)}, {call: "view", row: 39}, {call: "view", row: 2},
		{call: "start", row: 20}, {call: "start", row: 39}, {call: "to", at: geom.V2(100, 0)},
		{call: "view", row: 25}}
	for name, sheet := range map[string]string{"vertical-rl": rlRoot, "vertical-lr": lrRoot,
		"body's vertical-rl": "body { writing-mode: vertical-rl; }"} {
		t.Run(name, func(t *testing.T) {
			doc := verticalRootDoc(0, sheet+"#list { display: none; }")
			rows := doc.Body().Children()[5:]
			var script strings.Builder
			script.WriteString(`const rows = [...document.querySelectorAll(".row")], out = [];
const rects = () => rows.map(r => { const b = r.getBoundingClientRect(); return [b.x, b.y]; });
`)
			for _, st := range steps {
				switch st.call {
				case "to":
					fmt.Fprintf(&script, "scrollTo(%g, %g);\n", st.at.X, st.at.Y)
				case "view":
					fmt.Fprintf(&script, "rows[%d].scrollIntoView({block: 'nearest', inline: 'nearest'});\n", st.row)
				case "start":
					fmt.Fprintf(&script, "rows[%d].scrollIntoView({block: 'start', inline: 'nearest'});\n", st.row)
				}
				script.WriteString("out.push([[scrollX, scrollY], rects()]);\n")
			}
			script.WriteString("return out;")
			raw, err := chrome.Eval(doc, geom.V2(640, 400), script.String())
			require.NoError(t, err)
			var want []struct {
				Scroll [2]float32
				Rows   [][2]float32
			}
			var parsed [][2]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &parsed))
			for _, p := range parsed {
				var w struct {
					Scroll [2]float32
					Rows   [][2]float32
				}
				require.NoError(t, json.Unmarshal(p[0], &w.Scroll))
				require.NoError(t, json.Unmarshal(p[1], &w.Rows))
				want = append(want, w)
			}

			win := platform.NewWindow(platform.Size(640, 400))
			win.Add(doc)
			w := ember.New().AddEntities(win)
			w.RunFrames(3)
			require.Len(t, want, len(steps))
			for i, st := range steps {
				switch st.call {
				case "to":
					doc.SetScroll(doc.HTML(), st.at)
				case "view":
					doc.ScrollIntoView(rows[st.row])
				case "start":
					doc.ScrollToStart(rows[st.row])
				}
				w.RunFrames(60)
				what := fmt.Sprintf("step %d, %s %d %v", i, st.call, st.row, st.at)
				got := doc.Scroll(doc.HTML())
				assert.InDelta(t, want[i].Scroll[0], got.X, 1, "%s: scrollX", what)
				assert.InDelta(t, want[i].Scroll[1], got.Y, 1, "%s: scrollY", what)
				for j, r := range rows {
					b := r.Box()
					assert.InDelta(t, want[i].Rows[j][0], b.X, 1, "%s: row %d's x", what, j)
					assert.InDelta(t, want[i].Rows[j][1], b.Y, 1, "%s: row %d's y", what, j)
				}
			}
		})
	}
}
