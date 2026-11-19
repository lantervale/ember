//go:build !darwin

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/platform"
)

// ownsTitleBar is false off macOS: a window keeps the system's title bar
// above its content, which drags and zooms it, so there is nothing to
// keep clear and nothing for the program to drag (decision 0091).
const ownsTitleBar = false

func watchPresses() {}

func ownTitleBar(*glfw.Window) {}

func titleBarArea(*glfw.Window) platform.TitleBarArea { return platform.TitleBarArea{} }

func startDrag(*glfw.Window) {}

func doubleClickTitleBar(*glfw.Window) {}
