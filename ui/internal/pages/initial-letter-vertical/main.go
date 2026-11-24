// initial-letter in vertical text against Chrome's: drop caps of two and
// three lines in vertical-rl and vertical-lr, one raised, one boxed, one
// standing upright.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 14px; display: flex; gap: 14px; align-items: flex-start; }
p { margin: 0; height: 230px; writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; }
.i3::first-letter { initial-letter: 3; color: #a33; }
.i2::first-letter { initial-letter: 2; font-weight: 700; }
.raise::first-letter { initial-letter: 3 2; color: #36a; }
.boxed::first-letter { initial-letter: 2; padding: 2px 4px; margin-right: 4px; background-color: #def;
  border: 2px solid #357; }
.upright { text-orientation: upright; }
`

const text = "Over the crypt stairs the lantern swung, and every step rang hollow."

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{"i3"}, {"i2", "lr"}, {"raise"}, {"boxed", "lr"}, {"i2", "upright"}} {
		d.Body().Append(ui.P(ui.Class(c...), ui.Text(text)))
	}
	pages.Run(d, 520, 260)
}
