//go:build chrome

package ui_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"runtime"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// cjkFont is Go Regular with the ideographs from 一 drawn by its Latin capitals, 一 as
// A, 丁 as B and on: a font of ideographs, set upright in vertical lines, that ember and
// Chrome measure alike.
func cjkFont(t testing.TB) []byte {
	t.Helper()
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	glyphs := map[rune]uint16{}
	for r := rune(0x20); r < 0x2000; r++ {
		if g, lookupErr := f.GlyphIndex(&buf, r); lookupErr == nil && g != 0 {
			glyphs[r] = uint16(g)
		}
	}
	for i := range rune(26) {
		glyphs[0x4e00+i] = glyphs['A'+i]
	}
	return withCmap(t, goregular.TTF, cmap4(glyphs))
}

func cjkDocument(t *testing.T, sheet string) *ui.Document {
	t.Helper()
	font := cjkFont(t)
	return ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(`@font-face { font-family: CJK; src: url(cjk.ttf); } body { font-family: CJK; }`+sheet))
}

// wordSpans is each word of ws in a span of its own, a space between.
func wordSpans(ws ...string) []ui.Arg {
	var out []ui.Arg
	for i, w := range ws {
		if i > 0 {
			out = append(out, ui.Text(" "))
		}
		out = append(out, ui.Span(ui.Text(w)))
	}
	return out
}

func TestVerticalTextOrientationMatchesChrome(t *testing.T) {
	cases := map[string]func(t *testing.T) *ui.Document{
		"ideographs stand upright among turned Latin": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; height: 160px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("一丁丂七", "abc", "丄丅", "Latin", "丆万丈三上下")...)))
			return d
		},
		"upright stands every letter": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-lr; text-orientation: upright; height: 160px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("一丁丂", "abc", "Hi")...)))
			return d
		},
		"text-combine-upright sets digits across one character's room": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; height: 200px; } .c { text-combine-upright: all; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(ui.Span(ui.Text("一丁")),
				ui.Span(ui.Class("c"), ui.Text("12")), ui.Span(ui.Text("丂七")),
				ui.Span(ui.Class("c"), ui.Text("2024")), ui.Span(ui.Text("abc")))))
			return d
		},
		"sideways turns the ideographs too": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; text-orientation: sideways; height: 160px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("一丁丂七", "abc", "丄丅")...)))
			return d
		},
		"UAX 50 stands the symbols among turned letters": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; height: 300px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("a©b", "§2", "±x×y÷z", "¼½", "®™", "一«丁»")...)))
			return d
		},
	}
	matchVertical(t, cases)
}

func TestVerticalGlyphsMatchChrome(t *testing.T) {
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"a face's vertical advances set its ideographs": func(t *testing.T) *ui.Document {
			d := verticalDocument(t, false, `.v { writing-mode: vertical-rl; height: 200px; font-size: 20px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("一丁丂七", "abc", "丄丅丆", "万丈三上下")...)))
			return d
		},
		"brackets and stops in their vertical alternates": func(t *testing.T) *ui.Document {
			d := verticalDocument(t, false, `.v { writing-mode: vertical-lr; height: 200px; font-size: 20px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("「一丁」", "丂、七。", "「abc」")...)))
			return d
		},
		"VORG's origins leave the advances": func(t *testing.T) *ui.Document {
			d := verticalDocument(t, true, `.v { writing-mode: vertical-rl; height: 200px; font-size: 20px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(wordSpans("「一丁」", "丂七。")...)))
			return d
		},
	})
}

func TestVerticalLinesAlignAsChrome(t *testing.T) {
	sized := func(mode string) func(t *testing.T) *ui.Document {
		return func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: `+mode+`; height: 240px; } .big { font-size: 31px; }
				.small { font-size: 11px; } .mid { font-size: 23px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(ui.Text("一丁 ab "), ui.Span(ui.Class("big"), ui.Text("丂七 cd")),
				ui.Text(" 丄 "), ui.Span(ui.Class("small"), ui.Text("ef 丅丆")), ui.Text(" gh "),
				ui.Span(ui.Class("mid"), ui.Text("万 ij")), ui.Text(" 丈三上下 klm no pq"))))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"several font sizes on the central baseline in vertical-rl": sized("vertical-rl"),
		"several font sizes on the central baseline in vertical-lr": sized("vertical-lr"),
		"several font sizes upright": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; text-orientation: upright; height: 240px; }
				.big { font-size: 29px; } .small { font-size: 9px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(ui.Text("ab "), ui.Span(ui.Class("big"), ui.Text("cd")),
				ui.Text(" ef "), ui.Span(ui.Class("small"), ui.Text("gh")))))
			return d
		},
		"several font sizes sideways on the alphabetic baseline": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { writing-mode: vertical-rl; text-orientation: sideways; height: 240px; }
				.big { font-size: 29px; } .small { font-size: 9px; }`)
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(ui.Text("ab "), ui.Span(ui.Class("big"), ui.Text("cd")),
				ui.Text(" ef "), ui.Span(ui.Class("small"), ui.Text("gh")))))
			return d
		},
		"Go's letters in a vertical line of several sizes": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { writing-mode: vertical-rl; height: 240px; font-size: 15px; }
				.big { font-size: 27px; } .small { font-size: 10px; }`))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(ui.Text("Some "), ui.Span(ui.Class("big"), ui.Text("big")),
				ui.Text(" and "), ui.Span(ui.Class("small"), ui.Text("small words")), ui.Text(" on lines going down."))))
			return d
		},
		"inline flex and grid boxes of no line synthesize a central baseline": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { writing-mode: vertical-rl; width: 100px; height: 160px; line-height: 0;
				background: #c33; } .ib { display: inline-block; width: 100px; height: 40px; background: #3a3; }
				.f { display: inline-flex; } .g { display: inline-grid; } .f > div, .g > div { width: 60px; height: 40px;
				background: #36c; } .pad { padding-left: 12px; }`))
			d.Body().Append(ui.Div(ui.Class("v"), ui.Span(ui.Class("ib")), ui.Span(ui.Class("f"), ui.Div()),
				ui.Span(ui.Class("g", "pad"), ui.Div())))
			return d
		},
		"inline-blocks in lines taller than them": func(t *testing.T) *ui.Document {
			d := cjkDocument(t, `.v { height: 200px; line-height: 40px; } .rl { writing-mode: vertical-rl; }
				.lr { writing-mode: vertical-lr; } .ib { display: inline-block; inline-size: 30px; block-size: 12px;
				background: #3a6fd8; } .tall { block-size: 20px; vertical-align: top; }
				.low { vertical-align: bottom; } .odd { block-size: 13px; margin: 2px 5px 1px 3px; }
				.text { display: inline-block; font-size: 11px; background: #f0c040; }`)
			for _, mode := range []string{"rl", "lr"} {
				d.Body().Append(ui.Div(ui.Class("v", mode), ui.P(ui.Text("一丁 ab "), ui.Span(ui.Class("ib")),
					ui.Text(" 丂 "), ui.Span(ui.Class("ib", "tall")), ui.Text(" cd "), ui.Span(ui.Class("ib", "low")),
					ui.Text(" 七丄 "), ui.Span(ui.Class("ib", "odd")), ui.Text(" ef "),
					ui.Span(ui.Class("text"), ui.Text("万ab")), ui.Text(" gh 丅丆 ij"))))
			}
			return d
		},
	})
}

