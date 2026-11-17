package svg_test

import (
	"os/exec"
	"strings"
	"testing"
)

// svg reads pictures on servers and in the browser as on the desktop: it
// depends on the standard library and x/image alone, nothing of ember's
// and no cgo.
func TestSVGStaysPureGo(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Skipf("go list: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		switch {
		case dep == "C":
			t.Error("svg uses cgo")
		case dep == "github.com/lantervale/ember/svg", strings.HasPrefix(dep, "golang.org/x/image/"):
		case strings.Contains(strings.SplitN(dep, "/", 2)[0], "."):
			t.Errorf("svg depends on %s", dep)
		case dep == "net", strings.HasPrefix(dep, "net/"), dep == "os/exec":
			t.Errorf("svg depends on %s: it reads the bytes it is given and fetches nothing", dep)
		}
	}
}
