//go:build chrome

package layout_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func styled(sheet string, nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(nodes...)
	return d
}

// words is a paragraph with each word in a span of its own, so Chrome's boxes say where
// every word of every line went.
func words(style, text string) *ui.Node {
	args := []ui.Arg{ui.Style(style)}
	for i, w := range strings.Fields(text) {
		if i > 0 {
			args = append(args, ui.Text(" "))
		}
		args = append(args, ui.Span(ui.Text(w)))
	}
	return ui.P(args...)
}

func TestFloatsMatchChrome(t *testing.T) {
	const boxes = `.l { float: left; } .r { float: right; }
.box { width: 60px; height: 50px; background: #888; }
.wide { width: 240px; margin: 0; }`
	cases := map[string]*ui.Document{
		"text wraps round a left float": styled(boxes,
			ui.Div(ui.Class("wide"), ui.Div(ui.Class("l", "box")), ui.P(ui.Style("margin: 0"), ui.Text(sentence+" "+sentence))),
		),
		"text wraps round a right float with margins": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("r", "box"), ui.Style("margin: 4px 0 6px 10px")),
				ui.P(ui.Text(sentence+" "+sentence))),
		),
		"text between a left and a right float": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box")), ui.Div(ui.Class("r", "box"), ui.Style("height: 80px")),
				ui.P(ui.Style("margin: 0; text-align: center"), ui.Text(sentence+" "+sentence))),
		),
		"a float inside the paragraph it starts": styled(boxes,
			ui.P(ui.Class("wide"), ui.Span(ui.Class("l", "box")), ui.Text(sentence+" "+sentence)),
		),
		"a float in a line it fits beside": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("Title"), ui.Span(ui.Class("r"), ui.Text("today"))),
		),
		"a float inside a span": styled(boxes,
			ui.P(ui.Class("wide"), ui.Span(ui.Span(ui.Class("l", "box")), ui.Text(sentence)), ui.Text(" "+sentence)),
		),
		"a block clears the floats": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box")), ui.Div(ui.Class("r", "box"), ui.Style("height: 30px")),
				ui.P(ui.Style("margin: 0"), ui.Text("Beside.")),
				ui.P(ui.Style("clear: right; margin: 0"), ui.Text("Below the right one.")),
				ui.P(ui.Style("clear: both; margin: 8px 0"), ui.Text("Below both, with margins.")),
			),
		),
		"containers hold their floats or not": styled(boxes+` .held { overflow: hidden; } .root { display: flow-root; }
.frame { border: 2px solid #000; padding: 4px; margin-bottom: 8px; }`,
			ui.Div(ui.Class("wide", "frame"), ui.Div(ui.Class("l", "box")), ui.Text("Not held.")),
			ui.Div(ui.Class("wide", "frame", "held"), ui.Div(ui.Class("l", "box")), ui.Text("Held by overflow.")),
			ui.Div(ui.Class("wide", "frame", "root"), ui.Div(ui.Class("r", "box")), ui.Text("Held by flow-root.")),
			ui.Div(ui.Class("wide", "frame"), ui.Div(ui.Class("l", "box")), ui.Text("Cleared."),
				ui.Div(ui.Style("clear: both"))),
		),
		"three floated columns and a footer": styled(`.col { float: left; width: 33.33%; }
.col p { margin: 6px; } .foot { clear: both; border-top: 1px solid #000; }`,
			ui.Div(ui.Style("width: 420px"),
				ui.Div(ui.Class("col"), ui.P(ui.Text(sentence))),
				ui.Div(ui.Class("col"), ui.P(ui.Text("Short."))),
				ui.Div(ui.Class("col"), ui.P(ui.Text(sentence+" "+sentence))),
				ui.Div(ui.Class("foot"), ui.Text("The footer.")),
			),
		),
		"floats shrink to their text and wrap to new rows": styled(`
.tab { float: left; padding: 4px 10px; margin: 0 6px 6px 0; border: 1px solid #000; }
.bar { width: 200px; overflow: hidden; }`,
			ui.Div(ui.Class("bar"),
				ui.Div(ui.Class("tab"), ui.Text("Home")), ui.Div(ui.Class("tab"), ui.Text("Inventory")),
				ui.Div(ui.Class("tab"), ui.Text("Quests")), ui.Div(ui.Class("tab"), ui.Text("Map")),
				ui.Div(ui.Class("tab", "r"), ui.Text("Quit")),
			),
		),
		"a block with its own context stands beside the float": styled(boxes+` .own { overflow: hidden; background: #ccc; }`,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box"), ui.Style("margin-right: 8px")),
				ui.Div(ui.Class("own"), ui.Text(sentence)),
				ui.Div(ui.Class("own"), ui.Style("width: 200px"), ui.Text("Too wide to stand beside.")),
			),
		),
		"right to left, blocks with their own context beside floats": styled(boxes+`
.own { overflow: hidden; background: #ccc; }
.rtl { direction: rtl; } .s { float: inline-start; } .e { float: inline-end; }`,
			ui.Div(ui.Class("wide", "rtl"),
				ui.Div(ui.Class("r", "box"), ui.Style("margin-left: 8px")),
				ui.Div(ui.Class("own"), ui.Text(sentence)),
				ui.Div(ui.Class("own"), ui.Style("width: 120px; margin-right: 5px"), ui.Text("Narrow.")),
				ui.Div(ui.Class("own"), ui.Style("width: 200px"), ui.Text("Too wide to stand beside.")),
			),
			ui.Div(ui.Class("wide", "rtl"),
				ui.Div(ui.Class("l", "box")),
				ui.Div(ui.Class("own"), ui.Style("width: 100px"), ui.Text("Beside a left float.")),
				ui.Div(ui.Class("own"), ui.Style("display: flow-root; margin-inline-start: 6px"), ui.Text(sentence)),
			),
			ui.Div(ui.Class("wide", "rtl"),
				ui.Div(ui.Class("s", "box"), ui.Style("height: 30px")),
				ui.Div(ui.Class("e", "box"), ui.Style("width: 40px")),
				ui.Div(ui.Style("display: flex; height: 20px; background: #ccc")),
				ui.Table(ui.Tr(ui.Td(ui.Text("a table")))),
				ui.Div(ui.Class("own"), ui.Style("width: 60px; margin-inline-end: auto"), ui.Text("End.")),
			),
		),
		"right to left, -webkit-left and -webkit-right place blocks": styled(boxes+` .rtl { direction: rtl; }
.k { width: 100px; height: 10px; background: #888; }`,
			ui.Div(ui.Class("wide", "rtl"), ui.Style("text-align: -webkit-left"), ui.Div(ui.Class("k")),
				ui.Div(ui.Class("k"), ui.Style("margin-left: 10px")), ui.Div(ui.Class("k"), ui.Style("margin-right: 10px")),
				ui.P(ui.Style("width: 150px"), ui.Text("Words aligned left."))),
			ui.Div(ui.Class("wide"), ui.Style("text-align: -webkit-right"), ui.Div(ui.Class("k", "rtl")),
				ui.Div(ui.Class("rtl"), ui.Style("width: 150px; text-align: -webkit-left"), ui.Div(ui.Class("k"),
					ui.Style("width: 50px")))),
			ui.Div(ui.Class("wide", "rtl"), ui.Style("text-align: -webkit-center"), ui.Div(ui.Class("k")),
				ui.Div(ui.Class("k", "l")), ui.Div(ui.Class("k"), ui.Style("overflow: hidden; width: 60px"))),
		),
		"a word too long for the room beside a float goes below it": styled(boxes,
			ui.Div(ui.Style("width: 150px"),
				ui.Div(ui.Class("l", "box"), ui.Style("width: 100px; height: 30px")),
				ui.P(ui.Style("margin: 0"), ui.Text("Incomprehensibilities are rare.")),
			),
		),
		"floats in nested blocks reach the blocks after": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Style("padding-left: 10px"), ui.Div(ui.Class("l", "box"), ui.Style("height: 70px")),
					ui.Text("First block.")),
				ui.Div(ui.Style("margin: 0 0 0 20px"), ui.Text(sentence)),
				ui.Div(ui.Text(sentence)),
			),
		),
		"every word round two floats": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box"), ui.Style("height: 40px")),
				ui.Div(ui.Class("r", "box"), ui.Style("width: 90px; height: 64px")),
				words("margin: 0", sentence+" "+sentence),
				words("margin: 0; text-align: right", sentence)),
		),
		"every word of a justified text beside a float taller than a line": styled(boxes,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box"), ui.Style("margin: 0 8px 4px 0")),
				words("margin: 0; text-align: justify; font-size: 20px", sentence+" "+sentence)),
		),
		"a centred column of floats": styled(
			boxes+` .page { width: 300px; margin: 0 auto; } .side { float: right; width: 80px; }`,
			ui.Div(ui.Style("width: 500px"),
				ui.Div(ui.Class("page"),
					ui.Div(ui.Class("side"), ui.Text("A side note that wraps.")),
					ui.P(ui.Style("margin: 0"), ui.Text(sentence+" "+sentence)),
				),
			),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestFloatsInLinesMatchChrome(t *testing.T) {
	const boxes = `.l { float: left; } .r { float: right; }
.box { width: 60px; height: 30px; background: #888; }
.wide { width: 240px; margin: 0; } .rtl { direction: rtl; }`
	box := func(class, style string) *ui.Node { return ui.Span(ui.Class(class, "box"), ui.Style(style)) }
	cases := map[string]*ui.Document{
		"a left float after text, on the line it falls on": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("Some words "), box("l", ""), ui.Text(" then "+sentence)),
		),
		"a right float after text, on the line it falls on": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("Some words "), box("r", ""), ui.Text(" then "+sentence)),
		),
		"a float too wide for the rest of its line goes below it": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("The quick brown fox jumps "), box("l", "width: 120px"),
				ui.Text(" over "+sentence)),
			ui.P(ui.Class("wide"), ui.Text("The quick brown fox jumps "), box("r", "width: 120px"),
				ui.Text(" over "+sentence)),
		),
		"a float on a later line": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text(sentence+" Here "), box("l", ""), ui.Text(" "+sentence)),
		),
		"several floats on one line": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("One "), box("l", "width: 40px"), ui.Text(" two "),
				box("l", "width: 30px; height: 20px"), ui.Text(" three "), box("r", "width: 50px"),
				ui.Text(" "+sentence)),
		),
		"floats after text, one fitting and the next not": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("The quick brown fox "), box("l", "width: 50px"),
				box("r", "width: 90px"), ui.Text(" "+sentence)),
		),
		"a float in a block among text and an inline element": styled(boxes,
			ui.Div(ui.Class("wide"), ui.Text("Words before "), ui.Div(ui.Class("l", "box")),
				ui.Text(" and after "), ui.Span(ui.Text("in a span "), box("r", "height: 20px"), ui.Text(" more")),
				ui.Text(" "+sentence)),
		),
		"a float at the end of the text": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("A short line "), box("r", "")),
			ui.P(ui.Class("wide"), ui.Text(sentence+" "), box("l", "width: 200px")),
		),
		"a float after a line break": styled(boxes,
			ui.P(ui.Class("wide"), ui.Text("First line"), ui.Br(), box("l", ""), ui.Text("second line "+sentence)),
		),
		"a float after text beside an earlier float": styled(boxes,
			ui.Div(ui.Class("wide"), ui.Div(ui.Class("l", "box"), ui.Style("height: 50px")),
				ui.P(ui.Style("margin: 0"), ui.Text("Beside it "), box("r", "width: 40px"), ui.Text(" "+sentence))),
		),
		"right to left, floats after text": styled(boxes,
			ui.P(ui.Class("wide", "rtl"), ui.Text("Some words "), box("l", ""), ui.Text(" then "+sentence)),
			ui.P(ui.Class("wide", "rtl"), ui.Text("Some words "), box("r", ""), ui.Text(" then "+sentence)),
			ui.P(ui.Class("wide", "rtl"), ui.Text("The quick brown fox jumps "), box("r", "width: 120px"),
				ui.Text(" over "+sentence)),
		),
		"floats after text in columns": styled(boxes,
			ui.Div(ui.Style("columns: 2; column-gap: 12px; width: 300px"),
				ui.P(ui.Style("margin: 0"), ui.Text("Some words "), box("l", "width: 40px"),
					ui.Text(" then "+sentence+" More "), box("r", "width: 50px; height: 40px"),
					ui.Text(" "+sentence+" "+sentence))),
		),
		"floats after text on a trimmed first line": styled(boxes,
			ui.P(ui.Class("wide"), ui.Style("text-box: trim-both cap alphabetic; line-height: 24px"),
				ui.Text("Some words "), box("l", ""), ui.Text(" then "+sentence)),
			ui.P(ui.Class("wide"), ui.Style("text-box: trim-both cap alphabetic; line-height: 24px"),
				ui.Text("The quick brown fox jumps "), box("r", "width: 120px"), ui.Text(" over "+sentence)),
		),
		"every word round a float after text": styled(boxes,
			ui.Div(ui.Class("wide"), words("margin: 0", "The quick brown"),
				ui.P(ui.Style("margin: 0"), ui.Span(ui.Text("fox jumps ")), box("l", "width: 70px; height: 40px"),
					ui.Span(ui.Text("over the lazy dog, then naps in the warm afternoon sun.")))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestTextIndentMatchesChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"the first line starts in": styled(`p { width: 180px; text-indent: 24px; }`,
			ui.P(ui.Text(sentence)), ui.P(ui.Style("text-align: right"), ui.Text(sentence)),
			ui.P(ui.Style("text-align: center"), ui.Text("Short.")),
		),
		"a percentage of the block's width, and out when negative": styled(`p { width: 200px; margin-left: 40px; }`,
			ui.P(ui.Style("text-indent: 10%"), ui.Text(sentence)),
			ui.P(ui.Style("text-indent: -20px"), ui.Text(sentence)),
			ui.P(ui.Style("text-indent: -9999px; overflow: hidden; height: 20px"), ui.Text("Replaced by a picture")),
		),
		"inherited, and only a block's first line": styled(`div { width: 200px; text-indent: 2em; }`,
			ui.Div(ui.Text("An anonymous first line that wraps over two lines."),
				ui.P(ui.Text(sentence)), ui.Text("Text after the paragraph is not indented again."),
				ui.Span(ui.Style("display: inline-block; width: 120px"), ui.Text("An inline-block indents its own."))),
		),
		"shrink-to-fit boxes count it": styled(`span { display: inline-block; text-indent: 30px; border: 1px solid #000; }`,
			ui.Div(ui.Span(ui.Text("Indented")), ui.Text(" "), ui.Span(ui.Style("width: min-content"), ui.Text("Two words"))),
		),
		"every word of indented lines": styled(`p { width: 180px; }`,
			words("text-indent: 30px", sentence),
			words("text-indent: -12px; text-align: center; padding-left: 20px", sentence),
			ui.Div(ui.Style("float: right; width: 60px; height: 30px")),
			words("text-indent: 15%; text-align: right", sentence),
		),
		"beside a float": styled(`p { width: 200px; text-indent: 20px; margin: 0; }`,
			ui.Div(ui.Style("float: left; width: 50px; height: 40px")), ui.P(ui.Text(sentence)),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
