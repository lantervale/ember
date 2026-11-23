// Thai, Lao, Hangul and Hebrew shaping against Chrome's, in subsets of
// Noto Sans Thai, KR and Hebrew and Go's stand-ins for old faces with no
// layout tables (copies of internal/shape/testdata's): sara am
// decomposed, marks moved into the Private Use Area, jamo composed and
// tone marks moved, points drawn as presentation forms, a variation
// sequence's own glyph, and an Arabic word cut by elements joined.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

var (
	//go:embed noto-thai.ttf
	thai []byte
	//go:embed noto-hangul.ttf
	hangul []byte
	//go:embed noto-hebrew.ttf
	hebrew []byte
	//go:embed go-thai.ttf
	oldThai []byte
	//go:embed go-hebrew.ttf
	oldHebrew []byte
	//go:embed go-arabic.ttf
	arabic []byte
)

const hangulClass = "hangul"

const sheet = `
@font-face { font-family: Thai; src: url(thai.ttf); }
@font-face { font-family: Hangul; src: url(hangul.ttf); }
@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
@font-face { font-family: OldThai; src: url(old-thai.ttf); }
@font-face { font-family: OldHebrew; src: url(old-hebrew.ttf); }
@font-face { font-family: Joined; src: url(arabic.ttf); }
html { background-color: #fff; }
body { padding: 8px; font-size: 26px; color: #000; }
div { white-space: nowrap; margin-bottom: 4px; }
.thai { font-family: Thai; }
.hangul { font-family: Hangul; }
.hebrew { font-family: Hebrew; }
.old-thai { font-family: OldThai; }
.old-hebrew { font-family: OldHebrew; }
.joined { font-family: Joined; }
.big { font-size: 32px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":       {Data: []byte(sheet)},
		"thai.ttf":       {Data: thai},
		"hangul.ttf":     {Data: hangul},
		"hebrew.ttf":     {Data: hebrew},
		"old-thai.ttf":   {Data: oldThai},
		"old-hebrew.ttf": {Data: oldHebrew},
		"arabic.ttf":     {Data: arabic},
	}, "page.css"))
	for _, line := range [][2]string{
		{"thai", "ภาษาไทย น้ำ ทำ ก้ำ ปั่น"},
		{"old-thai", "ปั่น ญู ฐุ ที่ ก้ำ ป่"},
		{hangulClass, "한국어"},
		{hangulClass, "한글ᄀᆞᆫ"},
		{hangulClass, "가〮 가〯"},
		{"hebrew", "בְּרֵאשִׁית בָּרָא"},
		{"old-hebrew", "שׁ שּׁ בּ יִ אַ וֹ"},
		{"old-thai", "a︀b︁ aaa"},
		{"joined", "ب|big:ت|ب بب|big:ت"},
	} {
		var parts []ui.Arg
		for _, part := range strings.Split(line[1], "|") {
			if class, text, ok := strings.Cut(part, ":"); ok {
				parts = append(parts, ui.Span(ui.Class(class), ui.Text(text)))
			} else {
				parts = append(parts, ui.Text(part))
			}
		}
		d.Body().Append(ui.Div(append([]ui.Arg{ui.Class(line[0])}, parts...)...))
	}
	pages.Run(d, 420, 400)
}
