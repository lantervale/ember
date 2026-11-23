//go:build chrome

package ui_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestTypingFindsASelectsOptionAsChromes: letters typed within a second
// of each other find the option starting with them all, from the first
// selected on; one letter typed again goes on to the next option
// starting with it; past a second a letter starts again; leading spaces
// and disabled options are passed over. A listbox, multiple or not, and
// a closed dropdown alike, a listbox scrolled to the option found. A
// dropdown fires input and change for a new choice; a listbox no press or
// key has moved yet a change alone for every option found, and once one
// has, input and change for a new selection. The keys are stamped in
// Chrome with ui's own frame times.
func TestTypingFindsASelectsOptionAsChromes(t *testing.T) {
	var changes, inputs [3]int
	pantry := func(i int, args ...ui.Arg) *ui.Node {
		return ui.Select(append([]ui.Arg{ui.ID(fmt.Sprintf("s%d", i)),
			ui.OnChange(func() { changes[i]++ }), ui.OnInput(func() { inputs[i]++ }),
			opt("Apple"), opt("Avocado"), opt("Banana"), opt("Bx", ui.Disabled()), opt("Blueberry"),
			opt("Apricot"), opt("  Cherry"), opt("Ab"), opt("aa")}, args...)...)
	}
	sels := []*ui.Node{pantry(0, multiple(), ui.Attr("size", "4")), pantry(1, ui.Attr("size", "4")), pantry(2)}
	doc := ui.NewDocument(ui.BrowserSheets())
	doc.Body().Append(sels...)
	srv, _ := chrome.Serve(t, doc, nil, 0)
	win := platform.NewWindow(platform.Size(640, 300))
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(3)
	s := chrome.Open(t, srv.URL, geom.V2(640, 300), 1)
	s.Eval(`window.ev = {}; document.querySelectorAll("select").forEach(s => {
	for (const k of ["change", "input"]) s.addEventListener(k, () => { ev[s.id + k] = (ev[s.id + k] || 0) + 1; });
}); 1`, nil)
	base, frames := time.Now(), 0
	run := func(n int) {
		w.RunFrames(n)
		frames += n
	}
	typed := func(text string) {
		for _, r := range text {
			win.Input().AddText(r)
			run(1)
			s.KeyAt(string(r), 0, base.Add(time.Duration(frames)*time.Second/60))
			run(5)
		}
	}
	check := func(step string) {
		t.Helper()
		var want struct {
			Sel []string       `json:"sel"`
			Ev  map[string]int `json:"ev"`
			Top []float32      `json:"top"`
		}
		s.Eval(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => r({ev,
	top: [...document.querySelectorAll("select")].map(s => s.scrollTop),
	sel: [...document.querySelectorAll("select")].map(s => [...s.options].map(o => o.selected ? "1" : "0").join(""))}))))`,
			&want)
		for i, sel := range sels[:2] {
			if got := doc.Scroll(sel).Y; got != want.Top[i] {
				t.Errorf("%s: listbox %d: ui scrolled to %v, Chrome to %v", step, i, got, want.Top[i])
			}
		}
		for i, sel := range sels {
			got := []byte(strings.Repeat("0", len(dom.Options(sel))))
			for _, k := range doc.SelectedOptions(sel) {
				got[k] = '1'
			}
			if string(got) != want.Sel[i] {
				t.Errorf("%s: select %d: ui selects %s, Chrome %s", step, i, got, want.Sel[i])
			}
			id := fmt.Sprintf("s%d", i)
			if changes[i] != want.Ev[id+"change"] || inputs[i] != want.Ev[id+"input"] {
				t.Errorf("%s: select %d: ui fired %d change and %d input, Chrome %d and %d", step, i,
					changes[i], inputs[i], want.Ev[id+"change"], want.Ev[id+"input"])
			}
		}
	}
	for i, sel := range sels {
		doc.Focus(sel)
		run(2)
		s.Eval(fmt.Sprintf(`document.getElementById("s%d").focus(), 1`, i), nil)
		if i == 1 {
			win.Input().SetKey(input.KeyDown, true)
			run(1)
			win.Input().SetKey(input.KeyDown, false)
			s.KeyAt("ArrowDown", 0, base.Add(time.Duration(frames)*time.Second/60))
			run(1)
			check("select 1, Down")
		}
		for _, step := range []string{"aaaaa", "ap", "av", "b", "b", "b", "c", "aba", "aab", "blub", "Ap"} {
			run(70)
			typed(step)
			check(fmt.Sprintf("select %d, %q", i, step))
		}
	}
}
