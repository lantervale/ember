// List markers against Chrome's: counter(list-item) in reversed ols beside
// the markers Chrome numbers apart, the markers of items of a value
// attribute, the counter a list or an item names itself, and markers
// inside right-to-left items (bullets, numbers, strings, ::marker's
// content, pictures). Every reversed ol whose items write the counter has
// a start, and no item writing it a value: Chrome counts those once or
// twice, as the page loads.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

// dot is a 6×6 red picture, a list item's marker.
const dot = "data:image/png;base64," +
	"iVBORw0KGgoAAAANSUhEUgAAAAYAAAAGCAYAAADgzO9IAAAAEklEQVR4nGM44W77HxtmGEgJACYcUo3S" +
	"NAkeAAAAAElFTkSuQmCC"

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 8px; font-size: 14px; line-height: 18px; color: #2b2722; }
ol, ul { margin: 0 0 6px; padding: 0 0 0 30px; display: flex; gap: 14px; background-color: #f4f1ec;
  list-style-position: inside; }
.count li::before { content: "[" counter(list-item) "]"; color: #3c6fb0; }
.m li::marker { content: counter(list-item) ") "; color: #b0493c; }
.rtl { direction: rtl; padding: 0 30px 0 0; }
.c { list-style-type: circle; } .d { list-style-type: decimal; } .str { list-style-type: "-> "; }
.mm::marker { content: "M: "; color: #b0493c; }
.pic { list-style-image: url(` + dot + `); }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"list-counters.css": {Data: []byte(sheet)}}, "list-counters.css"))
	li := func(args ...ui.Arg) *ui.Node { return ui.Li(append(args, ui.Text("item"))...) }
	ol := func(args ...ui.Arg) *ui.Node { return ui.Ol(append([]ui.Arg{ui.Class("count")}, args...)...) }
	reversed, value := ui.Attr("reversed", ""), func(v string) ui.Arg { return ui.Attr("value", v) }
	rtl := func(class string) *ui.Node { return li(ui.Class(class)) }
	d.Body().Append(
		ol(reversed, ui.Attr("start", "7"), li(), li(), li()),
		ol(reversed, ui.Attr("start", "3"), ui.Class("m"), li(), li(), li()),
		ui.Ol(reversed, li(), li(value("9")), li()),
		ui.Ol(li(), li(value("9")), li()),
		ol(reversed, ui.Style("counter-reset: list-item 3"), li(), li(), li()),
		ol(reversed, ui.Attr("start", "4"), li(ui.Style("counter-increment: list-item 0")), li(), li()),
		ol(reversed, ui.Attr("start", "3"), li(), li(ui.Style("counter-set: list-item 6")), li()),
		ol(reversed, ui.Attr("start", "2"), li(), ui.Ol(li()), li()),
		ui.Ul(ui.Class("rtl"), rtl(""), rtl("c"), rtl("d"), rtl("str"), rtl("mm"), rtl("pic")),
		ui.Ol(ui.Class("rtl"), ui.Attr("start", "9"), rtl("d"), rtl("d"), rtl("d"), rtl("str")),
	)
	pages.Run(d, 440, 300)
}
