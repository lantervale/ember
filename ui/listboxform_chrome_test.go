//go:build chrome

package ui_test

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestAListBoxsSelectedRowsAreColouredAsChromes: a selected option's row
// is grey and its text near black, light blue under black once the
// listbox has the focus; a disabled option or a disabled listbox keeps
// its grey text; in either scheme.
func TestAListBoxsSelectedRowsAreColouredAsChromes(t *testing.T) {
	for _, scheme := range []string{"light", "dark"} {
		t.Run(scheme, func(t *testing.T) { listBoxColours(t, scheme) })
	}
}

func listBoxColours(t *testing.T, scheme string) {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(":root { color-scheme: "+scheme+" }"))
	focused := ui.Select(multiple(), ui.ID("focused"), opt("A", ui.Selected()), opt("B"),
		opt("C", ui.Selected(), ui.Disabled()))
	d.Body().Append(
		ui.Select(multiple(), opt("A", ui.Selected()), opt("B"), opt("C", ui.Selected(), ui.Disabled())),
		focused,
		ui.Select(multiple(), ui.Disabled(), opt("A", ui.Selected()), opt("B")),
		ui.Select(ui.Attr("size", "3"), ui.Optgroup("G", opt("A", ui.Selected())), opt("B")),
	)
	size := geom.V2(400, 200)
	win := platform.NewWindow(platform.Size(int(size.X), int(size.Y)))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	d.Focus(focused)
	w.RunFrames(2)
	if !d.Focused(focused) {
		t.Fatal("the listbox never took the focus")
	}
	props := []string{"color", "background-color"}
	styles, err := chrome.StylesAfter(d, size, props, `document.querySelector("#focused").focus();`)
	if err != nil {
		t.Fatal(err)
	}
	nodes := elementsOf(d)
	if len(nodes) != len(styles) {
		t.Fatalf("%d elements, Chrome styled %d", len(nodes), len(styles))
	}
	for i, n := range nodes {
		inks := ui.ShownInks(n)
		for k, p := range props {
			got, want := emberColour(inks[k]), chromeColour(t, styles[i][p])
			if linux, ok := linuxFocusedSelection(scheme, n, focused, p); ok {
				got = linux
			}
			if got != want {
				t.Errorf("element %d (%s %q) %s: %v, Chrome %v", i, n.Tag(), n.Text(), p, got, want)
			}
		}
	}
}

// linuxFocusedSelection is the colour Chrome on Linux and Windows gives
// property p of a selected option n of the focused listbox in the light
// scheme, where macOS's (and ui's) differ: white on its own active
// selection, #1967d2, or on #c8c8c8 for a disabled one
// (docs/05-ui/10-missing.md).
func linuxFocusedSelection(scheme string, n, focused *ui.Node, p string) ([4]int, bool) {
	if runtime.GOOS == "darwin" || scheme != "light" || n.Parent() != focused {
		return [4]int{}, false
	}
	switch kids := focused.Children(); {
	case (n == kids[0] || n == kids[2]) && p == "color":
		return [4]int{255, 255, 255, 100}, true
	case n == kids[0]:
		return [4]int{25, 103, 210, 100}, true
	case n == kids[2]:
		return [4]int{200, 200, 200, 100}, true
	}
	return [4]int{}, false
}

// TestAListBoxSubmitsAndValidatesAsChromes: a form holds every selected
// option's value, the label of one without, a disabled one's never; its
// selectedIndex is the first selected; a required listbox is
// invalid with none selected, an empty value counting as selected, and
// none of its options is a placeholder.
func TestAListBoxSubmitsAndValidatesAsChromes(t *testing.T) {
	d := ui.NewDocument()
	sels := []*ui.Node{
		ui.Select(multiple(), ui.Attr("name", "a"), opt("a", ui.Value("1"), ui.Selected()), opt("b", ui.Selected()),
			opt("c")),
		ui.Select(multiple(), ui.Attr("name", "b"), ui.Attr("required", ""), opt("a"), opt("b")),
		ui.Select(ui.Attr("size", "2"), ui.Attr("name", "c"), ui.Attr("required", ""), opt("", ui.Value(""), ui.Selected())),
		ui.Select(ui.Attr("size", "2"), ui.Attr("name", "d"), ui.Attr("required", ""), opt("e", ui.Value(""))),
		ui.Select(ui.Attr("size", "3"), ui.Attr("name", "e"), opt("x", ui.Selected()), opt("y", ui.Selected())),
		ui.Select(multiple(), ui.Attr("name", "f"), ui.Optgroup("G", ui.Disabled(), opt("g", ui.Selected())), opt("h")),
	}
	d.Body().Append(ui.Form(append([]ui.Arg{ui.ID("form")}, nodesAsArgs(sels)...)...))
	raw, err := chrome.Eval(d, geom.V2(640, 200), `const sels = [...document.querySelectorAll("select")];
return {
	values: sels.map(s => [...s.selectedOptions].map(o => o.value)),
	index: sels.map(s => s.selectedIndex),
	valid: sels.map(s => s.validity.valid),
	form: [...new FormData(document.querySelector("#form")).entries()].map(e => e.join("=")),
};`)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Values [][]string `json:"values"`
		Index  []int      `json:"index"`
		Valid  []bool     `json:"valid"`
		Form   []string   `json:"form"`
	}
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	d.Layout(geom.V2(640, 200))
	var form []string
	for i, s := range sels {
		var got []string
		for _, k := range d.SelectedOptions(s) {
			v, ok := dom.Options(s)[k].Attr("value")
			if !ok {
				v = dom.Options(s)[k].Text()
			}
			got = append(got, v)
		}
		if strings.Join(got, ",") != strings.Join(want.Values[i], ",") {
			t.Errorf("select %d: ui's selected options' values %q, Chrome's %q", i, got, want.Values[i])
		}
		if got := d.Selected(s); got != want.Index[i] {
			t.Errorf("select %d: ui's selectedIndex %d, Chrome's %d", i, got, want.Index[i])
		}
		if got := d.Valid(s); got != want.Valid[i] {
			t.Errorf("select %d: ui valid %v, Chrome %v", i, got, want.Valid[i])
		}
		name, _ := s.Attr("name")
		for _, v := range d.Values(s) {
			form = append(form, name+"="+v)
		}
	}
	if strings.Join(form, "&") != strings.Join(want.Form, "&") {
		t.Errorf("the form holds %q in ui, %q in Chrome", form, want.Form)
	}
}

func nodesAsArgs(ns []*ui.Node) []ui.Arg {
	out := make([]ui.Arg, len(ns))
	for i, n := range ns {
		out[i] = n
	}
	return out
}
