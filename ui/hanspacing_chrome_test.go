//go:build chrome

package ui_test

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestAFieldsHanSpacingStandsWhereChromesDoes selects in text fields of
// go-han.ttf, the stand-in face with halt, their text clear so that only
// the selection, green, and the caret, red, show: full-width punctuation
// side by side set half wide and the room text-autospace adds, in an
// input and along a textarea's lines, as Chrome sets them.
func TestAFieldsHanSpacingStandsWhereChromesDoes(t *testing.T) {
	face, err := os.ReadFile("../internal/shape/testdata/go-han.ttf")
	require.NoError(t, err)
	focused := ui.Attr("autofocus", "")
	for _, c := range []struct {
		name     string
		field    *ui.Node
		from, to int
	}{
		{"trimmed in an input", ui.Input(focused, ui.Attr("value", "漢」「漢。「漢、」漢")), 2, 8},
		{"after a trimmed bracket", ui.Input(focused, ui.Attr("value", "漢」「漢。「漢")), 3, 6},
		{"autospace in an input", ui.Input(focused, ui.Class("auto"), ui.Attr("value", "漢Ember字 1966か")), 1, 9},
		{"space-all", ui.Input(focused, ui.Class("all"), ui.Attr("value", "漢」「漢。「漢")), 2, 6},
		{"the caret after a trimmed bracket", ui.Input(focused, ui.Attr("value", "漢」「漢。「漢")), 6, 6},
		{"along a textarea's lines", ui.Textarea(focused, ui.Text("漢」「漢。「漢\n字」」字（「字")), 2, 13},
		{"in a wrapped textarea", ui.Textarea(focused, ui.Class("narrow"), ui.Text("漢字漢字」「漢字漢字」「字")), 3, 8},
		{"on a wrapped textarea's last line", ui.Textarea(focused, ui.Class("narrow"), ui.Text("漢字漢字」「漢字漢字」「字")),
			10, 12},
		{"the caret of a textarea", ui.Textarea(focused, ui.Text("漢」「漢\n字」」字")), 7, 7},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(`
@font-face { font-family: Han; src: url(han.ttf); }
body { margin: 0; padding: 10px; font-family: Han; font-size: 20px; }
input, textarea { font: inherit; width: 300px; height: 60px; color: transparent; caret-color: #ff0000;
  outline: none; border: none; padding: 0; resize: none; }
.narrow { width: 100px; height: 100px; }
.auto { text-autospace: normal; }
.all { text-spacing-trim: space-all; }
::selection { background-color: #00ff00; }`)}, "han.ttf": {Data: face}}, "page.css"))
			d.Body().Append(c.field)
			size := geom.V2(400, 130)
			landed(t, d, size)
			d.Focus(c.field)
			d.SetSelectionRange(c.field, c.from, c.to)
			d.Layout(size)
			var page bytes.Buffer
			require.NoError(t, export.Write(&page, d))
			out := filepath.Join(t.TempDir(), "field.png")
			require.NoError(t, chrome.Screenshot(page.Bytes(), size, out))
			f, openErr := os.Open(out)
			require.NoError(t, openErr)
			img, decodeErr := png.Decode(f)
			f.Close()
			require.NoError(t, decodeErr)

			if c.from == c.to {
				caret := ui.FieldCaretRect(d, c.field)
				want := spans(img, int(caret.Y+caret.H/2), caretInk)
				require.Len(t, want, 1, "Chrome's caret")
				assert.InDelta(t, float32(want[0][0]), caret.X, 1, "the caret stands where Chrome's does")
				return
			}
			got := ui.FieldSelectionRects(d, c.field)
			require.NotEmpty(t, got)
			for _, r := range got {
				y := int(r.Y + r.H/2)
				want := spans(img, y, selectedInk)
				require.Len(t, want, 1, "row %d: ember %v, Chrome %v", y, r, want)
				assert.InDelta(t, float32(want[0][0]), r.X, 1, "row %d starts where Chrome's does", y)
				assert.InDelta(t, float32(want[0][1]), r.X+r.W, 1, "row %d ends where Chrome's does", y)
			}
		})
	}
}
