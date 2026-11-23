//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// containerSite is a page laid out by its containers: a shell that
// stacks its side under a width, cards named inside a named page, units
// of each container in lengths and in calc(), style queries, and
// containers that shrink to fit what they hold.
const containerSite = `
body { margin: 0; font-size: 14px; line-height: 20px; }
.page { container: page / inline-size; padding: 8px; border: 2px solid #335; }
.shell { display: flex; gap: 10px; }
.side { flex: 0 0 30%; container: side / inline-size; --tone: dark; }
.main { flex: 1; container-type: inline-size; }
@container page (width < 600px) { .shell { flex-direction: column; } .side { flex-basis: auto; } }
.card { container: card / size; height: 120px; margin-bottom: 6px; padding: 4px; border: 1px solid; }
.title { height: 20px; width: 50cqw; margin-left: calc(10cqi - 4px); }
@container card (width > 250px) { .title { height: 30px; width: 80cqw; } }
@container side (width > 200px) { .card .title { padding-left: 5cqh; } }
@container page (min-width: 700px) and (max-width: 950px) { .main .card { height: 90px; } }
@container (height >= 100px) and (orientation: landscape) { .body { height: 40cqb; } }
@container style(--tone: dark) { .body { border-top: 3px solid; } }
@container side style(not (--tone: light)) { .body { padding-top: 2px; } }
@container (aspect-ratio > 3) { .body { margin-top: 1cqmin; margin-bottom: 1cqmax; } }
.row { display: flex; }
.chip { container-type: inline-size; padding: 3px; border: 1px solid; }
.ib { display: inline-block; container-type: inline-size; padding: 0 5px; min-width: 30px; }
.fl { float: left; container-type: inline-size; padding: 2px; }
.grid { display: grid; grid-template-columns: auto 1fr; }
.grid > .ci { container-type: inline-size; }
.abs { position: absolute; top: 2px; right: 4px; container-type: inline-size; }
.free { width: 10cqw; height: 5cqh; }
`

func containerDoc() *ui.Document {
	d := ui.NewDocument(ui.CSS(containerSite))
	card := func(text string) *ui.Node {
		return ui.Div(ui.Class("card"), ui.Div(ui.Class("title"), ui.Text(text)), ui.Div(ui.Class("body")))
	}
	d.Body().Append(
		ui.Div(ui.Class("page"),
			ui.Div(ui.Class("shell"),
				ui.Div(ui.Class("side"), card("Side")),
				ui.Div(ui.Class("main"), card("First"), card("Second"))),
			ui.Div(ui.Class("row"), ui.Div(ui.Class("chip"), ui.Text("flex item")), ui.Div(ui.Text("after"))),
			ui.Div(ui.Div(ui.Class("ib"), ui.Text("an inline block")), ui.Div(ui.Class("fl"), ui.Text("a float"))),
			ui.Div(ui.Class("grid"), ui.Div(ui.Class("ci"), ui.Text("auto column")), ui.Div(ui.Text("fr"))),
			ui.Div(ui.Class("abs"), ui.Text("placed"))),
		ui.Div(ui.Class("free")),
	)
	return d
}

