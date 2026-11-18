package desktop //nolint:testpackage // tests the driver's unexported rule for a resize the system holds the loop in

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
)

func TestAWindowResizedWhileTheSystemHoldsTheLoopIsDrawnAtOnce(t *testing.T) {
	win := platform.NewWindow(platform.Size(1024, 680))
	win.Sync(geom.V2(1024, 680), geom.V2(2048, 1360), true)
	n := &native{window: win, configured: [2]int{2048, 1360}}
	d := &driver{live: true}
	if d.lags(n) {
		t.Fatal("a window whose frame is the size of its framebuffer has nothing to draw again")
	}
	win.Sync(geom.V2(700, 900), geom.V2(1400, 1800), true)
	if !d.lags(n) {
		t.Fatal("a window resized between two frames of the loop shows its last frame stretched: it is drawn at once")
	}
	d.framing = true
	if d.lags(n) {
		t.Fatal("a resize from inside a frame is drawn by that frame")
	}
	d.framing, d.live = false, false
	if d.lags(n) {
		t.Fatal("a hidden run or a capture draws its frames from the loop alone")
	}
	d.live = true
	n.configured = [2]int{1400, 1800}
	if d.lags(n) {
		t.Fatal("once drawn at the new size, the window's second size event draws nothing")
	}
}

func TestWhatTheSystemAnimatesIsStartedBetweenFrames(t *testing.T) {
	win, gone := platform.NewWindow(), platform.NewWindow()
	n, closed := &native{window: win}, &native{window: gone}
	d := &driver{natives: map[*platform.Window]*native{win: n}}
	n.driver, closed.driver = d, d
	var ran []string
	d.framing = true
	n.between(func() { ran = append(ran, "zoom") })
	closed.between(func() { ran = append(ran, "closed") })
	if len(ran) != 0 {
		t.Fatalf("inside a frame nothing can draw the system's animation, so it waits for the frame's end: ran %v", ran)
	}
	d.framing = false
	d.runBetween()
	d.runBetween()
	if len(ran) != 1 || ran[0] != "zoom" {
		t.Fatalf("ran %v, want the open window's once, and nothing for a window closed meanwhile", ran)
	}
}
