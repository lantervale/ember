//go:build chrome

package chrome_test

import (
	"os"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestZProbe(t *testing.T) {
	out, err := chrome.EvalPage("file://"+os.Getenv("PROBE"), geom.V2(800, 600))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(out))
}
