//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// matchPages lays each page out in ui and in Chrome, a WPT test beside its reference.
func matchPages(t *testing.T, size geom.Vec2, pages map[string]*ui.Document) {
	t.Helper()
	for name, d := range pages {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, size)
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// WPT css/css-grid/abspos/positioned-grid-items-019.html: a fixed box in an inline grid,
// beside the reference's empty grid of the same tracks.
func TestWPTPositionedGridItems019MatchesChrome(t *testing.T) {
	const sheet = `#grid { display: inline-grid; grid: 50px 50px / 50px 50px; padding: 5px; border: 5px solid;
background-color: gray; margin: 50px }
#fixed { width: 50px; height: 50px; background-color: blue; position: fixed; top: 0px; left: 0px }`
	p := func() *ui.Node { return ui.P(ui.Text("The test passes if it has the same output than the reference.")) }
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": wptDoc(sheet, p(), ui.Div(ui.ID("grid"), ui.Div(ui.ID("fixed")))),
		"ref":  wptDoc(sheet, p(), ui.Div(ui.ID("fixed")), ui.Div(ui.ID("grid"))),
	})
}

// WPT css/css-grid/abspos/positioned-grid-items-025.html: positioned items of a right to
// left grid, placed from its padding edges by auto lines.
func TestWPTPositionedGridItems025MatchesChrome(t *testing.T) {
	const sheet = `#grid { display: grid; grid-template-columns: 50px 100px; height: 100px; width: 150px;
direction: rtl; padding-left: 50px; padding-right: 80px; position: absolute }
#grid > div { top: 0px; right: 0px; bottom: 0px; left: 0px; position: absolute; background-color: blue }`
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": wptDoc(sheet, ui.Div(ui.ID("grid"),
			ui.Div(ui.Style("grid-column: auto / 1")), ui.Div(ui.Style("grid-column: -1 / auto")))),
	})
}

// The line names of WPT css/css-grid/grid-definition/grid-auto-repeat-multiple-values-004.html
// and -005.html: an item starting at each.
//
//nolint:gochecknoglobals // the WPT pages' list
var autoRepeatStarts = []string{"u", "nonesuch", "u 1", "u -1", "u 4", "u 9", "u 2", "u -2", "u -6", "u -7",
	"v", "v 1", "v 2", "v -1", "v -2", "v 3", "v -3", "w", "w 1", "w 2", "w -1", "w -2", "w 3", "w -3",
	"x", "x 1", "x 2", "x -1", "x -2", "x 3", "x -3", "y", "y 1", "y 2", "y -1", "y -2", "y -3", "y 3",
	"z", "z 1", "z -1", "z 2", "z -2"}

func autoRepeatPage(sheet, class, prop string) *ui.Document {
	var nodes []*ui.Node
	for _, s := range autoRepeatStarts {
		nodes = append(nodes, ui.Div(ui.Class(class), ui.Div(ui.Style(prop+": "+s))))
	}
	return wptDoc(sheet, nodes...)
}

// WPT css/css-grid/grid-definition/grid-auto-repeat-multiple-values-004.html and -005.html:
// items placed by the names of an auto-fill repeat of several tracks, and the reference's
// tracks written out.
func TestWPTGridAutoRepeatMultipleValuesMatchChrome(t *testing.T) {
	const columns = `.columns { border: 1px solid black; display: grid; grid-auto-rows: 5px; grid-column-gap: 3px;
width: 94px } div > div { background: blue }`
	const rows = `.rows { border: 1px solid black; display: inline-grid; grid-auto-flow: column;
grid-auto-columns: 5px; grid-row-gap: 3px; width: min-content; height: 94px } div > div { background: blue }`
	const repeat = `[u] repeat(auto-fill, [v] 10px [w] 10px [x] 10px [y]) [z]`
	const written = `[u v] 10px [w] 10px [x] 10px [y v] 10px [w] 10px [x] 10px [y z]`
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"004 test": autoRepeatPage(columns+`.columns { grid-template-columns: `+repeat+` }`, "columns", "grid-column-start"),
		"004 ref":  autoRepeatPage(columns+`.columns { grid-template-columns: `+written+` }`, "columns", "grid-column-start"),
		"005 test": autoRepeatPage(rows+`.rows { grid-template-rows: `+repeat+` }`, "rows", "grid-row-start"),
		"005 ref":  autoRepeatPage(rows+`.rows { grid-template-rows: `+written+` }`, "rows", "grid-row-start"),
	})
}

