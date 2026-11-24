//go:build chrome

package ui_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

func groupedSelects() []*ui.Node {
	return []*ui.Node{
		ui.Select(ui.Option(ui.Text("Sword")),
			ui.Optgroup("Magic weapons, a label wider than any option",
				ui.Option(ui.Text("Wand")), ui.Option(ui.Text("Staff")))),
		ui.Select(ui.Option(ui.Text("Sword")), ui.Optgroup("M", ui.Option(ui.Text("Wand")))),
		ui.Select(ui.Option(ui.Text("Sword"))),
		ui.Select(ui.Optgroup("Cursed", ui.Disabled(), ui.Option(ui.Text("Skull")), ui.Option(ui.Text("Bone"))),
			ui.Option(ui.Text("Bow"))),
		ui.Select(ui.Option(ui.Text("Sword")), ui.Optgroup("Empty"), ui.Optgroup("Long", ui.Option(ui.Text("Wandwandwand")))),
		ui.Select(ui.Style("font-size: 20px"), ui.Optgroup("", ui.Option(ui.Text("Aa"))), ui.Option(ui.Text("Bb"))),
		ui.Select(ui.Attr("required", ""), ui.Optgroup("G", ui.Option(ui.Value(""), ui.Text("none")))),
		ui.Select(ui.Placeholder("Pick one"), ui.Optgroup("Ranged", ui.Option(ui.Text("Bow")))),
		ui.Select(ui.Optgroup("Empty"), ui.Optgroup("Long", ui.Option(ui.Text("Bone flail of the crypt")))),
		ui.Select(ui.Attr("required", ""), ui.Option(ui.Value(""), ui.Text("none")),
			ui.Optgroup("G", ui.Option(ui.Text("a")))),
	}
}

// TestASelectWithOptgroupsIsAsWideAsChromes: its width counts its
// options, those of a group behind four spaces, never the labels.
func TestASelectWithOptgroupsIsAsWideAsChromes(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(groupedSelects()...)
	// Chrome on Linux makes a select of the system font a point shorter
	// (21 for ui's and macOS's 22), the rows below it two higher.
	within := float32(chrome.Tolerance)
	if runtime.GOOS != "darwin" {
		within = 2
	}
	r := chrome.MatchWithin(t, d, geom.V2(640, 200), within)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestOptgroupsChooseAndDisableAsChromes: a select's options run across
// its groups as selectedIndex counts them, the first enabled one chosen
// where none is selected; a disabled optgroup and its options match
// :disabled, an enabled one :enabled; a required select's empty first
// option is a placeholder only outside any group.
func TestOptgroupsChooseAndDisableAsChromes(t *testing.T) {
	d := ui.NewDocument()
	sels := groupedSelects()
	d.Body().Append(sels...)
	raw, err := chrome.Eval(d, geom.V2(640, 200), `const sels = [...document.querySelectorAll("select")];
const items = [...document.querySelectorAll("optgroup, option:not([data-ember-implied])")];
return {
	index: sels.map(s => {
		const c = s.cloneNode(true);
		c.querySelectorAll("[selected]").forEach(o => o.removeAttribute("selected"));
		return c.selectedIndex;
	}),
	value: sels.map(s => s.value),
	count: sels.map(s => s.options.length),
	valid: sels.map(s => s.validity.valid),
	disabled: items.map(e => e.matches(":disabled")),
	enabled: items.map(e => e.matches(":enabled")),
};`)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Index    []int    `json:"index"`
		Count    []int    `json:"count"`
		Value    []string `json:"value"`
		Valid    []bool   `json:"valid"`
		Disabled []bool   `json:"disabled"`
		Enabled  []bool   `json:"enabled"`
	}
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	d.Layout(geom.V2(640, 200))
	for i, s := range sels {
		if _, ok := s.Attr("placeholder"); ok {
			// ui's own: the page holds an option standing for it.
			continue
		}
		if got := d.Selected(s); got != want.Index[i] {
			t.Errorf("select %d: ui chooses %d, Chrome %d", i, got, want.Index[i])
		}
		if got := d.Value(s); got != want.Value[i] {
			t.Errorf("select %d: ui's value %q, Chrome's %q", i, got, want.Value[i])
		}
		if got := len(dom.Options(s)); got != want.Count[i] {
			t.Errorf("select %d: ui has %d options, Chrome %d", i, got, want.Count[i])
		}
		if got := d.Valid(s); got != want.Valid[i] {
			t.Errorf("select %d: ui valid %v, Chrome %v", i, got, want.Valid[i])
		}
	}
	enabled, err := css.ParseSelectors(":enabled")
	if err != nil {
		t.Fatal(err)
	}
	k := 0
	d.Body().Walk(func(n *ui.Node) bool {
		if n.Tag() != dom.Optgroup && n.Tag() != dom.Option {
			return true
		}
		if got := n.State().Has(dom.Disabled); got != want.Disabled[k] {
			t.Errorf("%s %d: ui :disabled %v, Chrome %v", n.Tag(), k, got, want.Disabled[k])
		}
		if got := enabled[0].Matches(n); got != want.Enabled[k] {
			t.Errorf("%s %d: ui :enabled %v, Chrome %v", n.Tag(), k, got, want.Enabled[k])
		}
		k++
		return true
	})
}
