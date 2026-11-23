package desktop

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// pointsPerWindowsNotch is how far a WHEEL_DELTA of Windows scrolls in
// Chrome: three lines of 100/3 pixels (kScrollbarPixelsPerLine).
const pointsPerWindowsNotch = 100

// wholeNotches reports whether a turn of x, y is whole notches: a
// mouse's wheel clicks. A trackpad's or a smooth wheel's turn comes in
// fractions of one: Windows' WM_MOUSEWHEEL in parts of WHEEL_DELTA,
// Wayland's axis in its points over GLFW's ten a notch.
func wholeNotches(x, y float64) bool {
	return x == math.Trunc(x) && y == math.Trunc(y)
}

// windowsWheel is a turn of x, y as GLFW gives WM_MOUSEWHEEL's, in
// notches, and whether it is precise: one of fractions of WHEEL_DELTA,
// a precision touchpad's or a high-resolution wheel's, then in tenths
// of a point at Chrome's 100 points a notch, as input.AddPreciseWheel
// takes it.
func windowsWheel(x, y float64) (geom.Vec2, bool) {
	v := geom.V2(float32(x), float32(y))
	if wholeNotches(x, y) {
		return v, false
	}
	return v.Scale(pointsPerWindowsNotch / 10), true
}

// waylandWheel is a turn of x, y as GLFW gives wl_pointer's axis, its
// points over ten, and whether it is precise: a trackpad's, whose points
// aren't the compositor's whole notches of ten; its tenths of a point are
// input.AddPreciseWheel's unit already.
func waylandWheel(x, y float64) (geom.Vec2, bool) {
	return geom.V2(float32(x), float32(y)), !wholeNotches(x, y)
}
