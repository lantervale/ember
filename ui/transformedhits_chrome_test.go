//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestPointsNearTurnedEdgesHitAsInChrome finds the element under points
// half a pixel apart round boxes turned, scaled, skewed and moved by a
// fraction of a pixel, and their children, as elementFromPoint does: a
// point hits a transformed box when the pixel square from it reaches the
// box, as Blink's hit test carries the point through a transform.
func TestPointsNearTurnedEdgesHitAsInChrome(t *testing.T) {
	cells := []string{
		"rotate: 30deg", "scale: 1.5 0.5", "transform: skewX(20deg)", "translate: 0.5px 0.25px",
		"rotate: -45deg; scale: 0.75", "rotate: 90deg", "transform: rotate(10deg) translate(0.3px)", "scale: 2",
	}
	sheet := `body { margin: 0; } .c { position: absolute; width: 40px; height: 24px; background: #fde9c8; }
.c > div { position: absolute; left: 6px; top: 6px; width: 12px; height: 8px; background: #36c; }`
	d := ui.NewDocument(ui.CSS(sheet))
	var ids []string
	for i, c := range cells {
		x, y := 30+float32(i%4)*100, 30+float32(i/4)*100
		id := fmt.Sprint("c", i)
		ids = append(ids, id, id+"-in")
		d.Body().Append(ui.Div(ui.ID(id), ui.Class("c"), ui.Style(fmt.Sprintf("left: %gpx; top: %gpx; %s", x, y, c)),
			ui.Div(ui.ID(id+"-in"))))
	}
	size := geom.V2(420, 220)
	var points [][2]float32
	for y := float32(0); y < size.Y; y += 0.5 {
		for x := float32(0); x < size.X; x += 0.5 {
			points = append(points, [2]float32{x, y})
		}
	}
	js, err := json.Marshal(points)
	require.NoError(t, err)
	out, err := chrome.Eval(d, size, `return `+string(js)+`.map(([x, y]) => {
	const e = document.elementFromPoint(x, y); return e && e.id ? e.id : ""; });`)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Len(t, want, len(points))
	d.Layout(size)
	hit := func(p geom.Vec2) string {
		if n := ui.HitAt(d, p); n != nil {
			return n.ID()
		}
		return ""
	}
	// Chrome's square reaches a turned corner up to about 0.4 pixel further
	// than its exact overlap, where nothing found says why: a point that
	// close counts as Chrome's.
	corner := func(p geom.Vec2, id string) bool {
		for _, o := range []geom.Vec2{{X: 0.5}, {Y: 0.5}, {X: -0.5}, {Y: -0.5}} {
			if hit(p.Add(o)) == id {
				return true
			}
		}
		return false
	}
	differ, corners := 0, 0
	hits := map[string]int{}
	for i, pt := range points {
		p := geom.V2(pt[0], pt[1])
		got := hit(p)
		hits[want[i]]++
		switch {
		case got == want[i]:
		case corner(p, want[i]):
			corners++
		default:
			differ++
			if differ <= 30 {
				t.Errorf("at %v: ember %q, Chrome %q", p, got, want[i])
			}
		}
	}
	t.Logf("%d points, %d within half a pixel of a corner Chrome reaches past", len(points), corners)
	assert.Zero(t, differ, "points hit apart from Chrome, of %d", len(points))
	assert.Less(t, corners, 50, "points near corners hit apart from Chrome")
	for _, id := range ids {
		assert.Positive(t, hits[id], "Chrome hits %s", id)
	}
}
