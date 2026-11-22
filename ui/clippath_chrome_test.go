//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAClipPathIsHitAsInChrome(t *testing.T) {
	sheet := `
body { margin: 0; }
.cell { position: absolute; width: 128px; height: 108px; margin: 4px; background: #fde9c8; }
.cell > div { width: 168px; height: 148px; margin: -20px; background: #36c; }
.circle { left: 10px; top: 10px; clip-path: circle(40%); }
.ellipse { left: 160px; top: 10px; clip-path: ellipse(closest-side farthest-side at 30% 60%) content-box; }
.inset { left: 310px; top: 10px; clip-path: inset(10px 20% 15px round 12px 4px / 20px); }
.star { left: 10px; top: 150px; clip-path: polygon(evenodd, 50% 0, 79% 90%, 2% 35%, 98% 35%, 21% 90%); }
.path { left: 160px; top: 150px; clip-path: path(evenodd, "M10 10 h80 v60 h-80 z M30 30 h40 v20 h-40 z"); }
.box { left: 310px; top: 150px; clip-path: padding-box; }
.margin { left: 10px; top: 290px; clip-path: inset(-10px) margin-box; }
.turned { left: 160px; top: 290px; transform: rotate(30deg); clip-path: polygon(0 0, 100% 0, 0 100%); }
.xywh { left: 310px; top: 290px; clip-path: xywh(10px 10% 50% 30px round 6px); }
.over { left: 460px; top: 10px; clip-path: circle(30px at 0 0); }
.over > div { margin: 0; width: 60px; height: 60px; }
.empty { left: 460px; top: 150px; clip-path: circle(0); }
`
	// Chrome hits a clip-path of a box with a border or padding in a box
	// the size of its content box at its border box's corner, though it
	// paints it where ember does (the clip-path pixel page): these cells
	// have neither.
	classes := []string{"circle", "ellipse", "inset", "star", "path", "box", "margin", "turned", "xywh", "over", "empty"}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"c.css": {Data: []byte(sheet)}}, "c.css"))
	page := ui.Div(ui.ID("page"), ui.Style("position: absolute; inset: 0"))
	for _, c := range classes {
		page.Append(ui.Div(ui.ID(c), ui.Class("cell", c), ui.Div(ui.ID(c+"-in"))))
	}
	d.Body().Append(page)
	size := geom.V2(640, 440)
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
	hits, differ := map[string]int{}, 0
	for i, p := range points {
		got := ""
		if n := ui.HitAt(d, geom.V2(p[0], p[1])); n != nil {
			got = n.ID()
		}
		hits[want[i]]++
		if got != want[i] {
			differ++
			if differ <= 20 {
				t.Errorf("at %v: ember %q, Chrome %q", p, got, want[i])
			}
		}
	}
	assert.Zero(t, differ, "points hit apart from Chrome, of %d", len(points))
	for _, c := range classes[:len(classes)-1] {
		assert.Positive(t, hits[c+"-in"], fmt.Sprintf("Chrome hits %s's child somewhere", c))
	}
	assert.Zero(t, hits["empty"]+hits["empty-in"], "nothing of an empty clip is hit")
}
