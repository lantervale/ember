package nav //nolint:testpackage // white-box: the field's build counters and the level helpers of nav_test.go

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

// ruled is a grid whose solid cells a func decides: a type == can't
// compare, as a game's grid holding a func, a map or a slice is.
type ruled struct {
	*level
	solid func(x, y int) bool
	reads *int
}

func (r ruled) Solid(x, y int) bool {
	*r.reads++
	return r.solid(x, y)
}

func newRuled(l *level) ruled {
	return ruled{l, l.Solid, new(int)}
}

func TestAFieldOverAGridThatCannotBeComparedKeepsItsCells(t *testing.T) {
	r := newRuled(open(16))
	f := NewFlowField(r)
	f.SetTarget(at(8, 8))
	if *r.reads == 0 {
		t.Fatal("the first field reads the grid")
	}
	*r.reads = 0
	f.SetTarget(at(3, 3))
	if *r.reads != 0 {
		t.Fatalf("a new goal on the same grid at the same version reads none of it: %d reads", *r.reads)
	}
	if dir, ok := f.Toward(at(10, 3)); !ok || dir.X > -0.9 {
		t.Fatalf("towards the new goal: %v %v", dir, ok)
	}
	r.version++
	f.SetTarget(at(3, 4))
	if *r.reads == 0 {
		t.Fatal("a changed grid is read again")
	}
}

// ruledWindow is a window of the endless grid holding a func.
type ruledWindow struct {
	window
	solid func(x, y int) bool
	reads *int
}

func (w ruledWindow) Solid(x, y int) bool {
	*w.reads++
	return w.solid(x, y)
}

func (ruledWindow) Version() uint64 { return 0 }

func TestAFieldAroundCutsWindowsThatCannotBeCompared(t *testing.T) {
	var e endless
	reads := new(int)
	around := func(p geom.Vec2, cols, rows int) ruledWindow {
		w := e.around(p, cols, rows)
		return ruledWindow{w, w.Solid, reads}
	}
	f := NewFlowField(nil, Around(around, 32, 32))
	f.SetTarget(geom.V2(1000, 1000))
	*reads = 0
	f.SetTarget(geom.V2(1050, 1000))
	if e.cuts != 1 || *reads != 0 {
		t.Fatalf("the same window: %d cuts, %d reads", e.cuts, *reads)
	}
	f.SetTarget(geom.V2(1200, 1000))
	if e.cuts != 2 || *reads == 0 {
		t.Fatalf("a new window is read, never the old one's cells: %d cuts, %d reads", e.cuts, *reads)
	}
	if dir, ok := f.Toward(geom.V2(1150, 1000)); !ok || dir.X < 0.9 {
		t.Fatalf("towards the target in the new window: %v %v", dir, ok)
	}
}

func BenchmarkFlowFieldNewGoal128(b *testing.B) {
	f := NewFlowField(open(128))
	for i := 0; b.Loop(); i++ {
		f.SetTarget(at(64+i%2, 64))
	}
}
