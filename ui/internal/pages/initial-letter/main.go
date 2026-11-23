// initial-letter against Chrome's: drop caps of two and three lines, one
// raised, one sunk below its size, one boxed and coloured.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 14px; display: grid; grid-template-columns: 1fr 1fr; gap: 10px 16px; }
p { margin: 0; }
.i3::first-letter { initial-letter: 3; color: #a33; }
.i2::first-letter { initial-letter: 2; font-weight: 700; }
.raise::first-letter { initial-letter: 3 2; color: #36a; }
.boxed::first-letter { initial-letter: 2; padding: 2px 4px; margin-right: 4px; background-color: #def;
  border: 2px solid #357; }
`

const text = "Over the crypt stairs the lantern swung, and every step below rang hollow under the boots of " +
	"the three who went down into the dark."

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range []string{"i3", "i2", "raise", "boxed"} {
		d.Body().Append(ui.P(ui.Class(c), ui.Text(text)))
	}
	pages.Run(d, 520, 260)
}
