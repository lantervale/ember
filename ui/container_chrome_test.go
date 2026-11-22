//go:build chrome

package ui_test

import (
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
