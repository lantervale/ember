//go:build chrome

package chrome

import (
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lantervale/ember/geom"
)

// PixelsUnderMouse is Pixels for a page the mouse is on: ember's capture
// holds the mouse as mouse scripts it (EMBER_CAPTURE_MOUSE), and Chrome's
// page, opened in a Session at the capture's scale, is shot once act has
// moved and pressed Chrome's mouse the same way. name is the case's, for
// its report.
func PixelsUnderMouse(t testing.TB, pkg, name string, w, h int, mouse string, act func(*Session)) Diff {
	t.Helper()
	dir := t.TempDir()
	shot, page := filepath.Join(dir, "ember.png"), filepath.Join(dir, "page.html")
	run := func(env ...string) {
		cmd := exec.CommandContext(t.Context(), "go", "run", pkg)
		cmd.Env = append(pageEnv(os.Environ(), filepath.Join(dir, "crash")), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v: %s", pkg, err, out)
		}
	}
	run("EMBER_CAPTURE="+shot, "EMBER_CAPTURE_FRAME=10", "EMBER_CAPTURE_MOUSE="+mouse)
	run("EMBER_UI_EXPORT="+page, "EMBER_UI_SNAPSHOTS="+shot)
	a := load(t, shot)
	html, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	}))
	t.Cleanup(srv.Close)
	flags := []string{"--disable-gpu"}
	if runtime.GOOS != "darwin" {
		// as runAt: the system's bars lie over the content (decision 0181)
		flags = append(flags, "--enable-features=OverlayScrollbar")
	}
	s := Open(t, srv.URL, geom.V2(float32(w), float32(h)), float32(a.Bounds().Dx())/float32(w), flags...)
	act(s)
	b := s.Shot()
	if b.Bounds().Size() != a.Bounds().Size() {
		t.Fatalf("ember's capture is %v, Chrome's %v", a.Bounds().Size(), b.Bounds().Size())
	}
	d := Compare(a, b)
	reports := t.TempDir()
	if out := os.Getenv("EMBER_UI_REPORTS"); out != "" {
		reports = out
	}
	report := filepath.Join(reports, name+".png")
	// The report is for reading the difference; failing to write it
	// leaves the comparison as it is.
	if f, ferr := os.Create(report); ferr == nil { //nolint:gosec // the folder the gate names in EMBER_UI_REPORTS
		if ferr = png.Encode(f, d.Report); ferr != nil {
			t.Logf("%s: %v", report, ferr)
		}
		f.Close()
	}
	t.Logf("%s: mean difference %.4f, %d of %d pixels differ; report %s", name, d.Mean, d.Pixels, d.Total, report)
	return d
}

// WithinUnderMouse is Within for PixelsUnderMouse, under the threshold
// of its page.
func WithinUnderMouse(t testing.TB, pkg, name string, w, h int, mouse string, act func(*Session),
	threshold float64) Diff {
	t.Helper()
	d := PixelsUnderMouse(t, pkg, name, w, h, mouse, act)
	check(t, "page", filepath.Base(pkg), d, threshold)
	return d
}