// WPT css/css-grid/grid-model/grid-empty-floating-positioned-and-inline-001.html: empty
// grids floated, positioned and inline, laid out as the empty blocks of its reference.
func TestWPTGridEmptyFloatingPositionedAndInlineMatchesChrome(t *testing.T) {
	page := func(grid, inline string) *ui.Document {
		return wptDoc(`span { width: 100px; height: 100px }
.grid { display: `+grid+`; background-color: green }
.inline-grid { display: `+inline+`; background-color: green }`,
			ui.P(ui.Text("In this paragraph "),
				ui.Span(ui.Class("grid"), ui.Style("float: left")),
				ui.Text(" there should be "), ui.Span(ui.Class("grid"), ui.Style("float: right")),
				ui.Text(" two empty grids.  One will float to the left and one will float to the right.  "+
					"They should both be empty.")),
			ui.Br(ui.Attr("clear", "all")),
			ui.P(ui.Text("This grid should be positioned to the right. "),
				ui.Span(ui.Class("grid"), ui.Style("position: absolute; right: 8px"))),
			ui.P(ui.Style("margin-top: 115px"), ui.Span(ui.Style("float: left; background-color: orange")),
				ui.Span(ui.Class("inline-grid")), ui.Text(" The green grid should be to the right of the floating orange box.")))
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": page("grid", "inline-grid"),
		"ref":  page("block", "inline-block"),
	})
}

// WPT css/css-flexbox/fieldset-baseline-alignment.html: a fieldset's baseline in a flex
// line is its content's, below the legend, as an inline-block's is in a line.
func TestWPTFieldsetBaselineAlignmentMatchesChrome(t *testing.T) {
	fieldset := func(style string) *ui.Node {
		return ui.Fieldset(ui.Style(style), ui.Legend(ui.Text("legend")), ui.Text("content"))
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"flex": wptDoc("", ui.Div(ui.Style("display: flex; align-items: baseline"),
			ui.Span(ui.Text("baseline")), fieldset(""))),
		"inline-block": wptDoc("", ui.Div(ui.Span(ui.Text("baseline")), fieldset("display: inline-block"))),
	})
}

// WPT css/css-flexbox/abspos/flex-abspos-staticpos-fallback-justify-content-001.html: the
// static position of a positioned child of a flex row under justify-content's
// distributions, falling back to their alignment where it overflows, rows of floats
// cleared by brs.
func TestWPTFlexAbsposStaticPositionFallbackMatchesChrome(t *testing.T) {
	const sheet = `.container { display: flex; flex-flow: row; padding: 1px 2px; border: 1px solid black;
background: yellow; margin-bottom: 5px; margin-right: 5px; float: left }
br { clear: both }
.big > .container { height: 10px; width: 16px }
.small > .container { height: 2px; width: 4px }
.container > * { position: absolute; background: teal; height: 6px; width: 8px }`
	row := func(class string) *ui.Node {
		args := []ui.Arg{ui.Class(class)}
		for _, j := range []string{"space-between", "space-around", "space-evenly", "stretch"} {
			args = append(args, ui.Div(ui.Class("container"), ui.Style("justify-content: "+j), ui.Div()))
		}
		args = append(args, ui.Br())
		if class == "big" {
			args = append(args, ui.Br())
		}
		return ui.Div(args...)
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{"test": wptDoc(sheet, row("big"), row("small"))})
}
