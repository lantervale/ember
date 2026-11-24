//go:build chrome && gpu

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestBarsRestyledByTheirBoxsHoverAsChromes moves the mouse over boxes
// whose :hover restyles their bars' parts: as Chrome's, a part takes its
// box's hover only once the pointer reaches or leaves that part, with
// its bar and track, and keeps it after; a box whose own style changes
// with its hover restyles every part at once.
func TestBarsRestyledByTheirBoxsHoverAsChromes(t *testing.T) {
	const page = "./internal/pages/scrollbar-hover-late"
	at := func(x, y float32) chrome.Pointer { return chrome.Pointer{At: geom.V2(x, y)} }
	out := at(250, 132)
	for _, c := range []struct {
		name string
		path []chrome.Pointer
	}{
		{"content", []chrome.Pointer{out, at(60, 60)}},
		{"piece", []chrome.Pointer{out, at(60, 60), at(106, 80)}},
		{"thumb-and-out", []chrome.Pointer{out, at(60, 60), at(106, 25), at(60, 60), out}},
		{"piece-from-out", []chrome.Pointer{out, at(90, 106), out}},
		{"own-style", []chrome.Pointer{out, at(170, 60)}},
		{"own-style-and-out", []chrome.Pointer{out, at(170, 60), out}},
	} {
		t.Run(c.name, func(t *testing.T) {
			chrome.Since(t, 158, "a box's bars and border take the hover of the box under the pointer")
			chrome.WithinPointing(t, page, c.name, 260, 140, c.path, 0.0004)
		})
	}
}
