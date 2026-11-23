//go:build chrome

package layout_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/layout"
)

func TestCalcSizeHeightsLayOutAsChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { margin: 2px 0; background: #ddd; }
.pad { padding: 5px; border: 2px solid; }
.bb { box-sizing: border-box; }
`))
	for _, h := range []string{"auto", "calc-size(auto, size)", "calc-size(auto, size * 0.5)",
		"calc-size(auto, size + 20px)",
		"calc-size(fit-content, size * 2)", "calc-size(any, 30px)", "calc-size(50px, size * 2)",
		"calc-size(min-content, size - 4px)", "calc-size(calc-size(auto, size + 10px), size * 0.5)",
		"calc-size(auto, size * 0.5 + 10%)"} {
		d.Body().Append(
			ui.Div(ui.Style("height: "+h), ui.Text(sentence)),
			ui.Div(ui.Class("pad"), ui.Style("height: "+h+"; width: 200px; overflow: hidden"), ui.Text(sentence)),
			ui.Div(ui.Class("pad", "bb"), ui.Style("height: "+h+"; width: 200px"), ui.Text(sentence)),
		)
	}
	wrap := ui.NewDocument()
	wrap.Body().Append(ui.Div(ui.Style("height: 300px"), ui.Div(ui.Style("height: calc-size(auto, size + 10%)"),
		ui.Text(sentence))))
	for _, doc := range []*ui.Document{d, wrap} {
		r := chrome.Match(t, doc, geom.V2(800, 600))
		t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	}
}

// TestHeightsTransitionToAndFromAutoAsChrome runs height transitions to and
// from auto, paused at set times in Chrome and stepped as long in ui, and
// compares the boxes.
func TestHeightsTransitionToAndFromAutoAsChrome(t *testing.T) {
	sheet := `
.t { overflow: hidden; transition: height 1s linear; background: #ddd; }
.keywords { interpolate-size: allow-keywords; }
.zero { height: 0; } .zero.on { height: auto; }
.auto { height: auto; } .auto.on { height: 10px; }
.fixed { height: 40px; } .fixed.on { height: auto; }
.explicit { height: 0; } .explicit.on { height: calc-size(auto, size); }
.content { height: 12px; } .content.on { height: min-content; }
.half { height: calc-size(auto, size * 0.5); } .half.on { height: 100px; }
`
	classes := []string{"zero keywords", "auto keywords", "fixed keywords", "zero", "explicit", "content keywords",
		"half", "fixed"}
	for _, at := range []float32{0.25, 0.5, 0.8} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			d := ui.NewDocument(ui.CSS(sheet))
			var nodes []*ui.Node
			for _, c := range classes {
				n := ui.Div(ui.Class(append([]string{"t"}, strings.Fields(c)...)...), ui.Text(sentence+" "+sentence))
				nodes = append(nodes, n)
				d.Body().Append(n)
			}
			raw, err := chrome.Eval(d, geom.V2(400, 600), fmt.Sprintf(`
const els = Array.from(document.querySelectorAll(".t"));
els.forEach(e => getComputedStyle(e).height);
els.forEach(e => e.classList.add("on"));
els.forEach(e => getComputedStyle(e).height);
document.getAnimations().forEach(a => { a.pause(); a.currentTime = %v; });
return els.map(e => e.getBoundingClientRect().height);`, at*1000))
			if err != nil {
				t.Fatal(err)
			}
			var want []float32
			if err = json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			d.Layout(geom.V2(400, 600))
			tr := css.NewTransitions()
			tr.Compute(d.HTML(), d.Sheets())
			for _, n := range nodes {
				n.AddClass("on")
			}
			tr.Compute(d.HTML(), d.Sheets())
			tr.Step(at)
			tr.Compute(d.HTML(), d.Sheets())
			tree := layout.Layout(d.HTML(), geom.V2(400, 600), fonts.Of(d))
			for i, n := range nodes {
				got := tree.Of[n].Size.Y
				if abs(got-want[i]) > 0.5 {
					t.Errorf("%s at %v: ember %v, Chrome %v", classes[i], at, got, want[i])
				}
			}
		})
	}
}
