//go:build chrome

package ui_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const wholeSheet = `
@font-face { font-family: Featured; src: url(features.ttf); }
@font-face { font-family: Spaced; src: url(spaced.ttf); }
body { user-select: text; }
div { margin: 0 0 4px; font: 32px Featured, monospace; white-space: nowrap; }
div > span { display: inline-block; }
.spaced { font-family: Spaced, monospace; }
.c { color: #c00; } .u { text-decoration: underline; } .bg { background: #ff0; } .rel { position: relative; left: 3px; }
.lh { line-height: 50px; } .zero { padding: 0; border: 0 solid; margin: 0; } .base { vertical-align: baseline; }
.ls0 { letter-spacing: 0; }
.pad { padding-left: 1px; } .marg { margin-left: 2px; } .bord { border-left: 1px solid; } .end { padding-right: 1px; }
.v0 { vertical-align: 0; } .sub { vertical-align: sub; } .ls { letter-spacing: 1px; } .ws { word-spacing: 3px; }
.ffs { font-feature-settings: "kern" 1; } .kern { font-kerning: normal; } .bold { font-weight: 700; }
.nolig { font-variant-ligatures: none; }
`

// wholeLines is a Document of lines of text cut by inline elements:
// each line's parts are plain text or "classes:text", a span of those
// classes, "a/b:text" a span of b in one of a. A line starting with
// "spaced " is in spaced.ttf.
func wholeLines(t *testing.T, lines ...string) (*ui.Document, [][]*ui.Node) {
	t.Helper()
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(wholeSheet))
	var texts [][]*ui.Node
	for _, line := range lines {
		var args []ui.Arg
		if rest, ok := strings.CutPrefix(line, "spaced "); ok {
			args, line = append(args, ui.Class("spaced")), rest
		}
		var parts []ui.Arg
		for _, part := range strings.Split(line, "|") {
			classes, text, ok := strings.Cut(part, ":")
			if !ok {
				parts = append(parts, ui.Text(part))
				continue
			}
			el := ui.Text(text)
			cs := strings.Split(classes, "/")
			for i := len(cs) - 1; i >= 0; i-- {
				el = ui.Span(ui.Class(cs[i]), el)
			}
			parts = append(parts, el)
		}
		div := ui.Div(append(args, ui.Span(parts...))...)
		d.Body().Append(div)
		var nodes []*ui.Node
		div.Walk(func(n *ui.Node) bool {
			if n.IsText() {
				nodes = append(nodes, n)
			}
			return true
		})
		texts = append(texts, nodes)
	}
	return d, texts
}

func matchWhole(t *testing.T, lines ...string) {
	t.Helper()
	d, _ := wholeLines(t, lines...)
	size := geom.V2(500, 60*float32(len(lines)))
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestKerningAndLigaturesCrossAnInlineElementsEdgeAsInChrome(t *testing.T) {
	matchWhole(t,
		"A|c:V", "AVA|c:TAR|AVA", "c:A|c:V", "T|c:o", "c:T|o",
		"f|c:i", "of|c:fi|ce", "o|c:f|fice", "f|c:f|i", "c:f|c:f|c:i",
		"f|c:|i", "o|c: |A", "c/u:f|i", "c/u:f|bg:i", "AVATAR|c:LTd WAY|To",
	)
}

func TestShapingRunsOnWhereChromesDoesAcrossEdges(t *testing.T) {
	matchWhole(t,
		"f|c:i", "f|u:i", "f|bg:i", "f|rel:i", "f|lh:i", "f|zero:i", "f|base:i", "f|ls0:i",
		"A|u:V", "A|zero:V", "A|base:V", "o|c:ffi",
	)
}

func TestShapingBreaksWhereChromesDoesAtAnEdge(t *testing.T) {
	matchWhole(t,
		"f|pad:i", "f|marg:i", "f|bord:i", "end:f|i", "f|v0:i", "f|sub:i", "f|ls:i", "f|ws:i",
		"f|ffs:i", "f|kern:i", "f|bold:i", "f|nolig:i",
		"A|pad:V", "A|marg:V", "A|bord:V", "end:A|V", "A|v0:V", "A|ls:V", "A|ws:V", "A|ffs:V", "A|kern:V",
	)
}

func TestALigatureAndAlternatesReachAcrossASpaceAsInChrome(t *testing.T) {
	matchWhole(t,
		"spaced x x", "spaced z z", "spaced q q", "spaced a x x b", "spaced zz z q qq",
		"spaced x|c: x", "spaced x |c:x", "spaced z|c: z", "spaced z |c:z", "spaced q|c: q", "spaced q |c:q",
	)
}

// TestCaretsInAndBesideALigatureCutByElementsAreChromes compares where
// each character of each text node starts with Chrome's range rects: a
// ligature across an edge is the first element's, the characters of
// the rest at its end, as Chrome places them.
func TestCaretsInAndBesideALigatureCutByElementsAreChromes(t *testing.T) {
	lines := []string{"A|c:V", "of|c:fi|ce", "f|c:i", "o|c:f|fice", "f|c:f|i", "T|c:o", "spaced x|c: x", "spaced z|c: z"}
	d, texts := wholeLines(t, lines...)
	size := geom.V2(500, 60*float32(len(lines)))
	landed(t, d, size)
	out, err := chrome.Eval(d, size, `
const res = [];
for (const div of document.querySelectorAll("div")) {
  const walker = document.createTreeWalker(div, NodeFilter.SHOW_TEXT);
  let n; while ((n = walker.nextNode())) {
    const xs = [];
    for (let i = 0; i <= n.length; i++) {
      const q = document.createRange(); q.setStart(n, 0); q.setEnd(n, i);
      const b = q.getBoundingClientRect(); xs.push(i ? b.width : 0);
    }
    res.push(xs);
  }
}
return res;`)
	require.NoError(t, err)
	var want [][]float32
	require.NoError(t, json.Unmarshal(out, &want))
	k := 0
	for li, nodes := range texts {
		for _, n := range nodes {
			require.Less(t, k, len(want))
			for i := 1; i < len(want[k]); i++ {
				d.Select(ui.Position{Node: n, Offset: 0}, ui.Position{Node: n, Offset: i})
				var w float32
				for _, r := range d.SelectionRects() {
					w += r.W
				}
				assert.InDelta(t, want[k][i], w, 0.51, "line %q, %q to %d", lines[li], n.Text(), i)
			}
			k++
		}
	}
}
