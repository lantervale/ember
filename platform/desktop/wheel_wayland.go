//go:build linux && wayland

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/lantervale/ember/geom"
)

// wheelTurn is a turn of x, y, precise where it is off the compositor's
// notches.
func wheelTurn(_ *glfw.Window, x, y float64) (geom.Vec2, bool) { return waylandWheel(x, y) }

func watchWheels() {}

func forgetWheel(*glfw.Window) {}
