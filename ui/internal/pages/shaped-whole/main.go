// Text shaped whole across elements against Chrome's: ligatures and
// kerned pairs across the edge of inline elements of one font
// (go-features.ttf), each glyph drawn in the colour of the element its
// first character is in, edges that take room cutting them, a
// ligature and contextual alternates across a space (go-spaced.ttf),
// and lines broken inside them, their ends shaped again; both copies of
// internal/shape/testdata's.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

var (
	//go:embed go-features.ttf
	featured []byte
	//go:embed go-spaced.ttf
	spaced []byte
)

const sheet = `
@font-face { font-family: Featured; src: url(featured.ttf); }
@font-face { font-family: Spaced; src: url(spaced.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 30px Featured; color: #000; }
div { white-space: nowrap; margin-bottom: 2px; }
.spaced { font-family: Spaced; }
.r { color: #c00; } .b { color: #06c; } .u { text-decoration: underline; } .bg { background: #fd6; }
.pad { padding-left: 3px; } .sub { vertical-align: sub; } .ls { letter-spacing: 2px; }
.w { display: inline-block; vertical-align: top; white-space: normal; width: 40px; margin-right: 24px; }
.all { word-break: break-all; } .any { overflow-wrap: anywhere; } .w60 { width: 60px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":     {Data: []byte(sheet)},
		"featured.ttf": {Data: featured},
		"spaced.ttf":   {Data: spaced},
	}, "page.css"))
	// Each line is its parts: plain text, or "class:text", a span.
	for _, line := range []string{
		"of|r:fi|ce a|b:ffi|nity f|r:l|ow",
		"A|r:V|A|b:T|A|r:R L|b:T|d T|r:o",
		"u:of|fice A|bg:VA|TAR o|u:ff|i|r:ce",
		"of|pad:fi|ce A|pad:V f|sub:i A|sub:V",
		"of|ls:fi|ce A|ls:V|A",
		"spaced x x z z q q",
		"spaced x|r: x z|r: z q |b:q a x|bg: |x b",
	} {
		var args []ui.Arg
		if rest, ok := strings.CutPrefix(line, "spaced "); ok {
			args, line = append(args, ui.Class("spaced")), rest
		}
		for _, part := range strings.Split(line, "|") {
			class, text, ok := strings.Cut(part, ":")
			if !ok {
				args = append(args, ui.Text(part))
				continue
			}
			args = append(args, ui.Span(ui.Class(class), ui.Text(text)))
		}
		d.Body().Append(ui.Div(args...))
	}
	// Lines broken inside kerned pairs and ligatures.
	var boxes []ui.Arg
	for _, box := range []string{
		"w all:AVAVA", "w any:office", "w all w60:ToTo|r:AV", "w spaced:ax xb", "w spaced w60:q q|b: q",
	} {
		classes, text, _ := strings.Cut(box, ":")
		var args []ui.Arg
		for c := range strings.FieldsSeq(classes) {
			args = append(args, ui.Class(c))
		}
		for _, part := range strings.Split(text, "|") {
			if class, t, ok := strings.Cut(part, ":"); ok {
				args = append(args, ui.Span(ui.Class(class), ui.Text(t)))
			} else {
				args = append(args, ui.Text(part))
			}
		}
		boxes = append(boxes, ui.Span(args...))
	}
	d.Body().Append(ui.Div(boxes...))
	pages.Run(d, 460, 480)
}
