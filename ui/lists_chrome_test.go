//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// counterStyles define a style of each system and each descriptor, and
// the rules Chrome drops or reads as defining none.
const counterStyles = `
body { margin: 8px; font-size: 20px; }
p { margin: 0 0 6px; }
p span { display: inline-block; margin-right: 6px; }
@counter-style thumbs { system: cyclic; symbols: "a" "bb" "ccc"; }
@counter-style fixed3 { system: fixed 3; symbols: x y z; fallback: upper-roman; }
@counter-style stars { system: symbolic; symbols: "*" "+"; }
@counter-style base3 { system: numeric; symbols: "0" "1" "2"; negative: "(" ")"; pad: 4 "_"; }
@counter-style pairs { system: alphabetic; symbols: x y; range: 2 5, 8 infinite; fallback: lower-greek; }
@counter-style tally { system: additive; additive-symbols: 5 "V", 1 "I", 0 "o"; }
@counter-style framed { system: extends lower-roman; prefix: "["; suffix: "] "; negative: "~"; }
@counter-style wide { system: extends decimal; pad: 3 "0"; range: -5 50; fallback: upper-alpha; }
@counter-style loop1 { system: extends loop2; }
@counter-style loop2 { system: extends loop1; }
@counter-style lonely { system: alphabetic; symbols: x; }
@counter-style decimal { system: cyclic; symbols: "nope"; }
@counter-style circular { system: fixed; symbols: q; fallback: circular2; }
@counter-style circular2 { system: fixed; symbols: r; fallback: circular; }
@counter-style deaf { system: cyclic; symbols: "s"; speak-as: bullets; }
@counter-style lower-latin { system: cyclic; symbols: "L"; }
@counter-style ranged { system: extends upper-alpha; range: auto; }
@media (max-width: 10px) { @counter-style thumbs { system: cyclic; symbols: "m"; } }
`