// TestContainerQueriesFollowAResizeAsChromes lays one Document out at a
// window's sizes in turn, as a resize does, against a Chrome opened at
// each.
func TestContainerQueriesFollowAResizeAsChromes(t *testing.T) {
	d := containerDoc()
	for _, size := range []geom.Vec2{{X: 1100, Y: 700}, {X: 800, Y: 600}, {X: 560, Y: 500}, {X: 900, Y: 400},
		{X: 520, Y: 800}, {X: 1100, Y: 700}} {
		t.Run(fmt.Sprintf("%gx%g", size.X, size.Y), func(t *testing.T) {
			r := chrome.Match(t, d, size)
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
	assert.Empty(t, d.Warnings())
}

// TestContainerValuesAreChromes compares the computed values the
// containers decide with getComputedStyle's.
func TestContainerValuesAreChromes(t *testing.T) {
	props := []string{"width", "height", "margin-left", "margin-top", "margin-bottom", "padding-left", "padding-top",
		"border-top-width", "container-type", "container-name", "flex-direction"}
	for _, size := range []geom.Vec2{{X: 1000, Y: 700}, {X: 500, Y: 600}} {
		t.Run(fmt.Sprintf("%gx%g", size.X, size.Y), func(t *testing.T) {
			d := containerDoc()
			want, err := chrome.Styles(d, size, props)
			require.NoError(t, err)
			d.Layout(size)
			var got []*ui.Node
			d.HTML().Walk(func(n *ui.Node) bool {
				if !n.IsText() {
					got = append(got, n)
				}
				return true
			})
			require.Len(t, got, len(want))
			for i, n := range got {
				for _, p := range props {
					ours, chromes := computed(n, p), want[i][p]
					if p == "width" || p == "height" {
						if ours == "" {
							continue
						}
					}
					if a, aOK := pixels(ours); aOK {
						b, bOK := pixels(chromes)
						assert.True(t, bOK && math.Abs(a-b) < 0.05, "%s.%v %s: ember %s, Chrome %s", n.Tag(),
							n.Classes(), p, ours, chromes)
						continue
					}
					assert.Equal(t, chromes, ours, "%s.%v %s", n.Tag(), n.Classes(), p)
				}
			}
		})
	}
}

// computed is n's computed value of p as getComputedStyle writes it;
// "" for a width or height left to layout.
func computed(n *ui.Node, p string) string {
	s := dom.Data(n).Style
	px := func(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) + "px" }
	switch p {
	case "width", "height":
		l := s.Size.Width
		if p == "height" {
			l = s.Size.Height
		}
		if l.Kind != style.Fixed || l.IsPercent {
			return ""
		}
		return px(l.Px)
	case "margin-left":
		return px(s.Margin.Left.Px)
	case "margin-top":
		return px(s.Margin.Top.Px)
	case "margin-bottom":
		return px(s.Margin.Bottom.Px)
	case "padding-left":
		return px(s.Padding.Left.Px)
	case "padding-top":
		return px(s.Padding.Top.Px)
	case "border-top-width":
		return px(s.Border.Top)
	case "container-type":
		return []string{"normal", "inline-size", "size"}[s.ContainerType]
	case "container-name":
		if len(s.ContainerNames) == 0 {
			return "none"
		}
		return strings.Join(s.ContainerNames, " ")
	case "flex-direction":
		return []string{"row", "row-reverse", "column", "column-reverse"}[s.Direction]
	}
	return ""
}

func pixels(s string) (float64, bool) {
	v, ok := strings.CutSuffix(s, "px")
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	return f, err == nil
}

// TestStyleQueriesOnStandardPropertiesNeverHoldAsInChromes has Chrome lay
// out style() queries of standard properties beside one of a custom
// property: Chrome reads them and holds none, as ui, which warns.
func TestStyleQueriesOnStandardPropertiesNeverHoldAsInChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
.c { color: rgb(255, 0, 0); display: block; padding-left: 4px; container-type: inline-size; --x: 1 }
.a { width: 10px; height: 10px }
@container style(--x: 1) { .a { padding-right: 17px } }
@container style(color: rgb(255, 0, 0)) { .a { width: 11px } }
@container style(display: block) { .a { margin-left: 13px } }
@container style(padding-left: 4px) { .a { margin-top: 14px } }
@container style(display) { .a { padding-top: 16px } }
@container style(--x: 1) and style(padding-left: 4px) { .a { margin-bottom: 15px } }
@container style(--x: 1) or style(padding-left: 4px) { .a { height: 12px } }
`))
	d.Body().Append(ui.Div(ui.Class("c"), ui.Div(ui.Class("a"))))
	chrome.Match(t, d, geom.V2(400, 300))
	assert.Len(t, d.Warnings(), 6)
}

// TestKeyframesInAContainerRuleHoldWhateverTheContainerAsChromes has
// Chrome read @keyframes inside @container rules, whose conditions it
// never asks: the last of a name wins wherever it stands. ui's sheet,
// written out, keeps no @container round them, so Chrome reads the sheet
// as written.
func TestKeyframesInAContainerRuleHoldWhateverTheContainerAsChromes(t *testing.T) {
	sheet := `
.c { container-type: inline-size; width: 300px }
.n { container-type: inline-size; width: 50px }
@container (width > 100px) { @keyframes k { from { left: 9px } to { left: 9px } } }
@container (width > 10000px) { @keyframes k2 { from { left: 99px } to { left: 99px } } }
@keyframes k3 { from { left: 1px } to { left: 1px } }
@container (width > 10000px) { @keyframes k3 { from { left: 2px } to { left: 2px } } }
@container (width > 100px) { @keyframes k4 { from { left: 3px } to { left: 3px } } }
@keyframes k4 { from { left: 4px } to { left: 4px } }
.a, .b, .e, .f { position: relative; height: 2px; animation: k 1000s }
.b { animation-name: k2 }
.e { animation-name: k3 }
.f { animation-name: k4 }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range []string{"c", "n"} {
		d.Body().Append(ui.Div(ui.Class(c), ui.Div(ui.Class("a")), ui.Div(ui.Class("b")), ui.Div(ui.Class("e")),
			ui.Div(ui.Class("f"))))
	}
	raw, err := chrome.Eval(ui.NewDocument(), geom.V2(400, 300), `
const st = document.createElement("style");
st.textContent = `+strconv.Quote(sheet)+`;
document.head.appendChild(st);
const parts = '<div class="a"></div><div class="b"></div><div class="e"></div><div class="f"></div>';
document.body.innerHTML = '<div class="c">' + parts + '</div><div class="n">' + parts + '</div>';
return [...document.querySelectorAll(".a, .b, .e, .f")].map(e => e.getBoundingClientRect().left);`)
	require.NoError(t, err)
	var want []float32
	require.NoError(t, json.Unmarshal(raw, &want))
	d.Layout(geom.V2(400, 300))
	var got []float32
	d.HTML().Walk(func(n *ui.Node) bool {
		if n.HasClass("a") || n.HasClass("b") || n.HasClass("e") || n.HasClass("f") {
			got = append(got, n.Box().X)
		}
		return true
	})
	assert.Equal(t, want, got)
	assert.Empty(t, d.Warnings())
}
