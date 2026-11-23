//go:build chrome

package ui_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
)

// listBoxDriver plays the same pointer and keys on a Document in a
// window and on its export in Chrome.
type listBoxDriver struct {
	t       *testing.T
	doc     *ui.Document
	win     *platform.Window
	w       *world.World
	s       *chrome.Session
	sels    []*ui.Node
	changes []int
	inputs  []int
}

func newListBoxDriver(t *testing.T) *listBoxDriver {
	t.Helper()
	p := &listBoxDriver{t: t, doc: ui.NewDocument(ui.BrowserSheets(), ui.CSS(`body { margin: 10px; }`))}
	count := func(i int) []ui.Arg {
		return []ui.Arg{ui.ID(fmt.Sprintf("s%d", i)), ui.OnChange(func() { p.changes[i]++ }),
			ui.OnInput(func() { p.inputs[i]++ })}
	}
	p.sels = []*ui.Node{
		ui.Select(append(count(0), multiple(), ui.Attr("size", "5"), opt("Sword"), opt("Bow", ui.Selected()), opt("Wand"),
			opt("Skull", ui.Disabled()), opt("Staff"))...),
		ui.Select(append(count(1), multiple(), opt("Fists"),
			ui.Optgroup("Blades", opt("Sword"), opt("Dagger", ui.Disabled())), opt("Bow"))...),
		ui.Select(append(count(2), ui.Attr("size", "3"), opt("One"), opt("Two"), opt("Three"), opt("Four"),
			opt("Five"), opt("Six"))...),
	}
	p.changes, p.inputs = make([]int, len(p.sels)), make([]int, len(p.sels))
	p.doc.Body().Append(p.sels...)
	srv, _ := chrome.Serve(t, p.doc, nil, 0)
	p.win = platform.NewWindow(platform.Size(640, 300))
	p.win.Add(p.doc)
	p.w = ember.New().AddEntities(p.win)
	p.win.Input().SetMouse(geom.V2(630, 290))
	p.w.RunFrames(3)
	p.s = chrome.Open(t, srv.URL, geom.V2(640, 300), 1)
	p.s.Eval(`window.ev = {}; document.querySelectorAll("select").forEach(s => {
	for (const k of ["change", "input"]) s.addEventListener(k, () => { ev[s.id + k] = (ev[s.id + k] || 0) + 1; });
}); 1`, nil)
	return p
}

// command is the platform's toggling modifier, Command on macOS.
func command() (int, input.Key) {
	if runtime.GOOS == "darwin" {
		return chrome.Meta, input.KeyLeftSuper
	}
	return chrome.Ctrl, input.KeyLeftControl
}

func (p *listBoxDriver) hold(mods int, down bool) {
	in := p.win.Input()
	if mods&chrome.Shift != 0 {
		in.SetKey(input.KeyLeftShift, down)
	}
	if c, k := command(); mods&c != 0 {
		in.SetKey(k, down)
	}
}

// at is the middle of option i of select s in ui.
func (p *listBoxDriver) at(s, i int) geom.Vec2 {
	p.t.Helper()
	r, ok := p.doc.BoxOf(dom.Options(p.sels[s])[i])
	if !ok {
		p.t.Fatalf("select %d: no box for option %d", s, i)
	}
	return r.Center()
}

func (p *listBoxDriver) click(at geom.Vec2, mods int) {
	p.drag(mods, at)
}

// drag presses at the first point, moves through the rest and lets go
// at the last.
func (p *listBoxDriver) drag(mods int, through ...geom.Vec2) {
	in := p.win.Input()
	p.hold(mods, true)
	in.SetMouse(through[0])
	p.w.RunFrames(2)
	in.SetMouseButton(input.MouseLeft, true)
	p.w.RunFrames(1)
	p.s.Press(through[0].X, through[0].Y, mods)
	for _, at := range through[1:] {
		in.SetMouse(at)
		p.w.RunFrames(1)
		p.s.Move(at.X, at.Y, mods, true)
	}
	last := through[len(through)-1]
	in.SetMouseButton(input.MouseLeft, false)
	p.w.RunFrames(2)
	p.s.Release(last.X, last.Y, mods)
	p.hold(mods, false)
	p.w.RunFrames(1)
}

var keysOf = map[string]input.Key{ //nolint:gochecknoglobals // a test's table
	"ArrowUp": input.KeyUp, "ArrowDown": input.KeyDown, "Home": input.KeyHome, "End": input.KeyEnd,
	"PageUp": input.KeyPageUp, "PageDown": input.KeyPageDown, " ": input.KeySpace,
}

