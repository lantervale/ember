//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// Controls of appearance: none lose their own look's size and borders as
// Chrome's do: a checkbox and a radio are empty boxes, 0 by 0 unless
// sized, a select has no room for its chevron, a field and a button take
// the borders of Chrome's sheet, and a range is as tall as its track,
// which is as tall as its thumb unless sized. The selects' line-height
// and the field's and textarea's widths are set: Chrome on Linux makes a
// select's normal line a point shorter, and sizes a field by its
// system font's characters.
func TestControlsOfAppearanceNoneTakeChromesBoxes(t *testing.T) {
	const sheet = `.none { appearance: none; }
select { line-height: 16px; }
input:not([type]), textarea { width: 180px; }
.sized { width: 12px; height: 10px; border: 1px solid #000; }
input[type="range"] { width: 100px; }
.four::-webkit-slider-runnable-track { height: 4px; margin: 3px 0; }
.thumb::-webkit-slider-thumb { appearance: none; width: 10px; height: 24px; }
.padded::-webkit-slider-runnable-track { padding: 2px 0; border: 1px solid #000; }`
	d := ui.NewDocument(ui.CSS(sheet))
	none := ui.Class("none")
	d.Body().Append(
		ui.Div(ui.Input(ui.Type("checkbox"), none), ui.Input(ui.Type("checkbox"), ui.Checked(true), none),
			ui.Input(ui.Type("radio"), ui.Class("none", "sized")), ui.Input(ui.Type("checkbox"))),
		ui.Div(ui.Select(none, ui.Option(ui.Text("Walls"))), ui.Select(ui.Option(ui.Text("Walls")))),
		ui.Div(ui.Input(ui.Value("text"), none), ui.Button(ui.Text("Button"), none), ui.Textarea(none)),
		ui.Div(ui.Input(ui.Type("range"), none), ui.Input(ui.Type("range"), ui.Class("none", "four")),
			ui.Input(ui.Type("range"), ui.Class("none", "thumb")),
			ui.Input(ui.Type("range"), ui.Class("none", "thumb", "padded")),
			ui.Input(ui.Type("range"), ui.Class("none", "four", "thumb"))),
	)
	r := chrome.Match(t, d, geom.V2(640, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// appearance computes as Chrome computes it: none and auto, and the
// compatibility keywords as written, on any element. slider-vertical is
// left out: Chrome 156 computes it none, 157 as written, as ui does.
func TestAppearanceComputesAsChromes(t *testing.T) {
	words := []string{"none", "auto", "textfield", "menulist-button", "checkbox", "radio", "button", "listbox",
		"menulist", "meter", "progress-bar", "searchfield", "textarea", "base-select"}
	d := ui.NewDocument()
	var els []*ui.Node
	for _, w := range words {
		els = append(els, ui.Div(ui.Style("appearance: "+w)))
	}
	els = append(els, ui.Input(), ui.Button(), ui.Select(), ui.Textarea(), ui.Progress(), ui.Meter(),
		ui.Input(ui.Type("checkbox")), ui.Input(ui.Type("range")), ui.Div())
	d.Body().Append(els...)
	d.Layout(geom.V2(400, 300))
	// The page's own-look sheet keeps Chrome from drawing its controls;
	// out of the way, Chrome's cascade says what appearance computes to.
	got, err := chrome.StylesAfter(d, geom.V2(400, 300), []string{"appearance"},
		`document.querySelector("style[data-ember-own-look]").remove();`)
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range els {
		want := got[i+2]["appearance"]
		if ours := dom.Data(n).Style.Appearance.String(); ours != want {
			t.Errorf("%s %q: ours %s, Chrome's %s", n.Tag(), n.Style(), ours, want)
		}
	}
}
