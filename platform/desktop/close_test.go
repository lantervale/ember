package desktop //nolint:testpackage // tests the driver's unexported rules for windows closing

import (
	"testing"
	"time"

	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestALoopWithoutAWindowWaitsOutTheFixedStep(t *testing.T) {
	const step = float32(1.0 / 60)
	if wait := idle(0, step); wait < 16*time.Millisecond || wait > 17*time.Millisecond {
		t.Fatalf("a frame that just began waits a whole step: %v", wait)
	}
	if wait := idle(10*time.Millisecond, step); wait < 6*time.Millisecond || wait > 7*time.Millisecond {
		t.Fatalf("a frame 10 ms old waits what is left of the step: %v", wait)
	}
	if wait := idle(40*time.Millisecond, step); wait != 0 {
		t.Fatalf("a late frame runs at once: %v", wait)
	}
}

func TestTheDriverFollowsThePrimaryAsWindowsClose(t *testing.T) {
	w := world.New().AddPlugins(wm.Plugin{})
	main := platform.NewWindow(platform.Title("Main"), platform.OnClose(platform.CloseOnly))
	tool := platform.NewWindow(platform.Title("Tool"), platform.OnClose(platform.CloseOnly))
	w.AddEntities(main, tool)
	d := &driver{world: w, natives: map[*platform.Window]*native{}}
	w.AddView(&d.windows)
	nm, nt := &native{window: main, entity: main}, &native{window: tool, entity: tool}
	d.natives[main], d.natives[tool] = nm, nt
	if gone := d.left(); len(gone) != 0 {
		t.Fatalf("no window left the World: %v", gone)
	}
	if d.primary != nm || d.input() != main.Input() || d.title() != "Main" {
		t.Fatal("the first window is the primary: its input takes the pads and the scripts")
	}
	main.RequestClose()
	wm.Closing(w)
	if gone := d.left(); len(gone) != 1 || gone[0] != nm {
		t.Fatal("the primary window's native is destroyed like any other once its entity left")
	}
	if d.primary != nt || d.input() != tool.Input() || d.title() != "Tool" {
		t.Fatal("once it closes the oldest window left is the primary")
	}
	tool.RequestClose()
	wm.Closing(w)
	if gone := d.left(); len(gone) != 1 || gone[0] != nt || len(d.natives) != 0 {
		t.Fatal("the last window's native goes too")
	}
	if d.primary != nil || d.input() != nil {
		t.Fatal("without a window the driver has no primary and no input to feed")
	}
	if d.title() != "Tool" {
		t.Fatal("the crash log stays under the last primary's title")
	}
	if w.ShouldQuit() {
		t.Fatal("CloseOnly leaves the World running without a window")
	}
	if err := d.afterTick(nil); err != nil {
		t.Fatalf("a frame without a window ends like any other: %v", err)
	}
}

func TestNothingAsksTheProgramToQuitOrReopenBeforeTheSystemDoes(t *testing.T) {
	if appQuitRequested() || appReopened() {
		t.Fatal("nothing asked before the system did")
	}
}
