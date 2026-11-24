//go:build chrome

package ui_test

import (
	"runtime"
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
.boxed::first-letter { initial-letter: 2; padding: 2px 4px; margin-right: 4px; border: 2px solid #357; }
.tall { line-height: 26px; }
.tall::first-letter { initial-letter: 3 drop; }
.large { font-size: 20px; line-height: 1.5; }
.large::first-letter { initial-letter: 4; }
`))
	for _, c := range []string{"i3", "i2", "raise", "up", "boxed", "tall", "large"} {
		d.Body().Append(ui.P(append([]ui.Arg{ui.Class(c), ui.Text("Down ")}, words(prose+" "+prose)...)...))
	}
	// Chrome on Linux and Windows rounds a letter's ink out to whole
	// pixels, so its box is up to a pixel wider on each side than ui's
	// and macOS's; each first line keeps more room than that.
	within := float32(chrome.Tolerance)
	if runtime.GOOS != "darwin" {
		within = 2
	}
	r := chrome.MatchWithin(t, d, geom.V2(400, 1400), within)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// A drop cap of a fractional size sinks its size rounded down and rises
// over the first line by the rest, pushing the lines down, as Chrome 158.
func TestFractionalInitialLettersMatchChrome(t *testing.T) {
	chrome.Since(t, 158, "a fractional initial-letter sinks its size rounded down, no longer up")
	for _, size := range []string{"1.5", "2.3", "2.5", "2.7", "3.5", "2.5 drop", "2.5 raise", "2.5 3"} {
		t.Run(size, func(t *testing.T) {
			d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 360px; }
p { margin: 0 0 12px; }
.x::first-letter { initial-letter: ` + size + `; }
`))
			d.Body().Append(ui.P(append([]ui.Arg{ui.Class("x"), ui.Text("Down ")}, words(prose+" "+prose)...)...))
			within := float32(chrome.Tolerance)
			if runtime.GOOS != "darwin" {
				within = 2
			}
			r := chrome.MatchWithin(t, d, geom.V2(400, 600), within)
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
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

// TestFirstLineSizesMatchChrome sizes to their content blocks whose first
// line is styled, as Chrome measures them: every line in the first
// line's styles, the padding, border and margin in em of the inline
// elements on it resolved in its font; and vertical-align on a first
// line, which Chrome reads and lays out nothing by.
func TestFirstLineSizesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 380px; }
p { margin: 0; }
.big::first-line { font-size: 30px; }
.small::first-line { font-size: 8px; }
.min { width: min-content; margin-bottom: 6px; }
.max { width: max-content; margin-bottom: 6px; }
.fit { float: left; clear: left; margin-bottom: 6px; }
.lines { white-space: pre-line; }
.ems span { padding: 0 1em; margin: 0 0.5em; border: 0.125em solid #357; }
.ib { display: inline-block; }
.ib::first-line { font-size: 24px; }
.raised::first-line { vertical-align: 10px; }
.super::first-line { vertical-align: super; font-size: 20px; }
.after { clear: left; }
`))
	d.Body().Append(
		ui.Div(ui.Class("min"), ui.P(ui.Class("big"), ui.Span(ui.Text("aa")), ui.Text(" "), ui.Span(ui.Text("bbbbbb")))),
		ui.Div(ui.Class("min"), ui.P(ui.Class("small"), ui.Span(ui.Text("aa")), ui.Text(" "),
			ui.Span(ui.Text("bbbbbb")))),
		ui.Div(ui.Class("max"), ui.P(ui.Class("big", "lines"), ui.Span(ui.Text("aa bb")), ui.Text("\n"),
			ui.Span(ui.Text("cc dddddd")))),
		ui.Div(ui.Class("fit"), ui.P(ui.Class("small", "lines"), ui.Span(ui.Text("aa bb")), ui.Text("\n"),
			ui.Span(ui.Text("cc dddddd")))),
		ui.Div(ui.Class("fit"), ui.P(ui.Class("big", "ems"), ui.Text("aa "), ui.Span(ui.Text("bb")))),
		ui.Div(ui.Class("min"), ui.P(ui.Class("big", "ems"), ui.Text("aa "), ui.Span(ui.Text("bb")))),
		ui.P(ui.Class("big", "ems", "after"), ui.Text("aa "), ui.Span(ui.Text("bb")), ui.Text(" cc "),
			ui.Span(ui.Text("dd")), ui.Text(" over and over past the end of the line "), ui.Span(ui.Text("ee"))),
		ui.Div(ui.Span(ui.Class("ib"), ui.Span(ui.Text("an")), ui.Text(" "), ui.Span(ui.Text("inline"))),
			ui.Text(" "), ui.Span(ui.Text("after"))),
		ui.P(append([]ui.Arg{ui.Class("raised")}, words(prose)...)...),
		ui.P(append([]ui.Arg{ui.Class("super")}, words(prose)...)...),
	)
	r := chrome.Match(t, d, geom.V2(400, 700))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestFirstLettersAfterKeptSpacesMatchChrome takes the first letter of
// text whose spaces white-space keeps, the spaces before it going with
// it into its box, in its size; and the first letter of text a
// translation changes, taken from what the translation says.
func TestFirstLettersAfterKeptSpacesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; width: 380px; }
p { margin: 0 0 6px; float: left; clear: left; }
p::first-letter { font-size: 32px; }
.pre { white-space: pre; }
.wrap { white-space: pre-wrap; }
.line { white-space: pre-line; }
.boxed::first-letter { padding: 0 4px; border: 2px solid #357; }
.drop::first-letter { float: left; font-size: 40px; line-height: 1; }
`), ui.WithTranslator(translator{"greeting": "Willkommen", "quote": "«Ça va»", "night": "Úlfur"}))
	d.Body().Append(
		ui.P(ui.Class("pre"), ui.Text("   Lantern "), ui.Span(ui.Text("light"))),
		ui.P(ui.Class("pre"), ui.Text("  “Well” "), ui.Span(ui.Text("said"))),
		ui.P(ui.Class("wrap"), ui.Text("   Lantern "), ui.Span(ui.Text("light"))),
		ui.P(ui.Class("line"), ui.Text("   Lantern "), ui.Span(ui.Text("light"))),
		ui.P(ui.Class("pre", "boxed"), ui.Text("  Every "), ui.Span(ui.Text("step"))),
		ui.P(ui.Class("pre", "drop"), ui.Text("  Every "), ui.Span(ui.Text("step"))),
		ui.P(ui.Class("pre"), ui.Span(ui.Text("  ")), ui.Text("  Under "), ui.Span(ui.Text("boots"))),
		ui.P(ui.Text("greeting"), ui.Text(" "), ui.Span(ui.Text("friend"))),
		ui.P(ui.Text("quote"), ui.Text(" "), ui.Span(ui.Text("friend"))),
		ui.P(ui.Class("boxed"), ui.Text("night"), ui.Text(" "), ui.Span(ui.Text("friend"))),
		ui.P(ui.Attr("translate", "no"), ui.Text("night"), ui.Text(" "), ui.Span(ui.Text("friend"))),
	)
	r := chrome.Match(t, d, geom.V2(400, 700))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// translator says a few words otherwise.
type translator map[string]string

func (tr translator) Tr(m string) string {
	if s, ok := tr[m]; ok {
		return s
	}
	return m
}
