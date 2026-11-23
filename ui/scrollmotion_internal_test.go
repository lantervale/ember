package ui

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
)

func TestTheUsersGlidesLastAsChromiumsCurves(t *testing.T) {
	const frame = 1.0 / 60
	for _, c := range []struct {
		by    motion
		delta float32
		want  float32
	}{
		{byKeys, 40, 9 * frame}, {byKeys, 2000, 9 * frame},
		{byWheel, 54, 12 * frame}, {byWheel, 120, 12 * frame}, {byWheel, 300, 9 * frame},
		{byWheel, 480, 6 * frame}, {byWheel, 2000, 6 * frame},
		{byProgram, 400, 20 * frame}, {byProgram, 1e6, 90 * frame},
	} {
		assert.InDelta(t, c.want, glideTime(c.by, geom.V2(0, -c.delta)), 1e-6, "%v by %v", c.by, c.delta)
	}
}

func TestARetargetedGlideKeepsItsSpeed(t *testing.T) {
	g := newGlide(geom.Vec2{}, geom.V2(0, 54), byWheel)
	g.start = 0
	now := g.dur / 2
	before, speed := g.at(now), g.velocity(now)
	g.retarget(now, geom.V2(0, 108))
	assert.Equal(t, before, g.at(now), "it goes on from where it is")
	assert.InDelta(t, speed, g.velocity(now+1e-4), float64(speed)*0.02, "at the speed it had")
	assert.InDelta(t, 108, g.at(now+g.dur).Y, 1e-3, "to its new end")
	assert.LessOrEqual(t, g.dur, glideTime(byWheel, geom.V2(0, 108-before.Y)))

	g = newGlide(geom.Vec2{}, geom.V2(0, 300), byKeys)
	g.start = 0
	now = g.dur * 0.6
	at := g.at(now)
	g.retarget(now, at.Add(geom.V2(0, 2)))
	assert.Less(t, g.dur, glideTime(byKeys, geom.V2(0, 2)),
		"a fast glide sent just ahead ends as soon as its speed takes it there")
}

func TestAPageIsSevenEighthsOfTheSnapport(t *testing.T) {
	assert.InDelta(t, 87, pageStep(100), 1e-6)
	assert.InDelta(t, 262, pageStep(300), 1e-6)
	want := float32(525)
	if runtime.GOOS == "darwin" {
		want = 560
	}
	assert.InDelta(t, want, pageStep(600), 1e-6, "on macOS, at least all of it but 40 points")
	assert.InDelta(t, 1, pageStep(0.5), 1e-6)
}
