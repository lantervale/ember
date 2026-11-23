//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestAPressOnAFirstLetterIsChromes presses across the box of a
// ::first-letter, floated, boxed and taking quotes and kept spaces with
// it, and of a text node that is its first letter alone, and compares
// the place the press takes with Chrome's caretPositionFromPoint there;
// then drags from inside each letter, and selects a letter's node whole.
func TestAPressOnAFirstLetterIsChromes(t *testing.T) {
	const sheet = `p { width: 300px; }
.drop::first-letter { float: left; font-size: 40px; line-height: 34px; margin-right: 4px; }
.boxed::first-letter { font-size: 22px; padding: 1px 5px; margin-right: 3px; border: 2px solid #234; }
.quoted::first-letter { font-size: 24px; }
.pre { white-space: pre; }
.pre::first-letter { font-size: 30px; }
.alone::first-letter { font-size: 28px; }`
	ps := []*ui.Node{
		ui.P(ui.Class("drop"), ui.Text("Lantern light over the crypt stairs")),
		ui.P(ui.Class("boxed"), ui.Text("Every step rang hollow")),
		ui.P(ui.Class("quoted"), ui.Text("  “Well,” said the warden")),
		ui.P(ui.Class("pre"), ui.Text("  Under the boots")),
		ui.P(ui.Class("alone"), ui.Text("A"), ui.B(ui.Text("lone")), ui.Text(" letter")),
	}
	l := selectableDoc(t, sheet, ps...)
	var texts []*ui.Node
	l.doc.Body().Walk(func(n *ui.Node) bool {
		if n.IsText() {
			texts = append(texts, n)
		}
		return true
	})
	type caret struct {
		Node   int `json:"node"`
		Offset int `json:"offset"`
	}
	ours := func(p ui.Position) caret { return caret{slices.Index(texts, p.Node), p.Offset} }
	carets := func(at []geom.Vec2) []caret {
		t.Helper()
		points, err := json.Marshal(at)
		require.NoError(t, err)
		out, err := chrome.Eval(l.doc, geom.V2(640, 360), fmt.Sprintf(`
			const texts = [], w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
			while (w.nextNode()) texts.push(w.currentNode);
			return %s.map(p => {
				const c = document.caretPositionFromPoint(p.X, p.Y);
				return {node: texts.indexOf(c.offsetNode), offset: c.offset};
			});`, points))
		require.NoError(t, err)
		var cs []caret
		require.NoError(t, json.Unmarshal(out, &cs))
		return cs
	}
	var presses, drags []geom.Vec2
	var pressed, dragged []caret
	for _, p := range ps {
		b := dom.Data(p).FirstLetter.Box()
		for x := b.X + 1; x < b.X+b.W+4; x += 3 {
			at := geom.V2(x, b.Y+b.H/2)
			l.doc.ClearSelection()
			l.frames(30)
			l.down(at)
			l.up()
			presses, pressed = append(presses, at), append(pressed, ours(l.doc.Selection().Focus))
		}
		from, to := geom.V2(b.X+b.W*0.3, b.Y+b.H/2), geom.V2(b.X+b.W+40, b.Y+b.H/2)
		l.doc.ClearSelection()
		l.frames(30)
		l.dragged(from, to)
		s := l.doc.Selection()
		drags, dragged = append(drags, from, to), append(dragged, ours(s.Anchor), ours(s.Focus))
	}
	for i, c := range carets(presses) {
		assert.Equal(t, c, pressed[i], "pressed at %v", presses[i])
	}
	for i, c := range carets(drags) {
		assert.Equal(t, c, dragged[i], "dragged at %v", drags[i])
	}
	sameSelections(t, l.doc, geom.V2(640, 360), map[string][3]int{
		"a letter's node whole": {4, 0, 1},
		"a pre letter's spaces": {3, 0, 2},
		"a pre letter and more": {3, 1, 6},
	})
}
