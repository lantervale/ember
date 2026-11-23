//go:build !darwin && !windows && !(linux && wayland)

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/lantervale/ember/geom"
)

// wheelTurn is a turn of x, y in notches: GLFW gives X11's wheel as core
// button presses, its smooth scrolling unread, so nothing says a turn
// came from a trackpad.
func wheelTurn(_ *glfw.Window, x, y float64) (geom.Vec2, bool) {
	return geom.V2(float32(x), float32(y)), false
}

func watchWheels() {}

func forgetWheel(*glfw.Window) {}
