// The borders of iframes as Chrome draws them, beside boxes of the same:
// the default inset grey, and inset, outset, groove and ridge sides in
// black and in red.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 12px; margin-bottom: 12px; }
iframe, .box { width: 48px; height: 32px; background-color: #ffffff; }
.black-in { border: 4px inset black; }
.black-out { border: 4px outset black; }
.black-groove { border: 6px groove black; }
.black-ridge { border: 6px ridge black; }
.red-in { border: 4px inset #e03c31; }
.red-out { border: 4px outset #e03c31; }
.navy-in { border: 4px inset #000080; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	looks := []string{"", "black-in", "black-out", "black-groove", "black-ridge", "red-in", "red-out", "navy-in"}
	frames, boxes := ui.Div(ui.Class("row")), ui.Div(ui.Class("row"))
	for _, c := range looks {
		frames.Append(ui.Iframe(nil, ui.Class(c)))
		boxes.Append(ui.Div(ui.Class("box", c)))
	}
	d.Body().Append(frames, boxes)
	pages.Run(d, 560, 140)
}
