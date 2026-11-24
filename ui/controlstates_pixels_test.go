//go:build chrome && gpu

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestControlPartsFollowThePointerAsChromes moves the mouse over the
// parts of controls drawn from their CSS, in ember and in Chrome, and
// compares the pictures: a range's thumb and track hovered, a press on
// its track (the thumb jumping under the pointer, hovered, the track
// pressed) and on a vertical range's thumb dragged down, a progress's
// and a meter's value hovered.
func TestControlPartsFollowThePointerAsChromes(t *testing.T) {
	const page = "./internal/pages/control-states"
	at := func(x, y float32, down bool) chrome.Pointer { return chrome.Pointer{At: geom.V2(x, y), Down: down} }
	for _, c := range []struct {
		name string
		path []chrome.Pointer
		mean float64
	}{
		{"thumb", []chrome.Pointer{at(400, 230, false), at(68, 27, false)}, 0.0016},
		{"track", []chrome.Pointer{at(400, 230, false), at(180, 27, false), at(180, 27, true), at(190, 27, true)}, 0.0016},
		{"container", []chrome.Pointer{at(400, 230, false), at(150, 21, false)}, 0.0016},
		{"vertical-thumb", []chrome.Pointer{at(400, 230, false), at(26, 118, false), at(26, 118, true),
			at(26, 160, true)}, 0.0016},
		{"value", []chrome.Pointer{at(400, 230, false), at(290, 27, false)}, 0.0016},
		{"bar", []chrome.Pointer{at(400, 230, false), at(380, 27, false)}, 0.0016},
		{"meter", []chrome.Pointer{at(400, 230, false), at(280, 57, false)}, 0.0016},
	} {
		t.Run(c.name, func(t *testing.T) {
			chrome.WithinPointing(t, page, c.name, 440, 240, c.path, c.mean)
		})
	}
}
