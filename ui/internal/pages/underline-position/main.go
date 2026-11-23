// Underlines placed against Chrome's: text-underline-offset as a length,
// a share of the em and a percentage, negative and 0; text-underline-position
// from-font (the face's post table), under (the em box's bottom) and both
// with an offset; text-decoration-thickness from-font; an ancestor's
// placement kept over a descendant's; vertical lines underlined on their
// left, on their right for right and for Japanese, sideways ones off their
// alphabetic baseline; and a variable font's ink skipped at the weight
// drawn.
package main

import (
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/internal/varfont/varfonttest"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Weighty; src: url(weighty.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 20px Go; color: #000; margin: 0; }
.rows { display: flex; gap: 14px; align-items: flex-start; }
.rows > div > div { white-space: nowrap; margin-bottom: 2px; text-decoration: underline #c8473d; }
.none { text-decoration-skip-ink: none; }
.o3 { text-underline-offset: 3px; }
.o0 { text-underline-offset: 0; }
.neg { text-underline-offset: -2.5px; }
.em { text-underline-offset: 0.3em; }
.pc { text-underline-offset: 20%; }
.font { text-underline-position: from-font; }
.under { text-underline-position: under; }
.thin { text-decoration-thickness: from-font; }
.thick { text-decoration-thickness: 3px; }
.double { text-decoration-style: double; }
.big { font-size: 33px; }
.small { font-size: 15px; }
.keep span { text-underline-offset: 0; text-underline-position: under; font-size: 28px; }
.weighty { font-family: Weighty; font-size: 28px; }
.vert { display: flex; gap: 8px; align-items: flex-start; margin-top: 6px; }
.vert div { writing-mode: vertical-rl; height: 150px; font-size: 22px; text-decoration: underline #3a6fd8;
  text-decoration-skip-ink: none; white-space: nowrap; }
.vert .lr { writing-mode: vertical-lr; }
.vert .side { writing-mode: sideways-rl; }
.vert .up { text-orientation: upright; }
.left { text-underline-position: left; }
.right { text-underline-position: right; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":    {Data: []byte(sheet)},
		"weighty.ttf": {Data: varfonttest.Weighty()},
	}, "page.css"))
	line := func(class, text string) *ui.Node { return ui.Div(ui.Class(strings.Fields(class)...), ui.Text(text)) }
	vert := func(class, text string, args ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(strings.Fields(class)...), ui.Text(text)}, args...)...)
	}
	d.Body().Append(
		ui.Div(ui.Class("rows"),
			ui.Div(
				line("none", "auto gyp"),
				line("none o3", "offset 3px"),
				line("none o0", "offset 0 gyp"),
				line("neg", "minus 2.5px"),
				line("none em", "0.3em gyp"),
				line("none pc big", "20% of 33px"),
				line("none font", "from-font gyp"),
				line("none font o3 small", "from-font and 3px"),
				line("none under", "under gyp"),
				line("none under o3 big", "under 3px Qj"),
			),
			ui.Div(
				line("none thin", "from-font thin"),
				line("none thin big font", "thin, from-font"),
				line("none thin under", "thin, under"),
				line("thick o3", "3px thick, gyp"),
				line("none double o3", "double at 3px"),
				line("none double under small", "double under"),
				ui.Div(ui.Class("none", "pc", "keep"), ui.Text("20% "), ui.Span(ui.Text("its own"))),
				line("weighty", "gjpq ygg"),
				ui.Div(ui.Class("weighty"), ui.Style("font-weight: 900"), ui.Text("gjpq ygg")),
			),
		),
		ui.Div(ui.Class("vert"),
			vert("", "vertical lines"),
			vert("right", "on the right"),
			vert("o3", "offset 3px"),
			vert("right o3", "right, 3px"),
			vert("", "Japanese", ui.Attr("lang", "ja")),
			vert("left", "left, ja", ui.Attr("lang", "ja")),
			vert("under", "under, ko", ui.Attr("lang", "ko")),
			vert("lr", "vertical-lr"),
			vert("lr right thick", "lr, right"),
			vert("side", "sideways-rl"),
			vert("side right", "sideways right"),
			vert("up", "Upright"),
		),
	)
	pages.Run(d, 560, 450)
}
