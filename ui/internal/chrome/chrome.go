//go:build chrome

// Package chrome is the layout tests' reference: it lays an exported
// Document out in headless Google Chrome and compares every element's
// getBoundingClientRect with its Node.Box. Tests run with -tags chrome;
// EMBER_CHROME names the browser when it isn't in /Applications.
package chrome

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// Tolerance is how far, in points, an edge may be from Chrome's.
const Tolerance = 0.5

const script = `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const els = [document.documentElement, document.body,
		...document.body.querySelectorAll("*:not([data-ember-implied]):not(svg *)")];
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-boxes";
	out.textContent = JSON.stringify(els.map(e => {
		const r = e.getBoundingClientRect();
		const box = [r.x, r.y, r.width, r.height];
		const parts = e.getClientRects();
		if (parts.length > 1 && getComputedStyle(e).display !== "inline") {
			for (const p of parts) box.push(p.x, p.y, p.width, p.height);
		}
		return box;
	}));
	document.head.appendChild(out);
	// A pseudo-element has no box a script can ask for: each ::before and
	// ::after box is named as an anchor, and a fixed probe sized and placed
	// by that anchor is measured in its place. A pseudo-element without a
	// box anchors nothing, and its probe falls back far off the page.
	const probes = [];
	let rules = "";
	els.forEach((e, i) => ["before", "after"].forEach((pe, k) => {
		const content = getComputedStyle(e, "::" + pe).content;
		if (content === "none" || content === "normal") return;
		e.setAttribute("data-ember-pe", i);
		rules += '[data-ember-pe="' + i + '"]::' + pe + " { anchor-name: --ember-" + pe + "-" + i + "; }\n";
		const p = document.createElement("ember-probe");
		p.style.cssText = "all: initial; position: fixed; visibility: hidden; pointer-events: none; " +
			"position-anchor: --ember-" + pe + "-" + i + "; left: anchor(left, -99999px); top: anchor(top, -99999px); " +
			"width: anchor-size(width, 0px); height: anchor-size(height, 0px);";
		probes.push([i, k, p]);
	}));
	const sheet = document.createElement("style");
	sheet.textContent = rules;
	document.head.appendChild(sheet);
	for (const p of probes) document.documentElement.appendChild(p[2]);
	const pseudo = document.createElement("script");
	pseudo.type = "application/json";
	pseudo.id = "ember-pseudo-boxes";
	pseudo.textContent = JSON.stringify(probes.map(([i, k, p]) => {
		const r = p.getBoundingClientRect();
		return [i, k, r.x, r.y, r.width, r.height];
	}).filter(b => b[2] > -90000));
	document.head.appendChild(pseudo);
}));
</script>`

var (
	boxesJSON  = regexp.MustCompile(`<script type="application/json" id="ember-boxes">(.*?)</script>`)
	pseudoJSON = regexp.MustCompile(`<script type="application/json" id="ember-pseudo-boxes">(.*?)</script>`)
)

// generatedBox is the border box of an element's ::before box, or of its
// ::after box when after, as Chrome lays it out; element counts html,
// body and body's elements in document order.
type generatedBox struct {
	element int
	after   bool
	rect    geom.Rect
}

