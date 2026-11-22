//go:build chrome

package css_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// filtersCSS writes filters as getComputedStyle does.
func filtersCSS(fs []style.Filter) string {
	if len(fs) == 0 {
		return "none"
	}
	names := [...]string{"blur", "brightness", "contrast", "grayscale", "hue-rotate", "invert", "opacity", "saturate",
		"sepia", "drop-shadow"}
	var parts []string
	for _, f := range fs {
		var arg string
		switch f.Kind {
		case style.Blur:
			arg = px(f.Amount)
		case style.HueRotate:
			arg = num(f.Amount) + "deg"
		case style.DropShadow:
			s := f.Shadow
			arg = fmt.Sprintf("%s %s %s %s", rgb(s.Color), px(s.X), px(s.Y), px(s.Blur))
		case style.Brightness, style.Contrast, style.Grayscale, style.Invert, style.OpacityFilter, style.Saturate,
			style.Sepia:
			arg = num(f.Amount)
		}
		parts = append(parts, names[f.Kind]+"("+arg+")")
	}
	return strings.Join(parts, " ")
}

func TestFiltersComputeAsInChrome(t *testing.T) {
	css := `
div { color: #0a0; }
.all { filter: blur(2px) brightness(150%) contrast(0.5) grayscale(30%) hue-rotate(0.5turn) invert(0.25)
  opacity(40%) saturate(3) sepia(1) drop-shadow(#f00 2px 3px 4px); }
.bare { filter: blur() brightness() grayscale() hue-rotate() drop-shadow(1px -1px); }
.over { filter: grayscale(2) invert(150%) opacity(3) sepia(200%) brightness(4); }
.units { filter: blur(0.5em) hue-rotate(1rad) drop-shadow(1em 2px currentcolor); }
.math { font-size: 10px; filter: blur(calc(1em + 2px)) brightness(clamp(0.5, 3, 1.5)) hue-rotate(calc(45deg * 2))
  grayscale(min(30%, 50%)) drop-shadow(calc(1em / 2) max(1px, 3px)); }
.none { filter: blur(2px); filter: none; }
.bad { filter: sepia(1); filter: blur(-1px); filter: blur(1px) none; }
.webkit { -webkit-filter: invert(1); }
`
	d := ui.NewDocument(ui.CSS(css), ui.BrowserSheets())
	classes := []string{"all", "bare", "over", "units", "math", "none", "bad", "webkit"}
	for _, class := range classes {
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	raw, err := chrome.Eval(d, geom.V2(800, 600),
		`return Array.from(document.body.querySelectorAll("div")).map(e => getComputedStyle(e).filter);`)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	for i, n := range d.Body().Children() {
		if got := filtersCSS(dom.Data(n).Style.Filters); !same("filter", got, want[i]) {
			t.Errorf("%s: ember %q, Chrome %q", classes[i], got, want[i])
		}
	}
}
