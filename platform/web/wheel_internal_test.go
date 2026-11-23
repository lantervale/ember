package web

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
)

func TestAMousesNotchInChromeIsANotch(t *testing.T) {
	v, precise := wheelTurn(deltaPixel, geom.V2(0, 100), geom.V2(0, -120))
	assert.False(t, precise)
	assert.Equal(t, geom.V2(0, -1), v, "a notch down")
}

func TestATrackpadsPixelsArePrecise(t *testing.T) {
	v, precise := wheelTurn(deltaPixel, geom.V2(-4, 7), geom.V2(12, -21))
	assert.True(t, precise, "its wheelDelta is three times its pixels, no whole notch")
	assert.Equal(t, geom.V2(0.4, -0.7), v, "in tenths of a point")
}

func TestLinesAndPagesAreNotches(t *testing.T) {
	v, precise := wheelTurn(deltaLine, geom.V2(0, 3), geom.Vec2{})
	assert.False(t, precise)
	assert.Equal(t, geom.V2(0, -1), v, "Firefox's three lines a notch")
	_, precise = wheelTurn(deltaPage, geom.V2(0, 1), geom.Vec2{})
	assert.False(t, precise)
}
