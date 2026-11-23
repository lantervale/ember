//go:build chrome

package chrome

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
)

// Session is a page open in headless Chrome, driven step by step in real
// time through its DevTools pipe, where a page given to the other calls
// runs on a virtual clock until it settles: what a test watches change
// as time passes or as the page is scrolled, resized or asked for other
// media, with lazy pictures loading as a desktop Chrome loads them.
type Session struct {
	t       testing.TB
	cmd     *exec.Cmd
	in      *os.File
	out     *bufio.Reader
	id      int
	session string
	size    geom.Vec2
	scale   float32
}

// Serve serves d, exported, at the root of a server on 127.0.0.1, and
// the files of files under their names, each after delay; the func it
// returns tells the paths of those files asked for so far, in order.
func Serve(t testing.TB, d *ui.Document, files map[string][]byte,
	delay time.Duration) (*httptest.Server, func() []string) {
	t.Helper()
	var page bytes.Buffer
	if err := export.Write(&page, d); err != nil {
		t.Fatal(err)
	}
	asked := make(chan string, 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(page.Bytes())
			return
		}
		data, ok := files[r.URL.Path[1:]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		asked <- r.URL.RequestURI()
		time.Sleep(delay)
		w.Header().Set("Content-Type", http.DetectContentType(data))
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	var seen []string
	return srv, func() []string {
		for {
			select {
			case p := <-asked:
				seen = append(seen, p)
			default:
				return seen
			}
		}
	}
}

// Open opens url, served from this machine, in a viewport of size points
// at scale device pixels a point, Chrome run with flags besides its own,
// and waits for its load event. It skips the test where Go can't hand
// Chrome its pipe (Windows).
func Open(t testing.TB, url string, size geom.Vec2, scale float32, flags ...string) *Session {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chrome: no DevTools pipe on Windows")
	}
	unlock, err := lockChrome()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unlock)
	dir := t.TempDir()
	toChrome, theirIn, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	theirOut, fromChrome, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	//nolint:gosec // the Chrome found on this machine, run on our own page
	cmd := exec.CommandContext(ctx, browser(), append(flags, "--headless=new", "--remote-debugging-pipe",
		"--no-first-run", "--no-default-browser-check", "--disable-extensions",
		"--user-data-dir="+filepath.Join(dir, "profile"),
		// Headless Chrome loads lazy pictures at once unless told.
		"--blink-settings=lazyLoadEnabled=true",
		fmt.Sprintf("--window-size=%d,%d", int(size.X), int(size.Y)), "about:blank")...)
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome}
	if err = cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	toChrome.Close()
	fromChrome.Close()
	s := &Session{t: t, cmd: cmd, in: theirIn, out: bufio.NewReader(theirOut), size: size, scale: scale}
	t.Cleanup(func() {
		theirIn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cancel()
		theirOut.Close()
	})
	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	s.call("Target.getTargets", nil, &targets)
	for _, target := range targets.TargetInfos {
		if target.Type == "page" {
			var attached struct {
				SessionID string `json:"sessionId"`
			}
			s.call("Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &attached)
			s.session = attached.SessionID
			break
		}
	}
	if s.session == "" {
		t.Fatal("chrome: no page to drive")
	}
	s.Resize(size, scale)
	s.call("Page.navigate", map[string]any{"url": url}, nil)
	for range 500 {
		var ready string
		s.Eval(`document.readyState`, &ready)
		if ready == "complete" {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("chrome: the page never loaded")
	return nil
}

// call sends method with params and decodes its result into out.
func (s *Session) call(method string, params, out any) {
	s.t.Helper()
	s.id++
	msg := map[string]any{"id": s.id, "method": method, "params": params}
	if params == nil {
		msg["params"] = map[string]any{}
	}
	if s.session != "" {
		msg["sessionId"] = s.session
	}
	b, err := json.Marshal(msg)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err = s.in.Write(append(b, 0)); err != nil {
		s.t.Fatal(err)
	}
	for {
		var line []byte
		if line, err = s.out.ReadBytes(0); err != nil {
			s.t.Fatalf("chrome: %s: %v", method, err)
		}
		var resp struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err = json.Unmarshal(line[:len(line)-1], &resp); err != nil || resp.ID != s.id {
			continue
		}
		if resp.Error != nil {
			s.t.Fatalf("chrome: %s: %s", method, resp.Error.Message)
		}
		if out != nil {
			if err = json.Unmarshal(resp.Result, out); err != nil {
				s.t.Fatal(err)
			}
		}
		return
	}
}

// Eval evaluates the JavaScript expr in the page, waiting for a promise
// it gives, and decodes its value into out (nil to drop it).
func (s *Session) Eval(expr string, out any) {
	s.t.Helper()
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	s.call("Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &res)
	if res.ExceptionDetails != nil {
		s.t.Fatalf("chrome: %s: %s", expr, res.ExceptionDetails.Text)
	}
	if out != nil && res.Result.Value != nil {
		if err := json.Unmarshal(res.Result.Value, out); err != nil {
			s.t.Fatal(err)
		}
	}
}

// Resize gives the viewport size points at scale device pixels a point,
// as moving the window to another screen or resizing it would.
func (s *Session) Resize(size geom.Vec2, scale float32) {
	s.t.Helper()
	s.size, s.scale = size, scale
	s.call("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": int(size.X), "height": int(size.Y), "deviceScaleFactor": scale, "mobile": false}, nil)
}

// Media has the page answer the media feature name with value, as the
// system changing it would (prefers-color-scheme: dark…).
func (s *Session) Media(name, value string) {
	s.t.Helper()
	s.call("Emulation.setEmulatedMedia", map[string]any{
		"features": []map[string]string{{"name": name, "value": value}}}, nil)
}

// Mouse sends Chrome's input a mouse event of kind (mousePressed,
// mouseReleased, mouseMoved) at at, in the viewport's points: the left
// button, a press or a release the clicks'th of a run, a move holding the
// button where clicks is not 0.
func (s *Session) Mouse(kind string, at geom.Vec2, clicks int) {
	s.t.Helper()
	ev := map[string]any{"type": kind, "x": at.X, "y": at.Y, "button": "left", "clickCount": clicks}
	if kind == "mouseMoved" {
		ev["clickCount"] = 0
		if clicks == 0 {
			ev["button"] = "none"
		} else {
			ev["buttons"] = 1
		}
	}
	s.call("Input.dispatchMouseEvent", ev, nil)
}

// ShiftArrow presses and lets go the arrow key named key (ArrowDown…)
// with Shift held, as extending a selection by the keyboard does.
func (s *Session) ShiftArrow(key string) {
	s.t.Helper()
	s.Key(key, Shift)
}

// Shot is the viewport as Chrome shows it now, in device pixels.
func (s *Session) Shot() image.Image {
	s.t.Helper()
	var shot struct {
		Data string `json:"data"`
	}
	s.call("Page.captureScreenshot", map[string]any{"format": "png"}, &shot)
	b, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		s.t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		s.t.Fatal(err)
	}
	return img
}

// Modifier keys held through a Session's Press, Release, Move and Key,
// as DevTools numbers them.
const (
	Alt   = 1
	Ctrl  = 2
	Meta  = 4
	Shift = 8
)

// kindField and modifiers are the fields of DevTools' input events
// naming the event and the modifiers held.
const (
	kindField = "type"
	modifiers = "modifiers"
)

// Press presses the left button at (x, y), in CSS pixels, with the
// modifier keys mods held.
func (s *Session) Press(x, y float32, mods int) {
	s.t.Helper()
	s.mouse("mousePressed", x, y, mods, 1)
}

// Release lets the left button go at (x, y).
func (s *Session) Release(x, y float32, mods int) {
	s.t.Helper()
	s.mouse("mouseReleased", x, y, mods, 0)
}

// Move moves the pointer to (x, y), the left button held while pressed.
func (s *Session) Move(x, y float32, mods int, pressed bool) {
	s.t.Helper()
	buttons := 0
	if pressed {
		buttons = 1
	}
	s.mouse("mouseMoved", x, y, mods, buttons)
}

// Click presses and lets go the left button at (x, y).
func (s *Session) Click(x, y float32, mods int) {
	s.t.Helper()
	s.Press(x, y, mods)
	s.Release(x, y, mods)
}

func (s *Session) mouse(kind string, x, y float32, mods, buttons int) {
	s.t.Helper()
	params := map[string]any{kindField: kind, "x": x, "y": y, modifiers: mods, "buttons": buttons}
	if kind != "mouseMoved" || buttons != 0 {
		params["button"], params["clickCount"] = "left", 1
	}
	s.call("Input.dispatchMouseEvent", params, nil)
}

// keyCodes are the named keys Key presses, by the DOM's key, with their
// Windows virtual key codes and DOM codes.
//
//nolint:gochecknoglobals // a fixed table
var keyCodes = map[string]struct {
	vk   int
	code string
}{
	"ArrowUp": {38, "ArrowUp"}, "ArrowDown": {40, "ArrowDown"}, "ArrowLeft": {37, "ArrowLeft"},
	"ArrowRight": {39, "ArrowRight"}, "Home": {36, "Home"}, "End": {35, "End"}, "PageUp": {33, "PageUp"},
	"PageDown": {34, "PageDown"}, "Enter": {13, "Enter"}, "Escape": {27, "Escape"}, "Tab": {9, "Tab"},
	" ": {32, "Space"},
}

// Key presses and lets go the key the DOM's KeyboardEvent.key names,
// with mods held: a named key of keyCodes, or a letter or a digit, which
// types itself unless a modifier other than Shift is held.
func (s *Session) Key(key string, mods int) {
	s.t.Helper()
	down := map[string]any{kindField: "rawKeyDown", "key": key, modifiers: mods}
	if k, ok := keyCodes[key]; ok {
		down["windowsVirtualKeyCode"], down["code"] = k.vk, k.code
	} else if len(key) == 1 {
		up := strings.ToUpper(key)
		down["windowsVirtualKeyCode"], down["code"] = int(up[0]), "Key"+up
		if up[0] >= '0' && up[0] <= '9' {
			down["code"] = "Digit" + up
		}
	}
	if len(key) == 1 && mods&^Shift == 0 {
		down[kindField], down["text"] = "keyDown", key
	}
	s.call("Input.dispatchKeyEvent", down, nil)
	up := map[string]any{kindField: "keyUp", "key": key, modifiers: mods}
	for _, f := range []string{"windowsVirtualKeyCode", "code"} {
		if v, ok := down[f]; ok {
			up[f] = v
		}
	}
	s.call("Input.dispatchKeyEvent", up, nil)
}
