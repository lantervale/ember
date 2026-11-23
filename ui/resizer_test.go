package ui_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
)

// A press on a box's resizer drags its size: the style attribute takes
// the new width and height, the box follows, and no click reaches the
// box; a press beside the corner, or on a box that doesn't clip, resizes
// nothing.
func TestAResizerDragsItsBoxsSize(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { margin: 0; }
.r { width: 100px; height: 60px; overflow: hidden; resize: both; }
.v { resize: vertical; overflow: visible; }`))
	clicks := 0
	r := ui.Div(ui.Class("r"), ui.Style("color: red"), ui.OnClick(func() { clicks++ }))
	v := ui.Div(ui.Class("r", "v"))
	d.Body().Append(r, v)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(d)
	w := ember.New().AddEntities(win)
	in := win.Input()
	drag := func(from, to geom.Vec2) {
		in.SetMouse(from)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouse(to)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)
	}
	w.RunFrames(2)
	drag(geom.V2(96, 56), geom.V2(136, 86))
	if got, want := r.Style(), "color: red; width: 140px; height: 90px"; got != want {
		t.Errorf("style %q, want %q", got, want)
	}
	if b := r.Box(); b.W != 140 || b.H != 90 {
		t.Errorf("box %v, want 140×90", b)
	}
	if clicks != 0 {
		t.Errorf("%d clicks reached the box", clicks)
	}
	drag(geom.V2(100, 60), geom.V2(10, 10))
	if got := r.Style(); got != "color: red; width: 140px; height: 90px" {
		t.Errorf("a press off the corner resized it: %q", got)
	}
	b := v.Box()
	drag(geom.V2(b.X+b.W-4, b.Y+b.H-4), geom.V2(b.X+b.W+40, b.Y+b.H+40))
	if v.Style() != "" {
		t.Errorf("a box that doesn't clip took %q", v.Style())
	}
}

// A textarea's resizer is a page's: under BrowserSheets it takes
// resize: auto, both axes as Chrome's, and drags; a game's screen shows
// none, but where its sheet asks for one.
func TestATextareaResizesUnderBrowserSheetsAlone(t *testing.T) {
	for _, c := range []struct {
		name  string
		opts  []ui.DocumentOption
		moved bool
	}{
		{"page", []ui.DocumentOption{ui.BrowserSheets()}, true},
		{"screen", nil, false},
		{"screen asking", []ui.DocumentOption{ui.CSS("textarea { resize: auto; }")}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := ui.NewDocument(append(c.opts, ui.CSS("body { margin: 0; }"))...)
			ta := ui.Textarea()
			d.Body().Append(ta)
			win := platform.NewWindow(platform.Size(400, 300))
			win.Add(d)
			w := ember.New().AddEntities(win)
			in := win.Input()
			w.RunFrames(2)
			b := ta.Box()
			in.SetMouse(geom.V2(b.X+b.W-3, b.Y+b.H-3))
			w.RunFrames(1)
			in.SetMouseButton(input.MouseLeft, true)
			w.RunFrames(1)
			in.SetMouse(geom.V2(b.X+b.W+37, b.Y+b.H+27))
			w.RunFrames(1)
			in.SetMouseButton(input.MouseLeft, false)
			w.RunFrames(2)
			if got := ta.Box(); (got.W != b.W) != c.moved || (got.H != b.H) != c.moved {
				t.Errorf("box %v from %v, resized %v", got, b, c.moved)
			}
		})
	}
}
