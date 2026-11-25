//go:build gpu

package reftest_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/ui/internal/reftest"
)

// expected is the gate's list: the reftests that pass now.
const expected = "testdata/expected.txt"

// checkout is the WPT checkout: EMBER_WPT, else light's beside ember
// (task wpt:setup in light). Without one the gate is skipped.
func checkout(t *testing.T) string {
	t.Helper()
	root := os.Getenv("EMBER_WPT")
	if root == "" {
		root = filepath.Join("..", "..", "..", "..", "light", ".wpt", "wpt")
	}
	if _, err := os.Stat(filepath.Join(root, "MANIFEST.json")); err != nil {
		t.Skipf("no WPT checkout at %s (EMBER_WPT): %v", root, err)
	}
	return root
}

// survey draws the tests only lists ("" for every test of the
// directories EMBER_REFTEST_DIRS names by commas, else the default ones)
// and decides them; the pictures go to EMBER_REFTEST_OUT, else a test
// folder.
func survey(t *testing.T, root, only string) ([]reftest.Result, *reftest.Renderer) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "run")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "./run")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the runner: %v\n%s", err, out)
	}
	out := os.Getenv("EMBER_REFTEST_OUT")
	if out == "" {
		out = t.TempDir()
	}
	dirs := reftest.Dirs
	if d := os.Getenv("EMBER_REFTEST_DIRS"); d != "" {
		dirs = strings.Split(d, ",")
	}
	r := &reftest.Runner{Root: root, Dirs: dirs, Only: only, Renderer: reftest.Renderer{
		Binary: bin, Args: []string{"-worker", "-wpt", root}, Root: root, Out: out,
		Workers: min(4, runtime.NumCPU()), Timeout: 30 * time.Second,
	}}
	results, _, err := r.Run(context.WithoutCancel(t.Context()))
	if err != nil {
		t.Fatal(err)
	}
	return results, &r.Renderer
}

// TestReftestsStillPass draws every test of the gate's list marked ref
// and its references, and fails for each that no longer matches them.
func TestReftestsStillPass(t *testing.T) {
	root := checkout(t)
	want, err := reftest.ReadExpectations(expected)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	results, _ := survey(t, root, expected)
	seen := map[string]bool{}
	passing := 0
	for _, r := range results {
		seen[r.Path] = true
		if !want[r.Path].Ref {
			continue
		}
		if r.Status != reftest.Pass {
			t.Errorf("%s: %s: %s", r.Path, r.Status, r.Reason)
			continue
		}
		passing++
	}
	for p, e := range want {
		if e.Ref && !seen[p] {
			t.Errorf("%s: not in the checkout's manifest", p)
		}
	}
	t.Logf("%d reftests pass in %s (WPT %s)", passing, time.Since(start).Round(time.Second), reftest.Commit(root))
}
