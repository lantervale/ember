// A page drawn in the dark scheme: the canvas, its text, a link and a
// mark in their system colours, the controls a browser's sheet draws,
// enabled above and disabled below, a card of light-dark() colours and,
// beside it, one that keeps the light scheme.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
:root { color-scheme: dark; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: center; margin-bottom: 12px; }
.card { width: 120px; height: 40px; border: 4px solid light-dark(#3a6fd8, #f0c040);
  background-color: light-dark(#ffffff, #1e1e2e); }
.light { color-scheme: light; background-color: Canvas; border: 4px solid ButtonBorder; width: 60px; height: 40px; }
`

func controls(off bool) *ui.Node {
	arg := func(args ...ui.Arg) []ui.Arg {
		if off {
			args = append(args, ui.Attr("disabled", ""))
		}
		return args
	}
	return ui.Div(ui.Class("row"),
		ui.Input(arg(ui.Type("checkbox"), ui.Attr("checked", ""))...),
		ui.Input(arg(ui.Type("checkbox"))...),
		ui.Input(arg(ui.Type("radio"), ui.Attr("checked", ""))...),
		ui.Input(arg(ui.Type("radio"))...),
		ui.Input(arg(ui.Type("range"), ui.Value("30"))...),
		ui.Input(arg(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Attr("checked", ""))...),
		ui.Input(arg(ui.Type("checkbox"), ui.Attr("switch", ""))...),
		ui.Input(arg(ui.Style("width: 60px"))...),
		ui.Button(arg(ui.Style("width: 40px; height: 22px"))...),
	)
}

func main() {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(sheet))
	d.Body().Append(
		ui.Div(ui.Class("row"), ui.Div(ui.Class("card")), ui.Div(ui.Class("light")),
			ui.Mark(ui.Style("display: inline-block; width: 30px; height: 16px")),
			ui.Progress(ui.Value("40"), ui.Attr("max", "100"))),
		controls(false), controls(true),
	)
	pages.Run(d, 560, 160)
}
