// Package wm is how engine packages find a World's windows: the
// primary one, the one being presented, and its input. Games hold their
// windows instead.
package wm

import (
	"math"
	"slices"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type registry struct {
	world.Group[platform.Windowed]
	// open are the windows in the order they were added, which the Group
	// doesn't keep: the first is the primary.
	open    []platform.Windowed
	gone    []func(*platform.Window)
	current *platform.Window
	owners  map[*platform.Window][]Entity
	covers  map[*platform.Window]cover
}

func of(w *world.World) *registry {
	if r, ok := world.TryResource[registry](w); ok {
		return r
	}
	r := world.SetResource(w, &registry{})
	w.AddView(r)
	return r
}

func (r *registry) AddByInterface(o world.Identifier) {
	e, ok := o.(platform.Windowed)
	if !ok {
		return
	}
	r.Group.AddByInterface(o)
	if !slices.Contains(r.open, e) {
		r.open = append(r.open, e)
	}
}

// Remove forgets a window leaving the World: the next oldest becomes the
// primary and takes its pads, and whoever keeps something by window is
// told.
func (r *registry) Remove(o world.Identifier) {
	e, ok := r.Get(o.ID())
	if !ok {
		return
	}
	r.Group.Remove(o)
	win := e.GetWindow()
	primary := r.open[0] == e
	r.open = slices.DeleteFunc(r.open, func(x platform.Windowed) bool { return x == e })
	if primary && len(r.open) > 0 {
		inputfeed.Of(r.open[0].GetWindow().Input()).AdoptGamepads(win.Input())
	}
	if r.current == win {
		r.current = nil
	}
	delete(r.owners, win)
	delete(r.covers, win)
	for _, fn := range r.gone {
		fn(win)
	}
}

// OnGone has fn called with each window that leaves the World, for what
// an engine package keeps by window to go with it.
func OnGone(w *world.World, fn func(win *platform.Window)) {
	r := of(w)
	r.gone = append(r.gone, fn)
}

// Primary is the oldest window in the World: the first added, then, once
// that one is gone, the first added of those left. Nil when it has none.
func Primary(w *world.World) *platform.Window {
	if r := of(w); len(r.open) > 0 {
		return r.open[0].GetWindow()
	}
	return nil
}

// Count is how many windows the World has.
func Count(w *world.World) int { return len(of(w).open) }

// Resolve is win, or the primary window when win is nil.
func Resolve(w *world.World, win *platform.Window) *platform.Window {
	if win != nil {
		return win
	}
	return Primary(w)
}

// Of is the window e is under, the nearest window among its parents, nil
// when it is under none.
func Of(w *world.World, e interface{ GetBasicEntity() *world.BasicEntity }) *platform.Window {
	for p := e.GetBasicEntity().Parent(); p != nil; p = p.Parent() {
		if parent, ok := w.Entity(p.ID()); ok {
			if windowed, isWindow := parent.(platform.Windowed); isWindow {
				return windowed.GetWindow()
			}
		}
	}
	return nil
}

// Home is the window e belongs to: win when set, else the window e is
// under, else the primary.
func Home(w *world.World, e interface{ GetBasicEntity() *world.BasicEntity }, win *platform.Window) *platform.Window {
	if win != nil {
		return win
	}
	if under := Of(w, e); under != nil {
		return under
	}
	return Primary(w)
}

// SetCurrent is set by the runner around each window's presentation.
func SetCurrent(w *world.World, win *platform.Window) { of(w).current = win }

// Presenting reports whether a runner is presenting a window right now.
func Presenting(w *world.World) bool { return of(w).current != nil }

// Target is the window being presented, else the primary window.
func Target(w *world.World) *platform.Window {
	if c := of(w).current; c != nil {
		return c
	}
	return Primary(w)
}

var none input.State //nolint:gochecknoglobals // the one empty input a windowless entity reads

// Input is the primary window's input, which also holds the gamepads, or
// an idle state when the World has no window.
func Input(w *world.World) *input.State {
	if p := Primary(w); p != nil {
		return p.Input()
	}
	return &none
}

// Plugin answers the windows asked to close and begins every window's
// input frame before any other system runs, so a World run by RunFrames
// or Tick sees what tests and backends set, and emits the pads that
// connected and disconnected, the touches that began, moved and lifted,
// and the system's preferences that changed.
type Plugin struct{}

func (Plugin) Build(w *world.World) { w.AddSystems(&inputFrames{}) }

type inputFrames struct {
	preferences world.Group[platform.Preferred]
}

func (*inputFrames) Phase() world.Phase { return world.PreUpdate }
func (*inputFrames) Priority() int      { return math.MaxInt }

func (s *inputFrames) Update(f world.Frame) {
	Closing(f.World)
	for e := range s.preferences.All() {
		p := e.GetPreferences()
		if a, changed := p.TakeAppearance(); changed {
			world.Emit(f.World, platform.AppearanceChanged{Appearance: a})
		}
		if m, changed := p.TakeMotion(); changed {
			world.Emit(f.World, platform.MotionChanged{Motion: m})
		}
		if l, changed := p.TakeLanguages(); changed {
			world.Emit(f.World, platform.LanguagesChanged{Languages: l})
		}
	}
	for _, e := range of(f.World).open {
		in := e.GetWindow().Input()
		feed := inputfeed.Of(in)
		feed.BeginFrame()
		connected, disconnected := feed.GamepadChanges()
		for _, id := range connected {
			world.Emit(f.World, input.GamepadConnected{ID: id, Name: in.Gamepad(id).Name()})
		}
		for _, id := range disconnected {
			world.Emit(f.World, input.GamepadDisconnected{ID: id})
		}
		for _, t := range in.Touches() {
			switch t.Phase {
			case input.TouchBegan:
				world.Emit(f.World, input.TouchStarted{Touch: t})
			case input.TouchMoving:
				world.Emit(f.World, input.TouchMoved{Touch: t})
			case input.TouchLifted:
				world.Emit(f.World, input.TouchEnded{Touch: t})
			case input.TouchResting:
			}
		}
	}
	of(f.World).owned(f.World)
}
