//go:build chrome

package chrome

import (
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
)

// Pointer is one place of the mouse in a page's pointed comparison: where
// it is, in the window's points, and whether the left button is down.
type Pointer struct {
	At   geom.Vec2
	Down bool
}

// pointerSteps is how many of ember's steps each Pointer lasts.
const pointerSteps = 6

// captureMouse is ember's EMBER_CAPTURE_MOUSE script for path: the
// pointer at each place for pointerSteps steps, jumping between them, and
// the left button held through each run of places it is down for.
func captureMouse(path []Pointer) (string, int) {
	var parts []string
	held := -1
	for i, p := range path {
		from, to := i*pointerSteps, (i+1)*pointerSteps-1
		parts = append(parts, fmt.Sprintf("%g,%g@%d", p.At.X, p.At.Y, from), fmt.Sprintf("%g,%g@%d", p.At.X, p.At.Y, to))
		switch {
		case p.Down && held < 0:
			held = from
		case !p.Down && held >= 0:
			parts = append(parts, fmt.Sprintf("left:%d-%d", held, from))
			held = -1
		}
	}
	end := len(path) * pointerSteps
	if held >= 0 {
		parts = append(parts, fmt.Sprintf("left:%d-%d", held, end+pointerSteps))
	}
	return strings.Join(parts, ";"), end - 1
}

// Point moves Chrome's mouse through path, pressing the left button where
// a place is the first of a run that holds it down and letting it go at
// the first after.
func (s *Session) Point(path []Pointer) {
	s.t.Helper()
	down := false
	for _, p := range path {
		switch {
		case p.Down && !down:
			s.Mouse("mouseMoved", p.At, 0)
			s.Mouse("mousePressed", p.At, 1)
		case p.Down:
			s.Mouse("mouseMoved", p.At, 1)
		case down:
			s.Mouse("mouseMoved", p.At, 1)
			s.Mouse("mouseReleased", p.At, 1)
		default:
			s.Mouse("mouseMoved", p.At, 0)
		}
		down = p.Down
	}
}

// WithinPointing compares the page at pkg with Chrome, as Within does, with
// the mouse moved through path in both and its left button held where a
// place says: ember's capture plays it as EMBER_CAPTURE_MOUSE, and
// Chrome, given the exported page at rest, gets the same mouse events
// through its DevTools before its picture is taken. name tells this
// comparison's report and result from the page's others.
func WithinPointing(t testing.TB, pkg, name string, w, h int, path []Pointer, threshold float64) Diff {
	t.Helper()
	dir, reports := t.TempDir(), t.TempDir()
	if out := os.Getenv("EMBER_UI_REPORTS"); out != "" {
		reports = out
	}
	shot, page := filepath.Join(dir, "ember.png"), filepath.Join(dir, "page.html")
	run := func(env ...string) {
		cmd := exec.CommandContext(t.Context(), "go", "run", pkg)
		cmd.Env = append(pageEnv(os.Environ(), filepath.Join(dir, "crash")), "EMBER_UI_ZOOM=1")
		cmd.Env = append(cmd.Env, env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v: %s", pkg, err, out)
		}
	}
	mouse, last := captureMouse(path)
	run("EMBER_CAPTURE="+shot, fmt.Sprintf("EMBER_CAPTURE_FRAME=%d", last), "EMBER_CAPTURE_MOUSE="+mouse)
	run("EMBER_UI_EXPORT=" + page)
	a := load(t, shot)
	html, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	size := geom.V2(float32(w), float32(h))
	html, _ = fit(html, size, true)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = rw.Write(html)
	}))
	defer srv.Close()
	s := Open(t, srv.URL, size, float32(a.Bounds().Dx())/float32(w))
	// The page drawn before the pointer moves: what Chrome builds at its
	// first frame, such as a box's styled bars, then takes the moves as
	// they come.
	s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`, nil)
	s.Point(path)
	s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))`, nil)
	b := s.Shot()
	if b.Bounds().Size() != a.Bounds().Size() {
		t.Fatalf("ember's capture is %v, Chrome's %v", a.Bounds().Size(), b.Bounds().Size())
	}
	d := Compare(a, b)
	full := filepath.Base(pkg) + "@" + name
	report := filepath.Join(reports, full+".png")
	f, err := os.Create(report) //nolint:gosec // the folder the gate names in EMBER_UI_REPORTS
	if err == nil {
		if err = png.Encode(f, d.Report); err != nil {
			t.Logf("%s: %v", report, err)
		}
		f.Close()
	}
	t.Logf("%s: mean difference %.4f, %d of %d pixels differ; report %s", full, d.Mean, d.Pixels, d.Total, report)
	check(t, "page", full, d, threshold)
	return d
}
