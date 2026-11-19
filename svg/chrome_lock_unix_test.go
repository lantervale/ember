//go:build chrome && !windows

package svg_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// lockChrome takes the lock ember's other Chrome suites take
// (ui/internal/chrome): two headless Chromes started together on macOS
// can fail at teardown.
func lockChrome(t *testing.T) func() {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "ember-chrome.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		t.Fatal(err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
}
