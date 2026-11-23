//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
)

func TestTextPropertiesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
.caps { font-variant: small-caps; }
.mix { font-variant: slashed-zero all-petite-caps tabular-nums oldstyle-nums no-contextual common-ligatures; }
.none { font-variant: none; }
.asian { font-variant: ruby jis90 full-width; }
.pos { font-variant-position: super; font-variant-alternates: historical-forms; }
.font { font: italic small-caps bold 12px/1.5 Go; }
.reset { font-variant: tabular-nums; font: 14px Go; }
.longs { font-variant-caps: unicase; font-variant-numeric: ordinal diagonal-fractions proportional-nums; }
.back { font-variant: normal; }
.breaks { word-break: break-all; line-break: anywhere; tab-size: 3; }
.keep { word-break: keep-all; line-break: strict; tab-size: 2em; }
.word { word-break: break-word; line-break: loose; tab-size: 0; }
`))
	d.Body().Append(
		ui.Div(ui.Class("caps"), ui.Text("small caps"), ui.Span(ui.Text("inherited"))),
		ui.Div(ui.Class("mix"), ui.Text("mixed")),
		ui.Div(ui.Class("none"), ui.Text("none")),
		ui.Div(ui.Class("asian"), ui.Text("east asian")),
		ui.Div(ui.Class("pos"), ui.Text("position")),
		ui.Div(ui.Class("font"), ui.Text("font shorthand")),
		ui.Div(ui.Class("reset"), ui.Text("reset by font")),
		ui.Div(ui.Class("longs"), ui.Text("longhands"), ui.Span(ui.Class("back"), ui.Text("back to normal"))),
		ui.Div(ui.Class("breaks"), ui.Text("breaks"), ui.Span(ui.Text("inherited"))),
		ui.Div(ui.Class("keep"), ui.Text("keeps"), ui.Span(ui.Style("font-size: 20px"), ui.Text("ems inherited"))),
		ui.Div(ui.Class("word"), ui.Text("break-word")),
	)
	matchChrome(t, d)
}