func (p *listBoxDriver) key(name string, mods int) {
	in := p.win.Input()
	p.hold(mods, true)
	if k, ok := keysOf[name]; ok {
		in.SetKey(k, true)
		p.w.RunFrames(1)
		in.SetKey(k, false)
	} else {
		in.AddText([]rune(name)[0])
	}
	p.w.RunFrames(1)
	p.s.Key(name, mods)
	p.hold(mods, false)
	p.w.RunFrames(1)
}

// check compares every select's selection and its events so far.
func (p *listBoxDriver) check(step string) {
	p.t.Helper()
	var want struct {
		Sel []string       `json:"sel"`
		Ev  map[string]int `json:"ev"`
	}
	p.s.Eval(`({sel: [...document.querySelectorAll("select")].map(s => [...s.options].map(o => o.selected ? "1" : "0").join("")), ev})`, &want)
	for i, s := range p.sels {
		got := []byte(strings.Repeat("0", len(dom.Options(s))))
		for _, k := range p.doc.SelectedOptions(s) {
			got[k] = '1'
		}
		if string(got) != want.Sel[i] {
			p.t.Errorf("%s: select %d: ui selects %s, Chrome %s", step, i, got, want.Sel[i])
		}
		id := fmt.Sprintf("s%d", i)
		if p.changes[i] != want.Ev[id+"change"] || p.inputs[i] != want.Ev[id+"input"] {
			p.t.Errorf("%s: select %d: ui fired %d change and %d input, Chrome %d and %d", step, i,
				p.changes[i], p.inputs[i], want.Ev[id+"change"], want.Ev[id+"input"])
		}
	}
}

// TestAListBoxSelectsAsChromesUnderThePointerAndTheKeys: a press selects
// its option alone, Command (Control off macOS) turns one on or off,
// Shift selects a run from the anchor past disabled options, a drag the
// run it crosses; the arrows, Home, End and the page keys select the
// option they reach, with Shift a run, with Command they move alone and
// Space toggles; typing finds an option; a group's label and a disabled
// option take no press. Each change fires input then change once, as
// Chrome's.
func TestAListBoxSelectsAsChromesUnderThePointerAndTheKeys(t *testing.T) {
	p := newListBoxDriver(t)
	cmd, _ := command()
	p.check("at first")
	steps := []struct {
		name string
		do   func()
	}{
		{"press Bow", func() { p.click(p.at(0, 1), 0) }},
		{"command Sword", func() { p.click(p.at(0, 0), cmd) }},
		{"command Sword again", func() { p.click(p.at(0, 0), cmd) }},
		{"command Wand", func() { p.click(p.at(0, 2), cmd) }},
		{"shift Staff", func() { p.click(p.at(0, 4), chrome.Shift) }},
		{"press Skull", func() { p.click(p.at(0, 3), 0) }},
		{"press Wand", func() { p.click(p.at(0, 2), 0) }},
		{"Up", func() { p.key("ArrowUp", 0) }},
		{"Shift Up", func() { p.key("ArrowUp", chrome.Shift) }},
		{"Down", func() { p.key("ArrowDown", 0) }},
		{"command Down", func() { p.key("ArrowDown", cmd) }},
		{"Space", func() { p.key(" ", 0) }},
		{"Home", func() { p.key("Home", 0) }},
		{"Shift End", func() { p.key("End", chrome.Shift) }},
		{"type w", func() { p.key("w", 0) }},
		{"drag Sword to Wand", func() { p.drag(0, p.at(0, 0), p.at(0, 1), p.at(0, 2)) }},
		{"press Blades", func() {
			r, _ := p.doc.BoxOf(p.sels[1].Children()[1])
			p.click(geom.V2(r.X+r.W/2, r.Y+8), 0)
		}},
		{"press Sword in Blades", func() { p.click(p.at(1, 1), 0) }},
		{"Down past Dagger", func() { p.key("ArrowDown", 0) }},
		{"Up", func() { p.key("ArrowUp", 0) }},
		{"press Two", func() { p.click(p.at(2, 1), 0) }},
		{"command Three, one alone", func() { p.click(p.at(2, 2), cmd) }},
		{"PageDown", func() { p.key("PageDown", 0) }},
		{"PageDown again", func() { p.key("PageDown", 0) }},
		{"PageUp", func() { p.key("PageUp", 0) }},
		{"End", func() { p.key("End", 0) }},
	}
	for _, s := range steps {
		s.do()
		p.check(s.name)
	}
	var top float32
	p.s.Eval(`new Promise(r => setTimeout(() => r(document.querySelector("#s2").scrollTop), 1000))`, &top)
	p.w.RunFrames(3)
	if got := p.doc.Scroll(p.sels[2]).Y; got != top {
		t.Errorf("the scrolled listbox: ui at %v, Chrome at %v", got, top)
	}
}
