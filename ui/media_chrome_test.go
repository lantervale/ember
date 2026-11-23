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
// Document's scheme, reduced motion, contrast, forced colours and
// pointers, whatever the machine prefers, and lays the boxes their rules
// size out in both (decisions 0121, 0139).
func TestChromeLaysOutWhatADocumentPrefersAsIt(t *testing.T) {
	sheet := `.card { height: 10px; width: 50px; position: relative }
@media (prefers-color-scheme: dark) { .card { height: 20px } }
@media (prefers-reduced-motion: reduce) { .card { width: 30px } }
@media (min-width: 50vw) and (max-width: calc(100vw - 1px)) { .card { margin-left: 5px } }
@media (min-width: calc(100vw - 1px)) { .card { margin-top: 7px } }
@media (prefers-contrast: more) { .card { height: 12px } }
@media (prefers-contrast: less) { .card { height: 14px } }
@media (forced-colors: active) { .card { left: 3px } }
@media (any-hover: none) { .card { top: 2px } }
@media (min-width: 30ex) and (max-width: 80ch) { .card { min-width: 61px } }`
	dark := platform.NewPreferences()
	dark.SyncAppearance(platform.AppearanceDark)
	dark.SyncMotion(platform.MotionReduced)
	contrasted := platform.NewPreferences()
	contrasted.SyncContrast(platform.ContrastMore)
	contrasted.SyncForcedColors(platform.ForcedColorsActive)
	for name, d := range map[string]*ui.Document{
		"the system's contrast": ui.NewDocument(ui.CSS(sheet), ui.Preferences(contrasted)),
		"over the system's contrast": ui.NewDocument(ui.CSS(sheet), ui.Preferences(contrasted),
			ui.PrefersContrast(ui.ContrastLess)),
		"more contrast": ui.NewDocument(ui.CSS(sheet), ui.PrefersContrast(ui.ContrastMore)),
		"less contrast, forced colours": ui.NewDocument(ui.CSS(sheet), ui.PrefersContrast(ui.ContrastLess),
			ui.ForcedColors()),
		"touch only":      ui.NewDocument(ui.CSS(sheet), ui.TouchOnly()),
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
