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

// gridTrackCases are grids whose resolved grid-template-columns and -rows getComputedStyle
// writes: fixed, flexible and fractional tracks, line names, repeat() and auto-fill,
// collapsed auto-fit tracks, implicit tracks on either side, a grid of no template, right
// to left and vertical grids, and a block, whose value stays as declared.
//
//nolint:gochecknoglobals // the page's grids
var gridTrackCases = []struct{ style, items string }{
	{"grid-template-columns: 100px 1fr; width: 400px", "a"},
	{"grid-template-columns: [first] auto [last]", ""},
	{"grid-template-columns: repeat(auto-fill, 50px); width: 220px", "a"},
	{"grid-template-columns: repeat(auto-fit, 50px); width: 220px", "a"},
	{"grid-template-columns: [a] repeat(2, [b] 10px [c]) [d]; grid-template-rows: [x] 20px", "a"},
	{"grid-auto-columns: 30px; grid-auto-rows: 15px", "column3"},
	{"grid-template-columns: [s] 40px [e]; grid-auto-columns: 25px", "before"},
	{"grid-template-columns: 10px 20px [n] 30px; direction: rtl", "a"},
	{"grid-template-columns: 10px [n] 20px; grid-template-rows: 5px 1fr; writing-mode: vertical-lr; height: 100px", "a"},
	{"display: block; grid-template-columns: 100px 1fr", "a"},
}

func TestResolvedGridTracksMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { margin: 0 } .g { display: grid; margin-bottom: 4px }
.a { height: 10px } .column3 { grid-column: 3 } .before { grid-column: -3 / span 1 }`))
	var grids []*ui.Node
	for _, c := range gridTrackCases {
		var kids []ui.Arg
		if c.items != "" {
			kids = append(kids, ui.Div(ui.Class(c.items), ui.Text("x")))
		}
		g := ui.Div(append([]ui.Arg{ui.Class("g"), ui.Style(c.style)}, kids...)...)
		grids = append(grids, g)
		d.Body().Append(g)
	}
	size := geom.V2(600, 800)
	out, err := chrome.Eval(d, size, `return [...document.querySelectorAll(".g")].map(g => {
	const s = getComputedStyle(g); return s.gridTemplateColumns + " | " + s.gridTemplateRows })`)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(size)
	for i, g := range grids {
		cols, _ := d.ComputedValue(g, "grid-template-columns")
		rows, _ := d.ComputedValue(g, "grid-template-rows")
		assert.Equal(t, want[i], strings.Join([]string{cols, rows}, " | "), gridTrackCases[i].style)
	}
}
