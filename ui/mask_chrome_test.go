//go:build chrome

package ui_test

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAMaskedElementIsHitInItsBorderBoxAsInChrome(t *testing.T) {
	sheet := `
body { margin: 0; }
.cell { position: absolute; width: 128px; height: 108px; background: #fde9c8; border-radius: 40px; }
.cell > div { width: 168px; height: 148px; margin: -20px; background: #36c; }
.linear { left: 10px; top: 10px; mask-image: linear-gradient(#000, transparent 50%); }
.radial { left: 160px; top: 10px; mask: radial-gradient(#000 30%, transparent 30%) no-clip; }
.missing { left: 310px; top: 10px; mask-image: url(missing.png); }
.subtract { left: 10px; top: 150px; mask: linear-gradient(#000 0 0) subtract, linear-gradient(#000 0 0); }
.turned { left: 160px; top: 150px; transform: rotate(30deg); mask-image: linear-gradient(90deg, #000, transparent); }
`
	classes := []string{"linear", "radial", "missing", "subtract", "turned"}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"m.css": {Data: []byte(sheet)}}, "m.css"))
	page := ui.Div(ui.ID("page"), ui.Style("position: absolute; inset: 0"))
	for _, c := range classes {
		page.Append(ui.Div(ui.ID(c), ui.Class("cell", c), ui.Div(ui.ID(c+"-in"))))
	}
	d.Body().Append(page)
	size := geom.V2(480, 320)
	var points [][2]float32
	for y := float32(2); y < size.Y; y += 7 {
		for x := float32(3); x < size.X; x += 7 {
			points = append(points, [2]float32{x, y})
		}
	}
	js, err := json.Marshal(points)
	require.NoError(t, err)
	out, err := chrome.Eval(d, size,
		`return `+string(js)+`.map(([x, y]) => { const e = document.elementFromPoint(x, y); return e ? e.id : ""; });`)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Len(t, want, len(points))
	d.Layout(size)
	hits, differ, edges := map[string]int{}, 0, 0
	for i, p := range points {
		got := ""
		if n := ui.HitAt(d, geom.V2(p[0], p[1])); n != nil {
			got = n.ID()
		}
		hits[want[i]]++
		if got == "page" && want[i] == "turned-in" {
			// Chrome cuts what a turned box holds about half a pixel past its
			// edges.
			edges++
			continue
		}
		if got != want[i] {
			differ++
			if differ <= 20 {
				t.Errorf("at %v: ember %q, Chrome %q", p, got, want[i])
			}
		}
	}
	assert.Zero(t, differ, "points hit apart from Chrome, of %d", len(points))
	assert.LessOrEqual(t, edges, 8, "points on a turned edge")
	for _, c := range classes {
		assert.Positive(t, hits[c+"-in"], "Chrome hits %s's child where its mask is clear", c)
	}
}
