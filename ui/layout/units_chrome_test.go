//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const unitsSheet = `html { font: 18px/1.3 "Go Mono"; }
body { margin: 8px; font: 14px Go; }
.b { height: 6px; margin: 2px 0; background: #888; }
.mono { font-family: "Go Mono"; } .big { font-size: 22px; }
.bold { font-weight: bold; } .italic { font-style: italic; }
.tall { line-height: 1.7; } .fixed { line-height: 31px; }
.row { display: flex; width: 400px; } .row > div { height: 10px; background: #888; }
.outer { width: 360px; background: #ddd; }`

func bars(class string, widths ...string) *ui.Node {
	box := ui.Div(ui.Class(class))
	for _, w := range widths {
		box.Append(ui.Div(ui.Class("b"), ui.Style("width: "+w)))
	}
	return box
}

func TestUnitsOfTheFontMatchChrome(t *testing.T) {
	own := []string{"20ex", "20ch", "20cap", "10ic", "5lh"}
	root := []string{"20rex", "20rch", "20rcap", "10ric", "5rlh"}
	cases := map[string]*ui.Document{
		"sans":              styled(unitsSheet, bars("sans", own...)),
		"monospace":         styled(unitsSheet, bars("mono", own...)),
		"bold":              styled(unitsSheet, bars("bold", own...)),
		"italic":            styled(unitsSheet, bars("italic", own...)),
		"a larger size":     styled(unitsSheet, bars("big", own...)),
		"a line height":     styled(unitsSheet, bars("tall", "5lh", "2.5lh")),
		"a fixed line":      styled(unitsSheet, bars("fixed", "5lh")),
		"the root's":        styled(unitsSheet, bars("mono", root...)),
		"the root's, large": styled(unitsSheet, ui.Div(ui.Class("big", "tall"), bars("mono", root...))),
		"in a font-size": styled(unitsSheet,
			ui.Div(ui.Style("font-size: 3ex"), ui.Text("Sized by ex"), bars("", "10em", "20ex")),
			ui.Div(ui.Style("font-size: 2ch"), ui.Text("Sized by ch"), bars("", "10em")),
			ui.Div(ui.Style("font-size: 1.5cap"), ui.Text("Sized by cap"), bars("", "10em")),
			ui.Div(ui.Style("font-size: 0.8lh"), ui.Text("Sized by lh"), bars("", "10em")),
			ui.Div(ui.Style("font-size: 1.2rex"), ui.Text("Sized by rex"), bars("", "10em"))),
		"in a line-height": styled(unitsSheet,
			ui.Div(ui.Class("tall"), ui.Div(ui.Style("line-height: 1.5lh"), ui.Text("Lines by lh"), bars("", "3lh"))),
			ui.Div(ui.Style("line-height: 4ex"), ui.Text("Lines by ex")),
			ui.Div(ui.Style("line-height: 2rlh"), ui.Text("Lines by rlh"))),
		"in paddings, margins and calc": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Style("padding: 1ex 2ch; margin-left: 1cap"),
				ui.Div(ui.Class("b"), ui.Style("width: calc(100% - 4ch); margin-top: calc(1lh - 1ex)")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestUnitsOfTheViewportMatchChrome(t *testing.T) {
	units := []string{"10vw", "10vh", "10vmin", "10vmax", "10vi", "10vb", "10svw", "10svh", "10lvw", "10lvh",
		"10dvw", "10dvh", "10svmin", "10lvmax", "10dvmin", "calc(50vw - 10vh)"}
	page := func() *ui.Document {
		return styled(unitsSheet, bars("", units...),
			ui.Div(ui.Style("height: 20vh; width: 50vw; padding: 1vmin; margin: 1vh 2vw; background: #ccc"),
				ui.Div(ui.Style("font-size: 3vw; line-height: 6vh"), ui.Text("Sized by the viewport"))))
	}
	for name, size := range map[string]geom.Vec2{
		"wide": geom.V2(640, 400), "tall": geom.V2(520, 700), "small": geom.V2(500, 300),
	} {
		t.Run(name, func(t *testing.T) { chrome.Match(t, page(), size) })
	}
}

func TestMinMaxAndClampMatchChrome(t *testing.T) {
	widths := []string{"min(100%, 200px)", "min(50%, 200px)", "max(50%, 200px)", "max(20%, 3em)",
		"clamp(100px, 50%, 150px)", "clamp(100px, 20%, 150px)", "clamp(10em, 50%, 12em)",
		"calc(min(100%, 500px) - 20px)", "min(100% - 40px, 30em)", "max(10%, min(50%, 100px))",
		"min(10vw, 5em, 30%)", "clamp(1ex, 2ch, 3lh)", "calc(2 * max(1em, 5%))"}
	cases := map[string]*ui.Document{
		"widths": styled(unitsSheet, bars("outer", widths...)),
		"widths in a narrower box": styled(unitsSheet,
			ui.Div(ui.Style("width: 180px"), bars("", widths...))),
		"paddings, margins and heights": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Style("padding: min(5%, 30px) max(2%, 1em); height: clamp(40px, 10vh, 80px)"),
				ui.Div(ui.Class("b"), ui.Style("margin: max(1%, 4px) min(10%, 50px)")))),
		"a percentage of no height": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Div(ui.Style("height: min(50%, 80px); background: #888"), ui.Text("auto")))),
		"a percentage of a height": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Style("height: 100px"),
				ui.Div(ui.Style("height: min(50%, 80px); background: #888"), ui.Text("half")))),
		"flex bases and gaps": styled(unitsSheet,
			ui.Div(ui.Class("row"), ui.Style("gap: min(2%, 20px)"),
				ui.Div(ui.Style("flex: 0 0 min(30%, 200px)")), ui.Div(ui.Style("flex: 0 0 max(30%, 200px)")),
				ui.Div(ui.Style("flex: 1 1 clamp(10px, 5%, 30px)")))),
		"font sizes and line heights": styled(unitsSheet,
			ui.Div(ui.Style("font-size: clamp(12px, 4vw, 20px); line-height: max(1.2, 1.5)"), ui.Text("Clamped text")),
			ui.Div(ui.Style("font-size: min(2em, 20px); line-height: min(40px, 3em)"), ui.Text("Bounded text"))),
		"positions": styled(unitsSheet,
			ui.Div(ui.Class("outer"), ui.Style("position: relative; height: 80px"),
				ui.Div(ui.Style("position: absolute; left: min(10%, 100px); top: max(10%, 20px); "+
					"width: clamp(20px, 30%, 200px); height: 10px; background: #888")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}
