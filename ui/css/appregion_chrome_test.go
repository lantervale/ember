//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

// TestAppRegionComputesAsChromes: drag and no-drag are inherited, and none
// set on a box computes no-drag, so a box in a drag region set to none
// leaves it as one set to no-drag does.
func TestAppRegionComputesAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`#bar { app-region: drag } #back { app-region: no-drag } #none { app-region: none }
#again { app-region: drag } #plain-none { app-region: none }`))
	d.Body().Append(
		ui.Div(ui.ID("bar"), ui.Span(ui.ID("inner")),
			ui.Div(ui.ID("back"), ui.I(ui.ID("deep")), ui.Div(ui.ID("again"), ui.B(ui.ID("dragged")))),
			ui.Div(ui.ID("none"), ui.I(ui.ID("under-none")))),
		ui.Div(ui.ID("plain")), ui.Div(ui.ID("plain-none")))
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"app-region": func(s *style.Computed) string {
			return [...]string{"none", "drag", "no-drag"}[s.AppRegion]
		},
	}, 0)
}
