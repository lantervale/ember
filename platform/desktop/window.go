package desktop

import (
	"image"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
)

func flag(on bool) int {
	if on {
		return glfw.True
	}
	return glfw.False
}

func (n *native) SetResizable(on bool) {
	n.h.SetAttrib(glfw.Resizable, flag(on))
	n.syncTitleBar()
}
func (n *native) SetAlwaysOnTop(on bool) { n.h.SetAttrib(glfw.Floating, flag(on)) }

func (n *native) SetSizeLimits(lo, hi geom.Vec2) {
	limit := func(v geom.Vec2) (int, int) {
		if v == (geom.Vec2{}) {
			return glfw.DontCare, glfw.DontCare
		}
		return int(v.X), int(v.Y)
	}
	minW, minH := limit(lo)
	maxW, maxH := limit(hi)
	n.h.SetSizeLimits(minW, minH, maxW, maxH)
}

func (n *native) SetAspectRatio(width, height int) {
	if width <= 0 || height <= 0 {
		width, height = glfw.DontCare, glfw.DontCare
	}
	n.h.SetAspectRatio(width, height)
}

func (n *native) Maximize() { n.h.Maximize() }
func (n *native) Minimize() { n.h.Iconify() }
func (n *native) Restore()  { n.h.Restore() }

func (n *native) SetIcon(images []image.Image) { n.h.SetIcon(images) }

// SetFullscreen covers the monitor under the window's centre at its
// resolution, and puts the window back where it was when leaving: its
// position and size, maximized again if it was.
func (n *native) SetFullscreen(on bool) {
	switch {
	case on && n.h.GetMonitor() == nil:
		m := n.monitor()
		n.windowedMaximized = n.h.GetAttrib(glfw.Maximized) == glfw.True
		if n.windowedMaximized {
			n.h.Restore()
		}
		n.windowedX, n.windowedY = n.h.GetPos()
		n.windowedW, n.windowedH = n.h.GetSize()
		mode := m.GetVideoMode()
		n.h.SetMonitor(m, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
	case !on && n.h.GetMonitor() != nil:
		n.h.SetMonitor(nil, n.windowedX, n.windowedY, n.windowedW, n.windowedH, 0)
		if n.owns() {
			// GLFW sized the window around a title bar of its own again.
			ownTitleBar(n.h)
			n.h.SetSize(n.windowedW, n.windowedH)
		}
		if n.windowedMaximized {
			n.h.Maximize()
		}
	}
	n.sync()
	n.window.SyncState(n.h.GetAttrib(glfw.Maximized) == glfw.True, false)
}

func (n *native) monitor() *glfw.Monitor {
	x, y := n.h.GetPos()
	w, h := n.h.GetSize()
	cx, cy := x+w/2, y+h/2
	for _, m := range glfw.GetMonitors() {
		mx, my, mw, mh := m.GetWorkarea()
		if cx >= mx && cx < mx+mw && cy >= my && cy < my+mh {
			return m
		}
	}
	return glfw.GetPrimaryMonitor()
}

// owns reports whether n's content fills its title bar: a window that
// asked for it, shown, on a system that does it.
func (n *native) owns() bool { return ownsTitleBar && n.window.OwnTitleBar() && n.askedW == 0 }

// syncTitleBar tells the window what the system draws over its content.
func (n *native) syncTitleBar() {
	if n.owns() {
		n.window.SyncTitleBar(titleBarArea(n.h))
	}
}

// StartDrag hands the press to the system, which keeps its release.
func (n *native) StartDrag() {
	if n.scripted || !ownsTitleBar {
		return
	}
	n.window.Input().SetMouseButton(input.MouseLeft, false)
	n.between(func() { startDrag(n.h) })
}

func (n *native) DoubleClickTitleBar() {
	if !n.scripted {
		n.between(func() { doubleClickTitleBar(n.h) })
	}
}

// hints sets the creation hints for what a window asks for.
func hints(win *platform.Window) {
	glfw.WindowHint(glfw.Resizable, flag(win.Resizable()))
	glfw.WindowHint(glfw.Floating, flag(win.AlwaysOnTop()))
	glfw.WindowHint(glfw.Maximized, flag(win.Maximized()))
}

// apply sets what can only be set once the window exists.
func (n *native) apply(win *platform.Window) {
	if lo, hi := win.MinSize(), win.MaxSize(); lo != (geom.Vec2{}) || hi != (geom.Vec2{}) {
		n.SetSizeLimits(lo, hi)
	}
	if w, h := win.AspectRatio(); w > 0 && h > 0 {
		n.SetAspectRatio(w, h)
	}
	if icon := win.Icon(); len(icon) > 0 {
		n.SetIcon(icon)
	}
	if win.Fullscreen() {
		n.SetFullscreen(true)
	}
	n.h.SetMaximizeCallback(func(_ *glfw.Window, maximized bool) { win.SyncState(maximized, win.Minimized()) })
	n.h.SetIconifyCallback(func(_ *glfw.Window, minimized bool) { win.SyncState(win.Maximized(), minimized) })
}
