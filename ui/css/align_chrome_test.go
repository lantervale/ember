//go:build chrome

package css_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// alignDeclarations are values of the alignment properties and their shorthands, valid
// and not, as WPT's css/css-align/parsing tests try them.
//
//nolint:gochecknoglobals // a test's table
var alignDeclarations = map[string][]string{
	"align-items": {"normal", "safe normal", "stretch", "baseline", "first baseline", "last baseline", "center",
		"safe center", "unsafe center", "safe end", "self-start", "safe self-end", "flex-start", "safe flex-end",
		"flow-start", "left", "safe left", "anchor-center", "baseline last", "center safe", "legacy", "auto",
		"space-between", "safe baseline", "first last baseline", "first", "last"},
	"align-self": {"auto", "normal", "safe normal", "stretch", "baseline", "first baseline", "last baseline",
		"safe center", "unsafe end", "self-start", "left", "anchor-center", "safe anchor-center", "legacy"},
	"justify-self": {"auto", "normal", "safe normal", "stretch", "first baseline", "last baseline", "safe center",
		"unsafe end", "unsafe right", "anchor-center", "unsafe anchor-center", "flex-start", "safe stretch"},
	"justify-items": {"normal", "legacy", "safe center", "last baseline", "first baseline", "baseline", "stretch",
		"auto", "self-start", "unsafe self-end", "anchor-center"},
	"align-content": {"normal", "baseline", "first baseline", "last baseline", "safe center", "unsafe end",
		"space-between", "safe space-between", "stretch", "left", "safe flex-end", "self-start", "safe normal",
		"space-around safe"},
	"justify-content": {"normal", "baseline", "last baseline", "safe center", "unsafe end", "space-between",
		"stretch", "safe flex-start", "self-end", "safe normal", "unsafe"},
	"place-items": {"last baseline", "safe center", "safe center unsafe end", "first baseline last baseline",
		"center legacy", "legacy center", "last baseline safe end", "stretch safe", "normal auto"},
	"place-self": {"last baseline", "safe center", "safe center unsafe end", "first baseline last baseline",
		"auto unsafe self-start", "safe anchor-center normal"},
	"place-content": {"last baseline", "safe center", "safe center unsafe end", "first baseline last baseline",
		"baseline unsafe end", "space-between safe center", "baseline", "first baseline", "stretch space-around"},
}

// alignLonghands are the properties TestAlignmentValuesComputeAsInChrome compares.
//
//nolint:gochecknoglobals // a test's table
var alignLonghands = []string{"align-items", "align-self", "align-content", "justify-content", "justify-items",
	"justify-self"}

// alignWord is how Chrome serializes the computed alignment a.
func alignWord(a style.Align) string {
	var b strings.Builder
	switch {
	case a.Safe():
		b.WriteString("safe ")
	case a.Unsafe():
		b.WriteString("unsafe ")
	}
	b.WriteString(map[style.Align]string{
		style.AlignNormal: "normal", style.AlignAuto: "auto", style.AlignStretch: "stretch",
		style.AlignStart: "start", style.AlignEnd: "end", style.AlignFlexStart: "flex-start",
		style.AlignFlexEnd: "flex-end", style.AlignCenter: "center", style.AlignBaseline: "baseline",
		style.AlignSpaceBetween: "space-between", style.AlignSpaceAround: "space-around",
		style.AlignSpaceEvenly: "space-evenly", style.AlignAnchorCenter: "anchor-center",
		style.AlignLastBaseline: "last baseline",
	}[a.Position()])
	return b.String()
}

// TestAlignmentValuesComputeAsInChrome sets each of alignDeclarations on an element of
// its own in Chrome and in ui, and compares the six longhands each computes: an invalid
// value leaves them all at their initial values in both.
func TestAlignmentValuesComputeAsInChrome(t *testing.T) {
	type decl struct {
		Prop  string `json:"Prop"`
		Value string `json:"Value"`
	}
	var decls []decl
	for p, vs := range alignDeclarations {
		for _, v := range vs {
			decls = append(decls, decl{p, v})
		}
	}
	list, _ := json.Marshal(decls)
	longhands, _ := json.Marshal(alignLonghands)
	var want [][]string
	ask(t, geom.V2(400, 300), `
return `+string(list)+`.map(d => {
	const e = document.createElement("div");
	document.body.appendChild(e);
	e.style.setProperty(d.Prop, d.Value);
	const cs = getComputedStyle(e);
	return `+string(longhands)+`.map(p => cs.getPropertyValue(p));
});`, &want)
	d := ui.NewDocument()
	for _, dc := range decls {
		d.Body().Append(ui.Div(ui.Style(dc.Prop + ": " + dc.Value)))
	}
	d.Layout(geom.V2(400, 300))
	for i, n := range d.Body().Children() {
		st := dom.Data(n).Style
		got := []string{alignWord(st.AlignItems), alignWord(st.AlignSelf), alignWord(st.AlignContent),
			alignWord(st.JustifyContent), alignWord(st.JustifyItems), alignWord(st.JustifySelf)}
		for k, p := range alignLonghands {
			w := strings.NewReplacer("self-start", "start", "self-end", "end", "left", "start", "right", "end",
				"flow-start", "flex-start", "flow-end", "flex-end").Replace(want[i][k])
			if got[k] != w {
				t.Errorf("%s: %s: %s computes to %q, Chrome's %q", decls[i].Prop, decls[i].Value, p, got[k], want[i][k])
			}
		}
	}
}
