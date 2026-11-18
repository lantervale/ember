//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestGridAreasLayOutAsInChrome(t *testing.T) {
	named := func(classes ...string) []*ui.Node {
		var out []*ui.Node
		for _, c := range classes {
			out = append(out, ui.Div(ui.Class(c), ui.Text(c+" "+sentence)))
		}
		return out
	}
	const places = `.side { grid-area: side } .title { grid-area: title } .tools { grid-area: tools }
.text { grid-area: text } .end { grid-area: end } .foot { grid-area: foot }`
	cases := map[string]*ui.Document{
		"a page of named areas, its children in another order": styled(places+`
.page { display: grid; width: 700px; column-gap: 24px;
	grid-template: min-content 1fr min-content / 150px minmax(0, 1fr);
	grid-template-areas: 'notice notice' 'side main' 'foot foot' }
.main { grid-area: main; display: grid; grid-template: min-content min-content 1fr / minmax(0, 300px) min-content;
	grid-template-areas: 'title .' 'tools end' 'text end' }
.end { width: 90px }`,
			ui.Div(append([]ui.Arg{ui.Class("page")}, args(append(named("foot"),
				ui.Div(append([]ui.Arg{ui.Class("main")}, args(named("end", "text", "tools", "title"))...)...),
				named("side")[0]))...)...)),
		"the shorthand of strings and heights": styled(places+`
.page { display: grid; width: 500px; gap: 6px;
	grid-template: 'title title' 40px 'side text' 'foot foot' 30px / 120px 1fr }`,
			ui.Div(append([]ui.Arg{ui.Class("page")}, args(named("text", "foot", "title", "side"))...)...)),
		"lines by number and by name": styled(`
.page { display: grid; width: 400px; grid-template-columns: repeat(3, 1fr); grid-auto-rows: 30px;
	grid-template-areas: 'a a b' 'c . b' }
.a { grid-area: a } .b { grid-area: b } .c { grid-row: c; grid-column: c }
.d { grid-area: 3 / 1 / 4 / 4 } .e { grid-area: 4 / 2 } .g { grid-column-end: b; grid-row: 5 }`,
			ui.Div(append([]ui.Arg{ui.Class("page")}, args(named("g", "e", "d", "c", "b", "a"))...)...)),
		"an absolute box in an area": styled(`
.page { display: grid; position: relative; width: 400px; grid-template: 40px 60px / 100px 1fr;
	grid-template-areas: 'a b' 'c b' }
.b { grid-area: b; position: absolute; inset: 4px }`,
			ui.Div(ui.Class("page"), ui.Div(ui.Class("b")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 900))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}

func args(nodes []*ui.Node) []ui.Arg {
	out := make([]ui.Arg, len(nodes))
	for i, n := range nodes {
		out[i] = n
	}
	return out
}
