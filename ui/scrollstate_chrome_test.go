//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// scrollStateSheet has a sticky header and two scrollers answer
// scroll-state() queries, the widths of the boxes inside saying which
// held.
const scrollStateSheet = `
body { margin: 8px; font-size: 14px; }
.scroller { height: 80px; width: 160px; overflow: auto; margin-bottom: 8px; }
.tall { height: 300px; }
.wide { width: 400px; height: 10px; }
.head { position: sticky; top: 0; height: 20px; container-type: scroll-state; background: #eee; }
.side { position: sticky; left: 0; width: 30px; height: 20px; container-type: scroll-state; }
.self { container: self / scroll-state; }
i { display: block; height: 2px; width: 1px; }
@container scroll-state(stuck: top) { .a { width: 11px; } }
@container scroll-state(stuck) { .b { width: 12px; } }
@container scroll-state(stuck: none) { .c { width: 13px; } }
@container scroll-state(stuck: left) { .d { width: 14px; } }
@container self scroll-state(scrollable: top) { .e { width: 15px; } }
@container self scroll-state(scrollable: bottom) { .f { width: 16px; } }
@container self scroll-state(scrollable: x) { .g { width: 17px; } }
@container self scroll-state(scrollable: inline-end) { .h { width: 18px; } }
@container self scroll-state(scrollable: none) { .k { width: 19px; } }
@container self scroll-state(snapped: none) { .l { width: 20px; } }
@container self scroll-state((scrollable: block-start) and (scrollable: block-end)) { .m { width: 21px; } }
`

// TestScrollStateQueriesAnswerAsChromes scrolls each scroller of the page
// as Chrome's script does, and compares the widths of the boxes the
// queries size in each container.
func TestScrollStateQueriesAnswerAsChromes(t *testing.T) {
	probes := func() []ui.Arg {
		var out []ui.Arg
		for _, c := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "k", "l", "m"} {
			out = append(out, ui.I(ui.Class(c)))
		}
		return out
	}
	scrolls := []geom.Vec2{{}, {Y: 40}, {X: 30}, {X: 30, Y: 400}, {}}
	d := ui.NewDocument(ui.CSS(scrollStateSheet))
	var scrollers []*ui.Node
	for i := range scrolls {
		args := []ui.Arg{ui.Class("scroller", "self")}
		if i == 4 {
			args = []ui.Arg{ui.Class("scroller")}
		}
		s := ui.Div(append(args,
			ui.Div(append([]ui.Arg{ui.Class("head")}, probes()...)...),
			ui.Div(append([]ui.Arg{ui.Class("side")}, probes()...)...),
			ui.Div(probes()...),
			ui.Div(ui.Class("wide")), ui.Div(ui.Class("tall")))...)
		scrollers = append(scrollers, s)
	}
	d.Body().Append(scrollers...)
	script := "const offs = ["
	for _, at := range scrolls {
		script += fmt.Sprintf("[%g, %g],", at.X, at.Y)
	}
	script += `];
document.querySelectorAll(".scroller").forEach((s, i) => { s.scrollLeft = offs[i][0]; s.scrollTop = offs[i][1]; });
await new Promise(r => setTimeout(r, 300));
return [...document.querySelectorAll("i")].map(e => e.getBoundingClientRect().width);`
	raw, err := chrome.Eval(d, geom.V2(600, 800), script)
	require.NoError(t, err)
	var want []float32
	require.NoError(t, json.Unmarshal(raw, &want))
	d.Layout(geom.V2(600, 800))
	for i, s := range scrollers {
		d.SetScroll(s, scrolls[i])
	}
	d.Layout(geom.V2(600, 800))
	var got []float32
	d.HTML().Walk(func(n *ui.Node) bool {
		if n.Tag().String() == "i" {
			got = append(got, n.Box().W)
		}
		return true
	})
	require.Len(t, got, len(want))
	for i := range want {
		assert.InDelta(t, want[i], got[i], 0.01, "scroller %d, box %d", i/33, i%33)
	}
	assert.Empty(t, d.Warnings())
}
