//go:build gpu

package gpu //nolint:testpackage // white-box: counts what the renderer keeps by window

import (
	stdcolor "image/color"
	"testing"
	"time"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// litWindow is a window with what the renderer keeps textures for: a lit
// floor under a light and a probe, and a layer drawn as a pass.
func litWindow() *platform.Window {
	win := platform.NewWindow(platform.Size(200, 100), platform.Background(color.Black))
	win.Add(
		twod.NewCamera(light.Ambient(color.Black)),
		twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255}))),
		twod.NewLayer(1).Add(light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0)), light.NewProbe()),
		twod.NewLayer(2, twod.Blend(twod.Multiply)).Add(twod.NewSprite(filled(40, 40, stdcolor.RGBA{200, 0, 0, 255}))),
	)
	return win
}

func TestWhatTheRendererKeepsForAWindowLeavesWithIt(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	wm.OnGone(w, r.forget)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	frame := func(wins ...*platform.Window) {
		w.RunFrames(1)
		for _, win := range wins {
			r.answer(win)
			timers := r.timer(w, win)
			timers.On = true
			timers.Frame(func(string, time.Duration) {})
			r.window(w, win, target, timers, draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
			if err := wgpu.TakeError(); err != nil {
				t.Fatal(err)
			}
		}
	}
	kept := func() [6]int {
		l := &r.lights
		return [6]int{len(r.targets), len(r.timers), len(l.pictures), len(l.readbacks), len(l.lastRows), len(l.lastStretch)}
	}
	main := litWindow()
	w.AddEntities(main)
	frame(main)
	frame(main)
	base := kept()
	if base[0] == 0 || base[2] == 0 || base[3] == 0 {
		t.Fatalf("the scene must make a pass, light pictures and a readback to test their release: %v", base)
	}
	for range 25 {
		tool := litWindow()
		w.AddEntities(tool)
		frame(main, tool)
		frame(main, tool)
		if kept() == base {
			t.Fatal("a second window has textures of its own")
		}
		tool.RequestClose()
		frame(main)
		if got := kept(); got != base {
			t.Fatalf("after a window closed the renderer keeps %v, want what the first window alone had, %v", got, base)
		}
	}
	w.RemoveEntity(main)
	if got := kept(); got != ([6]int{}) {
		t.Fatalf("with no window left the renderer keeps %v", got)
	}
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
}
