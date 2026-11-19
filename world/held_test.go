package world_test

import (
	"runtime"
	"testing"
	"weak"

	"github.com/lantervale/ember/world"
)

func TestAChildRemovedIsNoLongerHeldByItsParent(t *testing.T) {
	parent := world.NewBasic()
	held := func() weak.Pointer[world.BasicEntity] {
		child := world.NewBasic()
		parent.AppendChild(child)
		parent.RemoveChild(child)
		return weak.Make(child)
	}()
	runtime.GC()
	if held.Value() != nil {
		t.Fatal("a parent still holds the child taken out of it")
	}
	runtime.KeepAlive(parent)
}

type carried struct{ what *[64]int }

func TestAnEventDroppedIsNoLongerHeldByTheWorld(t *testing.T) {
	w := world.New()
	held := func() weak.Pointer[[64]int] {
		what := new([64]int)
		world.Emit(w, carried{what})
		return weak.Make(what)
	}()
	for range 3 {
		w.EndFrame()
	}
	runtime.GC()
	if held.Value() != nil {
		t.Fatal("the World still holds what an event dropped frames ago carried")
	}
	runtime.KeepAlive(w)
}
