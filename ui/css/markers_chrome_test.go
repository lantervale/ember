//go:build chrome

package css_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestAMarkersCursorAndTransitionsAsInChrome has a list item's ::marker
// transition halfway (a negative delay of half its length) and asks
// Chrome and ui for its colour, its size and its cursor, and for the
// element over its bullet.
func TestAMarkersCursorAndTransitionsAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
li::marker { color: #000; font-size: 10px; cursor: pointer;
	transition: color 1000s linear -500s, font-size 1000s linear -500s; }
li.on::marker { color: #fff; font-size: 20px; }
li { cursor: default; }
`))
	item := ui.Li(ui.Text("item"))
	d.Body().Append(ui.Ul(item))
	d.Layout(geom.V2(800, 600))
	raw, err := chrome.Eval(d, geom.V2(800, 600), `
const li = document.querySelector("li");
const r = li.getBoundingClientRect();
const at = document.elementFromPoint(r.left - 12, r.top + 11);
li.classList.add("on");
const s = getComputedStyle(li, "::marker");
return [s.color, s.fontSize, s.cursor, at.tagName.toLowerCase(), r.left, r.top + 11];`)
	if err != nil {
		t.Fatal(err)
	}
	var want []any
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	win := platform.NewWindow(platform.Size(800, 600))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	left, _ := want[4].(float64)
	top, _ := want[5].(float64)
	at := d.ElementAt(geom.V2(float32(left)-12, float32(top)))
	item.AddClass("on")
	w.RunFrames(1)
	data := dom.Data(item)
	g := data.MarkerNode
	if g == nil {
		t.Fatal("no ::marker node")
	}
	got := []string{rgb(dom.Data(g).Shown.Color), fmt.Sprintf("%gpx", data.Marker.Font.Size), data.Marker.Cursor,
		at.Tag().String()}
	for i, name := range []string{"color", "font-size", "cursor", "element over the bullet"} {
		if !same(name, got[i], fmt.Sprint(want[i])) {
			t.Errorf("%s: ui %q, Chrome %q", name, got[i], want[i])
		}
	}
}
