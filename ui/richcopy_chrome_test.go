//go:build chrome

package ui_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const richCopySheet = `body { user-select: text; } .r { color: red; } .u { text-decoration: underline; }
.bg { background: #ffe; padding: 4px; } .big { font-size: 150%; letter-spacing: 1px; } .pw { white-space: pre-wrap; }
.up { text-transform: uppercase; } .mid { text-align: center; font-weight: 600; } .link { color: #0645ad; }`

// richCopyCase is a page and the selection a copy of it writes, from the
// character from of the text node holding at to the character to of the
// one holding till.
type richCopyCase struct {
	name      string
	body      []*ui.Node
	at        string
	from      int
	till      string
	to        int
	selectAll bool
}

//nolint:gochecknoglobals // the pages compared, a test table
var richCopyCases = []richCopyCase{
	{name: "across paragraphs", body: []*ui.Node{ui.P(ui.Text("First para here")), ui.P(ui.Text("Second para here"))},
		at: "First para here", from: 6, till: "Second para here", to: 6},
	{name: "in a bold", body: []*ui.Node{ui.P(ui.Text("Some "), ui.B(ui.Text("bold text")), ui.Text(" here"))},
		at: "bold text", from: 1, till: "bold text", to: 6},
	{name: "in a link", body: []*ui.Node{ui.P(ui.Text("Some "),
		ui.A(ui.Class("link"), ui.Attr("href", "/l"), ui.Text("link text")), ui.Text(" here"))},
		at: "link text", from: 1, till: "link text", to: 6},
	{name: "through elements", body: []*ui.Node{ui.P(ui.Text("Some "), ui.B(ui.Text("bold")), ui.Text(" text "),
		ui.A(ui.Class("link"), ui.Attr("href", "/x"), ui.Text("link")), ui.Span(ui.Class("r"), ui.Text(" red")))},
		at: "Some ", from: 2, till: " red", to: 2},
	{name: "into an italic in a colour", body: []*ui.Node{ui.P(ui.Text("Some "),
		ui.Span(ui.Class("r"), ui.Text("red "), ui.I(ui.Text("it")), ui.Text(" text")), ui.Text(" here"))},
		at: "red ", from: 1, till: "it", to: 1},
	{name: "a whole item", body: []*ui.Node{ui.Ul(ui.Li(ui.Text("one item")), ui.Li(ui.Text("two item")))},
		at: "one item", from: 0, till: "one item", to: 8},
	{name: "across items", body: []*ui.Node{ui.Ul(ui.Li(ui.Text("one item")), ui.Li(ui.Text("two item")))},
		at: "one item", from: 2, till: "two item", to: 3},
	{name: "part of an item", body: []*ui.Node{ui.Ul(ui.Li(ui.Text("one item")))},
		at: "one item", from: 1, till: "one item", to: 5},
	{name: "a nested item", body: []*ui.Node{ui.Ul(ui.Li(ui.Text("one")), ui.Li(ui.Ul(ui.Li(ui.Text("nested")))))},
		at: "nested", from: 0, till: "nested", to: 6},
	{name: "across cells", body: []*ui.Node{ui.Table(ui.Tbody(
		ui.Tr(ui.Td(ui.Text("cell one")), ui.Td(ui.Text("cell two"))), ui.Tr(ui.Td(ui.Text("c3")), ui.Td(ui.Text("c4")))))},
		at: "cell one", from: 2, till: "c4", to: 1},
	{name: "a whole cell", body: []*ui.Node{ui.Table(ui.Tbody(ui.Tr(ui.Td(ui.Text("cell one")),
		ui.Td(ui.Text("cell two")))))},
		at: "cell one", from: 0, till: "cell one", to: 8},
	{name: "round a picture", body: []*ui.Node{ui.P(ui.Text("pic "),
		ui.Img(ui.Attr("alt", "x"), ui.Attr("width", "20"), ui.Attr("height", "20")), ui.Text(" after"))},
		at: "pic ", from: 1, till: " after", to: 3},
	{name: "in a pre", body: []*ui.Node{ui.Pre(ui.Text("pre  text\n  two"))},
		at: "pre  text\n  two", from: 1, till: "pre  text\n  two", to: 12},
	{name: "collapsed spaces", body: []*ui.Node{ui.P(ui.Text("a   lot    of  spaces here"))},
		at: "a   lot    of  spaces here", from: 0, till: "a   lot    of  spaces here", to: 20},
	{name: "under a line", body: []*ui.Node{ui.Div(ui.Class("u"), ui.Text("under "), ui.Span(ui.Text("lined words")),
		ui.Text(" x"))}, at: "lined words", from: 0, till: "lined words", to: 6},
	{name: "on a background", body: []*ui.Node{ui.Div(ui.Class("bg"), ui.P(ui.Text("with background")))},
		at: "with background", from: 2, till: "with background", to: 9},
	{name: "a heading into a paragraph", body: []*ui.Node{ui.H1(ui.Text("Heading words")), ui.P(ui.Text("and text"))},
		at: "Heading words", from: 3, till: "and text", to: 3},
	{name: "from a paragraph's end", body: []*ui.Node{ui.P(ui.Text("end of para")), ui.P(ui.Text("next para"))},
		at: "end of para", from: 11, till: "next para", to: 4},
	{name: "to a paragraph's start", body: []*ui.Node{ui.P(ui.Text("first para")), ui.P(ui.Text("second"))},
		at: "first para", from: 0, till: "second", to: 0},
	{name: "a percentage", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Span(ui.Class("big"), ui.Text("big")),
		ui.Text(" post"))}, at: "pre ", from: 1, till: " post", to: 3},
	{name: "a style attribute", body: []*ui.Node{ui.P(ui.Text("pre "),
		ui.Span(ui.Style("color: blue; margin: 0"), ui.Text("styled")), ui.Text(" post"))},
		at: "pre ", from: 1, till: " post", to: 3},
	{name: "cased as written", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Span(ui.Class("up"), ui.Text("shout")),
		ui.Text(" post"))}, at: "pre ", from: 1, till: " post", to: 3},
	{name: "kept spaces", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Span(ui.Class("pw"), ui.Text("a  b   c")),
		ui.Text(" post"))}, at: "a  b   c", from: 0, till: " post", to: 3},
	{name: "a space at each end", body: []*ui.Node{ui.P(ui.Text("x lined words y"))},
		at: "x lined words y", from: 1, till: "x lined words y", to: 8},
	{name: "across a br", body: []*ui.Node{ui.P(ui.Text("line one"), ui.Br(), ui.Text("line two"))},
		at: "line one", from: 5, till: "line two", to: 4},
	{name: "in a code", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Code(ui.Text("some code")), ui.Text(" post"))},
		at: "some code", from: 1, till: "some code", to: 6},
	{name: "into a code", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Code(ui.Text("some code")), ui.Text(" post"))},
		at: "pre ", from: 1, till: "some code", to: 6},
	{name: "out of a div", body: []*ui.Node{ui.Div(ui.P(ui.Text("in div"))), ui.P(ui.Text("out"))},
		at: "in div", from: 1, till: "out", to: 2},
	{name: "centred and heavy", body: []*ui.Node{ui.P(ui.Class("mid"), ui.Text("centered"))},
		at: "centered", from: 1, till: "centered", to: 6},
	{name: "deep in bold and italic", body: []*ui.Node{ui.Div(ui.Class("r"), ui.B(ui.I(ui.Text("deep words"))))},
		at: "deep words", from: 1, till: "deep words", to: 6},
	{name: "in an underline", body: []*ui.Node{ui.P(ui.Text("x "), ui.U(ui.Text("under it")), ui.Text(" y"))},
		at: "under it", from: 1, till: "under it", to: 6},
	{name: "in a bold link", body: []*ui.Node{ui.P(ui.A(ui.Class("link"), ui.Attr("href", "/b"),
		ui.B(ui.Text("bold link"))))}, at: "bold link", from: 1, till: "bold link", to: 6},
	{name: "into a link", body: []*ui.Node{ui.P(ui.Text("pre "),
		ui.A(ui.Class("link"), ui.Attr("href", "/l"), ui.Text("link")), ui.Text(" post"))},
		at: "pre ", from: 1, till: "link", to: 2},
	{name: "a no-break space", body: []*ui.Node{ui.P(ui.Text("a\u00a0b c"))},
		at: "a\u00a0b c", from: 0, till: "a\u00a0b c", to: 5},
	{name: "across a small", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Small(ui.Text("tiny")), ui.Text(" post"))},
		at: "pre ", from: 1, till: " post", to: 3},
	{name: "across a mark", body: []*ui.Node{ui.P(ui.Text("pre "), ui.Mark(ui.Text("marked")), ui.Text(" post"))},
		at: "pre ", from: 1, till: " post", to: 3},
	{name: "across a strike-through", body: []*ui.Node{ui.P(ui.Text("pre "), ui.S(ui.Text("struck")),
		ui.Text(" post"))}, at: "pre ", from: 1, till: " post", to: 3},
	{name: "an important style", body: []*ui.Node{ui.P(ui.Text("pre "),
		ui.Span(ui.Style("color: blue !important; padding: 1px 2px"), ui.Text("loud")), ui.Text(" post"))},
		at: "pre ", from: 1, till: " post", to: 3},
	{name: "the whole page", body: []*ui.Node{ui.P(ui.Text("whole para")), ui.P(ui.Text("next"))}, selectAll: true},
}

