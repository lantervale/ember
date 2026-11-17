package collide //nolint:testpackage // white-box: shares the unexported thing helpers

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
)

// press is 150 soft 16×10 bodies, a game's monsters, come from the east
// at a hard 16×10 target off the crowd's cell lines, each walking at it a
// point a step until the crowd has settled round it.
func press(t *testing.T) (*Space, *thing, []*thing) {
	t.Helper()
	r := rand.New(rand.NewPCG(3, 4))
	target := newThing(13, 7, NewBody(Box(16, 10), Offset(0, -5), Group(1)))
	things := []*thing{target}
	for range 150 {
		things = append(things, newThing(300+r.Float32()*400, (r.Float32()-0.5)*200,
			NewBody(Box(16, 10), Offset(0, -5), Group(2), CollidesWith(1, 2), Soft(1))))
	}
	w, space := setup(things...)
	w.On(world.FixedUpdate, "walk", func(world.Frame) {
		for _, e := range things[1:] {
			if d := target.at.Sub(e.at); d.LenSq() > 1 {
				space.Move(e, d.Normalize())
			}
		}
	})
	w.RunFrames(1500)
	return space, target, things[1:]
}

func TestACrowdPressedFromOneSideStandsOffItsCellLines(t *testing.T) {
	space, _, crowd := press(t)
	on := 0
	for _, e := range crowd {
		x := space.BodyOf(e).Bounds().Center().X
		if geom.Abs(x-32*float32(math.Round(float64(x/32)))) < 2 {
			on++
		}
	}
	assert.LessOrEqual(t, float32(on)/float32(len(crowd)), float32(0.2),
		"%d of %d within 2 points of a 32-point line: about one in eight would be by chance", on, len(crowd))
}

func TestACrowdPressedRoundItsTargetOverlapsLittle(t *testing.T) {
	space, _, crowd := press(t)
	touching, deep := 0, 0
	for i, a := range crowd {
		ab := space.BodyOf(a).Bounds()
		for _, b := range crowd[i+1:] {
			bb := space.BodyOf(b).Bounds()
			w := min(ab.X+ab.W, bb.X+bb.W) - max(ab.X, bb.X)
			h := min(ab.Y+ab.H, bb.Y+bb.H) - max(ab.Y, bb.Y)
			if w <= 0 || h <= 0 {
				continue
			}
			touching++
			if w*h > ab.W*ab.H/3 {
				deep++
			}
		}
	}
	assert.LessOrEqual(t, float32(deep)/float32(touching), float32(0.05),
		"%d of %d touching pairs overlap by more than a third of a body", deep, touching)
}

func TestACrowdComingFromOneSideGoesRoundItsTarget(t *testing.T) {
	space, target, crowd := press(t)
	past := 0
	for _, e := range crowd {
		if space.BodyOf(e).Bounds().Center().X < target.at.X {
			past++
		}
	}
	assert.GreaterOrEqual(t, float32(past)/float32(len(crowd)), float32(0.3),
		"%d of %d came round to the target's far side", past, len(crowd))
}
