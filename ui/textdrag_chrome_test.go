//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const textDragSheet = `@font-face { font-family: Go; src: url(go.ttf); }
body { margin: 10px; font: 20px/30px Go; user-select: text; } p { margin: 0 0 10px; }
textarea, input { font: 20px Go; width: 300px; padding: 2px; border: 1px solid; margin: 0 0 10px; display: block; }
textarea { height: 60px; resize: none; }`

// textDragState is what a drag of selected text leaves, read alike from
// both: the text field focused and its selection, the fields' values,
// and the page's selection while no field holds the focus, its focus
// offset in #p, -1 elsewhere.
type textDragState struct {
	Focus string `json:"focus"`
	T     string `json:"t"`
	I     string `json:"i"`
	Range []int  `json:"range"`
	Page  string `json:"page"`
	Caret int    `json:"caret"`
}

const chromeTextDragState = `(() => {
	const a = document.activeElement, f = a && (a.id === "t" || a.id === "i") ? a : null, s = getSelection();
	const p = document.getElementById("p").firstChild;
	return JSON.stringify({focus: f ? f.id : "", t: document.getElementById("t").value,
		i: document.getElementById("i").value, range: f ? [f.selectionStart, f.selectionEnd] : [],
		page: f ? "" : s.toString(), caret: !f && s.focusNode === p ? s.focusOffset : -1});
})()`

// spot is a point in one of the page's texts: the middle of character i
// of the element id, or the edge before it when edge (after it when
// after), nudged by by.
type spot struct {
	id    string
	i     int
	edge  bool
	after bool
	by    geom.Vec2
}

func mid(id string, i int) spot        { return spot{id: id, i: i} }
func edgeOf(id string, i int) spot     { return spot{id: id, i: i, edge: true} }
func afterOf(id string, i int) spot    { return spot{id: id, i: i, after: true} }
func (s spot) nudge(x, y float32) spot { s.by = geom.V2(x, y); return s }

type textDragCase struct {
	name string
	// page selects characters of #p, or field (t or i) selects its own.
	page  [2]int
	field string
	sel   [2]int
	// value is #t's text, when not its own.
	value string
	press spot
	// hold has the press held still long enough to drag on macOS first.
	hold bool
	path []spot
	// copy holds the platform's copy key.
	copy bool
	mac  bool
}

// TestADragOfSelectedTextIsChromes drags the page's selected text and a
// field's in ember and in Chrome, Chrome handing its drag over to be
// dropped where the pointer is let go, and compares what each leaves.
func TestADragOfSelectedTextIsChromes(t *testing.T) {
	q, t5 := mid("q", 3), edgeOf("t", 5)
	for _, c := range []textDragCase{
		{name: "the page's text into a textarea", page: [2]int{4, 9}, press: mid("p", 6), hold: true,
			path: []spot{mid("p", 6).nudge(4, 0), t5}},
		{name: "the page's text into an input", page: [2]int{4, 9}, press: mid("p", 6), hold: true,
			path: []spot{mid("p", 6).nudge(0, 5), edgeOf("i", 2)}},
		{name: "the page's text onto text", page: [2]int{4, 9}, press: mid("p", 6), hold: true,
			path: []spot{mid("p", 6).nudge(4, 0), q}},
		{name: "a press beside the selection's start", page: [2]int{4, 9}, press: edgeOf("p", 4).nudge(-1, 0),
			hold: true, path: []spot{mid("p", 4).nudge(0, 6), t5}},
		{name: "a press and release", page: [2]int{4, 9}, press: mid("p", 6), hold: true},
		{name: "a move short of a drag", page: [2]int{4, 9}, press: mid("p", 6), hold: true,
			path: []spot{mid("p", 6).nudge(2, 2)}},
		{name: "a field's text into another", field: "t", sel: [2]int{4, 7}, press: mid("t", 5), hold: true,
			path: []spot{mid("t", 5).nudge(4, 0), edgeOf("i", 2)}},
		{name: "a field's text copied into another", field: "t", sel: [2]int{4, 7}, press: mid("t", 5), hold: true,
			copy: true, path: []spot{mid("t", 5).nudge(4, 0), edgeOf("i", 2)}},
		{name: "a field's text within it", field: "t", sel: [2]int{0, 3}, press: mid("t", 1), hold: true,
			path: []spot{mid("t", 1).nudge(4, 0), afterOf("t", 12)}},
		{name: "a field's text back onto itself", field: "t", sel: [2]int{4, 9}, press: mid("t", 5), hold: true,
			path: []spot{mid("t", 5).nudge(4, 0), mid("t", 7)}},
		{name: "a field's text onto the page", field: "t", sel: [2]int{4, 7}, press: mid("t", 5), hold: true,
			path: []spot{mid("t", 5).nudge(4, 0), q}},
		{name: "a field's press and release", field: "t", sel: [2]int{4, 7}, press: mid("t", 5), hold: true},
		{name: "a word moved out of the middle", field: "t", sel: [2]int{4, 7}, value: "one two, three",
			press: mid("t", 5), hold: true, path: []spot{mid("t", 5).nudge(4, 0), afterOf("i", 4)}},
		{name: "a quick long move takes the caret along", page: [2]int{4, 9}, press: mid("p", 6), mac: true,
			path: []spot{edgeOf("p", 10), edgeOf("p", 13)}},
		{name: "a quick long move in a field", field: "t", sel: [2]int{4, 7}, press: mid("t", 5), mac: true,
			path: []spot{edgeOf("t", 9), edgeOf("t", 11)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.mac && runtime.GOOS != "darwin" {
				t.Skip("Chrome waits before a drag of text on macOS only")
			}
			got, want := emberTextDrag(t, c), chromeTextDrag(t, c)
			assert.Equal(t, want, got)
		})
	}
}

func textDragOptions() []ui.DocumentOption {
	return []ui.DocumentOption{ui.CSS(textDragSheet),
		ui.FetchFonts(func(_, _ string) ([]byte, error) { return goregular.TTF, nil })}
}

func textDragNodes(c textDragCase) (map[string]*ui.Node, []*ui.Node) {
	value := c.value
	if value == "" {
		value = "one two three"
	}
	nodes := map[string]*ui.Node{
		"p": ui.P(ui.ID("p"), ui.Text("The quick brown fox")), "q": ui.P(ui.ID("q"), ui.Text("jumps over")),
		"t": ui.Textarea(ui.ID("t"), ui.Text(value)), "i": ui.Input(ui.ID("i"), ui.Attr("value", "field")),
	}
	return nodes, []*ui.Node{nodes["p"], nodes["q"], nodes["t"], nodes["i"]}
}

// textDragLive is the case's page in a window, its font loaded.
func textDragLive(t *testing.T, c textDragCase) (*live, map[string]*ui.Node) {
	t.Helper()
	nodes, order := textDragNodes(c)
	l := liveDocWith(t, textDragOptions(), order...)
	require.Eventually(t, func() bool {
		l.frames(1)
		return l.doc.FontsLoading() == 0
	}, 5*time.Second, 5*time.Millisecond, "the font never landed")
	l.frames(2)
	return l, nodes
}

// textDragPoints are the case's press and path in the page's points, as
// ember lays it out.
func textDragPoints(t *testing.T, c textDragCase) []geom.Vec2 {
	t.Helper()
	l, nodes := textDragLive(t, c)
	var out []geom.Vec2
	for _, s := range append([]spot{c.press}, c.path...) {
		var r geom.Rect
		if n := nodes[s.id]; s.id == "t" || s.id == "i" {
			l.doc.SetSelectionRange(n, s.i, s.i+1)
			rects := ui.FieldSelectionRects(l.doc, n)
			require.NotEmpty(t, rects)
			r = rects[0]
		} else {
			l.doc.Select(ui.Position{Node: textOf(n), Offset: s.i}, ui.Position{Node: textOf(n), Offset: s.i + 1})
			r = l.doc.SelectionRects()[0]
		}
		at := geom.V2(r.X+r.W/2, r.Y+r.H/2)
		switch {
		case s.edge:
			at.X = r.X + 0.5
		case s.after:
			at.X = r.X + r.W - 0.5
		}
		out = append(out, at.Add(s.by))
	}
	return out
}

func copyMods() (int, input.Key) {
	if runtime.GOOS == "darwin" {
		return chrome.Alt, input.KeyLeftAlt
	}
	return chrome.Ctrl, input.KeyLeftControl
}

func emberTextDrag(t *testing.T, c textDragCase) textDragState {
	t.Helper()
	pts := textDragPoints(t, c)
	l, nodes := textDragLive(t, c)
	if c.field != "" {
		l.doc.Focus(nodes[c.field])
		l.frames(1)
		l.doc.SetSelectionRange(nodes[c.field], c.sel[0], c.sel[1])
	} else {
		p := textOf(nodes["p"])
		l.doc.Select(ui.Position{Node: p, Offset: c.page[0]}, ui.Position{Node: p, Offset: c.page[1]})
	}
	l.frames(1)
	_, key := copyMods()
	if c.copy {
		l.win.Input().SetKey(key, true)
	}
	hold := 0
	if c.hold {
		hold = 20
	}
	l.dragFrom(pts[0], hold, pts[1:]...)
	l.up()
	l.win.Input().SetKey(key, false)
	l.frames(1)
	st := textDragState{T: l.doc.Value(nodes["t"]), I: l.doc.Value(nodes["i"]), Range: []int{}, Caret: -1}
	for _, id := range []string{"t", "i"} {
		if l.doc.Focused(nodes[id]) {
			from, to := l.doc.SelectionRange(nodes[id])
			st.Focus, st.Range = id, []int{from, to}
		}
	}
	if st.Focus == "" {
		s := l.doc.Selection()
		st.Page = s.Text
		if s.Focus.Node == textOf(nodes["p"]) {
			st.Caret = s.Focus.Offset
		}
	}
	return st
}

func chromeTextDrag(t *testing.T, c textDragCase) textDragState {
	t.Helper()
	pts := textDragPoints(t, c)
	_, order := textDragNodes(c)
	d := ui.NewDocument(textDragOptions()...)
	d.Body().Append(order...)
	srv, _ := chrome.Serve(t, d, map[string][]byte{"go.ttf": goregular.TTF}, 0)
	s := chrome.Open(t, srv.URL, geom.V2(640, 360), 1)
	s.Eval(`document.fonts.ready.then(() => 1)`, nil)
	s.InterceptDrags()
	if c.field != "" {
		s.Eval(fmt.Sprintf(`(() => { const f = document.getElementById(%q); f.focus();
			f.setSelectionRange(%d, %d); return 1; })()`, c.field, c.sel[0], c.sel[1]), nil)
	} else {
		s.Eval(fmt.Sprintf(`(() => { const p = document.getElementById("p").firstChild;
			getSelection().setBaseAndExtent(p, %d, p, %d); return 1; })()`, c.page[0], c.page[1]), nil)
	}
	mods := 0
	if c.copy {
		mods, _ = copyMods()
	}
	s.Press(pts[0].X, pts[0].Y, mods)
	if c.hold {
		time.Sleep(300 * time.Millisecond)
	}
	for _, p := range pts[1:] {
		// a frame or two apart, as ember takes each in a frame of its own:
		// Chrome coalesces the moves of one frame
		time.Sleep(35 * time.Millisecond)
		s.Move(p.X, p.Y, mods, true)
	}
	last := pts[len(pts)-1]
	s.Drop(last.X, last.Y, mods)
	var out string
	s.Eval(chromeTextDragState, &out)
	var st textDragState
	require.NoError(t, json.Unmarshal([]byte(out), &st))
	return st
}
