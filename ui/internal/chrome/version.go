//go:build chrome

package chrome

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// Version is the major version of the Chrome the tests run against, read
// from its user agent once: from the first Session's page, or a page of
// its own when none has opened. Asked while this process holds the Chrome
// lock and no Session has told it, it fails rather than wait on itself.
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
	defer known.Unlock()
	if known.set {
		return known.major, known.err
	}
	if lockHeld.Load() > 0 {
		return 0, errors.New("chrome: the version asked while this process holds the Chrome lock")
	}
	out, err := Eval(ui.NewDocument(), geom.V2(16, 16), "return navigator.userAgent")
	if err != nil {
		return 0, err
	}
	var agent string
	if err = json.Unmarshal(out, &agent); err != nil {
		return 0, err
	}
	known.major, known.err = majorOf(agent)
	known.set = true
	return known.major, known.err
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
// follow: ui tracks the newest stable Chrome, while CI may pin an older
// one. why says what major changed.
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
