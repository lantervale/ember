//go:build chrome

package ui_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/export"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// spans are the stretches of row y of img whose pixels are, as x from
// up to to.
func spans(img image.Image, y int, are func(r, g, b uint32) bool) [][2]int {
	var out [][2]int
	in := false
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		r, g, b, _ := img.At(x, y).RGBA()
		hit := are(r>>8, g>>8, b>>8)
		if hit && !in {
			out = append(out, [2]int{x, x})
		}
		if hit {
			out[len(out)-1][1] = x + 1
		}
		in = hit
	}
	return out
}

func selectedInk(r, g, b uint32) bool { return g > 200 && r < 60 && b < 60 }
func caretInk(r, g, b uint32) bool    { return r > 200 && g < 60 && b < 60 }

// TestAFieldsSelectionAndCaretStandWhereChromesDo selects in fields of
// text in both directions, in the Hebrew font both draw alike, its text
// clear so that only the selection, green, and the caret, red, show, and
// finds them in Chrome's screenshot, row by row: one stretch per run of
// selected characters side by side, the space where the text changes
// direction with the run it shows in, a tab's room, the caret of a
// right-to-left textarea where it is drawn, and none beside a selection.
func TestAFieldsSelectionAndCaretStandWhereChromesDo(t *testing.T) {
	font := hebrewFont(t)
	focused := ui.Attr("autofocus", "")
	rtl := ui.Attr("dir", "rtl")
	mixed := "שלום abc def עולם"
	for _, c := range []struct {
		name     string
		field    *ui.Node
		from, to int
	}{
		{"from hebrew into latin", ui.Input(focused, ui.Attr("value", mixed)), 2, 8},
		{"in a right-to-left input", ui.Input(focused, rtl, ui.Attr("value", mixed)), 2, 8},
		{"the space between the directions", ui.Input(focused, ui.Attr("value", "שלום abc")), 3, 6},
		{"in a right-to-left textarea", ui.Textarea(focused, rtl, ui.Text("abc שלום")), 1, 6},
		{"over two right-to-left lines", ui.Textarea(focused, rtl, ui.Text("abc שלום\nעולם def ghi")), 5, 14},
		{"over two lines", ui.Textarea(focused, ui.Text("abc def\nghi")), 5, 10},
		{"over an empty line", ui.Textarea(focused, ui.Text("abc\n\nghi")), 2, 7},
		{"a tab's room", ui.Textarea(focused, ui.Text("a\tb")), 0, 3},
		{"the caret of a right-to-left textarea", ui.Textarea(focused, rtl, ui.Text("abc שלום")), 6, 6},
		{"the caret of its second line", ui.Textarea(focused, rtl, ui.Text("abc שלום\nעולם def")), 11, 11},
		{"no caret beside a selection", ui.Input(focused, ui.Attr("value", "abc def")), 1, 4},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
				ui.CSS(`@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
body { margin: 0; padding: 10px; font-family: Hebrew; font-size: 20px; }
input, textarea { font: inherit; width: 300px; height: 60px; color: transparent; caret-color: #ff0000;
  outline: none; border: none; padding: 0; resize: none; }
::selection { background-color: #00ff00; }`))
			d.Body().Append(c.field)
			size := geom.V2(400, 100)
			landed(t, d, size)
			d.Focus(c.field)
			d.SetSelectionRange(c.field, c.from, c.to)
			d.Layout(size)
			var page bytes.Buffer
			require.NoError(t, export.Write(&page, d))
			out := filepath.Join(t.TempDir(), "field.png")
			require.NoError(t, chrome.Screenshot(page.Bytes(), size, out))
			f, err := os.Open(out)
			require.NoError(t, err)
			img, err := png.Decode(f)
			f.Close()
			require.NoError(t, err)

			if c.from != c.to {
				got := ui.FieldSelectionRects(d, c.field)
				require.NotEmpty(t, got)
				rows := map[int][]geom.Rect{}
				for _, r := range got {
					y := int(r.Y + r.H/2)
					rows[y] = append(rows[y], r)
				}
				for y, rs := range rows {
					want := spans(img, y, selectedInk)
					require.Len(t, want, len(rs), "row %d: ember %v, Chrome %v", y, rs, want)
					for i, w := range want {
						assert.InDelta(t, float32(w[0]), rs[i].X, 1, "row %d, stretch %d starts where Chrome's does", y, i)
						assert.InDelta(t, float32(w[1]), rs[i].X+rs[i].W, 1, "row %d, stretch %d ends where Chrome's does", y, i)
					}
				}
				assert.Empty(t, spans(img, int(got[0].Y+got[0].H/2), caretInk), "Chrome draws no caret beside a selection")
				return
			}
			caret := ui.FieldCaretRect(d, c.field)
			want := spans(img, int(caret.Y+caret.H/2), caretInk)
			require.Len(t, want, 1, "Chrome's caret")
			assert.InDelta(t, float32(want[0][0]), caret.X, 1, "the caret stands where Chrome's does")
		})
	}
}