// TestCounterStylesWriteEachValueAsChromes writes -3 to 14 in each style
// in an inline-block ::before. square and the disclosure triangles are
// left out: their glyphs are in none of the faces the page has, and
// Chrome draws them from a system font.
func TestCounterStylesWriteEachValueAsChromes(t *testing.T) {
	names := []string{"thumbs", "fixed3", "stars", "base3", "pairs", "tally", "framed", "wide", "loop1", "lonely",
		"decimal", "circular", "deaf", "nothing-such", "none", "disc", "circle", "lower-greek", "upper-alpha",
		"lower-alpha", "lower-latin", "ranged", "lower-roman", "upper-roman", "decimal-leading-zero", "Lower-Roman"}
	sheet := ""
	var ps []*ui.Node
	for _, name := range names {
		p := ui.P()
		for v := -3; v <= 14; v++ {
			p.Append(ui.Span(ui.Class("c-"+name), ui.Style(fmt.Sprintf("counter-reset: n %d", v)), ui.Text("|")))
		}
		ps = append(ps, p)
		sheet += fmt.Sprintf(".c-%s::before { content: counter(n, %s); }\n", name, name)
	}
	d := ui.NewDocument(ui.CSS(counterStyles + sheet))
	d.Body().Append(ps...)
	r := chrome.Match(t, d, geom.V2(900, 1400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestSymbolsWriteEachValueAsChromes writes -3 to 14 in counter styles
// symbols() makes, in counter() and in list items' markers, and compares
// list-style-type with getComputedStyle's: symbols() of one alphabetic
// string, of none, of a name or of a picture is dropped.
func TestSymbolsWriteEachValueAsChromes(t *testing.T) {
	funcs := []string{`symbols("*" "+")`, `symbols(cyclic "a" "bb")`, `SYMBOLS(Fixed 'x' 'y' 'z')`,
		`symbols(alphabetic "x" "y")`, `symbols(numeric "0" "1" "2")`, `symbols(symbolic "o")`,
		`symbols(alphabetic "x")`, `symbols()`, `symbols(cyclic x)`, `symbols(cyclic url(pic.png))`}
	sheet := "body { margin: 8px; font-size: 20px; } p, ol { margin: 0 0 6px; } p span { display: inline-block; }\n" +
		"ol { padding-left: 60px; list-style-type: lower-roman; }\n"
	var nodes []*ui.Node
	for i, f := range funcs {
		p := ui.P()
		for v := -3; v <= 14; v++ {
			p.Append(ui.Span(ui.Class(fmt.Sprintf("c%d", i)), ui.Style(fmt.Sprintf("counter-reset: n %d", v)),
				ui.Text("|")))
		}
		ol := ui.Ol(ui.Class(fmt.Sprintf("l%d", i)), ui.Attr("start", "-1"))
		for range 5 {
			ol.Append(ui.Li(ui.Text("item")))
		}
		nodes = append(nodes, p, ol)
		sheet += fmt.Sprintf(".c%d::before { content: counter(n, %s); }\n.l%d { list-style-type: %s; }\n", i, f, i, f)
	}
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(nodes...)
	r := chrome.Match(t, d, geom.V2(700, 1400))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	want, err := chrome.Styles(d, geom.V2(700, 1400), []string{"list-style-type"})
	require.NoError(t, err)
	i := 0
	d.HTML().Walk(func(n *ui.Node) bool {
		if n.IsText() {
			return true
		}
		if n.Tag() == dom.Ol {
			assert.Equal(t, want[i]["list-style-type"], listStyleType(dom.Data(n).Style), "%s %d", n.Tag(), i)
		}
		i++
		return true
	})
}

// TestInsideMarkersStandFirstOnTheirLineAsChromes lays markers inside
// their items, the first box of the first line: bullets, numbers,
// strings, ::marker's content, pictures and gradients, in an empty item
// and on a line of their own before a block, and gradients outside.
func TestInsideMarkersStandFirstOnTheirLineAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
ul { margin: 0; padding: 0 0 0 40px; list-style-position: inside; }
.big { font-size: 22px; }
.c { list-style-type: circle; } .s { list-style-type: square; } .d { list-style: decimal inside; }
.str { list-style-type: "->"; }
.g { list-style-image: linear-gradient(red, blue); }
.gb { list-style-image: radial-gradient(red, blue); font-size: 30px; }
.pic { list-style-image: url(pic.png); }
.m::marker { content: "M"; }
.big-marker::marker { font-size: 26px; }
.o { list-style: outside linear-gradient(red, blue); }
.blk p { margin: 0; }
.narrow { width: 90px; }
`), ui.Images(fstest.MapFS{"pic.png": {Data: pngOf(12, 9)}}))
	li := func(class string, kids ...ui.Arg) *ui.Node {
		return ui.Li(append([]ui.Arg{ui.Class(strings.Fields(class)...)}, kids...)...)
	}
	x := func() ui.Arg { return ui.Text("item text") }
	d.Body().Append(ui.Ul(
		li("", x()), li("big", x()), li("c", x()), li("s", x()), li("d", x()), li("str", x()), li("g", x()),
		li("gb", x()), li("pic", x()), li("m", x()), li("big-marker d", x()), li(""), li("d"),
		li("blk", ui.P(x())), li("d blk", ui.Text("lead"), ui.P(x())), li("o", x()),
		li("d narrow", ui.Text("words that wrap past the first line")),
		li("", ui.Span(ui.Text("in a span"))),
	))
	r := chrome.Match(t, d, geom.V2(500, 700))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestReversedCountersAreDroppedAsChromes has the numbers stand in each
// item's ::before, as wide as Chrome writes them: Chrome 157 reads no
// reversed() (CSS.supports says false), so the declaration is dropped and
// the counter made where it is first incremented.
func TestReversedCountersAreDroppedAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 20px; }
ol, ul { margin: 0 0 4px; }
li::before { content: counter(list-item) " " counter(c) "|"; display: inline-block; }
.r { counter-reset: reversed(c); }
.r > li { counter-increment: c -1; }
.r2 { counter-reset: reversed(c) 7; }
.r2 > li { counter-increment: c -2; }
.set { counter-reset: reversed(c); }
.set > li { counter-increment: c -1; }
.set > .to { counter-set: c 10; }
`))
	items := func(n int, args ...ui.Arg) []ui.Arg {
		for range n {
			args = append(args, ui.Li(ui.Text("x")))
		}
		return args
	}
	d.Body().Append(
		ui.Ol(items(3, ui.Attr("reversed", ""), ui.Attr("start", "10"))...),
		ui.Ul(items(3, ui.Class("r"))...),
		ui.Ul(items(3, ui.Class("r2"))...),
		ui.Ul(append(items(2, ui.Class("set")), ui.Li(ui.Class("to"), ui.Text("x")), ui.Li(ui.Text("x")))...),
	)
	r := chrome.Match(t, d, geom.V2(600, 800))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	assert.Len(t, d.Warnings(), 3, "each reversed() is refused")
}