// TestARichCopyIsChromes compares the markup a copy writes with the one
// Chrome's copy writes of the same selection, which a drag of it
// carries, the attributes the export adds for Chrome left out and its
// links read against the page's address.
func TestARichCopyIsChromes(t *testing.T) {
	ours := regexp.MustCompile(` data-ember-[a-z-]+="[^"]*"`)
	for _, c := range richCopyCases {
		t.Run(c.name, func(t *testing.T) {
			d := ui.NewDocument(ui.CSS(richCopySheet))
			d.Body().Append(c.body...)
			d.Layout(geom.V2(400, 400))
			if c.selectAll {
				d.SelectAll()
			} else {
				d.Select(ui.Position{Node: textNode(d, c.at), Offset: c.from},
					ui.Position{Node: textNode(d, c.till), Offset: c.to})
			}
			srv, _ := chrome.Serve(t, d, nil, 0)
			s := chrome.Open(t, srv.URL, geom.V2(400, 400), 1)
			want := strings.ReplaceAll(ours.ReplaceAllString(s.CopiedHTML(), ""), srv.URL, "")
			assert.Equal(t, want, d.SelectionHTML())
		})
	}
}

// textNode is d's text node of text s.
func textNode(d *ui.Document, s string) *ui.Node {
	var found *ui.Node
	d.Body().Walk(func(n *ui.Node) bool {
		if found == nil && n.IsText() && n.Text() == s {
			found = n
		}
		return found == nil
	})
	return found
}
