// Skipped ink and small capitals against Chrome's: underlines through
// the daggers, slashes and low lines auto leaves whole and all skips,
// overlines over accents and lines through, none of them skipped; a
// face's own small capitals (go-features.ttf, a copy of
// internal/shape/testdata's) and Go's made-up ones underlined; and small
// capitals in text fields, a closed select and the alt text of a picture
// that didn't load.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed go-features.ttf
var featured []byte

const sheet = `
@font-face { font-family: Featured; src: url(featured.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 22px Go; color: #000; }
div { white-space: nowrap; margin-bottom: 4px; }
.f { font-family: Featured; }
.under { text-decoration: underline; }
.over { text-decoration: overline; }
.through { text-decoration: line-through; }
.thick { text-decoration-thickness: 3px; }
.all { text-decoration-skip-ink: all; }
.none { text-decoration-skip-ink: none; }
.row { display: flex; gap: 8px; align-items: flex-start; font-size: 14px; }
.caps { font-variant-caps: small-caps; }
.all-caps { font-variant-caps: all-small-caps; }
input { width: 150px; }
select { width: 120px; }
textarea { width: 140px; resize: none; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":     {Data: []byte(sheet)},
		"featured.ttf": {Data: featured},
	}, "page.css"))
	line := func(class, text string) *ui.Node { return ui.Div(ui.Class(strings.Fields(class)...), ui.Text(text)) }
	const ink = "g†‡j y_q/p\\j _Q| §¶"
	d.Body().Append(
		line("under thick", ink),
		line("under thick all", ink),
		line("under thick none", ink),
		line("over thick", "ÉÀ Ñ ÖÂ"),
		line("over thick all", "ÉÀ Ñ ÖÂ"),
		line("through all", "gypsy H/A_x †"),
		line("f under caps", "Small Capitals, jumpy"),
		line("f under all-caps all", "All Small Quips"),
		line("under caps", "Made up in Go, jumpy"),
		ui.Div(ui.Class("row"),
			ui.Input(ui.Type("text"), ui.Class("caps"), ui.Value("Field of Small caps")),
			ui.Input(ui.Type("text"), ui.Class("all-caps", "f"), ui.Value("Featured Field")),
		),
		ui.Div(ui.Class("row"),
			ui.Textarea(ui.Class("caps"), ui.Attr("rows", "2"), ui.Text("a textarea\nin Small caps")),
			ui.Select(ui.Class("caps"), ui.Option(ui.Text("Closed select")), ui.Option(ui.Text("Second"))),
		),
		ui.Div(ui.Class("row"),
			ui.Img(ui.Attr("src", "missing.png"), ui.Class("caps"), ui.Attr("alt", "Broken in Small caps"),
				ui.Style("width: 170px; height: 40px")),
			ui.Img(ui.Attr("src", "missing.png"), ui.Class("caps"), ui.Attr("alt", "Unsized caps")),
		),
	)
	pages.Run(d, 440, 420)
}
