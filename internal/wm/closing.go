package wm

import (
	"slices"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// answer is what rule does to a window asked to close: whether it closes,
// and whether the program ends. primary is whether the window is the
// primary one, last whether it is the only one.
func answer(rule platform.CloseRule, primary, last bool) (bool, bool) {
	switch rule {
	case platform.CloseQuitsIfPrimary:
		return !primary, primary
	case platform.CloseQuits:
		return false, true
	case platform.CloseQuitsIfLast:
		return true, last
	case platform.CloseOnly:
		return true, false
	case platform.CloseAsks:
	}
	return false, false
}

// Closing answers every window asked to close since it last ran, oldest
// first: it emits WindowCloseRequested, then follows the window's Closing
// rule, removing the window with a WindowClosed, quitting the World, both
// or neither. The World runs it as each frame begins; a runner that must
// know before the frame whether the World quits runs it itself.
func Closing(w *world.World) {
	// left are the windows not closed yet: inside a system the World
	// removes them only once it returns.
	left := slices.Clone(of(w).open)
	for _, e := range slices.Clone(left) {
		win := e.GetWindow()
		if !win.TakeCloseRequest() {
			continue
		}
		world.Emit(w, platform.WindowCloseRequested{Window: win})
		closes, quits := answer(win.Closing(), left[0] == e, len(left) == 1)
		if closes {
			left = slices.DeleteFunc(left, func(x platform.Windowed) bool { return x == e })
			world.Emit(w, platform.WindowClosed{Window: win})
			w.RemoveEntity(e)
		}
		if quits {
			w.Quit()
		}
	}
}
