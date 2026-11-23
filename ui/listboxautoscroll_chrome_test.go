//go:build chrome

package ui_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestAListBoxsKeysScrollItAtOnceAsChromes: the option a key reaches is
// scrolled into view at once, by the least that shows it, where the
// same keys glide a scrolling div: Chrome's listbox scrolls in one
// animation frame.
func TestAListBoxsKeysScrollItAtOnceAsChromes(t *testing.T) {
	args := []ui.Arg{ui.ID("k"), multiple(), ui.Attr("size", "4")}
	for i := range 30 {
		args = append(args, opt(fmt.Sprintf("Option %d", i)))
	}
	lb := ui.Select(args...)
	doc := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { margin: 10px; }`))
	doc.Body().Append(lb)
	srv, _ := chrome.Serve(t, doc, nil, 0)
	win := platform.NewWindow(platform.Size(640, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(3)
	doc.Focus(lb)
	w.RunFrames(2)
	s := chrome.Open(t, srv.URL, geom.V2(640, 300), 1)
	s.Eval(`document.getElementById("k").focus(), 1`, nil)
	for _, k := range []string{"ArrowDown", "ArrowDown", "ArrowDown", "ArrowDown", "ArrowDown", "PageDown", "End",
		"ArrowUp", "Home"} {
		in := win.Input()
		in.SetKey(keysOf[k], true)
		w.RunFrames(1)
		in.SetKey(keysOf[k], false)
		w.RunFrames(1)
		got := doc.Scroll(lb).Y
		s.Eval(`(() => { const k = document.getElementById("k"); window.S = [k.scrollTop];
	(function f() { S.push(k.scrollTop); if (S.length < 20) requestAnimationFrame(f); })(); return 1; })()`, nil)
		s.Key(k, 0)
		time.Sleep(400 * time.Millisecond)
		var tops []float32
		s.Eval(`S`, &tops)
		final := tops[len(tops)-1]
		for _, top := range tops {
			if top != tops[0] && top != final {
				t.Errorf("%s: Chrome passed by %v on its way to %v: %v", k, top, final, tops)
				break
			}
		}
		if got != final {
			t.Errorf("%s: ui scrolled to %v, Chrome to %v", k, got, final)
		}
	}
}

// TestADragNearAListBoxsEdgeScrollsItAsChromes: once a held press moves
// over a listbox, a pointer within 20 points of its top or bottom edge,
// or past it, scrolls it each frame by how far it lies into that belt,
// as Chrome's autoscroll; the options a still pointer comes over wait
// for it to move. Chrome's frames are its own, so the pace a frame is
// compared, measured over Chrome's animation frames.
func TestADragNearAListBoxsEdgeScrollsItAsChromes(t *testing.T) {
	args := []ui.Arg{ui.ID("k"), multiple(), ui.Attr("size", "4")}
	for i := range 300 {
		args = append(args, opt(fmt.Sprintf("Option %d", i)))
	}
	lb := ui.Select(args...)
	doc := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { margin: 10px; } #k { margin-top: 150px; }`))
	doc.Body().Append(lb)
	srv, _ := chrome.Serve(t, doc, nil, 0)
	win := platform.NewWindow(platform.Size(640, 400))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	in := win.Input()
	in.SetMouse(geom.V2(630, 390))
	w.RunFrames(3)
	s := chrome.Open(t, srv.URL, geom.V2(640, 400), 1)
	r, _ := doc.BoxOf(lb)
	x := r.X + 20
	for _, c := range []struct {
		name string
		from float32
		at   float32
	}{
		{"past the bottom", 0, r.Y + r.H + 10},
		{"just past the bottom", 0, r.Y + r.H + 0.5},
		{"inside the bottom belt", 0, r.Y + r.H - 12},
		{"above the belt", 0, r.Y + r.H - 21},
		{"past the top", 2000, r.Y - 30},
		{"inside the top belt", 2000, r.Y + 5},
	} {
		up := c.from > 0
		row := r.Y + 1 + 8.5
		if up {
			row = r.Y + 1 + 17*3 + 8.5
		}
		next := row + 17
		if up {
			next = row - 17
		}
		doc.SetScroll(lb, geom.V2(0, c.from))
		s.Eval(fmt.Sprintf(`document.getElementById("k").scrollTop = %v, 1`, c.from), nil)
		in.SetMouse(geom.V2(x, row))
		w.RunFrames(2)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouse(geom.V2(x, next))
		w.RunFrames(1)
		start := doc.Scroll(lb).Y
		in.SetMouse(geom.V2(x, c.at))
		w.RunFrames(10)
		got := (doc.Scroll(lb).Y - start) / 10
		picked := doc.SelectedOptions(lb)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)

		s.Press(x, row, 0)
		s.Move(x, next, 0, true)
		time.Sleep(100 * time.Millisecond)
		s.Eval(`(() => { const k = document.getElementById("k"); window.S = [];
	(function f() { S.push(k.scrollTop); if (S.length < 12) requestAnimationFrame(f); })(); return 1; })()`, nil)
		s.Move(x, c.at, 0, true)
		time.Sleep(400 * time.Millisecond)
		var tops []float32
		s.Eval(`S`, &tops)
		var chromes []int
		s.Eval(`[...document.getElementById("k").options].flatMap((o, i) => o.selected ? [i] : [])`, &chromes)
		s.Release(x, c.at, 0)
		time.Sleep(100 * time.Millisecond)
		if start != tops[0] {
			t.Errorf("%s: the press left ui at %v, Chrome at %v", c.name, start, tops[0])
		}
		if !slices.Equal(picked, chromes) {
			t.Errorf("%s: ui selects %v, Chrome %v", c.name, picked, chromes)
		}
		var steps []float32
		for i := 2; i < len(tops); i++ {
			steps = append(steps, tops[i]-tops[i-1])
		}
		slices.Sort(steps)
		want := steps[len(steps)/2]
		if d := got - want; d < -0.5 || d > 0.5 {
			t.Errorf("%s: ui scrolls %v a frame, Chrome %v (%v)", c.name, got, want, tops)
		}
	}
}
