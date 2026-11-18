//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

func TestThePicturesOfSpritesRemovedLeaveTheRenderer(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100), platform.Background(color.Black))
	win.Add(twod.NewCamera(twod.At(100, 50)))
	layer := twod.NewLayer(0)
	win.Add(layer)
	w.AddEntities(win)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	frame := func() {
		w.RunFrames(1)
		r.sweep()
		r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
		if err := wgpu.TakeError(); err != nil {
			t.Fatal(err)
		}
	}
	stays := twod.NewSprite(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 8, 8))), twod.At(20, 20))
	layer.Add(stays)
	frame()
	held := r.images.len()
	const sizes = 30
	for i := range sizes {
		sprite := twod.NewSprite(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 16+i, 16+i))), twod.At(100, 50))
		layer.Add(sprite)
		frame()
		w.RemoveEntity(sprite)
	}
	// The last sprite's picture may still be named by a draw list's spare
	// room, which the next sprite drawn there takes.
	deadline := time.Now().Add(5 * time.Second)
	for r.images.len() > held+1 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		frame()
	}
	if got := r.images.len(); got > held+1 {
		t.Fatalf("the renderer holds %d pictures after %d sprites came and went, want the %d it had", got, sizes, held)
	}
	runtime.KeepAlive(stays)
}
