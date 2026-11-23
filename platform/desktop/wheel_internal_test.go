package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
)

func TestAWindowsWheelOfWholeNotchesIsAMouses(t *testing.T) {
	v, precise := windowsWheel(0, -2)
	assert.False(t, precise)
	assert.Equal(t, geom.V2(0, -2), v, "two notches")
}

func TestAWindowsWheelOfPartsOfANotchIsPrecise(t *testing.T) {
	v, precise := windowsWheel(0, 30.0/120)
	assert.True(t, precise, "a precision touchpad's or a high-resolution wheel's turn")
	assert.InDelta(t, 2.5, v.Y, 1e-6, "25 points, a quarter of Chrome's 100 a notch, in tenths")
}

func TestAWaylandAxisOffTheCompositorsNotchesIsPrecise(t *testing.T) {
	_, precise := waylandWheel(0, 1)
	assert.False(t, precise, "ten points, a notch")
	v, precise := waylandWheel(-0.35, 0)
	assert.True(t, precise, "a trackpad's three and a half points")
	assert.InDelta(t, -0.35, v.X, 1e-6)
}
