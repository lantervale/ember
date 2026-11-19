//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestSheetsChangedWhileADocumentShowsLayOutAsChromes changes one
// Document's sheets step by step, as a page's scripts do, and lays it
// out against a Chrome given the sheets as they then stand: a rule
// added later wins or loses by order and specificity as Chrome's, and a
// sheet removed leaves the look before it.
func TestSheetsChangedWhileADocumentShowsLayOutAsChromes(t *testing.T) {
	for name, opts := range map[string][]ui.DocumentOption{"a game's sheet": nil,
		"a browser's sheet": {ui.BrowserSheets()}} {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(append(opts, ui.CSS(`
				.card { width: 200px; height: 20px; padding: 4px; margin: 2px 0; border: 1px solid #335; }
				#first { width: 150px; }
				.row { display: flex; width: 300px; }
				.row > div { flex: 1; height: 12px; }`))...)
			d.Body().Append(
				ui.Div(ui.ID("first"), ui.Class("card")),
				ui.Div(ui.Class("card", "wide")),
				ui.Div(ui.Class("row"), ui.Div(), ui.Div(ui.Class("big")), ui.Div()),
			)
			size := geom.V2(800, 600)
			step := func(name string) {
				t.Run(name, func(t *testing.T) {
					r := chrome.Match(t, d, size)
					t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
				})
			}
			step("as made")
			base := d.StyleSheets()[0]

			later := d.AddSheet(`.card { width: 260px; } div.wide { height: 44px; } #first { margin-left: 30px; }`)
			step("a sheet added after: the same specificity wins by order, a weaker one loses")

			early := d.InsertSheet(base, `.card { width: 90px; padding: 9px; } #first.card { width: 111px; }
				.big { flex: 3 !important; }`)
			step("a sheet inserted before: it loses by order, wins by specificity and importance")

			require.NoError(t, later.InsertRule(0, `.wide { height: 70px; width: 120px; }`))
			require.NoError(t, later.InsertRule(4,
				`@media (min-width: 700px) { .row > div { height: 30px; } .card { width: 240px; } }`))
			require.NoError(t, later.InsertRule(5, `@media (min-width: 900px) { .row { width: 10px; } }`))
			step("rules inserted first and last in a sheet")

			require.NoError(t, later.DeleteRule(1))
			step("a rule deleted")

			early.Replace(`.row { width: 400px; } .card { height: 5px; }`)
			step("a sheet replaced")

			later.SetMedia("(max-width: 500px)")
			step("a sheet under a media that does not hold")
			size = geom.V2(480, 600)
			step("the window resized under it")
			size = geom.V2(800, 600)
			later.SetMedia("")

			early.SetDisabled(true)
			step("a sheet disabled")
			early.SetDisabled(false)

			later.Remove()
			early.Remove()
			step("the sheets removed: the look before them")
			assert.Empty(t, d.Warnings())
		})
	}
}
