package ember_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The svg reader is linked only by what imports it: ui/svg, which
// registers it with ui, and the programs importing that. ui itself, its
// web fetcher, twod and a game that shows no SVG carry none of it.
func TestSVGIsLinkedOnlyByWhatImportsIt(t *testing.T) {
	const svg = "github.com/lantervale/ember/svg"
	deps := func(pkg string) []string {
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Skipf("go list %s: %v\n%s", pkg, err, out)
		}
		return strings.Fields(string(out))
	}
	for _, pkg := range []string{
		".", "./ui", "./ui/web", "./ui/kit", "./twod", "./examples/11-ui-elements", "./games/2048",
	} {
		if slices.Contains(deps(pkg), svg) {
			t.Errorf("%s links the svg reader without importing ui/svg", pkg)
		}
	}
	if !slices.Contains(deps("./ui/svg"), svg) {
		t.Error("./ui/svg does not link the svg reader")
	}
}
