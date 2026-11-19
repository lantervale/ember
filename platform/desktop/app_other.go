//go:build !darwin

package desktop

// Only macOS asks a program, not a window, to quit or to show itself
// again: elsewhere a program is its windows.
func watchApp() {}

func appQuitRequested() bool { return false }

func appReopened() bool { return false }
