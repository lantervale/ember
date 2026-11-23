//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// A progress and a meter of appearance none stand in a line of text as
// Chrome's do, and a switch of appearance none is the empty box Chrome's
// checkbox is.
func TestBarsOfAppearanceNoneStandInALineAsChromes(t *testing.T) {
	const sheet = `body { font: 16px "Go"; } .none { appearance: none; }
progress, meter { width: 60px; height: 14px; } .tall { height: 30px; }
.small { font-size: 10px; }`
	d := ui.NewDocument(ui.CSS(sheet))
	none := ui.Class("none")
	d.Body().Append(
		ui.Div(ui.Text("xg"), ui.Progress(ui.Value("0.5"), none), ui.Text("xg"), ui.Meter(ui.Value("0.5"), none),
			ui.Text("xg"), ui.Progress(ui.Value("0.5")), ui.Meter(ui.Value("0.5"))),
		ui.Div(ui.Text("xg"), ui.Progress(ui.Class("none", "tall")), ui.Meter(ui.Class("none", "small"))),
		ui.Div(ui.Text("xg"), ui.Input(ui.Type("checkbox"), ui.Attr("switch", ""), none),
			ui.Input(ui.Type("checkbox"), ui.Attr("switch", ""), ui.Checked(true), none)),
	)
	r := chrome.Match(t, d, geom.V2(400, 200))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// A block ::before or ::after in a checkbox or a radio starts at the top
// of its content box, as in Chrome, whatever the control's appearance.
func TestGeneratedBlocksInACheckboxStartAtItsTop(t *testing.T) {
	const sheet = `body { font: 16px "Go"; } .none { appearance: none; }
input { width: 40px; height: 40px; border: 1px solid #000; margin: 4px; }
.b::before { content: "B"; display: block; }
.a::after { content: "A"; display: block; height: 10px; }
.i::before { content: "I"; }`
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(
		ui.Div(ui.Input(ui.Type("checkbox"), ui.Class("none", "b")), ui.Input(ui.Type("radio"), ui.Class("none", "b", "a")),
			ui.Input(ui.Type("checkbox"), ui.Class("b")), ui.Input(ui.Type("radio"), ui.Class("a")),
			ui.Input(ui.Type("checkbox"), ui.Class("none", "i"))),
	)
	r := chrome.Match(t, d, geom.V2(400, 200))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// ::before and ::after on a progress and a meter stand on lines above
// and below the bar, overflowing the box, and on a range at the start
// and the end of a row with the track between them, as in Chrome.
func TestGeneratedContentOnBarsAndRangesAsChromes(t *testing.T) {
	const sheet = `body { font: 16px "Go"; } .none { appearance: none; }
progress, meter { width: 80px; height: 14px; } input[type="range"] { width: 120px; }
.b::before { content: "B"; } .a::after { content: "A"; }
.bb::before { content: "BB"; display: block; }
.big::before { content: "BIG"; font-size: 30px; }
.framed { padding: 3px; border: 2px solid #000; height: 30px; } .auto { height: auto; }
.track::-webkit-slider-runnable-track { height: 6px; margin: 2px 0; }
.track::-webkit-slider-thumb { appearance: none; width: 10px; height: 24px; }
.blocky::before { content: "B"; display: block; width: 20px; }`
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(
		ui.Div(ui.Text("x"), ui.Input(ui.Type("range"), ui.Class("none", "big", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Input(ui.Type("range"), ui.Class("none", "track", "b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Input(ui.Type("range"), ui.Class("none", "blocky", "a"), ui.Attr("dir", "rtl")),
			ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Progress(ui.Class("none", "framed", "b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Meter(ui.Value("0.2"), ui.Class("none", "auto", "b")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Meter(ui.Value("0.2"), ui.Class("a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Progress(ui.Value("0.5"), ui.Class("none", "b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Meter(ui.Value("0.5"), ui.Class("none", "b")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Progress(ui.Value("0.5"), ui.Class("b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Input(ui.Type("range"), ui.Class("none", "b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Input(ui.Type("range"), ui.Class("b", "a")), ui.Text("x")),
		ui.Div(ui.Text("x"), ui.Progress(ui.Value("0.5"), ui.Class("none", "bb")), ui.Text("x")),
	)
	r := chrome.Match(t, d, geom.V2(400, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// A vertical range lays its ::before, its slider container and its
// ::after down it, up it in sideways-lr and slider-vertical, as Chrome's.
func TestGeneratedContentDownVerticalRangesAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`body { margin: 8px; font: 16px "Go"; display: flex; gap: 10px; align-items: flex-start; }
input::before { content: "ab"; background: #f00; } input::after { content: "C"; background: #00f; }
.b::before { display: block; width: 10px; height: 6px; }
.a::after { display: block; inline-size: 7px; block-size: 4px; }
.rl { writing-mode: vertical-rl; } .lr { writing-mode: vertical-lr; } .sl { writing-mode: sideways-lr; }
.sv { appearance: slider-vertical; width: 20px; height: 100px; } .none { appearance: none; }`))
	for _, c := range [][]string{{"rl"}, {"lr"}, {"sl"}, {"sv"}, {"rl", "b", "a"}, {"lr", "b"}, {"sl", "a"},
		{"sv", "b", "a"}, {"rl", "none", "b"}} {
		d.Body().Append(ui.Input(ui.Type("range"), ui.Class(c...)))
	}
	r := chrome.Match(t, d, geom.V2(600, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
