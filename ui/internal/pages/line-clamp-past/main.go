// What a line-clamp hides past its last line, as Chrome behind its
// CSSLineClamp flag: a bordered span's lines and a float after the clamp
// are not drawn, a positioned box still is; -webkit-line-clamp draws all
// of them; line-clamp: auto clamps at the last line a max-height holds.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722; }
p { margin: 0; }
.c { line-clamp: 2; width: 180px; margin-bottom: 14px; position: relative; background-color: #f4f1ec; }
.w { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.auto { line-clamp: auto; max-height: 40px; }
.s { background-color: #dfe9f6; border: 2px solid #3c5a8a; padding: 0 2px; }
.f { float: right; width: 26px; height: 22px; background-color: #3c8a4f; }
.a { position: absolute; right: 4px; bottom: -10px; width: 40px; height: 8px; background-color: #b0493c; }
`

const words = "The lantern swings over the crypt stairs and its light runs ahead of the hero, " +
	"down to the door where the old key waits"

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"line-clamp-past.css": {Data: []byte(sheet)}}, "line-clamp-past.css"))
	clamped := func(class string) *ui.Node {
		return ui.Div(ui.Class("c", class), ui.Text("Lit: "), ui.Span(ui.Class("s"), ui.Text(words)))
	}
	blocks := func(class string) *ui.Node {
		return ui.Div(ui.Class("c", class), ui.P(ui.Text("First line")), ui.P(ui.Text("Second line")),
			ui.Div(ui.Class("f")), ui.P(ui.Text("Third line"), ui.Span(ui.Class("a"))))
	}
	d.Body().Append(
		clamped(""), clamped("w"),
		blocks(""), blocks("w"),
		ui.Div(ui.Class("c", "auto"), ui.Text(words+" "+words)),
	)
	pages.Run(d, 420, 360)
}