// TestReversedListItemCountersAsChromes writes counter(list-item) in
// items' ::before and in some ::marker's content, beside the markers
// Chrome numbers apart from the counter: a reversed ol's markers count
// down from its start, else from its number of items; an item's value
// attribute numbers its marker alone, and a list or an item whose own
// counter properties name list-item gets nothing implicit from the
// counter. No item writing the counter has a value attribute, and every
// reversed ol whose items write it has a start: Chrome resets the counter
// of one with none to 1 and leaves it be at a value when it first counts,
// as ui does, but to one over its number of items, and to the value in an
// item's own scope, when it counts again, as it does at times while the
// page loads. ui's first count of those is unit tested (css's
// TestReversedListsCountTheirMarkersApart).
func TestReversedListItemCountersAsChromes(t *testing.T) {
	d := listItemCounters()
	r := chrome.Match(t, d, geom.V2(700, 900))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// listItemCounters is TestReversedListItemCountersAsChromes's document,
// its numbers written in roman and in letters as well, for their widths
// to tell them apart.
func listItemCounters() *ui.Document {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 14px; }
ol, ul { margin: 0 0 4px; padding: 0; display: flex; gap: 12px; list-style: upper-roman inside; }
li::before {
  content: "[" counter(list-item) counter(list-item, upper-roman) counter(list-item, lower-alpha) "]";
  display: inline-block;
}
.m::marker { content: counter(list-item, upper-roman) ") "; }
.bare li::before { content: none; }
`))
	li := func(args ...ui.Arg) *ui.Node { return ui.Li(append(args, ui.Text("x"))...) }
	items := func(n int, args ...ui.Arg) []ui.Arg {
		for range n {
			args = append(args, li())
		}
		return args
	}
	style := ui.Style
	rev := func(start string, args ...ui.Arg) *ui.Node {
		return ui.Ol(append([]ui.Arg{ui.Attr("reversed", ""), ui.Attr("start", start)}, args...)...)
	}
	d.Body().Append(
		ui.Ol(items(4, ui.Class("bare"), ui.Attr("reversed", ""))...),
		ui.Ol(ui.Class("bare"), ui.Attr("reversed", ""), li(style("counter-increment: list-item 0")), li(),
			li(style("display: none")), ui.Div(li()), li(ui.Attr("value", "9")), li()),
		rev("7", items(4)...),
		rev("0", items(2)...),
		rev("-3", items(2)...),
		rev("3", li(ui.Class("m")), li(ui.Class("m")), li(ui.Class("m"))),
		rev("3", ui.Class("bare"), li(), li(ui.Attr("value", "7")), li()),
		ui.Ol(ui.Class("bare"), li(), li(ui.Attr("value", "7")), li()),
		ui.Ol(ui.Class("bare"), ui.Attr("start", "3"), li(), li(ui.Attr("value", "0")), li()),
		rev("4", li(style("counter-increment: list-item 0")), li(), li()),
		rev("4", li(style("counter-increment: none")), li(), li()),
		rev("5", li(style("counter-increment: list-item -2")), li(), li()),
		ui.Ol(ui.Attr("start", "5"), li(style("counter-increment: list-item 3")), li(), li()),
		rev("3", li(), li(style("counter-set: list-item 6")), li()),
		ui.Ol(ui.Class("bare"), li(), li(ui.Attr("value", "9"), style("counter-set: list-item 3")), li()),
		rev("3", ui.Class("bare"), li(), li(ui.Attr("value", "5"), style("counter-increment: list-item 3")), li()),
		rev("3", li(), li(style("counter-increment: list-item 2; counter-set: list-item 9")), li()),
		rev("3", li(), li(style("counter-reset: list-item 4; counter-increment: list-item 2")), li()),
		ui.Ol(append(items(5, ui.Attr("reversed", "")), style("counter-reset: list-item 3"))...),
		rev("6", ui.Div(li()), li(), li(style("display: none")), li()),
		rev("6", li(), li(), ui.Ol(li(), li()), li()),
		rev("6", li(), li(), ui.Div(style("counter-reset: list-item 20")), li()),
		ui.Ul(items(2, ui.Attr("reversed", ""))...),
	)
	return d
}

// TestInsideMarkersOfRightToLeftItemsAsChromes lays markers inside
// right-to-left items: at the right end of their first line, bullets,
// numbers, strings, ::marker's content and pictures, before a block and
// beside text whose bidi reorders it.
func TestInsideMarkersOfRightToLeftItemsAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
ul, ol { margin: 0 0 6px; padding: 0 40px 0 0; list-style-position: inside; }
.d { list-style-type: decimal; } .c { list-style-type: circle; } .str { list-style-type: "->"; }
.m::marker { content: "M: "; }
.pic { list-style-image: url(pic.png); }
.blk p { margin: 0; }
.narrow { width: 120px; }
.center { text-align: center; } .start { text-align: left; }
`), ui.Images(fstest.MapFS{"pic.png": {Data: pngOf(12, 9)}}))
	li := func(class string, kids ...ui.Arg) *ui.Node {
		return ui.Li(append([]ui.Arg{ui.Class(strings.Fields(class)...)}, kids...)...)
	}
	x := func() ui.Arg { return ui.Span(ui.Text("item text")) }
	d.Body().Append(
		ui.Ul(ui.Attr("dir", "rtl"),
			li("", x()), li("c", x()), li("d", x()), li("str", x()), li("m", x()), li("pic", x()),
			li("", ui.Span(ui.Text("(item) text!"))), li("d", ui.Span(ui.Text("12 item, text"))), li(""), li("d"),
			li("blk", ui.P(x())), li("d blk", ui.Span(ui.Text("lead")), ui.P(x())),
			li("d narrow", ui.Span(ui.Text("words that wrap past the first line"))),
			li("d center", x()), li("d start", x()),
		),
		ui.Ol(li("", ui.Attr("dir", "rtl"), x()), li("", ui.Style("direction: rtl"), ui.Text("item"))),
	)
	r := chrome.Match(t, d, geom.V2(500, 800))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestMarkersTakeThePointerAsChromes asks ui and Chrome for the element
