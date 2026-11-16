package collide //nolint:testpackage // white-box: shares the unexported thing helpers and newHorde

import (
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

// horde is n enemies of 16×10 boxes spread over a disc as bench2d's 20k
// scene spreads them, each moved a step towards the origin by step.
type horde struct {
	w      *world.World
	space  *Space
	things []*thing
}

func newHorde(n int, radius float32, opts ...BodyOption) *horde {
	r := rand.New(rand.NewPCG(1, 2))
	h := &horde{w: world.New(), space: NewSpace(Grid(32))}
	h.w.AddSystems(NewTracker(h.space))
	opts = append([]BodyOption{Offset(0, -5), Group(2), CollidesWith(2)}, opts...)
	for range n {
		at := geom.FromAngle(r.Float32() * 2 * geom.Pi).Scale(radius * geom.Sqrt(r.Float32()))
		t := newThing(at.X, at.Y, NewBody(Box(16, 10), opts...))
		h.things = append(h.things, t)
		h.w.AddEntities(t)
	}
	h.w.On(world.FixedUpdate, "step", func(world.Frame) {
		for _, t := range h.things {
			h.space.Move(t, t.at.Scale(-1).Normalize().Scale(1))
		}
	})
	return h
}

func benchHorde(b *testing.B, settle int, opts ...BodyOption) {
	benchHordeOf(b, 20000, 1712, settle, opts...)
}

func benchHordeOf(b *testing.B, n int, radius float32, settle int, opts ...BodyOption) {
	b.Helper()
	h := newHorde(n, radius, opts...)
	h.w.RunFrames(settle)
	before := cpu()
	for b.Loop() {
		h.w.RunFrames(1)
	}
	b.ReportMetric(float64(cpu()-before)/float64(b.N)/1e6, "cpu-ms/op")
}

func BenchmarkHorde20k(b *testing.B) { benchHorde(b, 60) }

// BenchmarkSoftHorde20k is the horde soft, still spread out, and
// BenchmarkSoftPile20k once it has settled into one pile on its goal.
func BenchmarkSoftHorde20k(b *testing.B) { benchHorde(b, 60, Soft(1)) }
func BenchmarkSoftPile20k(b *testing.B)  { benchHorde(b, 2000, Soft(1)) }

// BenchmarkSoftHorde300Wide is a level's few hundred monsters spread over
// a span as wide as a level, and BenchmarkSoftPile300Wide once they have
// piled on their goal.
func BenchmarkSoftHorde300Wide(b *testing.B) { benchHordeOf(b, 300, 1300, 60, Soft(1)) }
func BenchmarkSoftPile300Wide(b *testing.B)  { benchHordeOf(b, 300, 1300, 1500, Soft(1)) }
