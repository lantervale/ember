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

//nolint:gochecknoglobals // a test sheet
var turnedSheet = `
body { margin: 0; }
.cell { position: absolute; width: 80px; height: 50px; background: #fde9c8; }
.cell > div { width: 30px; height: 20px; margin: 10px; background: #36c; }
#moved { left: 10px; top: 10px; translate: 20px 50%; }
#turned { left: 150px; top: 20px; rotate: 30deg; }
#scaled { left: 290px; top: 20px; scale: 1.5 0.5; transform-origin: 0 0; }
#all { left: 430px; top: 30px; translate: -10px 5px; rotate: -20deg; scale: 0.8;
  transform: skewX(10deg) translate(5px); }
#nested { left: 20px; top: 130px; rotate: 90deg; }
#nested > div { translate: 15px; scale: 2; rotate: 45deg; }
#axis { left: 170px; top: 130px; rotate: 0 0 -1 60deg; transform-origin: right bottom; }
#none { left: 310px; top: 130px; translate: none; rotate: none; scale: none; }
#mirror { left: 450px; top: 130px; scale: -1 1; translate: 0 20px; }
`

//nolint:gochecknoglobals // the cells of turnedSheet
var turnedCells = []string{"moved", "turned", "scaled", "all", "nested", "axis", "none", "mirror"}

func turnedDoc() *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(turnedSheet)}}, "t.css"))
	for _, c := range turnedCells {
		d.Body().Append(ui.Div(ui.ID(c), ui.Class("cell"), ui.Div(ui.ID(c+"-in"))))
	}
	return d
}

// TestIndividualTransformsTurnBoxesAsChromes compares the box round each
// element as translate, rotate and scale turn it, in that order, before
// transform, about transform-origin, with getBoundingClientRect.
func TestIndividualTransformsTurnBoxesAsChromes(t *testing.T) {
	d := turnedDoc()
	size := geom.V2(600, 260)
	out, err := chrome.Eval(d, size, `return [...document.querySelectorAll(".cell, .cell > div")].map(e => {
	const r = e.getBoundingClientRect(); return [r.x, r.y, r.width, r.height]; });`)
	require.NoError(t, err)
	var want [][4]float32
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(size)
	require.Len(t, want, 2*len(turnedCells))
	for i, w := range want {
		id := turnedCells[i/2]
		if i%2 == 1 {
			id += "-in"
		}
		got, ok := ui.TransformedBox(d, d.ByID(id))
		require.True(t, ok, id)
		r := geom.Rect{X: w[0], Y: w[1], W: w[2], H: w[3]}
		assert.InDelta(t, r.X, got.X, 0.5, "%s: ember %v, Chrome %v", id, got, r)
		assert.InDelta(t, r.Y, got.Y, 0.5, "%s: ember %v, Chrome %v", id, got, r)
		assert.InDelta(t, r.W, got.W, 0.5, "%s: ember %v, Chrome %v", id, got, r)
		assert.InDelta(t, r.H, got.H, 0.5, "%s: ember %v, Chrome %v", id, got, r)
	}
}

// TestIndividualTransformsAreHitAsInChrome finds the element under
// points across the turned cells, as elementFromPoint does.
func TestIndividualTransformsAreHitAsInChrome(t *testing.T) {
	d := turnedDoc()
	size := geom.V2(600, 260)
	var points [][2]float32
	for y := float32(2); y < size.Y; y += 5 {
		for x := float32(3); x < size.X; x += 5 {
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
	hit := func(p geom.Vec2) string {
		if n := ui.HitAt(d, p); n != nil {
			return n.ID()
		}
		return ""
	}
	hits, differ := map[string]int{}, 0
	for i, pt := range points {
		p := geom.V2(pt[0], pt[1])
		hits[want[i]]++
		if got := hit(p); got != want[i] {
			differ++
			if differ <= 20 {
				t.Errorf("at %v: ember %q, Chrome %q", p, got, want[i])
			}
		}
	}
	assert.Zero(t, differ, "points hit apart from Chrome, of %d", len(points))
	for _, c := range turnedCells {
		assert.Positive(t, hits[c+"-in"], "Chrome hits %s's child somewhere", c)
	}
}

// TestIndividualTransformsGrowScrollableOverflowAsChromes has turned,
// moved and scaled children, their own overflow with them, reach as far
// in their scrollers as in Chrome's, where transform does the same.
func TestIndividualTransformsGrowScrollableOverflowAsChromes(t *testing.T) {
	sheet := `body { margin: 0; } .s { width: 100px; height: 100px; overflow: auto; position: relative; }
.s > div { width: 50px; height: 50px; } .wide { width: 200px; height: 10px; }`
	kids := []string{
		"transform: translate(100px, 80px)",
		"translate: 100px 80px",
		"rotate: 45deg; margin-left: 80px",
		"scale: 3",
		"scale: 3; transform-origin: 0 0",
		"translate: -100px -80px",
		"translate: 100px 80px; overflow: hidden",
		"translate: 100px 80px",
		"position: absolute; translate: 100px 80px",
		"rotate: 90deg; translate: 0 120px; transform-origin: 0 0",
		"display: flex; translate: 60px",
	}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte(sheet)}}, "s.css"))
	for i, k := range kids {
		child := ui.Div(ui.Style(k))
		if i == 6 || i == 7 {
			child.Append(ui.Div(ui.Class("wide")))
		}
		d.Body().Append(ui.Div(ui.Class("s"), child))
	}
	size := geom.V2(400, 1400)
	out, err := chrome.Eval(d, size, `return [...document.querySelectorAll(".s")].map(e =>
	[e.scrollWidth, e.scrollHeight, e.clientWidth, e.clientHeight]);`)
	require.NoError(t, err)
	var want [][4]float32
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(size)
	require.Len(t, want, len(kids))
	for i, s := range d.Body().Children() {
		got := ui.ScrollSize(d, s)
		got = geom.V2(max(got.X, want[i][2]), max(got.Y, want[i][3]))
		assert.InDelta(t, want[i][0], got.X, 1, "%s: scrollWidth, ember %v, Chrome %v", kids[i], got, want[i])
		assert.InDelta(t, want[i][1], got.Y, 1, "%s: scrollHeight, ember %v, Chrome %v", kids[i], got, want[i])
	}
}