// at each point of a grid round each item's first line: an outside
// ::marker is its item's to the pointer over its text's box, a bullet's
// as the bullet and the space after it, the line's height, a picture's
// over the picture.
func TestMarkersTakeThePointerAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
ul, ol { margin: 0 0 6px; padding: 0 60px; }
li { margin-bottom: 4px; }
.c { list-style-type: circle; } .s { list-style-type: square; } .d { list-style-type: decimal; }
.str { list-style-type: "->"; } .m::marker { content: "Mm "; }
.big { font-size: 26px; } .tall { line-height: 34px; } .small::marker { font-size: 9px; }
.pic { list-style-image: url(pic.png); }
.rtl { direction: rtl; }
`), ui.Images(fstest.MapFS{"pic.png": {Data: pngOf(12, 9)}}))
	var items []*ui.Node
	li := func(class string) *ui.Node {
		n := ui.Li(ui.ID(fmt.Sprintf("i%d", len(items))), ui.Class(strings.Fields(class)...), ui.Text("item"))
		items = append(items, n)
		return n
	}
	d.Body().Append(
		ui.Ul(ui.ID("u"), li(""), li("c"), li("s"), li("big"), li("tall"), li("small"), li("str"), li("m"), li("pic")),
		ui.Ol(ui.ID("o"), li("d"), li("d big")),
		ui.Ul(ui.ID("r"), ui.Class("rtl"), li(""), li("big"), li("d"), li("pic"), li("str"), li("tall")),
	)
	size := geom.V2(400, 700)
	d.Layout(size)
	var points [][2]float32
	for _, n := range items {
		b := n.Box()
		from, to := b.X-50, b.X+2
		if strings.Contains(n.Parent().ID(), "r") {
			from, to = b.X+b.W-2, b.X+b.W+50
		}
		for y := b.Y - 1.5; y < b.Y+40; y++ {
			for x := from + 0.5; x < to; x++ {
				points = append(points, [2]float32{x, y})
			}
		}
	}
	js, err := json.Marshal(points)
	require.NoError(t, err)
	out, err := chrome.Eval(d, size,
		`return `+string(js)+`.map(([x, y]) => {
const e = document.elementFromPoint(Math.floor(x), Math.floor(y));
return e ? e.id : "";
});`)
	require.NoError(t, err)
	var want []string
	require.NoError(t, json.Unmarshal(out, &want))
	require.Len(t, want, len(points))
	// each item's marker as the points outside its box that find it span
	type span struct{ ember, chrome geom.Rect }
	spans := map[string]*span{}
	grow := func(r *geom.Rect, p [2]float32) {
		q := geom.Rect{X: p[0] - 0.5, Y: p[1] - 0.5, W: 1, H: 1}
		if r.W == 0 {
			*r = q
			return
		}
		x0, y0 := min(r.X, q.X), min(r.Y, q.Y)
		*r = geom.Rect{X: x0, Y: y0, W: max(r.X+r.W, q.X+q.W) - x0, H: max(r.Y+r.H, q.Y+q.H) - y0}
	}
	for i, p := range points {
		got := ""
		if n := d.ElementAt(geom.V2(p[0], p[1])); n != nil {
			got = n.ID()
		}
		for _, id := range []string{got, want[i]} {
			if id == "" || id[0] != 'i' || spans[id] != nil {
				continue
			}
			spans[id] = &span{}
		}
		inside := func(id string) bool {
			var b geom.Rect
			for _, n := range items {
				if n.ID() == id {
					b = n.Box()
				}
			}
			return p[0] >= b.X && p[0] < b.X+b.W && p[1] >= b.Y && p[1] < b.Y+b.H
		}
		if got != "" && got[0] == 'i' && !inside(got) {
			grow(&spans[got].ember, p)
		}
		if want[i] != "" && want[i][0] == 'i' && !inside(want[i]) {
			grow(&spans[want[i]].chrome, p)
		}
	}
	for _, n := range items {
		s := spans[n.ID()]
		if s == nil {
			t.Errorf("%s: no marker takes the pointer", n.ID())
			continue
		}
		e, c := s.ember, s.chrome
		if max(math.Abs(float64(e.X-c.X)), math.Abs(float64(e.Y-c.Y)), math.Abs(float64(e.X+e.W-c.X-c.W)),
			math.Abs(float64(e.Y+e.H-c.Y-c.H))) > 1 {
			t.Errorf("%s %v: ember's marker takes the pointer over %v, Chrome's over %v", n.ID(), n.Classes(),
				s.ember, s.chrome)
		}
	}
}

// TestListTypeAttributesStyleAsChromes compares the list-style-type the
// type attributes of ol, ul and li give with getComputedStyle's.
func TestListTypeAttributesStyleAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(".greek { list-style-type: lower-greek; }"))
	li := func(args ...ui.Arg) *ui.Node { return ui.Li(append(args, ui.Text("x"))...) }
	d.Body().Append(
		ui.Ol(ui.Attr("type", "a"), li(), li(ui.Attr("type", "I")), li(ui.Attr("type", "disc"))),
		ui.Ol(ui.Attr("type", "A"), li(ui.Attr("type", "i")), li(ui.Attr("type", "1"))),
		ui.Ol(ui.Attr("type", "i"), li(ui.Attr("type", "A"))),
		ui.Ol(ui.Attr("type", "I"), li(ui.Attr("type", "Circle"))),
		ui.Ol(ui.Attr("type", "1"), ui.Class("greek"), li()),
		ui.Ol(ui.Attr("type", "x"), li(ui.Attr("type", "SQUARE")), li(ui.Attr("type", "none"))),
		ui.Ol(ui.Attr("type", "disc"), li(ui.Attr("type", ""))),
		ui.Ul(ui.Attr("type", "SQUARE"), li(ui.Attr("type", "a")), li(ui.Attr("type", "circle"))),
		ui.Ul(ui.Attr("type", "circle"), li(ui.Attr("type", "disc"))),
		ui.Ul(ui.Attr("type", "none"), li(), ui.Ul(ui.Attr("type", "a"), li())),
		ui.Ul(ui.Attr("type", "1"), li()),
	)
	props := []string{"list-style-type"}
	want, err := chrome.Styles(d, geom.V2(400, 800), props)
	require.NoError(t, err)
	d.Layout(geom.V2(400, 800))
	i := 0
	d.HTML().Walk(func(n *ui.Node) bool {
		if n.IsText() {
			return true
		}
		if i < len(want) {
			assert.Equal(t, want[i]["list-style-type"], listStyleType(dom.Data(n).Style), "%s %d", n.Tag(), i)
		}
		i++
		return true
	})
	assert.Len(t, want, i)
}

// listStyleType is st's list-style-type as getComputedStyle writes it.
func listStyleType(st *style.Computed) string {
	switch st.ListStyle {
	case style.ListCounter:
		return st.ListName
	case style.ListString:
		return strconv.Quote(st.ListName)
	case style.ListDisc, style.ListNone, style.ListCircle, style.ListSquare, style.ListDisclosureClosed,
		style.ListDisclosureOpen:
	}
	return []string{"disc", "none", "disclosure-closed", "disclosure-open", "circle", "square"}[st.ListStyle]
}

// TestContentURLReplacesAnElementAsChromes has content: url() alone
// replace an element by its picture, sized as an img; content of a string,
// or of a url() and more, leaves an element as it is.
func TestContentURLReplacesAnElementAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
.pic { content: url(pic.png); }
.framed { border: 3px solid; padding: 2px; }
.wide { width: 90px; }
.tall { display: block; height: 60px; }
.text { content: "text"; }
.both { content: url(pic.png) "and"; }
`), ui.Images(fstest.MapFS{"pic.png": {Data: pngOf(30, 20)}}))
	d.Body().Append(
		ui.Div(ui.Class("pic"), ui.Text("gone"), ui.Span(ui.Text("inside"))),
		ui.Div(ui.Class("pic", "framed"), ui.Text("gone")),
		ui.P(ui.Text("a "), ui.Span(ui.Class("pic"), ui.Text("word")), ui.Text(" in a line")),
		ui.Div(ui.Class("pic", "wide"), ui.Text("gone")),
		ui.Span(ui.Class("pic", "tall"), ui.Text("gone")),
		ui.Div(ui.Class("text"), ui.Text("kept")),
		ui.Div(ui.Class("both"), ui.Text("kept")),
		ui.Img(ui.Class("pic"), ui.Attr("width", "50")),
	)
	r := chrome.Match(t, d, geom.V2(500, 500))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestCheckboxesAndRadiosTakeBeforeAndAfterAsChromes gives a checkbox
// and a radio inline ::before and ::after boxes, laid out in their box and
// past it as in Chrome; a range, a progress bar, a meter and a text field
// take none. The text field has a width: left to itself it is as wide as
// the machine's system font makes Chrome's.
func TestCheckboxesAndRadiosTakeBeforeAndAfterAsChromes(t *testing.T) {
	for _, display := range []string{"inline", "inline-block; width: 10px; height: 6px"} {
		t.Run(display, func(t *testing.T) {
			d := ui.NewDocument(ui.CSS(`
body { margin: 8px; font-size: 16px; }
p { margin: 0 0 10px; }
input[type=text] { width: 120px; }
input::before { content: "B"; display: ` + display + `; }
input::after { content: "A"; display: ` + display + `; }
input:checked::after { content: "checked"; }
`))
			for _, c := range []*ui.Node{
				ui.Input(ui.Attr("type", "checkbox")), ui.Input(ui.Attr("type", "radio")),
				ui.Input(ui.Attr("type", "checkbox"), ui.Attr("checked", "")), ui.Input(ui.Attr("type", "text")),
			} {
				d.Body().Append(ui.P(ui.Text("x "), c, ui.Text(" y")))
			}
			r := chrome.Match(t, d, geom.V2(500, 300))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
