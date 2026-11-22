//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestChromeLaysOutWhatADocumentPrefersAsIt has Chrome emulate each
// Document's scheme and reduced motion, whatever the machine prefers,
// and lays the boxes their rules size out in both (decision 0121).
func TestChromeLaysOutWhatADocumentPrefersAsIt(t *testing.T) {
	sheet := `.card { height: 10px; width: 50px }
@media (prefers-color-scheme: dark) { .card { height: 20px } }
@media (prefers-reduced-motion: reduce) { .card { width: 30px } }
@media (min-width: 50vw) and (max-width: calc(100vw - 1px)) { .card { margin-left: 5px } }
@media (min-width: calc(100vw - 1px)) { .card { margin-top: 7px } }`
	dark := platform.NewPreferences()
	dark.SyncAppearance(platform.AppearanceDark)
	dark.SyncMotion(platform.MotionReduced)
	for name, d := range map[string]*ui.Document{
		"light":           ui.NewDocument(ui.CSS(sheet)),
		"dark":            ui.NewDocument(ui.CSS(sheet), ui.ColorScheme(platform.AppearanceDark)),
		"reduced":         ui.NewDocument(ui.CSS(sheet), ui.ReducedMotion()),
		"the system's":    ui.NewDocument(ui.CSS(sheet), ui.Preferences(dark)),
		"over the system": ui.NewDocument(ui.CSS(sheet), ui.Preferences(dark), ui.ColorScheme(platform.AppearanceLight)),
	} {
		t.Run(name, func(t *testing.T) {
			d.Body().Append(ui.Div(ui.Class("card")))
			chrome.Match(t, d, geom.V2(400, 300))
		})
	}
}
