//go:build chrome

package css_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestLogicalPropertiesInEachWritingModeMatchChrome(t *testing.T) {
	src := `
.a { margin-block-start: 1px; margin-block-end: 2px; margin-inline-start: 3px; margin-inline-end: 4px;
  padding-block: 5px 6px; padding-inline: 7px 8px; inline-size: 30px; block-size: 40px; min-inline-size: 10px;
  max-block-size: 90px; border-inline-start: 2px solid; border-block-end-width: 3px; border-block-end-style: solid;
  border-start-end-radius: 6px; border-end-start-radius: 9px; overflow-inline: hidden; overflow-block: scroll;
  position: relative; inset-block-start: 11px; inset-inline-end: 12px; }
.b { margin: 0; margin-inline: 5px 15px; margin-left: 1px; }
.c { margin-top: 3px; margin-block-start: 9px; }
.combine { text-combine-upright: all; text-orientation: upright; }
.sideways { text-orientation: sideways; }
.legacy { writing-mode: tb-rl; }
`
	props := map[string]func(*style.Computed) string{
		"writing-mode": func(s *style.Computed) string {
			return [...]string{"horizontal-tb", "vertical-rl", "vertical-lr", "sideways-rl", "sideways-lr"}[s.Mode]
		},
		"text-orientation": func(s *style.Computed) string {
			return [...]string{"mixed", "upright", "sideways"}[s.Orientation]
		},
		"text-combine-upright": func(s *style.Computed) string {
			return map[bool]string{false: "none", true: "all"}[s.Combine]
		},
	}
	for _, p := range []string{"min-width", "min-height", "max-width", "max-height",
		"margin-left", "padding-right", "border-left-width", "border-bottom-width",
		"border-right-style", "border-top-right-radius", "border-bottom-left-radius", "border-top-left-radius"} {
		props[p] = everything[p]
	}
	for _, p := range []string{"margin-top", "margin-right", "padding-top", "padding-left", "padding-bottom",
		"margin-bottom", "border-top-width", "border-right-width", "overflow-x", "overflow-y"} {
		if f, ok := compared[p]; ok {
			props[p] = f
		} else {
			props[p] = everything[p]
		}
	}
	for p, f := range props {
		if f == nil {
			t.Fatalf("no way to read %s", p)
		}
	}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"a.css": {Data: []byte(src)}}, "a.css"))
	for _, mode := range []string{"horizontal-tb", "vertical-rl", "vertical-lr", "sideways-rl", "sideways-lr"} {
		for _, dir := range []string{"ltr", "rtl"} {
			box := ui.Div(ui.Style(fmt.Sprintf("writing-mode: %s; direction: %s", mode, dir)),
				ui.Div(ui.Class("a")), ui.Div(ui.Class("b")), ui.Div(ui.Class("c")),
				ui.Span(ui.Class("combine"), ui.Text("12")), ui.Span(ui.Class("sideways"), ui.Text("ab")))
			d.Body().Append(box)
		}
	}
	d.Body().Append(ui.Div(ui.Class("legacy"), ui.Div(ui.Class("a"))))
	matchChromeProps(t, d, props, 2)
}
