package ember_test

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// labelled is a window with a label, as a tool window has.
func labelled(text string, opts ...platform.Option) *platform.Window {
	win := platform.NewWindow(append([]platform.Option{platform.Title(text), platform.Size(320, 200)}, opts...)...)
	doc := ui.NewDocument(kit.Theme())
	doc.Body().Append(kit.Label(text))
	win.Add(doc, twod.NewCamera())
	return win
}

// drawn is how much ui win drew in the last frame.
func drawn(w *world.World, win *platform.Window) int {
	if l := draw.FrontFor(w, win); l != nil {
		return len(l.Vertices)
	}
	return 0
}

func TestAWindowAddedToARunningWorldShowsAndTakesInput(t *testing.T) {
	w := ember.New().AddEntities(labelled("main"))
	w.RunFrames(3)
	later := labelled("later")
	w.AddEntities(later)
	w.RunFrames(2)
	if drawn(w, later) == 0 {
		t.Fatal("a window added after the World started draws its label")
	}
	later.Input().SetKey(input.KeyA, true)
	w.RunFrames(1)
	if !later.Input().Pressed(input.KeyA) {
		t.Fatal("its input frames begin like any window's")
	}
}

type eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
}

func TestWhatBelongsToNoWindowMovesToTheNextPrimary(t *testing.T) {
	main, tool := labelled("main", platform.OnClose(platform.CloseOnly)), labelled("tool")
	loose := ui.NewDocument(kit.Theme())
	loose.Body().Append(kit.Label("loose"))
	cam := &eye{BasicEntity: world.NewBasic()}
	w := ember.New().AddEntities(main, tool, loose, cam)
	w.RunFrames(2)
	if twod.CanvasFor(w, nil).Window() != main || cam.RendersIn() != main {
		t.Fatal("what names no window is the primary's")
	}
	alone := drawn(w, tool)

	main.RequestClose()
	w.RunFrames(3)
	if w.Has(main) || w.ShouldQuit() || wm.Primary(w) != tool {
		t.Fatal("the primary closed alone, and the tool window is the primary")
	}
	if c := twod.CanvasFor(w, nil); c.Window() != tool || c != twod.CanvasFor(w, tool) {
		t.Fatal("the canvas of no window in particular is the new primary's own")
	}
	if cam.RendersIn() != tool {
		t.Fatal("a camera under no window draws in the new primary")
	}
	if drawn(w, tool) <= alone {
		t.Fatal("a Document under no window shows in the new primary, with its own")
	}
	if draw.FrontFor(w, main) != nil {
		t.Fatal("nothing is drawn for a closed window")
	}
}

func TestAWorldRunsOnWithNoWindowAndTakesOneLater(t *testing.T) {
	only := labelled("only", platform.OnClose(platform.CloseOnly))
	loose := ui.NewDocument(kit.Theme())
	loose.Body().Append(kit.Label("loose"))
	w := ember.New().AddEntities(only, loose)
	w.RunFrames(2)
	only.Input().SetKey(input.KeyF3, true)
	w.RunFrames(2)
	only.RequestClose()
	w.RunFrames(5)
	if w.ShouldQuit() || wm.Count(w) != 0 {
		t.Fatal("the World runs on with no window, its debug overlay shown and a Document under none")
	}
	if c := twod.CanvasFor(w, nil); c == nil || c.Window() != nil {
		t.Fatal("the canvas of no window in particular is nobody's")
	}
	again := labelled("again")
	w.AddEntities(again)
	w.RunFrames(2)
	if twod.CanvasFor(w, nil).Window() != again || drawn(w, again) == 0 {
		t.Fatal("a window opened later is the primary and draws")
	}
}

func TestWindowsOpenedAndClosedLeaveNothingBehind(t *testing.T) {
	w := ember.New().AddEntities(labelled("main"))
	w.RunFrames(2)
	const rounds = 40
	heap := func() uint64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	before := heap()
	var closed []weak.Pointer[platform.Window]
	var docs []weak.Pointer[ui.Document]
	func() {
		for range rounds {
			win := labelled("tool")
			sprite := twod.NewPolygon(nil)
			win.Add(sprite)
			w.AddEntities(win)
			win.Input().SetMouse(win.Size().Scale(0.5))
			w.RunFrames(2)
			win.RequestClose()
			w.RunFrames(1)
			if w.Has(win) {
				t.Fatal("a secondary window asked to close leaves the World")
			}
			closed = append(closed, weak.Make(win))
			for _, c := range win.Children() {
				if doc, ok := c.(*ui.Document); ok {
					docs = append(docs, weak.Make(doc))
				}
			}
		}
	}()
	if len(docs) != rounds {
		t.Fatalf("%d Documents for %d windows", len(docs), rounds)
	}
	held, heldDocs := rounds, rounds
	deadline := time.Now().Add(5 * time.Second)
	for held+heldDocs > 2 && time.Now().Before(deadline) {
		runtime.GC()
		w.RunFrames(1)
		held, heldDocs = 0, 0
		for _, p := range closed {
			if p.Value() != nil {
				held++
			}
		}
		for _, p := range docs {
			if p.Value() != nil {
				heldDocs++
			}
		}
	}
	if held > 1 || heldDocs > 1 {
		t.Fatalf("of %d closed windows the engine still holds %d, and %d of their Documents", rounds, held, heldDocs)
	}
	// A Document's faces rasterise some 600 KB of glyphs each.
	if after := heap(); after > before+8<<20 {
		t.Fatalf("the heap grew %d KB over %d windows opened and closed", (after-before)>>10, rounds)
	}
}
