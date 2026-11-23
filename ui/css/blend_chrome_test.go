//go:build chrome

package css_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

//nolint:gochecknoglobals // a lookup table
var blendNames = [...]string{"normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge",
	"color-burn", "hard-light", "soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity"}

func TestBlendModesComputeAsInChrome(t *testing.T) {
	css := `
div { color: #0a0; }
.all { mix-blend-mode: hard-light; isolation: isolate; background-blend-mode: screen, color-burn, normal;
  backdrop-filter: blur(2px) brightness(150%) hue-rotate(0.5turn) drop-shadow(1px 2px); }
.modes { mix-blend-mode: soft-light; mix-blend-mode: exclusion; background-blend-mode: hue; }
.bad { mix-blend-mode: luminosity; mix-blend-mode: plus-lighter; isolation: isolate; isolation: auto;
  backdrop-filter: invert(1); backdrop-filter: blur(-2px); }
.none { backdrop-filter: sepia(1); backdrop-filter: none; background-blend-mode: color, saturation; }
.layers { background-image: linear-gradient(red, blue), none, none; background-blend-mode: darken, lighten; }
`
	d := ui.NewDocument(ui.CSS(css))
	classes := []string{"all", "modes", "bad", "none", "layers"}
	for _, class := range classes {
		d.Body().Append(ui.Div(ui.Class(class)))
	}
	raw, err := chrome.Eval(d, geom.V2(800, 600), `return Array.from(document.body.querySelectorAll("div")).map(e => {
  const s = getComputedStyle(e);
  return [s.mixBlendMode, s.isolation, s.backgroundBlendMode, s.backdropFilter].join(" | ");
});`)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	for i, n := range d.Body().Children() {
		s := dom.Data(n).Style
		isolation := "auto"
		if s.Isolate {
			isolation = "isolate"
		}
		var blends []string
		for l := range s.BackgroundLayers() {
			blends = append(blends, blendNames[s.BackgroundLayer(l).Blend])
		}
		got := strings.Join([]string{blendNames[s.BlendMode], isolation, strings.Join(blends, ", "),
			filtersCSS(s.BackdropFilters)}, " | ")
		if !same("blend", got, want[i]) {
			t.Errorf("%s: ember %q, Chrome %q", classes[i], got, want[i])
		}
	}
}
