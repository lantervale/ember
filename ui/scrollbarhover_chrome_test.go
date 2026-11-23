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
// pressed; the box's own :hover holds over its bars and content.
func TestScrollbarPartsUnderTheMouseAsChromes(t *testing.T) {
	const pkg = "./internal/pages/scrollbar-hover"
	for _, c := range []struct {
		name    string
		at      geom.Vec2
		pressed bool
	}{
		{"thumb", geom.V2(106, 45), false},
		{"button", geom.V2(106, 95), false},
		{"piece", geom.V2(40, 106), false},
		{"content", geom.V2(50, 50), false},
		{"thumb-pressed", geom.V2(106, 45), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			mouse := fmt.Sprintf("%g,%g@0", c.at.X, c.at.Y)
			if c.pressed {
				mouse += ";left:4-30"
			}
			chrome.WithinUnderMouse(t, pkg, "scrollbar-hover-"+c.name, 200, 140, mouse, func(s *chrome.Session) {
				s.Mouse("mouseMoved", c.at, 0)
				if c.pressed {
					s.Mouse("mousePressed", c.at, 1)
				}
				time.Sleep(100 * time.Millisecond)
			}, 0.0004)
		})
	}
}
