package collide //nolint:testpackage // white-box: the unexported crowd, its cells and density

import (
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// widePile is n soft bodies on one point and four more at the corners of
// a span as wide as a level, 2,750 × 2,048 points.
func widePile(n int) []*thing {
	things := pile(n, geom.V2(1400, 1000))
	for _, c := range []geom.Vec2{{}, {X: 2750}, {Y: 2048}, {X: 2750, Y: 2048}} {
		things = append(things, newThing(c.X, c.Y, NewBody(Box(16, 10), Soft(1))))
	}
	return things
}

func TestACrowdOverAWideSpanKeepsCellsAtItsBodiesScale(t *testing.T) {
	_, space := setup(widePile(150)...)
	space.crowd.rebuild(space)
	assert.InDelta(t, 32, space.crowd.cell, 0.01, "twice the largest body's side, however far apart the bodies are")
}

// densest is how much soft bodies cover of the fullest square of side
// cell, the squares laid from the origin whatever the crowd's own cells.
func densest(things []*thing, cell float32) float32 {
	cover := map[[2]int]float32{}
	var most float32
	for _, t := range things {
		key := [2]int{int(geom.Floor(t.at.X / cell)), int(geom.Floor(t.at.Y / cell))}
		cover[key] += 16 * 10 / (cell * cell)
		most = max(most, cover[key])
	}
	return most
}

func TestBodiesPiledOnOnePointOfAWideSpanSpreadAsOnASmallOne(t *testing.T) {
	things := widePile(60)
	w, space := setup(things...)
	require.Greater(t, densest(things, 32), float32(1), "sixty on one point overfill their cell")
	w.RunFrames(300)
	assert.LessOrEqual(t, densest(things, 32), float32(1.5),
		"no square of twice a body's side is covered past half again its capacity, as decision 0028 holds")
	assert.LessOrEqual(t, fullest(space), float32(1), "they spread until no cell is over capacity")
	for i, a := range things {
		for _, b := range things[i+1:] {
			assert.NotEqual(t, a.at, b.at, "no two stay on the same spot")
		}
	}
}

func TestAQueryWiderThanTheCrowdFindsWhatTouchesIt(t *testing.T) {
	things := widePile(20)
	_, space := setup(things...)
	space.crowd.rebuild(space)
	all := space.crowd.collect(nil, geom.Rect{X: -100, Y: -100, W: 3000, H: 2300})
	assert.Len(t, all, len(things), "an area over every cell finds every body")
	corner := space.crowd.collect(nil, geom.Rect{X: 2700, Y: 2000, W: 100, H: 100})
	require.Len(t, corner, 1, "and a small one only the bodies under it")
	assert.Equal(t, space.BodyOf(things[len(things)-1]), corner[0])
}

func TestACrowdTooSpreadForItsCellsGrowsThemWithinItsGrid(t *testing.T) {
	far := []*thing{
		newThing(0, 0, NewBody(Box(16, 10), Soft(1))),
		newThing(100_000, 100_000, NewBody(Box(16, 10), Soft(1))),
	}
	_, space := setup(far...)
	space.crowd.rebuild(space)
	c := space.crowd
	assert.LessOrEqual(t, int(c.cols)*int(c.rows), maxCells)
	assert.Greater(t, c.cell, float32(32))
}

// within counts the pairs of things whose centres are within d.
func within(things []*thing, d float32) int {
	n := 0
	for i, a := range things {
		for _, b := range things[i+1:] {
			if a.at.Dist(b.at) < d {
				n++
			}
		}
	}
	return n
}

func TestACrowdPressingOnTheHeroStandsApart(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	hero := newThing(1375, 1024, NewBody(Box(16, 16)))
	var foes []*thing
	for range 150 {
		foes = append(foes, newThing(r.Float32()*2750, r.Float32()*2048, NewBody(Box(16, 10), Soft(1))))
	}
	w, space := setup(append(foes, hero)...)
	for range 1500 {
		for _, f := range foes {
			space.Move(f, hero.at.Sub(f.at).Normalize().Scale(2))
		}
		w.RunFrames(1)
	}
	assert.Equal(t, geom.V2(1375, 1024), hero.at, "a crowd never pushes the hero")
	assert.Zero(t, within(foes, 0.01), "none stand on the very same point as another")
	pairs := len(foes) * (len(foes) - 1) / 2
	assert.Less(t, within(foes, 2), pairs/100, "fewer than one pair in a hundred within 2 points")
}
