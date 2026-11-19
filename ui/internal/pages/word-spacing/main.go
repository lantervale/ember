// Word spacing against Chrome's: wider and tighter spaces, a percentage
// of each element's own font size, no-break spaces, kept runs of spaces,
// justified lines, decorations and shadows across the wider spaces, and
// list markers.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
p, pre, ol { margin: 0 0 6px; }
.wide { word-spacing: 12px; }
.tight { word-spacing: -2px; }
.half { word-spacing: 50%; }
.big { font-size: 24px; }
.red { color: #c8473d; }
.under { text-decoration: underline #3a6fd8; }
.through { text-decoration: line-through; }
.shadow { text-shadow: 2px 2px 0 #3a6fd8; }
.just { width: 300px; text-align: justify; word-spacing: 4px; }
.tracked { letter-spacing: 2px; word-spacing: 1em; }
pre { word-spacing: 6px; }
ol { word-spacing: 20px; }
li::marker { color: #c8473d; }
`

const sentence = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"word-spacing.css": {Data: []byte(sheet)}}, "word-spacing.css"))
	d.Body().Append(
		ui.P(ui.Text("Words as the font spaces them, for scale.")),
		ui.P(ui.Class("wide"), ui.Text("Words twelve points further apart.")),
		ui.P(ui.Class("tight"), ui.Text("Words two points closer than the font has them.")),
		ui.P(ui.Class("half"), ui.Text("Half an em more, "),
			ui.Span(ui.Class("big"), ui.Text("half a bigger em")), ui.Text(" and back.")),
		ui.P(ui.Class("wide"), ui.Text("No\u00a0break\u00a0spaces widen, "),
			ui.Span(ui.Class("red"), ui.Text("a span's spaces too, ")),
			ui.Span(ui.Style("word-spacing: normal"), ui.Text("and these do not."))),
		ui.P(ui.Class("wide", "under"), ui.Text("A line under "),
			ui.Span(ui.Class("through"), ui.Text("the wider spaces")), ui.Text(" as well.")),
		ui.P(ui.Class("wide", "shadow"), ui.Text("A shadow spaced as its text.")),
		ui.P(ui.Class("tracked"), ui.Text("Letters and words apart.")),
		ui.P(ui.Class("just"), ui.Text(sentence+" "+sentence)),
		ui.Pre(ui.Text("kept  spaces   each\n  six points   wider")),
		ui.Ol(ui.Li(ui.Text("A number before its wider space")), ui.Li(ui.Text("And the next one"))),
	)
	pages.Run(d, 420, 440)
}
