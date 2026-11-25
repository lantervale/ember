//go:build chrome && gpu

package ui_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// The parts of styled bars take their :hover and :active as Chrome's
// under a real mouse: the part the pointer is on, the bar it is on, the
// track while the pointer is on a piece or the thumb, and the thumb
// pressed, and the viewport's bars too.
func TestScrollbarPartsUnderTheMouseAsChromes(t *testing.T) {
	const hover, viewport = "./internal/pages/scrollbar-hover", "./internal/pages/scrollbar-viewport"
	for _, c := range []struct {
		name    string
		pkg     string
		at      geom.Vec2
		pressed bool
	}{
		{"thumb", hover, geom.V2(106, 45), false},
		{"button", hover, geom.V2(106, 95), false},
		{"piece", hover, geom.V2(40, 106), false},
		{"content", hover, geom.V2(50, 50), false},
		{"thumb-pressed", hover, geom.V2(106, 45), true},
		{"viewport-thumb", viewport, geom.V2(313, 95), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.pkg == hover {
				chrome.UnlikePinned(t, "a short styled thumb is shorter than 26 points")
			}
			mouse := fmt.Sprintf("%g,%g@0", c.at.X, c.at.Y)
			if c.pressed {
				mouse += ";left:4-30"
			}
			w, h := 200, 140
			if c.pkg == viewport {
				w, h = 320, 260
			}
			chrome.WithinUnderMouse(t, c.pkg, "scrollbar-hover-"+c.name, w, h, mouse, func(s *chrome.Session) {
				s.Mouse("mouseMoved", c.at, 0)
				if c.pressed {
					s.Mouse("mousePressed", c.at, 1)
				}
				time.Sleep(100 * time.Millisecond)
			}, 0.0004)
		})
	}
}
