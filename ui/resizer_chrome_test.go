//go:build chrome

package ui_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestResizersDragTheirBoxAsChromes drags the resizer of boxes the user
// may resize, in ui and in Chrome, out and then far in: each box takes
// the size Chrome gives it, along the axes its resize names (block and
// inline by its writing mode), in the size box-sizing names, within its
// min and max sizes, no smaller than its resizer's corner, from the
// bottom-left corner right to left, and its style attribute says the
// same width and height as Chrome writes there, a textarea's margins
// beside them.
func TestResizersDragTheirBoxAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { margin: 10px; }
.r { width: 100px; height: 60px; overflow: auto; margin: 6px; border: 3px solid #888; padding: 4px; }
.styled::-webkit-scrollbar { width: 12px; height: 12px; background-color: #444; }
.styled > div { height: 200px; }
textarea { resize: both; }`))
	styles := []string{"resize: both", "resize: horizontal", "resize: vertical", "resize: block", "resize: inline",
		"resize: both; box-sizing: border-box", "resize: both; min-width: 90px; max-height: 80px",
		"resize: both; direction: rtl", "resize: block; writing-mode: vertical-rl", "resize: both; padding: 0; border: 0"}
	var boxes []*ui.Node
	for _, st := range styles {
		boxes = append(boxes, ui.Div(ui.Class("r"), ui.Style(st), ui.Text("x")))
	}
	boxes = append(boxes, ui.Div(ui.Class("r", "styled"), ui.Style("resize: both"), ui.Div()),
		ui.Textarea(ui.Style("margin: 5px 7px")))
	for _, b := range boxes {
		d.Body().Append(b)
	}
	size := geom.V2(500, 1500)
	srv, _ := chrome.Serve(t, d, nil, 0)
	win := platform.NewWindow(platform.Size(int(size.X), int(size.Y)))
	win.Add(d)
	w := ember.New().AddEntities(win)
	in := win.Input()
	in.SetMouse(geom.V2(490, 1490))
	w.RunFrames(3)
	s := chrome.Open(t, srv.URL, size, 1)
	pinned, err := chrome.Pinned()
	if err != nil {
		t.Fatalf("Chrome's version: %v", err)
	}
	drag := func(from, by geom.Vec2) {
		to := from.Add(by)
		in.SetMouse(from)
		w.RunFrames(2)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouse(from.Add(by.Scale(0.5)))
		w.RunFrames(1)
		in.SetMouse(to)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)
		s.Press(from.X, from.Y, 0)
		s.Move(from.X+by.X/2, from.Y+by.Y/2, 0, true)
		s.Move(to.X, to.Y, 0, true)
		s.Release(to.X, to.Y, 0)
	}
	check := func(step string) {
		t.Helper()
		var want []struct {
			X, Y, W, H    float32
			Width, Height string
		}
		s.Eval(`[...document.body.children].map(e => { const r = e.getBoundingClientRect();
	return {X: r.x, Y: r.y, W: r.width, H: r.height, Width: e.style.width, Height: e.style.height}; })`, &want)
		for i, b := range boxes {
			if pinned && b.Tag() == dom.Textarea {
				continue // textareaNarrower
			}
			got := b.Box()
			if math.Abs(float64(got.W-want[i].W)) > 0.5 || math.Abs(float64(got.H-want[i].H)) > 0.5 ||
				math.Abs(float64(got.Y-want[i].Y)) > 0.5 {
				t.Errorf("%s: box %d (%s): ui %v, Chrome %v", step, i, b.Style(), got, want[i])
			}
			if w, h := declaredIn(b.Style(), "width"), declaredIn(b.Style(), "height"); w != want[i].Width ||
				h != want[i].Height {
				t.Errorf("%s: box %d: ui's style says width %q and height %q, Chrome's %q and %q", step, i, w, h,
					want[i].Width, want[i].Height)
			}
		}
	}
	check("at first")
	for i, b := range boxes {
		r := b.Box()
		corner, way := geom.V2(r.X+r.W-4, r.Y+r.H-4), geom.V2(1, 1)
		if i == 7 {
			corner, way = geom.V2(r.X+4, r.Y+r.H-4), geom.V2(-1, 1)
		}
		drag(corner, geom.V2(40*way.X, 30))
		check(fmt.Sprintf("box %d out", i))
		r = b.Box()
		corner = geom.V2(r.X+r.W-4, r.Y+r.H-4)
		if i == 7 {
			corner = geom.V2(r.X+4, r.Y+r.H-4)
		}
		drag(corner, geom.V2(-300*way.X, -300))
		check(fmt.Sprintf("box %d in", i))
	}
}

// TestTextareasResizeAsChromes drags the corner of textareas no sheet
// gives a resize, in ui and in Chrome: on a page read as a browser's
// (BrowserSheets) each takes Chrome's resize: auto and the size Chrome
// gives it, rows and cols, right to left, vertical, scrolled and
// disabled alike; on a game's screen none resizes, in either.
func TestTextareasResizeAsChromes(t *testing.T) {
	chrome.UnlikePinned(t, textareaNarrower)
	for _, page := range []bool{true, false} {
		t.Run(fmt.Sprint("browser sheets ", page), func(t *testing.T) {
			var opts []ui.DocumentOption
			if page {
				opts = append(opts, ui.BrowserSheets())
			}
			d := ui.NewDocument(append(opts, ui.CSS(`body { margin: 10px; } textarea { margin: 6px; font: 14px/18px Go; }`))...)
			boxes := []*ui.Node{ui.Textarea(), ui.Textarea(ui.Attr("rows", "4"), ui.Attr("cols", "10")),
				ui.Textarea(ui.Attr("dir", "rtl")), ui.Textarea(ui.Style("writing-mode: vertical-lr")),
				ui.Textarea(ui.Text(strings.Repeat("line\n", 12))), ui.Textarea(ui.Attr("disabled", ""))}
			for _, b := range boxes {
				d.Body().Append(ui.Div(b))
			}
			size := geom.V2(500, 900)
			srv, _ := chrome.Serve(t, d, nil, 0)
			win := platform.NewWindow(platform.Size(int(size.X), int(size.Y)))
			win.Add(d)
			w := ember.New().AddEntities(win)
			in := win.Input()
			in.SetMouse(geom.V2(490, 890))
			w.RunFrames(3)
			s := chrome.Open(t, srv.URL, size, 1)
			for i, b := range boxes {
				r := b.Box()
				from, way := geom.V2(r.X+r.W-3, r.Y+r.H-3), float32(1)
				if i == 2 {
					from, way = geom.V2(r.X+3, r.Y+r.H-3), -1
				}
				to := from.Add(geom.V2(40*way, 30))
				in.SetMouse(from)
				w.RunFrames(2)
				in.SetMouseButton(input.MouseLeft, true)
				w.RunFrames(1)
				in.SetMouse(to)
				w.RunFrames(1)
				in.SetMouseButton(input.MouseLeft, false)
				w.RunFrames(2)
				s.Press(from.X, from.Y, 0)
				s.Move(to.X, to.Y, 0, true)
				s.Release(to.X, to.Y, 0)
				var want []struct {
					W, H          float32
					Width, Height string
				}
				s.Eval(`[...document.querySelectorAll('textarea')].map(e => { const r = e.getBoundingClientRect();
	return {W: r.width, H: r.height, Width: e.style.width, Height: e.style.height}; })`, &want)
				got := b.Box()
				if math.Abs(float64(got.W-want[i].W)) > 0.5 || math.Abs(float64(got.H-want[i].H)) > 0.5 {
					t.Errorf("textarea %d: ui %v, Chrome %vx%v", i, got, want[i].W, want[i].H)
				}
				if w, h := declaredIn(b.Style(), "width"), declaredIn(b.Style(), "height"); w != want[i].Width ||
					h != want[i].Height {
					t.Errorf("textarea %d: ui's style says width %q and height %q, Chrome's %q and %q", i, w, h,
						want[i].Width, want[i].Height)
				}
				if page == (got == r) {
					t.Errorf("textarea %d: %v from %v", i, got, r)
				}
			}
		})
	}
}

// textareaNarrower is how the pinned Chrome for Testing sizes a textarea
// unlike the stable Chrome ui follows (158.0.8127.37 on a Mac).
const textareaNarrower = "a textarea of its cols is a point narrower"

// declaredIn is the value of the last declaration of property in the style
// attribute css, empty with none.
func declaredIn(css, property string) string {
	v := ""
	for _, d := range strings.Split(css, ";") {
		if k, val, ok := strings.Cut(d, ":"); ok && strings.TrimSpace(k) == property {
			v = strings.TrimSpace(val)
		}
	}
	return v
}
