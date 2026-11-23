//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// Bars their sheets style are as thick as their ::-webkit-scrollbar says,
// each way, none for display: none, and a stable gutter is kept whether
// the content overflows or not: the content beside them is laid out in
// what is left, as in Chrome.
func TestStyledScrollbarsTakeTheirRoomAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: scroll; margin: 4px; }
.s > div { width: 300px; height: 300px; }
.thick::-webkit-scrollbar { width: 22px; height: 9px; }
.thin::-webkit-scrollbar:vertical { width: 4px; }
.thin::-webkit-scrollbar:horizontal { height: 30px; }
.none::-webkit-scrollbar { display: none; }
.auto { overflow: auto; }
.auto > div { width: 50%; height: 20px; }
.stable { scrollbar-gutter: stable; }
.stable::-webkit-scrollbar { width: 11px; height: 11px; }
.hidden { overflow: hidden; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{"thick"}, {"thin"}, {"none"}, {"auto", "thick"}, {"auto", "stable"},
		{"hidden", "stable"}, {"auto", "thin"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div(ui.Class("in"))))
	}
	r := chrome.Match(t, d, geom.V2(640, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// scrollbar-gutter: stable both-edges keeps the bar's room at the start
// too, in blocks, flex and grid containers, either direction, its bar
// styled or not, and absolutely placed children stand in what is left.
// (Right to left, a box that also overflows sideways starts scrolled by
// its gutters in Chrome, a gap 10-missing keeps.)
func TestBothEdgesGuttersAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: auto; margin: 4px; position: relative;
  scrollbar-gutter: stable both-edges; }
.s > div { height: 20px; }
.s > .wide { width: 150%; }
.s > .abs { position: absolute; left: 3px; right: 5px; top: 30px; }
.tall > div:first-child { height: 200px; }
.scroll { overflow: scroll; }
.hidden { overflow-x: hidden; overflow-y: hidden; }
.flex { display: flex; }
.flex > div { flex: 1; }
.grid { display: grid; grid-template-columns: 1fr 2fr; justify-content: end; }
.rtl { direction: rtl; }
.rtl > .wide { width: 50%; }
.styled::-webkit-scrollbar { width: 9px; height: 9px; }
.styled::-webkit-scrollbar-thumb { background-color: #333; }
.inline { display: inline-block; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{}, {"tall"}, {"scroll"}, {"hidden"}, {"flex"}, {"grid"}, {"rtl"}, {"rtl", "tall"},
		{"styled"}, {"styled", "tall", "rtl"}, {"flex", "rtl"}, {"inline"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"s"}, c...)...),
			ui.Div(), ui.Div(ui.Class("wide")), ui.Div(ui.Class("abs"))))
	}
	r := chrome.Match(t, d, geom.V2(640, 1400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// scrollbar-color asks for the system's bars, which Chrome on macOS lays
// over the content, keeping no gutter, whether ::-webkit-scrollbar styles
// them or not; the bars it leaves alone keep theirs.
func TestColouredScrollbarsTakeNoRoomAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: scroll; margin: 4px; }
.s > div { width: 300px; height: 300px; }
.c { scrollbar-color: #c33 #ddd; }
.styled::-webkit-scrollbar { width: 22px; height: 22px; }
.auto { overflow: auto; }
.stable { overflow: auto; scrollbar-gutter: stable; }
.stable > div { height: 20px; width: auto; }
.inherit { scrollbar-color: red blue; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{"c"}, {"c", "styled"}, {"styled"}, {"c", "auto"}, {"c", "stable"}, {"stable"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div()))
	}
	d.Body().Append(ui.Div(ui.Class("inherit"), ui.Div(ui.Class("s"), ui.Div())))
	r := chrome.Match(t, d, geom.V2(640, 900))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
