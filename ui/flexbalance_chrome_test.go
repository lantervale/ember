//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestFlexBalanceValuesAreChromes compares the computed flex-wrap,
// flex-direction and flex-line-count of each way to write them with
// getComputedStyle's, WPT css/css-flexbox/balance's parsing tests.
func TestFlexBalanceValuesAreChromes(t *testing.T) {
	styles := []string{
		"flex-wrap: balance", "flex-wrap: wrap balance", "flex-wrap: balance wrap", "flex-wrap: wrap-reverse balance",
		"flex-wrap: balance wrap-reverse", "flex-wrap: nowrap balance", "flex-wrap: balance balance",
		"flex-wrap: wrap wrap", "flex-wrap: balance 1", "flex-wrap: wrap-reverse",
		"flex-flow: balance", "flex-flow: row balance", "flex-flow: row wrap balance", "flex-flow: balance wrap row",
		"flex-flow: column balance", "flex-flow: balance column", "flex-flow: column wrap-reverse balance",
		"flex-flow: balance wrap-reverse row-reverse", "flex-flow: wrap row balance", "flex-flow: balance column wrap",
		"flex-flow: row balance column", "flex-flow: balance nowrap",
		"flex-line-count: 2", "flex-line-count: round(up, calc(7/2))", "flex-line-count: calc(NaN)",
		"flex-line-count: 0", "flex-line-count: 1.5", "flex-line-count: -1", "flex-line-count: auto",
	}
	d := ui.NewDocument()
	for _, s := range styles {
		d.Body().Append(ui.Div(ui.Style("flex-line-count: 3; " + s)))
	}
	props := []string{"flex-wrap", "flex-direction", "flex-line-count"}
	want, err := chrome.Styles(d, geom.Vec2{X: 400, Y: 300}, props)
	require.NoError(t, err)
	d.Layout(geom.Vec2{X: 400, Y: 300})
	var got []*ui.Node
	d.HTML().Walk(func(n *ui.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	require.Len(t, got, len(want))
	for i, n := range got[2:] {
		for _, p := range props {
			ours, _ := d.ComputedValue(n, p)
			assert.Equal(t, want[i+2][p], ours, "%s: %s", styles[i], p)
		}
	}
}
