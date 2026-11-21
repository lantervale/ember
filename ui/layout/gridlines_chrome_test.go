//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestGridLineNamesLayOutAsInChrome(t *testing.T) {
	grid := func(classes ...string) *ui.Node {
		kids := []ui.Arg{ui.Class("page")}
		for _, c := range classes {
			kids = append(kids, ui.Div(ui.Class(c), ui.Text(c)))
		}
		return ui.Div(kids...)
	}
	const page = `.page { display: grid; position: relative; width: 600px; grid-auto-rows: 20px; grid-auto-columns: 30px }
.page > * { min-width: 0 }`
	cases := map[string]*ui.Document{
		"a full-bleed page": styled(page+`
.page { grid-template-columns: [full-start] minmax(1rem, 1fr) [content-start] minmax(0, 20rem) [content-end] minmax(1rem, 1fr) [full-end] }
.page > * { grid-column: content } .page > .full { grid-column: full }`,
			grid("a", "full", "b", "c")),
		"names in repeat(), by number and by span": styled(page+`
.page { grid-template-columns: [a] 50px repeat(3, [col] 40px [gap]) [b] 60px [end] }
.i { grid-column: col 2 } .j { grid-column: col -1 / span gap 2 } .k { grid-column: span col 2 / b }
.l { grid-column: 2 / span col } .m { grid-column: a / gap 2 } .n { grid-column: gap / end }
.o { grid-column: -1 col } .p { grid-column: end -1 / a } .q { grid-column: col 5 } .r { grid-column: -5 col / span 2 }
.s { grid-column: span gap 3 / 3 } .t { grid-column: span 2 / span col } .u { grid-column: gap 2 / gap 2 }`,
			grid("i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u")),
		"names in auto-fill and auto-fit repeats": styled(page+`
.page { grid-template-columns: [first] repeat(auto-fill, [c] 80px [d]) [last] 40px }
.fit { grid-template-columns: [first] repeat(auto-fit, [c] 80px) [last]; margin-top: 10px }
.i { grid-column: c 3 } .j { grid-column: last } .k { grid-column: c -1 / last } .l { grid-column: d 2 / span c 2 }
.m { grid-column: first / c 2 }`,
			grid("i", "j", "k", "l", "m"), ui.Div(ui.Class("page", "fit"), ui.Div(ui.Class("j"), ui.Text("j")),
				ui.Div(ui.Class("i"), ui.Text("i")))),
		"the lines of named areas": styled(page+`
.page { grid-template-columns: 100px 1fr 80px; grid-template-areas: 'head head .' 'side main main' }
.i { grid-column: head-start / main-end } .j { grid-row: main-start; grid-column: side-end }
.k { grid-column: main / head-end; grid-row: head } .l { grid-area: side-start / main-start / side-end / main }`,
			grid("i", "j", "k", "l")),
		"areas from line names": styled(page+`
.page { grid-template-columns: 40px [content-start] 100px 1fr [content-end] 50px;
	grid-template-rows: [content-start] 30px [content-end] 20px [aside-start] 20px [aside-end] }
.i { grid-area: content } .j { grid-area: aside / content } .k { grid-row: content; grid-column: 1 }`,
			grid("i", "j", "k")),
		"lines named by nothing": styled(page+`
.page { grid-template-columns: 100px 100px; grid-template-areas: 'a b c' }
.i { grid-column-end: nope; grid-row: 2 } .j { grid-area: nowhere } .k { grid-column: nope 2; grid-row: 3 }
.l { grid-column: span nope / 2; grid-row: 4 } .m { grid-row: x -2; grid-column: 1 } .n { grid-column: 1 / span nope 2; grid-row: 5 }
.o { grid-column: -1; grid-row: 6 } .p { grid-column: c; grid-row: 7 }`,
			grid("i", "j", "k", "l", "m", "n", "o", "p")),
		"the shorthand with names": styled(page+`
.page { grid-template: [top] 'a a' 40px [mid] 'b c' 30px [bottom] / [l] 100px [m] 1fr [r] }
.i { grid-row: mid / bottom; grid-column: m / r } .j { grid-row: top; grid-column: l } .k { grid-area: a }
.n { grid-template: [r1] 30px [r2] 40px [r3] / [c1] 100px [c2] }
.n > .i { grid-area: r2 / c1 / r3 / c2 }`,
			grid("i", "j", "k"), ui.Div(ui.Class("page", "n"), ui.Div(ui.Class("i"), ui.Text("i")))),
		"absolute boxes by name": styled(page+`
.page { grid-template-columns: [a] 100px [b] 200px [c]; grid-template-rows: [top] 40px [bottom] 40px }
.page > * { position: absolute; inset: 0 }
.i { grid-column: b / c; grid-row: top / bottom } .j { grid-column: b / nope; grid-row: bottom }
.k { grid-column: span b / c } .l { grid-column: nope / b; grid-row: nope }`,
			grid("i", "j", "k", "l")),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 900))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}
