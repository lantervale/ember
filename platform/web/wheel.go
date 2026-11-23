package web

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// WheelEvent's deltaMode.
const (
	deltaPixel = 0
	deltaLine  = 1
	deltaPage  = 2
)

// wheelTurn is a WheelEvent of mode turning by delta, its legacy
// wheelDelta of notches as 120 each, as input takes it: in notches, up
// and left positive, else, precise, in tenths of a point. The browser
// counts a mouse's notches in lines (Firefox) or in pixels whose
// wheelDelta is whole notches (Chrome, 100 pixels each); a trackpad's or
// a smooth wheel's pixels have a wheelDelta of three times their own.
func wheelTurn(mode int, delta, wheelDelta geom.Vec2) (geom.Vec2, bool) {
	switch mode {
	case deltaLine:
		return delta.Scale(-1.0 / 3), false
	case deltaPage:
		return delta.Scale(-1 / 0.3), false
	}
	if notches(wheelDelta.X) && notches(wheelDelta.Y) && wheelDelta != (geom.Vec2{}) {
		return delta.Scale(-1.0 / 100), false
	}
	return delta.Scale(-1.0 / 10), true
}

// notches reports whether a wheelDelta is whole notches of 120.
func notches(v float32) bool { return math.Mod(float64(v), 120) == 0 }
