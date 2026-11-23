//go:build chrome

package chrome

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

var (
	versionOnce  sync.Once
	versionMajor int
	versionErr   error
)

// Version is the major version of the Chrome the tests run against, read
// from its user agent once.
func Version() (int, error) {
	versionOnce.Do(func() {
		out, err := Eval(ui.NewDocument(), geom.V2(16, 16), "return navigator.userAgent")
		if err != nil {
			versionErr = err
			return
		}
		var agent string
		if err = json.Unmarshal(out, &agent); err != nil {
			versionErr = err
			return
		}
		versionMajor, versionErr = majorOf(agent)
	})
	return versionMajor, versionErr
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
