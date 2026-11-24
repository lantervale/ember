//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const verticalSelectionSheet = `@font-face { font-family: CJK; src: url(cjk.ttf); }
body { margin: 8px; user-select: text; font: 16px/20px CJK; display: flex; gap: 14px; align-items: flex-start; }
p { margin: 0; block-size: 70px; inline-size: 150px; }
.rl { writing-mode: vertical-rl; } .lr { writing-mode: vertical-lr; } .slr { writing-mode: sideways-lr; }
.srl { writing-mode: sideways-rl; } .h { writing-mode: horizontal-tb; }`

// verticalSelectionPage is a live page of paragraphs in each vertical
// writing mode, of turned Latin and of upright ideographs (cjkFont's,
// which both draw alike), and its paragraphs.
func verticalSelectionPage(t *testing.T) (*live, []*ui.Node) {
	t.Helper()
	font := cjkFont(t)
	ps := []*ui.Node{
		ui.P(ui.Class("rl"), ui.Text("The quick brown fox jumps over the lazy dog")),
		ui.P(ui.Class("lr"), ui.Text("Lantern light over the crypt stairs")),
		ui.P(ui.Class("slr"), ui.Text("Every step rang hollow below")),
		ui.P(ui.Class("srl"), ui.Text("Sideways down the wall")),
		ui.P(ui.Class("rl"), ui.Text("一丁丂七 丄丅丆 Ab 万丈三上下 不与丏丐丑")),
		ui.P(ui.Class("lr"), ui.Text("丏丐 mixed 丑丒 lines 专且丕世")),
	}
	l := liveDocWith(t, []ui.DocumentOption{ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(verticalSelectionSheet)}, ps...)
	for range 200 {
		if l.doc.FontsLoading() == 0 {
			break
		}
		l.frames(1)
	}
	require.Zero(t, l.doc.FontsLoading())
	l.frames(2)
	return l, ps
}

type verticalCaret struct {
	Node   int `json:"node"`
	Offset int `json:"offset"`
}

// verticalTexts are d's text nodes in tree order, as Chrome's walk lists them.
func verticalTexts(d *ui.Document) []*ui.Node {
	var texts []*ui.Node
	d.Body().Walk(func(n *ui.Node) bool {
		if n.IsText() {
			texts = append(texts, n)
		}
		return true
	})
	return texts
}

// TestAPressOnVerticalLinesIsChromes presses across paragraphs of each
// vertical writing mode, past their lines' ends and between them, and
// compares the place each press takes with Chrome's
// caretPositionFromPoint there.
func TestAPressOnVerticalLinesIsChromes(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Chrome 156 on Windows runs the test face's last lines further
		// down than ui and Chrome on macOS: 242 of 1,428 presses past ui's
		// line end land inside Chrome's (docs/05-ui/10-missing.md).
		t.Skip("the test face's vertical lines are as long as Chrome on macOS's")
	}
	l, ps := verticalSelectionPage(t)
	texts := verticalTexts(l.doc)
	var at []geom.Vec2
	var got []verticalCaret
	for _, p := range ps {
		b := p.Box()
		for x := b.X + 1; x < b.X+b.W; x += 5 {
			for y := b.Y + 1; y < b.Y+b.H; y += 9 {
				l.down(geom.V2(x, y))
				l.up()
				f := l.doc.Selection().Focus
				at, got = append(at, geom.V2(x, y)), append(got, verticalCaret{slices.Index(texts, f.Node), f.Offset})
			}
		}
	}
	want := chromeCarets(t, l.doc, at)
	misses := 0
	for i := range want {
		if !assert.Equal(t, want[i], got[i], "pressed at %v", at[i]) {
			misses++
		}
	}
	t.Logf("%d presses, %d unlike Chrome's", len(at), misses)
}

func chromeCarets(t *testing.T, d *ui.Document, at []geom.Vec2) []verticalCaret {
	t.Helper()
	points, err := json.Marshal(at)
	require.NoError(t, err)
	out, err := chrome.Eval(d, geom.V2(640, 360), fmt.Sprintf(`
		const texts = [], w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
		while (w.nextNode()) texts.push(w.currentNode);
		return %s.map(p => {
			const c = document.caretPositionFromPoint(p.X, p.Y);
			return {node: texts.indexOf(c.offsetNode), offset: c.offset};
		});`, points))
	require.NoError(t, err)
	var cs []verticalCaret
	require.NoError(t, json.Unmarshal(out, &cs))
	return cs
}

// characterAt is a point of the rectangle a selection of character i of
// p's text draws, across its middle and a third of the way along from its
// start, where no rounding can take it to the character's other edge;
// of the next where a line's end drops it.
func characterAt(d *ui.Document, p *ui.Node, i int) geom.Vec2 {
	defer d.ClearSelection()
	n := p.Children()[0]
	d.Select(ui.Position{Node: n, Offset: i}, ui.Position{Node: n, Offset: i + 1})
	for len(d.SelectionRects()) == 0 {
		i++
		d.Select(ui.Position{Node: n, Offset: i}, ui.Position{Node: n, Offset: i + 1})
	}
	r := d.SelectionRects()[0]
	along := r.Y + r.H/3
	if slices.Contains(p.Classes(), "slr") {
		along = r.Y + r.H*2/3
	}
	return geom.V2(r.X+r.W/2, along)
}

// chromeSession opens d in Chrome, its fonts loaded and nothing selected.
func chromeSession(t *testing.T, d *ui.Document) *chrome.Session {
	t.Helper()
	srv, _ := chrome.Serve(t, d, map[string][]byte{"cjk.ttf": cjkFont(t)}, 0)
	s := chrome.Open(t, srv.URL, geom.V2(640, 360), 1)
	s.Eval(`document.fonts.ready`, nil)
	// Chrome takes the first press of a page as one at the next place the
	// pointer goes: a click on the margin takes it.
	s.Mouse("mouseMoved", geom.V2(2, 2), 0)
	s.Mouse("mousePressed", geom.V2(2, 2), 1)
	s.Mouse("mouseReleased", geom.V2(2, 2), 1)
	return s
}

// chromeAct runs act on s from nothing selected, and returns
// getSelection().toString() after it.
func chromeAct(s *chrome.Session, act func()) string {
	s.Eval(`getSelection().removeAllRanges();
		new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`, nil)
	act()
	var text string
	s.Eval(`getSelection().toString()`, &text)
	return text
}

// dragSteps is how many moves a drag makes on its way.
const dragSteps = 10

// TestADragAcrossVerticalLinesSelectsAsChrome drags along and across the
// lines of each vertical writing mode, and from one paragraph into the
// next, and compares the text selected with Chrome's after the same
// mouse events.
func TestADragAcrossVerticalLinesSelectsAsChrome(t *testing.T) {
	l, ps := verticalSelectionPage(t)
	type drag struct{ from, to geom.Vec2 }
	var drags []drag
	for _, p := range ps {
		at := func(i int) geom.Vec2 { return characterAt(l.doc, p, i) }
		drags = append(drags, drag{at(1), at(6)}, drag{at(3), at(18)}, drag{at(21), at(8)})
	}
	drags = append(drags, drag{characterAt(l.doc, ps[0], 12), characterAt(l.doc, ps[2], 9)})
	var got []string
	for _, dr := range drags {
		l.doc.ClearSelection()
		l.frames(30)
		l.down(dr.from)
		for i := 1; i <= dragSteps; i++ {
			l.moveTo(dr.from.Add(dr.to.Sub(dr.from).Scale(float32(i) / dragSteps)))
		}
		l.up()
		got = append(got, l.doc.Selection().Text)
	}
	s := chromeSession(t, l.doc)
	for i, dr := range drags {
		want := chromeAct(s, func() {
			s.Mouse("mouseMoved", dr.from, 0)
			s.Mouse("mousePressed", dr.from, 1)
			for j := 1; j <= dragSteps; j++ {
				s.Mouse("mouseMoved", dr.from.Add(dr.to.Sub(dr.from).Scale(float32(j)/dragSteps)), 1)
			}
			s.Mouse("mouseReleased", dr.to, 1)
		})
		assert.Equal(t, want, got[i], "dragged from %v to %v", dr.from, dr.to)
	}
}

// TestADoubleClickOnVerticalLinesTakesChromesWord double clicks inside
// words of each vertical writing mode and compares the word taken with
// Chrome's after the same clicks.
func TestADoubleClickOnVerticalLinesTakesChromesWord(t *testing.T) {
	l, ps := verticalSelectionPage(t)
	var at []geom.Vec2
	// inside Latin words: Chrome takes words of ideographs by ICU's dictionary
	for i, chars := range [][]int{{1, 11}, {1, 9}, {1, 9}, {1, 9}, {9}, {5, 14}} {
		for _, c := range chars {
			at = append(at, characterAt(l.doc, ps[i], c))
		}
	}
	var got []string
	for _, a := range at {
		l.doc.ClearSelection()
		l.frames(30)
		for range 2 {
			l.down(a)
			l.up()
		}
		got = append(got, l.doc.Selection().Text)
	}
	s := chromeSession(t, l.doc)
	for i, a := range at {
		w := chromeAct(s, func() {
			s.Mouse("mouseMoved", a, 0)
			s.Mouse("mousePressed", a, 1)
			s.Mouse("mouseReleased", a, 1)
			s.Mouse("mousePressed", a, 2)
			s.Mouse("mouseReleased", a, 2)
		})
		assert.Equal(t, w, got[i], "double clicked at %v", a)
	}
}

// TestShiftArrowsMoveAlongAndAcrossVerticalLinesAsChrome selects each
// vertical paragraph from its start to the middle of its second line,
// then moves the selection's end with Shift and each arrow key, and
// compares the text then selected with Chrome's after the same key.
// Chrome extends a selection only while it is not collapsed: no move
// here takes its end back to its start.
func TestShiftArrowsMoveAlongAndAcrossVerticalLinesAsChrome(t *testing.T) {
	l, ps := verticalSelectionPage(t)
	keys := map[string]input.Key{"ArrowUp": input.KeyUp, "ArrowDown": input.KeyDown,
		"ArrowLeft": input.KeyLeft, "ArrowRight": input.KeyRight}
	s := chromeSession(t, l.doc)
	for i, p := range ps {
		n := p.Children()[0]
		end := secondLineMiddle(t, l.doc, n)
		for name, k := range keys {
			l.doc.Select(ui.Position{Node: n, Offset: 0}, ui.Position{Node: n, Offset: end})
			l.frames(1)
			l.held(input.KeyLeftShift, k)
			got := l.doc.Selection().Text
			want := chromeAct(s, func() {
				s.Eval(fmt.Sprintf(`{
					const n = document.querySelectorAll("p")[%d].firstChild;
					getSelection().setBaseAndExtent(n, 0, n, %d);
				}`, i, end), nil)
				s.ShiftArrow(name)
			})
			assert.Equal(t, want, got, "paragraph %d, from its start to %d, Shift+%s", i, end, name)
		}
	}
}

// secondLineMiddle is the place in n's text before the character
// nearest the middle of its second line.
func secondLineMiddle(t *testing.T, d *ui.Document, n *ui.Node) int {
	t.Helper()
	defer d.ClearSelection()
	size := len([]rune(n.Text()))
	d.Select(ui.Position{Node: n, Offset: 0}, ui.Position{Node: n, Offset: size})
	lines := d.SelectionRects()
	require.GreaterOrEqual(t, len(lines), 2)
	mid := geom.V2(lines[1].X+lines[1].W/2, lines[1].Y+lines[1].H/2)
	best, near := 0, float32(1e9)
	for i := range size {
		d.Select(ui.Position{Node: n, Offset: i}, ui.Position{Node: n, Offset: i + 1})
		if rs := d.SelectionRects(); len(rs) > 0 {
			if far := geom.V2(rs[0].X+rs[0].W/2, rs[0].Y+rs[0].H/2).Sub(mid).Len(); far < near {
				best, near = i, far
			}
		}
	}
	return best
}

// TestASelectionOfVerticalLinesIsDrawnInChromesRects selects spans of
// each vertical paragraph, within a line and across lines, and compares
// the rectangles drawn with Chrome's range rects.
func TestASelectionOfVerticalLinesIsDrawnInChromesRects(t *testing.T) {
	l, ps := verticalSelectionPage(t)
	l.doc.Select(ui.Position{Node: ps[0].Children()[0], Offset: 3}, ui.Position{Node: ps[0].Children()[0], Offset: 4})
	space := l.doc.SelectionRects()[0].H
	require.Greater(t, space, float32(2))
	for i, p := range ps {
		n := p.Children()[0]
		for _, span := range [][2]int{{2, 7}, {4, 26}, {0, 1}} {
			l.doc.Select(ui.Position{Node: n, Offset: span[0]}, ui.Position{Node: n, Offset: span[1]})
			out, err := chrome.Eval(l.doc, geom.V2(640, 360), `
				return [...getSelection().getRangeAt(0).getClientRects()].map(r => [r.x, r.y, r.width, r.height])`)
			require.NoError(t, err)
			var all [][4]float32
			require.NoError(t, json.Unmarshal(out, &all))
			var want []geom.Rect
			for _, r := range all {
				if r[3] > 0 {
					want = append(want, geom.Rect{X: r[0], Y: r[1], W: r[2], H: r[3]})
				}
			}
			got := l.doc.SelectionRects()
			name := fmt.Sprintf("paragraph %d, %d to %d", i, span[0], span[1])
			if !assert.Len(t, got, len(want), "%s: ember %v, Chrome %v", name, got, want) {
				continue
			}
			for j, w := range want {
				g := got[j]
				assert.InDelta(t, w.X+w.W/2, g.X+g.W/2, 1.01, "%s: rect %d on Chrome's line: %v %v", name, j, g, w)
				assert.GreaterOrEqual(t, g.W, w.W-0.01, "%s: rect %d as wide as its line, Chrome's as its glyphs", name, j)
				if extra := g.H - w.H; extra > 0.51 {
					// a space more past the line's end, below it or, up a line of sideways-lr, above
					assert.InDelta(t, space, extra, 0.51, "%s: rect %d is Chrome's and a space: %v %v", name, j, g, w)
					assert.True(t, abs32(w.Y-g.Y) < 0.51 || abs32(w.Y+w.H-g.Y-g.H) < 0.51,
						"%s: rect %d starts or ends where Chrome's does: %v %v", name, j, g, w)
					continue
				}
				assert.InDelta(t, w.Y, g.Y, 0.51, "%s: rect %d starts where Chrome's does: %v %v", name, j, g, w)
				assert.InDelta(t, w.H, g.H, 0.51, "%s: rect %d as long as Chrome's: %v %v", name, j, g, w)
			}
		}
	}
}

// TestTheMarksOfVerticalLinesAreChromes selects the room of kept tabs,
// the empty lines of preformatted text and a br alone on its line, in
// each vertical writing mode, and compares each line's part with
// Chrome's range rects there: on the same line, from the same place,
// a tab's room as long as Chrome's.
func TestTheMarksOfVerticalLinesAreChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { margin: 8px; user-select: text; font-size: 16px; line-height: 20px; display: flex;
	gap: 10px; align-items: flex-start; }
pre, p { margin: 0; font: inherit; block-size: 120px; } .wrap { white-space: pre-wrap; }
.rl { writing-mode: vertical-rl; } .lr { writing-mode: vertical-lr; } .slr { writing-mode: sideways-lr; }`))
	d.Body().Append(
		ui.Pre(ui.Class("rl"), ui.Text("a\tb")),
		ui.Pre(ui.Class("lr", "wrap"), ui.Text("one\t\ttwo")),
		ui.Pre(ui.Class("slr"), ui.Text("x\ty")),
		ui.Pre(ui.Class("rl"), ui.Text("ab\n\ncd\n\n\nef")),
		ui.Pre(ui.Class("lr"), ui.Text("ab\n\ncd")),
		ui.Pre(ui.Class("slr"), ui.Text("ab\n\n\ncd")),
		ui.P(ui.Class("rl"), ui.Text("ab"), ui.Br(), ui.Br(), ui.Text("cd")),
		ui.P(ui.Class("lr"), ui.Text("ab"), ui.Br(), ui.Br(), ui.Br(), ui.Text("cd")),
		ui.P(ui.Class("slr"), ui.Text("ab"), ui.Br(), ui.Br(), ui.Text("cd")),
	)
	size := geom.V2(400, 300)
	d.Layout(size)
	texts := verticalTexts(d)
	spans := map[string][4]int{
		"a tab whole":                   {0, 0, 0, 3},
		"the tab alone":                 {0, 1, 0, 2},
		"from the tab":                  {0, 1, 0, 3},
		"two tabs":                      {1, 2, 1, 6},
		"the second tab":                {1, 4, 1, 5},
		"a tab up the line":             {2, 1, 2, 2},
		"empty lines, vertical-rl":      {3, 1, 3, 11},
		"empty lines, vertical-lr":      {4, 0, 4, 5},
		"empty lines, sideways-lr":      {5, 1, 5, 6},
		"a br alone, vertical-rl":       {6, 1, 7, 1},
		"brs alone, vertical-lr":        {8, 1, 9, 1},
		"a br alone, sideways-lr":       {10, 1, 11, 1},
		"from the start of a br's line": {6, 0, 7, 1},
	}
	for name, span := range spans {
		d.Select(ui.Position{Node: texts[span[0]], Offset: span[1]}, ui.Position{Node: texts[span[2]], Offset: span[3]})
		out, err := chrome.Eval(d, size, `
			return [...getSelection().getRangeAt(0).getClientRects()].map(r => [r.x, r.y, r.width, r.height])`)
		require.NoError(t, err)
		var all [][4]float32
		require.NoError(t, json.Unmarshal(out, &all))
		var want []geom.Rect
		for _, r := range all {
			want = append(want, geom.Rect{X: r[0], Y: r[1], W: r[2], H: r[3]})
		}
		g, w := verticalLines(d.SelectionRects()), verticalLines(want)
		if !assert.Len(t, g, len(w), "%s: ember %v, Chrome %v", name, g, w) {
			continue
		}
		upward := span[0] == 2 || span[0] == 5 || span[0] == 10
		for i := range w {
			assert.InDelta(t, w[i].X+w[i].W/2, g[i].X+g[i].W/2, 1.01, "%s: line %d is Chrome's: %v %v", name, i, g[i], w[i])
			if upward {
				assert.InDelta(t, w[i].Y+w[i].H, g[i].Y+g[i].H, 0.51, "%s: line %d starts where Chrome's does: %v %v",
					name, i, g[i], w[i])
			} else {
				assert.InDelta(t, w[i].Y, g[i].Y, 0.51, "%s: line %d starts where Chrome's does: %v %v", name, i, g[i], w[i])
			}
			if span[0] < 3 {
				assert.InDelta(t, w[i].H, g[i].H, 0.51, "%s: line %d as long as Chrome's: %v %v", name, i, g[i], w[i])
			}
		}
	}
}

// verticalLines joins the rectangles of each vertical line into one, by
// their middle across the line, from right to left.
func verticalLines(rs []geom.Rect) []geom.Rect {
	var out []geom.Rect
	for _, r := range rs {
		i := slices.IndexFunc(out, func(o geom.Rect) bool { return abs32(o.X+o.W/2-r.X-r.W/2) < 3 })
		if i < 0 {
			out = append(out, r)
			continue
		}
		o := out[i]
		y0, y1 := min(o.Y, r.Y), max(o.Y+o.H, r.Y+r.H)
		out[i] = geom.Rect{X: o.X, Y: y0, W: o.W, H: y1 - y0}
	}
	slices.SortFunc(out, func(a, b geom.Rect) int { return int(b.X - a.X) })
	return out
}
