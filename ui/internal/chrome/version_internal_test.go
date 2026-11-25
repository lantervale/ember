//go:build chrome

package chrome

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
)

func TestMajorOfAUserAgent(t *testing.T) {
	for agent, want := range map[string]int{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) " +
			"HeadlessChrome/158.0.8127.37 Safari/537.36": 158,
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36": 156,
	} {
		if got, err := majorOf(agent); err != nil || got != want {
			t.Errorf("%q: %d, %v; want %d", agent, got, err, want)
		}
	}
	if _, err := majorOf("Firefox/140.0"); err == nil {
		t.Error("a user agent without Chrome's version has none")
	}
}

func TestVersionIsTheBrowsers(t *testing.T) {
	v, err := Version()
	if err != nil || v < 100 {
		t.Fatalf("Chrome %d, %v", v, err)
	}
}

// forget empties the known version until t ends, as at the process's start.
func forget(t *testing.T) {
	t.Helper()
	known.Lock()
	set, major, err := known.set, known.major, known.err
	known.set, known.major, known.err = false, 0, nil
	known.Unlock()
	t.Cleanup(func() {
		known.Lock()
		known.set, known.major, known.err = set, major, err
		known.Unlock()
	})
}

// pinAs has Pinned answer on until t ends.
func pinAs(t *testing.T, on bool) {
	t.Helper()
	pinnedState.Lock()
	asked, was, err := pinnedState.asked, pinnedState.on, pinnedState.err
	pinnedState.asked, pinnedState.on, pinnedState.err = true, on, nil
	pinnedState.Unlock()
	t.Cleanup(func() {
		pinnedState.Lock()
		pinnedState.asked, pinnedState.on, pinnedState.err = asked, was, err
		pinnedState.Unlock()
	})
}

// The pinned build is told by its executable: a page's user agent says
// 158.0.0.0 of every 158.
func TestThePinnedChromeIsToldByItsExecutable(t *testing.T) {
	v := strings.TrimSpace(pinned)
	self, err := os.Executable()
	require.NoError(t, err)
	for said, want := range map[string]bool{
		"Google Chrome for Testing " + v:       true,
		"Google Chrome for Testing " + v + "1": false,
		"Google Chrome 158.0.8127.37":          false,
	} {
		t.Setenv("EMBER_FAKE_CHROME", "version:"+said)
		got, askErr := isPinned(self)
		require.NoError(t, askErr)
		assert.Equal(t, want, got, said)
	}

	dir := t.TempDir()
	exe := filepath.Join(dir, "chrome.exe")
	require.NoError(t, os.WriteFile(exe, nil, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, v), 0o750))
	got, err := isPinned(exe)
	require.NoError(t, err)
	assert.True(t, got, "Windows' chrome.exe beside its version's folder")
}

func TestOnlyThePinnedChromeSkipsWhatItDrawsUnlike(t *testing.T) {
	pinAs(t, true)
	skipped := false
	t.Run("pinned, this system", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		UnlikePinned(t, "draws it otherwise", runtime.GOOS)
	})
	assert.True(t, skipped)
	t.Run("pinned, another system", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		UnlikePinned(t, "draws it otherwise", "plan9")
	})
	assert.False(t, skipped)

	pinAs(t, false)
	t.Run("stable", func(t *testing.T) {
		t.Cleanup(func() { skipped = t.Skipped() })
		UnlikePinned(t, "draws it otherwise")
	})
	assert.False(t, skipped)
}

func TestASessionTellsTheVersionItHoldsTheLockFor(t *testing.T) {
	forget(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><p>x</p>"))
	}))
	defer srv.Close()
	Open(t, srv.URL, geom.V2(100, 100), 1)
	asked := make(chan error, 1)
	go func() {
		v, err := Version()
		if err == nil && v < 100 {
			err = fmt.Errorf("Chrome %d", v)
		}
		asked <- err
	}()
	select {
	case err := <-asked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the version waited for the lock its own session holds")
	}
}
