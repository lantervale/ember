// Package gpu is the WebGPU presenter. It observes the World through views
// (it is never a system) and turns render components, the Environment,
// immediate buffers and the 2D lists into a lit, shadowed, bloomed and
// tonemapped frame.
package gpu

import (
	"os"

	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// Settings tune quality versus cost.
type Settings struct {
	MSAA       int
	ShadowSize int
	// RenderScale renders the 3D scene below native resolution (0.5..1).
	RenderScale float32
}

// Plugin installs the renderer and the pure render plugin it depends on.
type Plugin struct {
	Settings Settings
}

func (p Plugin) Build(a *world.World) {
	a.AddPlugins(render.Plugin{}, twod.Plugin{}, light.Plugin{})
	ui := &webgpu2D{}
	sc := newScene3D(p.Settings, ui)
	ui.scene = sc
	a.AddView(&sc.drawables).
		AddView(&sc.lights).
		AddView(&sc.cameras).
		AddView(&sc.looks).
		AddView(&sc.gizmos).
		AddView(&sc.billboards)
	a.AddPresenter(ui)
	wm.OnGone(a, ui.forget)
}

// forget releases what the renderer keeps for win, a window that left the
// World: its passes' textures, light pictures, 3D targets, timers and
// probes on their way back.
func (r *webgpu2D) forget(win *platform.Window) {
	for tex, t := range r.targets {
		if t.win == win {
			r.forgetGroups()
			t.release()
			delete(r.targets, tex)
		}
	}
	if t := r.timers[win]; t != nil {
		t.Release()
		delete(r.timers, win)
	}
	l := &r.lights
	if p := l.pictures[win]; p != nil {
		r.forgetGroups()
		p.release()
		delete(l.pictures, win)
	}
	if rb := l.readbacks[win]; rb != nil {
		rb.Release()
		delete(l.readbacks, win)
	}
	delete(l.lastRows, win)
	delete(l.lastStretch, win)
	if r.scene == nil {
		return
	}
	if s := r.scene.surfaces[win]; s != nil {
		s.releaseTargets()
		if s.shadow != nil {
			s.shadow.Release()
		}
		delete(r.scene.surfaces, win)
	}
}

// timing is whether the GPU's passes are timed this frame: one frame in
// gpuSample while w's F3 profiler shows, since timer queries stall the
// driver and F3 would show their cost as the game's, or every frame under
// EMBER_STATS (but EMBER_STATS=cpu).
func timing(w *world.World) bool {
	return statsRun() || twod.ProfilerShown(w) && w.Time().Frame%gpuSample == 0
}

// gpuSample is how many frames apart F3 times the GPU.
const gpuSample = 30

// statsRun is whether EMBER_STATS asks for the GPU's timings.
func statsRun() bool {
	v := os.Getenv("EMBER_STATS")
	return v != "" && v != "cpu"
}
