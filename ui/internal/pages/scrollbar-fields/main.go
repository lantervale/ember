// Textareas keeping scrollbar-gutter: stable both-edges: their text
// between a gutter at each edge, its lines overflowing or not, left to
// right and right to left, their bars styled or ui's own.
package main

import (
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; padding: 8px; }
textarea { display: inline-block; vertical-align: top; width: 140px; height: 90px; margin: 4px; padding: 2px;
  border: 1px solid #6b8f5a; background-color: #e8e0d0; color: #3a332c; resize: none;
  scrollbar-gutter: stable both-edges; }
.styled::-webkit-scrollbar { width: 10px; background-color: #5a1f1f; }
.styled::-webkit-scrollbar-thumb { background-color: #4f7cff; }
.stable { scrollbar-gutter: stable; }
.rtl { direction: rtl; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	long, short := strings.Repeat("A line of text that wraps. ", 8), "Short"
	for _, c := range []struct {
		class, text string
	}{
		{"styled", long}, {"styled", short}, {"styled rtl", long}, {"styled stable", short},
		{"", "Short and plain"}, {"rtl", short},
	} {
		d.Body().Append(ui.Textarea(ui.Class(strings.Fields(c.class)...), ui.Text(c.text)))
	}
	pages.Run(d, 480, 230)
}
