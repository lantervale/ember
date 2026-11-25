// Text no face of the engine has, drawn with the machine's faces under
// ui.SystemFonts as Chrome asks macOS for them: Japanese in its language
// and without one, Korean, Arabic joined right to left, bold Japanese,
// and Japanese and Chinese in serif.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; font-size: 20px; color: #000; }
p { margin: 0 0 6px; white-space: nowrap; }
.bold { font-weight: bold; }
.serif { font-family: serif; }
`

func main() {
	d := ui.NewDocument(ui.SystemFonts(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	for _, line := range []struct{ lang, class, text string }{
		{"ja", "", "日本語のテキスト、「漢字」とかな。"},
		{"", "", "ひらがなとカタカナ、漢字。"},
		{"ko", "", "한국어 텍스트입니다."},
		{"ar", "", "النص العربي هنا"},
		{"ja", "bold", "太字の日本語です。"},
		{"ja", "serif", "明朝体の日本語。"},
		{"zh-Hans", "serif", "简体中文的宋体。"},
	} {
		args := []ui.Arg{ui.Text(line.text)}
		if line.lang != "" {
			args = append(args, ui.Attr("lang", line.lang))
		}
		if line.class != "" {
			args = append(args, ui.Class(line.class))
		}
		d.Body().Append(ui.P(args...))
	}
	fonts.Of(d).Await()
	pages.Run(d, 360, 260)
}
