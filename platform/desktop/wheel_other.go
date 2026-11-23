//go:build !darwin

package desktop

import "github.com/go-gl/glfw/v3.3/glfw"

// wheelPrecise reports false: GLFW gives X11's and Windows' wheels as
// notches, and nothing says a turn came from a trackpad.
func wheelPrecise(*glfw.Window) bool { return false }

func forgetWheel(*glfw.Window) {}
