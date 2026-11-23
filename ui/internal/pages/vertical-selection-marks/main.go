// A selection's marks on vertical lines against Chrome's: the empty lines of
// preformatted text in vertical-rl and sideways-lr, a br alone on its line in
// vertical-lr, and the room of tabs kept between words, each marked across its line,
// a space long down it, or up it in sideways-lr.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { margin: 0; padding: 12px; font-size: 16px; line-height: 22px; color: #2b2722; display: flex; gap: 14px;
	align-items: flex-start; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
pre, p { margin: 0; font: inherit; inline-size: 200px; border: 1px solid #c8473d; }
.rl { writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; }
.slr { writing-mode: sideways-lr; }
.ink::selection { background-color: #3c6fb0; color: #ffffff; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"marks.css": {Data: []byte(sheet)}}, "marks.css"))
	first := ui.Pre(ui.Class("rl"), ui.Text("pre text\n\nwith empty\tlines\n\n\nbetween"))
	last := ui.Pre(ui.Class("slr", "ink"), ui.Text("up the\tlines\n\nand a\ttab\n\nending"))
	d.Body().Append(
		first,
		ui.P(ui.Class("lr"), ui.Text("Before it"), ui.Br(), ui.Br(), ui.Text("after an empty line"), ui.Br(), ui.Br(),
			ui.Br(), ui.Text("and two")),
		last,
		ui.Pre(ui.Class("rl"), ui.Text("After\tthe selection.")),
	)
	d.Select(ui.Position{Node: first.Children()[0], Offset: 4}, ui.Position{Node: last.Children()[0], Offset: 29})
	pages.Run(d, 520, 240)
}
