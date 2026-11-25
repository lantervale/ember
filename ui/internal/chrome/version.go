//go:build chrome

package chrome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/export"
)

// Version is the major version of the Chrome the tests run against, read
// from its user agent once: from the first Session's page, or a page of
// its own when none has opened. Asked while this process holds a Chrome
// slot and no Session has told it, it reads it in another slot only if
// one is free, and fails rather than wait, maybe on itself.
func Version() (int, error) { return version() }

//nolint:gochecknoglobals // one Chrome per process, its version read once
var known struct {
	sync.Mutex
	set   bool
	major int
	err   error
}

// knowAgent records the version a page's user agent says, unless one is
// known already.
func knowAgent(agent string) {
	known.Lock()
	defer known.Unlock()
	if !known.set {
		known.major, known.err = majorOf(agent)
		known.set = true
	}
}

func version() (int, error) {
	known.Lock()
	set, major, err := known.set, known.major, known.err
	known.Unlock()
	if set {
		return major, err
	}
	agent, err := readAgent()
	if err != nil {
		return 0, err
	}
	knowAgent(agent)
	known.Lock()
	defer known.Unlock()
	return known.major, known.err
}

// readAgent reads Chrome's user agent from a page of its own, in a slot
// of its own.
func readAgent() (string, error) {
	var unlock func()
	var err error
	if slotsHeld.Load() > 0 {
		if unlock, err = takeSlot(false); errors.Is(err, errNoSlot) {
			return "", errors.New("chrome: the version asked while this process holds a Chrome slot and none is free")
		}
	} else {
		unlock, err = lockChrome()
	}
	if err != nil {
		return "", err
	}
	defer unlock()
	var page bytes.Buffer
	if err = export.Write(&page, ui.NewDocument(), export.Head(EvalScript("return navigator.userAgent"))); err != nil {
		return "", err
	}
	out, err := runRaw(page.Bytes(), geom.V2(minWidth, minWidth), dumpDOM, css.Media{})
	if err != nil {
		out, err = runRaw(page.Bytes(), geom.V2(minWidth, minWidth), dumpDOM, css.Media{})
	}
	if err != nil {
		return "", err
	}
	res, err := evalResult(out)
	if err != nil {
		return "", err
	}
	var agent string
	err = json.Unmarshal(res, &agent)
	return agent, err
}

var agentVersion = regexp.MustCompile(`Chrome/(\d+)\.`)

func majorOf(agent string) (int, error) {
	m := agentVersion.FindStringSubmatch(agent)
	if m == nil {
		return 0, fmt.Errorf("no Chrome version in %q", agent)
	}
	return strconv.Atoi(m[1])
}

// Since skips t on a Chrome older than major, whose behaviour ui doesn't
// follow: ui tracks the newest stable Chrome, which CI pins
// (ui/internal/chrome/VERSION) and task chrome fetches; a machine running
// an older one skips. why says what major changed.
func Since(t testing.TB, major int, why string) {
	t.Helper()
	v, err := Version()
	if err != nil {
		t.Fatalf("Chrome's version: %v", err)
	}
	if v < major {
		t.Skipf("Chrome %d: %s since Chrome %d", v, why, major)
	}
}

// Pinned reports whether the Chrome the tests run against is the Chrome
// for Testing build CI pins (VERSION): a page's user agent gives no more
// than the major version, so the executable is asked, once.
func Pinned() (bool, error) {
	pinnedState.Lock()
	defer pinnedState.Unlock()
	if !pinnedState.asked {
		pinnedState.on, pinnedState.err = isPinned(browser())
		pinnedState.asked = true
	}
	return pinnedState.on, pinnedState.err
}

//nolint:gochecknoglobals // one Chrome per process, asked once
var pinnedState struct {
	sync.Mutex
	asked, on bool
	err       error
}

// isPinned reports whether the Chrome at exe is the pinned build: its
// --version says so, or, on Windows, where it prints nothing, the folder
// of its resources beside it is named by the version.
func isPinned(exe string) (bool, error) {
	v := strings.TrimSpace(pinned)
	if info, err := os.Stat(filepath.Join(filepath.Dir(exe), v)); err == nil && info.IsDir() {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "--version").Output()
	if err != nil {
		return false, fmt.Errorf("chrome: %s --version: %w", exe, err)
	}
	said := strings.Fields(string(out))
	return strings.Contains(string(out), "Chrome for Testing ") && slices.Contains(said, v), nil
}

// UnlikePinned skips t on the pinned Chrome for Testing build on systems
// (every system when none is named), which differs there from the
// stable Chrome ui follows: why says how. Elsewhere t runs.
func UnlikePinned(t testing.TB, why string, systems ...string) {
	t.Helper()
	if len(systems) > 0 && !slices.Contains(systems, runtime.GOOS) {
		return
	}
	on, err := Pinned()
	if err != nil {
		t.Fatalf("Chrome's version: %v", err)
	}
	if on {
		t.Skipf("Chrome for Testing %s on %s: %s", strings.TrimSpace(pinned), runtime.GOOS, why)
	}
}
