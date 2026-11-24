//go:build chrome

package chrome

import (
	"encoding/json"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// Major is the major version of the Chrome the tests run, read once
// from its user agent.
//
//nolint:gochecknoglobals // read once per test process
var Major = sync.OnceValues(func() (int, error) {
	raw, err := Eval(ui.NewDocument(), geom.V2(100, 100), "return navigator.userAgent")
	if err != nil {
		return 0, err
	}
	var agent string
	if err = json.Unmarshal(raw, &agent); err != nil {
		return 0, err
	}
	return majorOf(agent), nil
})

var chromeVersion = regexp.MustCompile(`Chrome/(\d+)\.`)

// majorOf is the major version a Chrome user agent names, 0 for none.
func majorOf(agent string) int {
	m := chromeVersion.FindStringSubmatch(agent)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

// Since skips t where Chrome is older than version, the one that
// brought what t compares; why says what that version changed.
func Since(t testing.TB, version int, why string) {
	t.Helper()
	v, err := Major()
	if err != nil {
		t.Fatalf("reading Chrome's version: %v", err)
	}
	if v < version {
		t.Skipf("Chrome %d: %s since Chrome %d", v, why, version)
	}
}
