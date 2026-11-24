//go:build chrome

package ui_test

import (
	"runtime"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func opt(text string, args ...ui.Arg) *ui.Node {
	return ui.Option(append([]ui.Arg{ui.Text(text)}, args...)...)
}

func multiple() ui.Arg { return ui.Attr("multiple", "") }

func listBoxes() []*ui.Node {
	return []*ui.Node{
		ui.Select(multiple(), opt("Sword"), opt("Bow", ui.Selected()), opt("Wand")),
		ui.Select(ui.Attr("size", "3"), opt("One"), opt("Two"), opt("Three"), opt("Four"), opt("Five")),
		ui.Select(multiple(), opt("Fists"),
			ui.Optgroup("Blades", opt("Sword"), opt("Dagger", ui.Disabled())),
			ui.Optgroup("Cursed", ui.Disabled(), opt("Skull"))),
		ui.Select(multiple(), ui.Style("font-size: 20px"), opt("Aa")),
		ui.Select(ui.Attr("size", "2"), opt("x")),
		ui.Select(ui.Attr("size", "3"), ui.Style("font-size: 20px; line-height: 40px"), opt("A"), ui.Optgroup("x")),
		ui.Select(multiple(), ui.Style("font-weight: 700"), opt("Sword"),
			ui.Optgroup("G", opt("Bow", ui.Selected()), opt("Staff", ui.Selected(), ui.Disabled()))),
		ui.Select(ui.Attr("size", "2"), ui.Style("height: 60px; width: 90px"), opt("Tall")),
		ui.Div(ui.Select(ui.Attr("size", "1"), ui.Style("height: 24px"), opt("Dropdown")),
			ui.Select(ui.Attr("size", "0"), ui.Style("height: 24px"), opt("Dropdown"))),
	}
}

// TestAListBoxIsLaidOutAsChromes: an inline-block of size rows (4 with
// multiple alone), each as tall as its tallest option or group label, a
// group's label a bold row before its options, indented 20px; as wide as
// its widest row, bottom on the text's.
func TestAListBoxIsLaidOutAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.BrowserSheets())
	boxes := listBoxes()
	if runtime.GOOS == "windows" {
		// Chrome on Windows sets the bold listbox's group label in a
		// taller line, 20 points to ui's and macOS's 17, raising the
		// box 12 (docs/05-ui/10-missing.md).
		boxes = slices.Delete(boxes, 6, 7)
	}
	d.Body().Append(boxes...)
	r := chrome.Match(t, d, geom.V2(640, 240))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestAGameListBoxIsLaidOutAsChromes is the same in a game's sheet.
func TestAGameListBoxIsLaidOutAsChromes(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(listBoxes()...)
	r := chrome.Match(t, d, geom.V2(640, 240))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
