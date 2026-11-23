//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestZZDiagSelectHeight(t *testing.T) {
	font := hebrewFont(t)
	doc := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(`@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
body { font-family: Hebrew; } select, button, input, div { display: block; margin-bottom: 4px; }`))
	nodes := []*ui.Node{
		ui.Select(ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("line-height: 20px"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("font-size: 20px"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("font-family: Arial"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("padding: 0; border: 0"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("padding: 0; border: 0; background: none"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("font-size: 16px"), ui.Option(ui.Text("abc"))),
		ui.Select(ui.Style("font-size: 12px"), ui.Option(ui.Text("abc"))),
		ui.Button(ui.Text("abc")),
		ui.Input(ui.Attr("value", "abc")),
		ui.Div(ui.Text("abc")),
		ui.Div(ui.Style("font-size: 13.333333px"), ui.Text("abc")),
		ui.Div(ui.Style("font-size: 13.333333px; font-family: Arial"), ui.Text("abc")),
		ui.Div(ui.Style("font-size: 16px"), ui.Text("abc")),
		ui.Div(ui.Style("font-size: 12px"), ui.Text("abc")),
		ui.Div(ui.Style("font-size: 20px"), ui.Text("abc")),
	}
	doc.Body().Append(nodes...)
	out, err := chrome.Eval(doc, geom.V2(400, 700), `return [...document.body.children].map(e => {
  const s = getComputedStyle(e); const r = e.getBoundingClientRect();
  return e.tagName + " h=" + r.height + " lh=" + s.lineHeight + " fs=" + s.fontSize +
    " ff=" + s.fontFamily + " pad=" + s.paddingTop + " app=" + s.appearance; })`)
	if err != nil {
		t.Fatal(err)
	}
	t.Errorf("%s", out)
	doc.Layout(geom.V2(400, 700))
	for _, n := range nodes {
		r, _ := doc.BoxOf(n)
		t.Errorf("ember %v", r)
	}
}
