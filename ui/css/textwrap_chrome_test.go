//go:build chrome

package css_test

import (
	"encoding/json"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
)

func TestTextWrapPropertiesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
.balance { text-wrap: balance; }
.pretty { text-wrap: nowrap pretty; }
.after { text-wrap: balance; white-space: pre; }
.before { white-space: pre; text-wrap: balance; }
.breaks { white-space: preserve-breaks nowrap; }
.mode { white-space: pre-line; text-wrap-mode: nowrap; }
.collapse { white-space-collapse: preserve; }
.wrap { text-wrap: wrap; }
.stable { text-wrap-style: stable; }
.hy { hyphens: none; hyphenate-character: "="; }
.hy2 { hyphens: auto; hyphenate-character: "\2010"; }
.box { text-box: cap alphabetic; }
.trim { text-box-trim: trim-start; }
.edge { text-box-edge: ex text; }
.end { text-box: trim-end text; }
.normal { text-box: normal; }
.spacing { text-spacing-trim: space-all; text-autospace: normal; interpolate-size: allow-keywords; }
.first { text-spacing-trim: space-first; text-box: text trim-start; }
`))
	d.Body().Append(
		ui.Div(ui.Class("balance"), ui.Text("balance"), ui.Span(ui.Text("inherited"))),
		ui.Div(ui.Class("pretty"), ui.Text("pretty")),
		ui.Div(ui.Class("after"), ui.Text("after")),
		ui.Div(ui.Class("before"), ui.Text("before")),
		ui.Div(ui.Class("breaks"), ui.Text("breaks")),
		ui.Div(ui.Class("mode"), ui.Text("mode"), ui.Span(ui.Class("wrap"), ui.Text("wrap"))),
		ui.Div(ui.Class("collapse"), ui.Text("collapse")),
		ui.Div(ui.Class("stable"), ui.Text("stable")),
		ui.Div(ui.Class("hy"), ui.Text("hy"), ui.Span(ui.Text("inherited"))),
		ui.Div(ui.Class("hy2"), ui.Text("hy2")),
		ui.Div(ui.Class("box"), ui.Text("box"), ui.Span(ui.Text("not inherited"))),
		ui.Div(ui.Class("trim"), ui.Text("trim")),
		ui.Div(ui.Class("edge"), ui.Text("edge")),
		ui.Div(ui.Class("end"), ui.Text("end")),
		ui.Div(ui.Class("box normal"), ui.Text("normal")),
		ui.Div(ui.Class("spacing"), ui.Text("spacing"), ui.Span(ui.Text("inherited"))),
		ui.Div(ui.Class("first"), ui.Text("first")),
	)
	matchChrome(t, d)
}

// TestTextWrapValuesAreReadAsChrome asks Chrome which values it reads
// for the text-wrap, white-space, hyphens and text-box properties, and
// checks ui reads the same.
func TestTextWrapValuesAreReadAsChrome(t *testing.T) {
	decls := [][2]string{
		{"text-wrap", "balance"}, {"text-wrap", "pretty"}, {"text-wrap", "stable"}, {"text-wrap", "nowrap balance"},
		{"text-wrap", "pretty wrap"}, {"text-wrap", "auto"}, {"text-wrap", "wrap nowrap"}, {"text-wrap", "avoid-orphans"},
		{"text-wrap-style", "balance"}, {"text-wrap-style", "wrap"}, {"text-wrap-mode", "nowrap"},
		{"text-wrap-mode", "balance"},
		{"white-space", "preserve nowrap"}, {"white-space", "nowrap preserve"},
		{"white-space", "preserve-breaks"}, {"white-space", "collapse balance"}, {"white-space", "wrap"},
		{"white-space", "pre pre"}, {"white-space-collapse", "preserve-spaces"}, {"white-space-collapse", "discard"},
		{"white-space-collapse", "preserve-breaks"},
		{"hyphens", "auto"}, {"hyphens", "manual"}, {"hyphens", "all"}, {"hyphenate-character", "auto"},
		{"hyphenate-character", `"-"`}, {"hyphenate-character", "none"}, {"hyphenate-character", `"a" "b"`},
		{"text-box-trim", "trim-both"}, {"text-box-trim", "both"}, {"text-box-edge", "cap"},
		{"text-box-edge", "text"}, {"text-box-edge", "cap alphabetic"}, {"text-box-edge", "alphabetic cap"},
		{"text-box-edge", "ex alphabetic"}, {"text-box-edge", "text alphabetic"}, {"text-box-edge", "ideographic"},
		{"text-box-edge", "cap ideographic"}, {"text-box-edge", "auto auto"},
		{"text-box", "normal"}, {"text-box", "cap alphabetic"}, {"text-box", "trim-start cap alphabetic"},
		{"text-box", "cap alphabetic trim-end"}, {"text-box", "cap trim-end alphabetic"}, {"text-box", "cap"},
		{"text-box", "none"}, {"text-box", "auto"}, {"text-box", "trim-both auto"}, {"text-box", "normal text"},
		{"text-spacing-trim", "trim-start"}, {"text-spacing-trim", "trim-both"}, {"text-autospace", "normal"},
		{"text-autospace", "auto"}, {"interpolate-size", "allow-keywords"}, {"interpolate-size", "auto"},
		{"height", "calc-size(auto, size)"}, {"height", "calc-size(auto, size * 2 + 10%)"}, {"height", "calc-size(any, 10px)"},
		{"height", "calc-size(any, size)"}, {"height", "calc-size(50%, size - 1em)"}, {"height", "calc-size(auto)"},
		{"height", "calc-size(auto, 10)"}, {"height", "calc-size(calc-size(fit-content, size), size)"},
		{"height", "calc-size(none, size)"}, {"height", "calc-size(stretch, size)"},
	}
	list, err := json.Marshal(decls)
	if err != nil {
		t.Fatal(err)
	}
	var want []bool
	ask(t, geom.V2(400, 300), "return "+string(list)+".map(([p, v]) => CSS.supports(p, v));", &want)
	for i, dv := range decls {
		_, warns := css.Parse("t.css", "p { "+dv[0]+": "+dv[1]+"; }")
		if got := len(warns) == 0; got != want[i] {
			t.Errorf("%s: %s: ui reads it %v, Chrome %v (%v)", dv[0], dv[1], got, want[i], warns)
		}
	}
}
