//go:build chrome

package pages_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPagesDrawAndHitTheSameCulled runs every page, the kit's cards and
// the pictures compared with Chrome's alike, with EMBER_UI_LOOK: drawn
// headless and scrolled about, each paint and each hit test done both
// walking every box and skipping those out of view (decision 0101). A
// page fails where the two draw or hit differently. It builds every page,
// so it runs with the Chrome suites (-tags chrome), though it needs no
// Chrome.
func TestPagesDrawAndHitTheSameCulled(t *testing.T) {
	t.Parallel()
	mains, err := filepath.Glob("*/main.go")
	require.NoError(t, err)
	require.NotEmpty(t, mains)
	bin := t.TempDir()
	args := []string{"build", "-o", bin + "/"}
	for _, m := range mains {
		args = append(args, "./"+filepath.Dir(m))
	}
	if out, buildErr := exec.Command("go", args...).CombinedOutput(); buildErr != nil {
		t.Fatalf("building the pages: %v: %s", buildErr, out)
	}
	for _, m := range mains {
		name := filepath.Dir(m)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			run := exec.Command(filepath.Join(bin, name))
			run.Dir = name
			run.Env = append(os.Environ(), "EMBER_UI_LOOK=1")
			if msg, runErr := run.CombinedOutput(); runErr != nil {
				t.Fatalf("%v: %s", runErr, msg)
			}
		})
	}
}