func browser() string {
	if p := os.Getenv("EMBER_CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}

// Screenshot writes Chrome's picture of page in a viewport of size
// points to png, to set beside ember's drawing of the same document.
func Screenshot(page []byte, size geom.Vec2, out string) error {
	if _, err := run(page, size, "--screenshot="+out); err != nil {
		return err
	}
	return crop(out, int(size.X*deviceScale+0.5), int(size.Y*deviceScale+0.5))
}

// ScreenshotScaled is Screenshot on a display of scale device pixels per
// point, such as 2 to set beside ember's capture on a Retina display.
func ScreenshotScaled(page []byte, size geom.Vec2, scale float32, out string) error {
	deviceScale = scale
	defer func() { deviceScale = 1 }()
	return Screenshot(page, size, out)
}

// crop cuts the picture at path to w×h pixels from its top-left, when a
// narrow page ran in a wider window.
func crop(path string, w, h int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	if img.Bounds().Dx() <= w && img.Bounds().Dy() <= h {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	f, err = os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// Boxes lays page out in a viewport of size points, and returns the
// border boxes of html, body and body's elements in document order.
func Boxes(page []byte, size geom.Vec2) ([]geom.Rect, error) {
	rects, _, _, err := boxes(page, size)
	return rects, err
}

// boxes is Boxes with the parts of each block Chrome cut between
// columns, its getClientRects: none for a block in one piece, or for an
// inline element, whose rects are its lines'; and with the ::before and
// ::after boxes Chrome generated.
func boxes(page []byte, size geom.Vec2) ([]geom.Rect, [][]geom.Rect, []generatedBox, error) {
	out, err := run(page, size, dumpDOM)
	if err != nil {
		return nil, nil, nil, err
	}
	m := boxesJSON.FindSubmatch(out)
	if m == nil {
		return nil, nil, nil, fmt.Errorf("chrome: no boxes in the page:\n%s", out)
	}
	var raw [][]float32
	if err = json.Unmarshal(m[1], &raw); err != nil {
		return nil, nil, nil, err
	}
	var generated []generatedBox
	if pm := pseudoJSON.FindSubmatch(out); pm != nil {
		var list [][]float32
		if err = json.Unmarshal(pm[1], &list); err != nil {
			return nil, nil, nil, err
		}
		for _, g := range list {
			generated = append(generated, generatedBox{element: int(g[0]), after: g[1] == 1,
				rect: geom.Rect{X: g[2], Y: g[3], W: g[4], H: g[5]}})
		}
	}
	rects := make([]geom.Rect, len(raw))
	parts := make([][]geom.Rect, len(raw))
	for i, r := range raw {
		if len(r) < 4 || len(r)%4 != 0 {
			return nil, nil, nil, fmt.Errorf("chrome: box %d has %d numbers", i, len(r))
		}
		rects[i] = geom.Rect{X: r[0], Y: r[1], W: r[2], H: r[3]}
		for j := 4; j < len(r); j += 4 {
			parts[i] = append(parts[i], geom.Rect{X: r[j], Y: r[j+1], W: r[j+2], H: r[j+3]})
		}
	}
	return rects, parts, generated, nil
}

// run runs one Chrome at a time across test processes: two headless
// Chromes started together on macOS can fail at teardown. A failed run is
// tried once more.
func run(page []byte, size geom.Vec2, action string) ([]byte, error) {
	unlock, err := lockChrome()
	if err != nil {
		return nil, err
	}
	defer unlock()
	out, err := runOnce(page, size, action)
	if err != nil {
		out, err = runOnce(page, size, action)
	}
	return out, err
}

// lockPath is the file every run of Chrome locks, across test processes.
func lockPath() string { return filepath.Join(os.TempDir(), "ember-chrome.lock") }

// waitNote tells that a run waits for another's Chrome; waitAfter is how
// long it waits before saying so, and again between notes.
//
//nolint:gochecknoglobals // the lock test swaps them to watch the notes quickly
var (
	waitNote  = func(s string) { fmt.Fprintln(os.Stderr, s) }
	waitAfter = 5 * time.Second
)

// lockChrome takes the Chrome lock and writes who holds it into the file:
// full suites in several sessions queue on it for an hour, and a run
// waiting behind them says whom it waits for instead of seeming to hang.
func lockChrome() (func(), error) {
	f, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err = lockFile(f, false); err != nil {
		done := make(chan struct{})
		go func() {
			start := time.Now()
			for {
				select {
				case <-done:
					return
				case <-time.After(waitAfter):
					holder, _ := os.ReadFile(lockPath())
					waitNote(fmt.Sprintf("chrome: waiting for the Chrome lock %s for %s, held by %s",
						lockPath(), time.Since(start).Round(time.Second), strings.TrimSpace(string(holder))))
				}
			}
		}()
		err = lockFile(f, true)
		close(done)
		if err != nil {
			f.Close()
			return nil, err
		}
	}
	// The holder's note only tells those waiting whom for: a failed write
	// leaves the lock held all the same.
	_ = f.Truncate(0)
	note := fmt.Sprintf(
		"pid %d (%s) since %s",
		os.Getpid(),
		filepath.Base(os.Args[0]),
		time.Now().Format(time.TimeOnly),
	)
	_, _ = f.WriteAt([]byte(note), 0)
	return func() {
		_ = f.Truncate(0)
		// Closing the file releases the lock too, should unlocking fail.
		_ = unlockFile(f)
		f.Close()
	}, nil
}

// dumpDOM is the action that prints the laid-out page.
const dumpDOM = "--dump-dom"

// minWidth is the narrowest window headless Chrome keeps.
const minWidth = 500

// minNarrowHeight is the shortest window a narrow page's picture is taken
// in: Chrome for Testing on CI runners drew only the first line of narrow,
// short pages (160×104, 480×120). Wide pages keep their window, where a
// modal dialog centres as in ui.
const minNarrowHeight = 500

// fit makes page lay out in a viewport of size as ui's does: the root
// never shows a scrollbar, and a page smaller than the window it opens in
// keeps its size on the root. It returns the window to open: at least
// minWidth wide, and for a narrow page's picture minNarrowHeight tall,
// cropped back to size. The page of a Document whose viewport scrolls
// (export.ScrollingViewport) is left as it is, so it needs a viewport at
// least minWidth wide to dump its boxes; its picture is taken in a frame.
func fit(page []byte, size geom.Vec2, shot bool) ([]byte, geom.Vec2) {
	rule := "html { scrollbar-width: none; }"
	window := size
	if size.X < minWidth {
		window.X = minWidth
		if shot {
			window.Y = max(window.Y, minNarrowHeight)
		}
	}
	if bytes.Contains(page, []byte(" "+export.ScrollingViewport)) {
		if shot {
			return framed(page, size), window
		}
		return page, window
	}
	if window.X > size.X {
		rule += fmt.Sprintf(" html { width: %gpx; }", size.X)
	}
	if shot && window != size {
		// Chrome's window is bigger than the page, so the initial
		// containing block is too: for a picture, html becomes the
		// containing block of absolute boxes at the page's size, so right
		// and bottom resolve as ui's do. Box dumps leave html's own box
		// as ui has it.
		rule += fmt.Sprintf(" html { width: %gpx; height: %gpx; position: relative; }", size.X, size.Y)
	}
	return bytes.Replace(page, []byte("</head>"), []byte("<style>"+rule+"</style>\n</head>"), 1), window
}

// framed is the page of a Document whose viewport scrolls, in a frame of
// size filling another page, for its picture: headless Chrome's
// screenshot of a window scrolled itself comes out blank, while a frame's
// viewport scrolls, holds fixed boxes and paints its canvas as a window's.
func framed(page []byte, size geom.Vec2) []byte {
	return fmt.Appendf(nil, `<!DOCTYPE html><html><body style="margin: 0">`+
		`<iframe style="display: block; border: 0; width: %gpx; height: %gpx" srcdoc="%s"></iframe></body></html>`,
		size.X, size.Y, html.EscapeString(string(page)))
}

// margin is how much bigger than its viewport Chrome's window is when it
// dumps the DOM, for the browser's own parts, measured once; a screenshot
// run's viewport is its window.
//
//nolint:gochecknoglobals // measured once per test process
var (
	margin     geom.Vec2
	marginOnce sync.Once
)

var viewportJSON = regexp.MustCompile(`<script type="application/json" id="ember-viewport">(.*?)</script>`)

// measureMargin opens an empty page and reads the viewport Chrome gave
// it, to size later windows so their viewports are as asked.
func measureMargin() {
	page := []byte(
		`<!DOCTYPE html><html><head><script>addEventListener("load", () => {
	const o = document.createElement("script");
	o.type = "application/json";
	o.id = "ember-viewport";
	o.textContent = JSON.stringify([innerWidth, innerHeight]);
	document.head.appendChild(o);
});</script></head><body></body></html>`,
	)
	want := geom.V2(800, 600)
	out, err := runRaw(page, want, dumpDOM)
	if err != nil {
		return
	}
	m := viewportJSON.FindSubmatch(out)
	if m == nil {
		return
	}
	var got [2]float32
	if json.Unmarshal(m[1], &got) == nil {
		margin = geom.V2(want.X-got[0], want.Y-got[1])
	}
}

func runOnce(page []byte, size geom.Vec2, action string) ([]byte, error) {
	page, size = fit(page, size, action != dumpDOM)
	if action == dumpDOM {
		marginOnce.Do(measureMargin)
		size = size.Add(margin)
	}
	return runRaw(page, size, action)
}

// runRaw runs Chrome once on page with a window of size.
func runRaw(page []byte, size geom.Vec2, action string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "ember-chrome-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "page.html")
	if err = os.WriteFile(file, page, 0o600); err != nil { //nolint:gosec // a file in our own temporary folder
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// --run-all-compositor-stages-before-draw: headless shoots when the
	// virtual time runs out, before its tiles are rasterized on a loaded
	// machine, which left pages half drawn.
	cmd := exec.CommandContext( //nolint:gosec // the Chrome found on this machine, run on our own page
		ctx,
		browser(),
		"--headless=new",
		"--disable-gpu",
		"--run-all-compositor-stages-before-draw",
		fmt.Sprintf("--force-device-scale-factor=%g", deviceScale),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--user-data-dir="+filepath.Join(dir, "profile"),
		fmt.Sprintf("--window-size=%d,%d", int(size.X), int(size.Y)),
		"--virtual-time-budget=10000",
		action,
		"file://"+file,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if bytes.Contains(stderr.Bytes(), []byte("Teardown watchdog expired")) && done(action, out) {
			return out, nil
		}
		return nil, fmt.Errorf("chrome: %w: %s", err, stderr.String())
	}
	return out, nil
}

// done reports whether Chrome finished action before it exited: a whole
// dumped page, or a picture that decodes. On a loaded Mac, Chrome's
// teardown watchdog kills it after its work is written, which is no
// failure of the page.
func done(action string, out []byte) bool {
	if action == dumpDOM {
		return bytes.Contains(out, []byte("</html>"))
	}
	path, ok := strings.CutPrefix(action, "--screenshot=")
	if !ok {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = png.DecodeConfig(f)
	return err == nil
}

// deviceScale is the device pixels per point Chrome lays out at; 1
// unless MatchScaled sets it.
var deviceScale float32 = 1 //nolint:gochecknoglobals // set for the length of one MatchScaled

// MatchScaled is Match on a display of scale device pixels per point,
// where both round font metrics to device pixels.
func MatchScaled(t testing.TB, d *ui.Document, size geom.Vec2, scale float32) Report {
	t.Helper()
	deviceScale = scale
	fonts.Of(d).SetScale(scale)
	defer func() { deviceScale = 1; fonts.Of(d).SetScale(1) }()
	return Match(t, d, size)
}

// Report is what Match compared.
type Report struct {
	Boxes    int
	MaxDelta float32
}

// Match lays d out in a viewport of size points both in ui and in
// Chrome, and fails t for every element whose box differs by more than
// Tolerance on an edge, and so for every ::before and ::after box: one of
// the two alone having it fails too. Report.Boxes counts them all.
func Match(t testing.TB, d *ui.Document, size geom.Vec2) Report {
	t.Helper()
	var page bytes.Buffer
	if err := export.Write(&page, d, export.Head(script)); err != nil {
		t.Fatal(err)
	}
	want, wantParts, wantGenerated, err := boxes(page.Bytes(), size)
	if err != nil {
		t.Fatal(err)
	}
	d.Layout(size)
	var got []*dom.Node
	d.HTML().Walk(func(n *dom.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(got), len(want))
	}
	r := Report{Boxes: len(got)}
	for i, n := range got {
		g, w := n.Box(), want[i]
		delta := edgeDelta(g, w)
		got, parts := cutParts(n, g), wantParts[i]
		if len(got) != len(parts) {
			t.Errorf("%s: ember cuts it in %d parts %v, Chrome in %d %v", path(n), len(got), got, len(parts), parts)
		} else {
			for k := range parts {
				delta = max(delta, edgeDelta(got[k], parts[k]))
			}
		}
		r.MaxDelta = max(r.MaxDelta, delta)
		if delta > Tolerance {
			t.Errorf("%s: ember %v %v, Chrome %v %v", path(n), g, got, w, parts)
		}
	}
	chromes := map[*dom.Node]geom.Rect{}
	for _, g := range wantGenerated {
		data := dom.Data(got[g.element])
		node, name := data.Before, "::before"
		if g.after {
			node, name = data.After, "::after"
		}
		if node == nil {
			t.Errorf("%s%s: Chrome has a box %v, ember none", path(got[g.element]), name, g.rect)
			continue
		}
		chromes[node] = g.rect
	}
	for _, n := range got {
		for _, node := range []*dom.Node{dom.Data(n).Before, dom.Data(n).After} {
			if node == nil {
				continue
			}
			name := path(n) + "::" + node.Pseudo().String()
			w, ok := chromes[node]
			if !ok {
				t.Errorf("%s: ember has a box %v, Chrome none", name, node.Box())
				continue
			}
			r.Boxes++
			delta := edgeDelta(node.Box(), w)
			r.MaxDelta = max(r.MaxDelta, delta)
			if delta > Tolerance {
				t.Errorf("%s: ember %v, Chrome %v", name, node.Box(), w)
			}
		}
	}
	return r
}

func edgeDelta(g, w geom.Rect) float32 {
	return max(abs(g.X-w.X), abs(g.Y-w.Y), abs(g.X+g.W-w.X-w.W), abs(g.Y+g.H-w.Y-w.H))
}

// cutParts are the parts of n's box that columns cut, at box in the
// document's space: what Chrome's getClientRects gives for a block cut
// between columns, where its getBoundingClientRect is their union.
func cutParts(n *dom.Node, at geom.Rect) []geom.Rect {
	b, ok := dom.Data(n).Placed.(*box.Box)
	if !ok || !b.Broken {
		return nil
	}
	out := make([]geom.Rect, len(b.Fragments))
	for i, f := range b.Fragments {
		out[i] = geom.Rect{X: at.X + f.X, Y: at.Y + f.Y, W: f.W, H: f.H}
	}
	return out
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }

func path(n *dom.Node) string {
	s := n.Tag().String()
	if n.ID() != "" {
		s += "#" + n.ID()
	}
	if p := n.Parent(); p != nil {
		return fmt.Sprintf("%s > %s:%d", path(p), s, n.Index())
	}
	return s
}

var stylesJSON = regexp.MustCompile(`<script type="application/json" id="ember-styles">(.*?)</script>`)

// Styles lays d out in Chrome in a viewport of size points, and returns
// getComputedStyle's value of each of props for html, body and body's
// elements, in document order.
func Styles(d *ui.Document, size geom.Vec2, props []string) ([]map[string]string, error) {
	return StylesAfter(d, size, props, "")
}

// StylesAfter is Styles with script run in the page first, such as a
// class change that starts transitions, paused at a time.
func StylesAfter(d *ui.Document, size geom.Vec2, props []string, script string) ([]map[string]string, error) {
	names, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	head := `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const els = [document.documentElement, document.body,
		...document.body.querySelectorAll("*:not([data-ember-implied]):not(svg *)")];
	` + script + `
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-styles";
	out.textContent = JSON.stringify(els.map(e => {
		const s = getComputedStyle(e);
		return Object.fromEntries(` + string(names) + `.map(p => [p, s.getPropertyValue(p)]));
	}));
	document.head.appendChild(out);
}));
</script>`
	var page bytes.Buffer
	if err = export.Write(&page, d, export.Head(head)); err != nil {
		return nil, err
	}
	out, err := run(page.Bytes(), size, dumpDOM)
	if err != nil {
		return nil, err
	}
	m := stylesJSON.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("chrome: no styles in the page:\n%s", out)
	}
	var styles []map[string]string
	if err = json.Unmarshal(m[1], &styles); err != nil {
		return nil, err
	}
	return styles, nil
}

var evalJSON = regexp.MustCompile(`<script type="application/json" id="ember-eval">(.*?)</script>`)

// Eval lays d out in Chrome in a viewport of size points, evaluates the
// JavaScript expression expr once the fonts are ready, and returns its
// value as JSON.
func Eval(d *ui.Document, size geom.Vec2, expr string) ([]byte, error) {
	head := `<script>
addEventListener("load", () => document.fonts.ready.then(() => {
	const out = document.createElement("script");
	out.type = "application/json";
	out.id = "ember-eval";
	out.textContent = JSON.stringify((() => { ` + expr + ` })());
	document.head.appendChild(out);
}));
</script>`
	var page bytes.Buffer
	if err := export.Write(&page, d, export.Head(head)); err != nil {
		return nil, err
	}
	out, err := run(page.Bytes(), size, dumpDOM)
	if err != nil {
		return nil, err
	}
	m := evalJSON.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("chrome: no result in the page:\n%s", out)
	}
	return m[1], nil
}
