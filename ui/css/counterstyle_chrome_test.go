//go:build chrome

package css_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestComplexCounterStylesWriteAsChromes sets each value Chrome writes
// in a complex predefined style beside the text ui writes for it, in the
// same font, and has Chrome measure the two: a digit, a marker, a zero or
// a space more or less shows as a width.
func TestComplexCounterStylesWriteAsChromes(t *testing.T) {
	names := []string{"japanese-informal", "japanese-formal", "korean-hangul-formal", "korean-hanja-informal",
		"korean-hanja-formal", "simp-chinese-informal", "simp-chinese-formal", "trad-chinese-informal",
		"trad-chinese-formal", "cjk-ideographic", "ethiopic-numeric", "cjk-decimal"}
	values := []int{-10000, -9999, -1011, -10, -1, 0, 1, 2, 9, 10, 11, 12, 19, 20, 21, 99, 100, 101, 110, 111, 115,
		120, 1000, 1001, 1010, 1011, 1100, 1110, 1111, 2015, 2025, 9999, 10000, 10001, 12345, 100000, 1000000,
		1010101, 10000000, 10000001, 10010000, 100000000, 100000010, 110000, 1000000000, 2000010001, 2147483647,
		-2147483647}
	sheet := "body { margin: 0; font-size: 16px; } p { margin: 0; white-space: pre; } span { display: inline-block; }\n"
	type pair struct {
		name string
		n    int
		text string
	}
	var pairs []pair
	var ps []*ui.Node
	for i, name := range names {
		sheet += fmt.Sprintf(".s%d::before { content: counter(n, %s); }\n", i, name)
		for _, n := range values {
			text := css.Represent(name, n)
			pairs = append(pairs, pair{name, n, text})
			ps = append(ps, ui.P(
				ui.Span(ui.Class(fmt.Sprintf("s%d", i)), ui.Style(fmt.Sprintf("counter-reset: n %d", n))),
				ui.Span(ui.Text(text))))
		}
	}
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(ps...)
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("p")).map(p =>
		[p.children[0].getBoundingClientRect().width, p.children[1].getBoundingClientRect().width])`)
	require.NoError(t, err)
	var widths [][2]float64
	require.NoError(t, json.Unmarshal(raw, &widths))
	require.Len(t, widths, len(pairs))
	for i, p := range pairs {
		assert.InDelta(t, widths[i][0], widths[i][1], 0.01, "%s %d: ui writes %q", p.name, p.n, p.text)
	}
}
