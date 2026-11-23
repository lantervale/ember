//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const prose = "The lantern swung over the crypt stairs, and every step below it rang hollow under the boots of the " +
	"three who went down."

// words is text cut at its spaces, each word in a span of its own, so
// Chrome's boxes tell where each word stands.
func words(text string) []ui.Arg {
	var out []ui.Arg
	start := 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != ' ' {
			continue
		}
		if i > start {
			out = append(out, ui.Span(ui.Text(text[start:i])))
		}
		if i < len(text) {
			out = append(out, ui.Text(" "))
		}
		start = i + 1
	}
	return out
}

func TestFirstLettersMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 360px; }
p { margin: 0 0 10px; }
.big::first-letter { font-size: 200%; color: #a00; }
.drop::first-letter { float: left; font-size: 3.2em; line-height: 1; margin: 2px 6px 0 0; }
.boxed::first-letter { padding: 2px 4px; border: 2px solid #357; margin-right: 3px; background-color: #def; }
.punct::first-letter { font-size: 24px; }
.raised::first-letter { vertical-align: 6px; letter-spacing: 4px; }
.cased::first-letter { text-transform: uppercase; font-weight: 700; }
.before::before { content: "Once "; }
.before::first-letter { font-size: 28px; }
div.outer::first-letter { font-size: 30px; }
li::first-letter { font-size: 22px; }
.cell { display: table-cell; width: 160px; }
.cell::first-letter, .ib::first-letter { font-size: 26px; }
.ib { display: inline-block; width: 150px; }
.none::first-letter { font-size: 40px; }
`))
	d.Body().Append(
		ui.P(append([]ui.Arg{ui.Class("big")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("drop")}, words(prose+" "+prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("boxed")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("punct")}, words(`"(Well," said the warden, "go on.")`)...)...),
		ui.P(append([]ui.Arg{ui.Class("raised")}, words("raised letters "+prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("cased")}, words("quiet now, "+prose)...)...),
		ui.P(ui.Class("big"), ui.B(ui.Text("Bold")), ui.Text(" "), ui.Span(ui.Text("start")), ui.Text(" "),
			ui.Span(ui.Text("here"))),
		ui.P(append([]ui.Arg{ui.Class("before")}, words("upon a time")...)...),
		ui.Div(ui.Class("outer"), ui.P(append([]ui.Arg{ui.Style("margin: 0")}, words("nested block")...)...)),
		ui.Ol(ui.Li(words("listed one")...), ui.Li(words("(second) one")...)),
		ui.Div(ui.Div(ui.Class("cell"), ui.Span(ui.Text("cell")), ui.Text(" "), ui.Span(ui.Text("text")))),
		ui.Div(ui.Span(ui.Class("ib"), ui.Span(ui.Text("inline")), ui.Text(" "), ui.Span(ui.Text("block")))),
		ui.P(ui.Class("none"), ui.Img(ui.Style("width: 10px; height: 10px")), ui.Span(ui.Text("after")),
			ui.Text(" "), ui.Span(ui.Text("picture"))),
	)
	r := chrome.Match(t, d, geom.V2(400, 900))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestInitialLettersMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 360px; }
p { margin: 0 0 12px; }
.i3::first-letter { initial-letter: 3; }
.i2::first-letter { initial-letter: 2; color: #a00; }
.raise::first-letter { initial-letter: 3 2; }
.up::first-letter { initial-letter: 2 raise; }
.boxed::first-letter { initial-letter: 2; padding: 2px 4px; margin-right: 6px; border: 2px solid #357; }
.tall { line-height: 26px; }
.tall::first-letter { initial-letter: 3 drop; }
.half::first-letter { initial-letter: 2.5; }
.large { font-size: 20px; line-height: 1.5; }
.large::first-letter { initial-letter: 4; }
`))
	for _, c := range []string{"i3", "i2", "raise", "up", "boxed", "tall", "half", "large"} {
		d.Body().Append(ui.P(append([]ui.Arg{ui.Class(c), ui.Text("Down ")}, words(prose+" "+prose)...)...))
	}
	r := chrome.Match(t, d, geom.V2(400, 1400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestFirstLinesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 360px; }
p { margin: 0 0 10px; }
.big::first-line { font-size: 22px; letter-spacing: 1px; }
.spaced::first-line { word-spacing: 6px; font-weight: 700; }
.tall::first-line { line-height: 40px; }
.small::first-line { font-size: 11px; }
.ems span { font-size: 0.75em; }
.ems::first-line { font-size: 24px; }
.cased::first-line { text-transform: uppercase; }
div.outer::first-line { font-size: 20px; }
.both::first-line { font-size: 20px; }
.both::first-letter { font-size: 40px; float: left; line-height: 1; }
li::first-line { font-size: 20px; }
.ib { display: inline-block; width: 150px; }
.ib::first-line { font-size: 20px; }
`))
	d.Body().Append(
		ui.P(append([]ui.Arg{ui.Class("big")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("spaced")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("tall")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("small")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("ems")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("cased")}, words(prose)...)...),
		ui.Div(ui.Class("outer"), ui.P(append([]ui.Arg{ui.Style("margin: 0")}, words(prose)...)...)),
		ui.P(append([]ui.Arg{ui.Class("both")}, words(prose+" "+prose)...)...),
		ui.Ul(ui.Li(words(prose)...)),
		ui.Div(ui.Span(append([]ui.Arg{ui.Class("ib")}, words("an inline block of a few words")...)...)),
	)
	r := chrome.Match(t, d, geom.V2(400, 1000))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
