package pages

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/cull"
)

// look draws d headless in a window of w×h points and looks all over it:
// the pointer goes over a grid of points, the wheel turned at each one,
// down then back up, so the viewport and every scroll container under it
// scroll, and the element at each point is asked for. Every paint and hit
// test is done both ways (cull.Check), and a difference panics.
func look(d *ui.Document, w, h int) {
	defer cull.Set(cull.Check, nil)()
	win := platform.NewWindow(platform.Size(w, h), platform.FixedSize())
	win.Add(d)
	world := ember.New().AddEntities(win)
	world.RunFrames(3)
	in := win.Input()
	const step = 29
	for _, lines := range []float32{2, 5, -3} {
		for y := float32(1); y < float32(h); y += step {
			for x := float32(1); x < float32(w); x += step {
				at := geom.V2(x, y)
				d.ElementAt(at)
				in.SetMouse(at)
				in.AddWheel(geom.V2(0, lines))
				world.RunFrames(1)
			}
		}
	}
}
