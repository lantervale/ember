// Text shaped whole across elements right to left and in sideways
// lines against Chrome's: Arabic's lam-alef and contextual alternates
// across the edge of inline elements (go-arabic.ttf), Hebrew kerned
// across one (a subset of Noto Sans Hebrew), runs of either direction
// in one line, Arabic broken anywhere keeping its joined forms, and
// Latin kerned and ligated in sideways lines (go-features.ttf); copies
// of internal/shape/testdata's.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

var (
	//go:embed go-arabic.ttf
	arabic []byte
	//go:embed noto-hebrew.ttf
	hebrew []byte
	//go:embed go-features.ttf
	featured []byte
)

const sheet = `
@font-face { font-family: Joined; src: url(arabic.ttf); }
@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
@font-face { font-family: Featured; src: url(featured.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 30px Joined; color: #000; }
div { white-space: nowrap; margin-bottom: 2px; direction: rtl; }
.hebrew { font-family: Hebrew; } .featured { font-family: Featured; } .ltr { direction: ltr; }
.r { color: #c00; } .b { color: #06c; } .u { text-decoration: underline; } .bg { background: #fd6; }
.pad { padding-right: 3px; } .sub { vertical-align: sub; }
.w { display: inline-block; vertical-align: top; white-space: normal; width: 40px; margin-left: 24px;
  overflow-wrap: anywhere; }
.side { display: inline-block; vertical-align: top; writing-mode: sideways-rl; margin-left: 12px; }
.turned { display: inline-block; vertical-align: top; writing-mode: vertical-rl; text-orientation: sideways;
  margin-left: 12px; }
`

// parts appends to args the parts of line: plain text, or "class:text",
// a span of that class.
func parts(args []ui.Arg, line string) []ui.Arg {
	for _, part := range strings.Split(line, "|") {
		class, text, ok := strings.Cut(part, ":")
		if !ok {
			args = append(args, ui.Text(part))
			continue
		}
		args = append(args, ui.Span(ui.Class(class), ui.Text(text)))
	}
	return args
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":     {Data: []byte(sheet)},
		"arabic.ttf":   {Data: arabic},
		"hebrew.ttf":   {Data: hebrew},
		"featured.ttf": {Data: featured},
	}, "page.css"))
	// Each line is its classes, a semicolon, then its parts.
	for _, line := range []string{
		";ل|r:ا ب|b:ل|r:ا|ب يل|u:ا",
		";ب|r:م ب|b:مم ل|pad:ا ل|sub:ا",
		";ب|r:َ|ب سس|bg:س|b:س",
		"hebrew;ד|r:ך בד|b:ב|r:ד ד|pad:ך",
		"hebrew;דך|r: AV|b:A ד|r:ך",
		"ltr;ab ل|r:ا| cd",
		"ltr hebrew;AV ד|r:ך AV",
	} {
		classes, text, _ := strings.Cut(line, ";")
		var args []ui.Arg
		for c := range strings.FieldsSeq(classes) {
			args = append(args, ui.Class(c))
		}
		d.Body().Append(ui.Div(parts(args, text)...))
	}
	var boxes []ui.Arg
	for _, box := range []string{
		"w:ببببب", "w hebrew:דךד|r:ך", "side featured:A|r:V|b:A of|r:fi|ce", "turned featured:T|r:o A|pad:V",
	} {
		classes, text, _ := strings.Cut(box, ":")
		var args []ui.Arg
		for c := range strings.FieldsSeq(classes) {
			args = append(args, ui.Class(c))
		}
		boxes = append(boxes, ui.Span(parts(args, text)...))
	}
	d.Body().Append(ui.Div(boxes...))
	pages.Run(d, 460, 480)
}
