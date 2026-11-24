//go:build chrome && gpu

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestViewportBarsRestyledByBodysHoverAsChromes moves the mouse over a
// page whose viewport's bars body's :hover restyles: unlike a box's,
// which wait for the pointer to reach or leave a part, Chrome restyles
// every part of the viewport's as body's hover changes, the pointer on
// the bars leaving body unhovered, as it does when body's own style
// changes (decision 0213).
func TestViewportBarsRestyledByBodysHoverAsChromes(t *testing.T) {
	const page = "./internal/pages/scrollbar-viewport-hover"
	at := func(x, y float32) chrome.Pointer { return chrome.Pointer{At: geom.V2(x, y)} }
	out, body, own := at(150, 80), at(150, 30), at(30, 20)
	thumb, piece := at(294, 20), at(294, 150)
	for _, c := range []struct {
		name string
		path []chrome.Pointer
	}{
		{"body", []chrome.Pointer{out, body}},
		{"thumb", []chrome.Pointer{out, body, thumb}},
		{"thumb-and-out", []chrome.Pointer{out, body, thumb, body, out}},
		{"piece-from-out", []chrome.Pointer{out, piece, out}},
		{"thumb-to-piece", []chrome.Pointer{out, body, thumb, piece}},
		{"piece-to-body", []chrome.Pointer{out, piece, body}},
		{"own-style", []chrome.Pointer{out, own}},
		{"own-style-and-out", []chrome.Pointer{out, own, out}},
	} {
		t.Run(c.name, func(t *testing.T) {
			chrome.WithinPointing(t, page, c.name, 300, 220, c.path, 0.0008)
		})
	}
}
