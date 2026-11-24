//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
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
	chrome.Since(t, 158, "scrollbar-color's bars lie over the content on every runner")
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

// A box with both gutters scrolls sideways within them, to the content's
// end and no further; a right-to-left box with a stable gutter that
// overflows sideways starts where Chrome starts it.
func TestGuttersBoundTheSidewaysScrollAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: auto; margin: 4px; position: relative; }
.s > div { height: 20px; width: 300px; }
.both { scrollbar-gutter: stable both-edges; }
.stable { scrollbar-gutter: stable; }
.tall > div:first-child { height: 200px; }
.rtl { direction: rtl; }
.styled::-webkit-scrollbar { width: 9px; height: 9px; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range []struct {
		classes []string
		scroll  geom.Vec2
	}{
		{[]string{"both"}, geom.V2(1e6, 0)}, {[]string{"both", "tall"}, geom.V2(1e6, 0)},
		{[]string{"both", "styled"}, geom.V2(1e6, 0)}, {[]string{"both", "rtl"}, geom.V2(-1e6, 0)},
		{[]string{"stable", "rtl"}, geom.V2(0, 0)}, {[]string{"stable", "rtl", "tall"}, geom.V2(0, 0)},
		{[]string{"both", "rtl"}, geom.V2(0, 0)}, {[]string{"stable", "rtl", "styled"}, geom.V2(0, 0)},
		{[]string{"stable", "rtl"}, geom.V2(-1e6, 0)}, {[]string{"stable"}, geom.V2(1e6, 0)},
		{[]string{"stable", "rtl", "styled"}, geom.V2(-1e6, 0)}, {[]string{"stable", "tall"}, geom.V2(1e6, 0)},
	} {
		s := ui.Div(ui.Class(append([]string{"s"}, c.classes...)...), ui.Div(), ui.Div())
		d.Body().Append(s)
		dom.Data(s).Scroll = c.scroll
	}
	r := chrome.Match(t, d, geom.V2(640, 1300))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// In a vertical writing mode a stable gutter is the horizontal bar's, at
// the block end, and both-edges keeps one at the top as well.
func TestVerticalGuttersAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: auto; margin: 4px; position: relative;
  writing-mode: vertical-rl; }
.s > div { width: 20px; }
.s > .abs { position: absolute; top: 3px; bottom: 5px; left: 30px; }
.both { scrollbar-gutter: stable both-edges; }
.stable { scrollbar-gutter: stable; }
.wide > div:first-child { width: 200px; }
.lr { writing-mode: vertical-lr; }
.styled::-webkit-scrollbar { width: 9px; height: 11px; }
.flex { display: flex; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range []struct {
		classes []string
		scroll  geom.Vec2
	}{
		{[]string{"stable"}, geom.Vec2{}}, {[]string{"both"}, geom.Vec2{}}, {[]string{"both", "wide"}, geom.Vec2{}},
		{[]string{"both", "lr"}, geom.Vec2{}}, {[]string{"both", "styled"}, geom.Vec2{}},
		{[]string{"stable", "styled"}, geom.Vec2{}}, {[]string{"both", "flex"}, geom.Vec2{}},
		{[]string{"both", "wide"}, geom.V2(-1e6, 0)}, {[]string{"stable", "wide"}, geom.V2(-1e6, 0)},
		{[]string{"both", "wide", "lr"}, geom.V2(1e6, 0)}, {[]string{"stable", "wide", "lr"}, geom.V2(1e6, 0)},
	} {
		s := ui.Div(ui.Class(append([]string{"s"}, c.classes...)...), ui.Div(), ui.Div(), ui.Div(ui.Class("abs")))
		d.Body().Append(s)
		dom.Data(s).Scroll = c.scroll
	}
	r := chrome.Match(t, d, geom.V2(640, 1200))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// scrollbar-width: thin asks for the system's bars as scrollbar-color
// does, which Chrome on macOS lays over the content, its
// ::-webkit-scrollbar rules ignored.
func TestThinScrollbarsTakeNoRoomAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: scroll; margin: 4px; scrollbar-width: thin; }
.s > div { width: 300px; height: 300px; }
.styled::-webkit-scrollbar { width: 22px; height: 22px; }
.auto { overflow: auto; }
.auto > div { height: 20px; width: auto; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{}, {"styled"}, {"auto", "styled"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div()))
	}
	r := chrome.Match(t, d, geom.V2(640, 400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