// fallsBack names the case whose text the test face lacks a character
// of (™, past cjkFont's range), drawn from the system's fallback face:
// Chrome on Linux's sets the standing ™ a point shorter than ui's.
const fallsBack = "UAX 50 stands the symbols among turned letters"

func matchVertical(t *testing.T, cases map[string]func(t *testing.T) *ui.Document) {
	t.Helper()
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			d := build(t)
			size := geom.V2(800, 600)
			landed(t, d, size)
			within := float32(chrome.Tolerance)
			if runtime.GOOS != "darwin" && name == fallsBack {
				within = 1
			}
			r := chrome.MatchWithin(t, d, size, within)
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestVerticalFlexPicturesAsChrome(t *testing.T) {
	pic := func(w, h int) string {
		var b bytes.Buffer
		if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
			t.Fatal(err)
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	}
	square, tall := pic(100, 100), pic(20, 50)
	page := func(sheet, src string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(sheet))
			d.Body().Append(ui.Div(ui.Class("f"), ui.Img(ui.Attr("src", src)), ui.Div(ui.Class("abs"))))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"a picture in a vertical-lr flex row": page(`.f { writing-mode: vertical-lr; display: flex; }`, square),
		"a picture in a vertical-rl flex row": page(`.f { writing-mode: vertical-rl; display: flex; }`, square),
		"a picture of a set width in a vertical flex row of a set width": page(`.f { position: relative;
			display: flex; writing-mode: vertical-lr; width: 100px; align-items: start; } img { width: 40px; }
			.abs { position: absolute; left: 40px; width: 60px; height: 100px; }`, tall),
		"an orthogonal table as a flex item": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.f { display: flex; width: 0; } .t { display: table; writing-mode: vertical-rl;
				min-width: 0; width: 100px; height: 100px; } .c { display: table-cell; } .w { width: 100px; }`))
			d.Body().Append(ui.Div(ui.Class("f"), ui.Div(ui.Class("t"), ui.Div(ui.Class("c"), ui.Div(ui.Class("w"))))))
			return d
		},
		"a table as a flex column's item": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.f { display: flex; flex-direction: column; height: 0; } .t { display: table;
				min-height: 0; width: 100px; height: 100px; } .c { display: table-cell; } .w { height: 100px; }`))
			d.Body().Append(ui.Div(ui.Class("f"), ui.Div(ui.Class("t"), ui.Div(ui.Class("c"), ui.Div(ui.Class("w"))))))
			return d
		},
		"a vertical flow in an inline flex box of a percentage height": func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.t { width: 100px; height: 100px; line-height: 0; }
				.f { display: inline-flex; height: 100%; } .v { writing-mode: vertical-rl; }
				span { display: inline-block; inline-size: 100px; block-size: 50px; }`))
			d.Body().Append(ui.Div(ui.Class("t"), ui.Div(ui.Class("f"), ui.Div(ui.Class("v"), ui.Span(), ui.Span()))))
			return d
		},
	})
}
