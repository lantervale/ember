//go:build chrome && gpu

package reftest_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/reftest"
)

// settle waits for the page's fonts, then for two frames, so what
// Chrome shows is what it finished laying out.
const settle = `document.fonts.ready.then(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))))` //nolint:lll // one script

// pageWatch is the longest one page may take in Chrome, waiting for the
// Chrome lock included, before the run stops: a Chrome that dies without
// closing its pipe leaves the session waiting for an answer for ever,
// holding the lock every other run waits for.
const pageWatch = 5 * time.Minute

// TestReftestsMatchChrome draws each test of the gate's list in ember
// and in headless Chrome at 800×600, 1 pixel a point, and fails for each
// marked chrome whose mean difference passes reftest.ChromeThreshold.
// With EMBER_REFTEST_SURVEY set it compares every test of the default
// directories that ember draws instead, and fails for none. With
// EMBER_REFTEST_CHROME naming a file it writes the comparisons there
// after each test, for command run's -chrome, and a run that stopped
// takes up where it did, past the tests the file has.
func TestReftestsMatchChrome(t *testing.T) {
	root := checkout(t)
	only := expected
	want := map[string]reftest.Expectation{}
	if os.Getenv("EMBER_REFTEST_SURVEY") != "" {
		only = ""
	} else {
		var err error
		if want, err = reftest.ReadExpectations(expected); err != nil {
			t.Fatal(err)
		}
	}
	results, renderer := survey(t, root, only)
	srv := httptest.NewServer(reftest.Handler(root))
	t.Cleanup(srv.Close)
	version, err := chrome.Version()
	if err != nil {
		t.Fatal(err)
	}
	path := os.Getenv("EMBER_REFTEST_CHROME")
	out := reftest.ChromeResults{Chrome: version, Results: map[string]*reftest.ChromeResult{}}
	if data, readErr := os.ReadFile(path); path != "" && readErr == nil {
		if jsonErr := json.Unmarshal(data, &out); jsonErr != nil || out.Chrome != version {
			t.Fatalf("%s is of another run (Chrome %d, %v): remove it", path, out.Chrome, jsonErr)
		}
	}
	for _, r := range results {
		if r.Status != reftest.Pass && r.Status != reftest.Fail || out.Results[r.Path] != nil {
			continue
		}
		t.Run(r.Path, func(t *testing.T) {
			res := shoot(t, srv.URL, renderer, r.Path)
			out.Results[r.Path] = res
			if only != "" && want[r.Path].Chrome && !res.OK {
				t.Errorf("mean difference from Chrome %d %.4f, more than %.4f", version, res.Mean, res.Threshold)
			}
			if path != "" {
				write(t, path, out)
			}
		})
	}
}

// shoot compares ember's picture of the test at p with Chrome's.
func shoot(t *testing.T, server string, renderer *reftest.Renderer, p string) *reftest.ChromeResult {
	t.Helper()
	watch := time.AfterFunc(pageWatch, func() {
		panic(fmt.Sprintf("%s: Chrome took over %s", p, pageWatch))
	})
	defer watch.Stop()
	ember, err := reftest.LoadPNG(renderer.Picture(p))
	if err != nil {
		t.Fatal(err)
	}
	s := chrome.Open(t, server+"/"+p, geom.V2(reftest.Width, reftest.Height), 1, "--hide-scrollbars")
	s.Eval(settle, nil)
	d := chrome.Compare(ember, s.Shot())
	return &reftest.ChromeResult{Mean: d.Mean, Threshold: reftest.ChromeThreshold, Pixels: d.Pixels,
		OK: d.Mean <= reftest.ChromeThreshold}
}

func write(t *testing.T, path string, out reftest.ChromeResults) {
	t.Helper()
	data, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
