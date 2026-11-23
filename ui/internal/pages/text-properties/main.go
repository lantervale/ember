// Text properties against Chrome's: small capitals synthesised in each
// variant, tab stops in kept text, words broken anywhere, letter spacing
// as a percentage, and decorations that skip the descenders and cast the
// text's shadow.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
p, pre { margin: 0 0 6px; }
.caps { font-variant: small-caps; }
.all { font-variant-caps: all-small-caps; color: #c8473d; }
.uni { font-variant-caps: unicase; }
.big { font: small-caps 22px Go; letter-spacing: 1px; }
pre { tab-size: 6; background-color: #f3efe6; }
.stops { tab-size: 60px; }
.narrow { width: 150px; word-break: break-all; background-color: #f3efe6; }
.any { width: 120px; line-break: anywhere; }
.tracked { letter-spacing: 15%; }
.under { text-decoration: underline #3a6fd8; font-size: 22px; }
.ink { text-decoration: underline #3a6fd8; text-decoration-skip-ink: none; font-size: 22px; }
.cast { text-decoration: underline overline line-through; text-shadow: 2px 3px 0 #3a6fd8; color: #c8473d; }
.blur { text-decoration: underline; text-shadow: 0 2px 2px #3a6fd8; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"text.css": {Data: []byte(sheet)}}, "text.css"))
	d.Body().Append(
		ui.P(ui.Class("caps"), ui.Text("The quick brown fox jumps over the lazy dog.")),
		ui.P(ui.Class("all"), ui.Text("All small capitals, 42 of them.")),
		ui.P(ui.Class("uni"), ui.Text("The Quick Brown Fox Jumps")),
		ui.P(ui.Class("big"), ui.Text("Bigger Small Capitals")),
		ui.Pre(ui.Text("name\tkind\tsize\nember\tengine\t12\ttall\nlantervale\tgame\t7")),
		ui.Pre(ui.Class("stops"), ui.Text("a\tb\tc\tdd\teee")),
		ui.P(ui.Class("narrow"), ui.Text("assets/tilesets/dungeon_walls_north_east.tsj breaks anywhere")),
		ui.P(ui.Class("any"), ui.Text("Anywhere, even (here) and-there.")),
		ui.P(ui.Class("tracked"), ui.Text("Letters fifteen percent apart")),
		ui.P(ui.Class("under"), ui.Text("Jumpy quaggy pygmy gypsy")),
		ui.P(ui.Class("ink"), ui.Text("Jumpy quaggy pygmy gypsy")),
		ui.P(ui.Class("cast"), ui.Text("Lines cast the shadow too")),
		ui.P(ui.Class("blur"), ui.Text("A blurred shadow under a line")),
	)
	pages.Run(d, 420, 490)
}
