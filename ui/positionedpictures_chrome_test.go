//go:build chrome

package ui_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"

	"github.com/lantervale/ember/ui"
)

// widePicture is a 40×16 PNG as a data URL: wider than tall, so a picture turned shows.
func widePicture(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 40, 16))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func TestPicturesLeavingVerticalBoxesAsChrome(t *testing.T) {
	wide := widePicture(t)
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { width: 120px; height: 90px; padding: 3px 7px 5px 2px; border: 2px solid;
				margin: 10px 20px; writing-mode: vertical-rl; } .v > p { margin: 0; inline-size: 30px; block-size: 20px; }
				.a { position: absolute; margin: 1px 4px 2px 3px; background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(), ui.Img(ui.Class("a"), ui.Attr("src", wide)), ui.P()))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"of its natural size":                page(``),
		"of its natural size in vertical-lr": page(`.v { writing-mode: vertical-lr; }`),
		"of its natural size in sideways-rl": page(`.v { writing-mode: sideways-rl; }`),
		"of its natural size in sideways-lr": page(`.v { writing-mode: sideways-lr; }`),
		"of a width alone":                   page(`.a { width: 60px; }`),
		"of a height alone":                  page(`.a { height: 30px; }`),
		"of an inline size alone":            page(`.a { inline-size: 30px; }`),
		"of a block size alone in vertical-lr": page(`.v { writing-mode: vertical-lr; }
			.a { block-size: 30px; }`),
		"of a width and an aspect-ratio":         page(`.a { width: 30px; aspect-ratio: 1 / 2; }`),
		"of a height and an aspect-ratio":        page(`.a { height: 30px; aspect-ratio: 3; }`),
		"of an aspect-ratio alone":               page(`.a { aspect-ratio: 1; }`),
		"of a max-width":                         page(`.a { max-width: 20px; }`),
		"of a max-height":                        page(`.a { max-height: 8px; }`),
		"of a min-width":                         page(`.a { min-width: 80px; }`),
		"of a min-height in vertical-lr":         page(`.v { writing-mode: vertical-lr; } .a { min-height: 40px; }`),
		"of a max-inline-size":                   page(`.a { max-inline-size: 10px; }`),
		"stretched by its insets":                page(`.a { left: 10px; right: 50px; }`),
		"stretched by its insets in vertical-lr": page(`.v { writing-mode: vertical-lr; } .a { top: 10px; bottom: 40px; }`),
		"of an object-fit":                       page(`.a { width: 30px; height: 30px; object-fit: contain; }`),
		"of padding and a border":                page(`.a { padding: 2px 5px 1px 3px; border: 3px solid; }`),
		"held by a horizontal box":               page(`.a { writing-mode: horizontal-tb; }`),
	})
}

func TestReplacedBoxesLeavingVerticalBoxesAsChrome(t *testing.T) {
	page := func(sheet string, el func() *ui.Node) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { width: 120px; height: 90px; border: 2px solid; margin: 10px 20px;
				writing-mode: vertical-rl; } .v > p { margin: 0; inline-size: 30px; block-size: 20px; }
				.a { position: absolute; border: 0; background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(), el(), ui.P()))
			return d
		}
	}
	canvas := func() *ui.Node { return ui.Canvas(nil, ui.Class("a"), ui.Attr("width", "50"), ui.Attr("height", "20")) }
	frame := func() *ui.Node { return ui.Iframe(ui.NewDocument(), ui.Class("a")) }
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"a canvas":                     page(``, canvas),
		"a canvas of a height":         page(`.a { height: 40px; }`, canvas),
		"a canvas in vertical-lr":      page(`.v { writing-mode: vertical-lr; }`, canvas),
		"an iframe":                    page(``, frame),
		"an iframe of a width":         page(`.a { width: 100px; }`, frame),
		"an iframe in sideways-lr":     page(`.v { writing-mode: sideways-lr; }`, frame),
		"an iframe of an aspect-ratio": page(`.a { width: 60px; aspect-ratio: 2; }`, frame),
	})
}

func TestPicturesLeavingHorizontalBoxesOfVerticalPagesAsChrome(t *testing.T) {
	wide := widePicture(t)
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`html { writing-mode: vertical-rl; } .h { writing-mode: horizontal-tb;
				width: 90px; height: 120px; border: 2px solid; margin: 10px 20px; } .h > p { margin: 0; height: 20px; }
				.a { position: absolute; background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("h"), ui.P(), ui.Img(ui.Class("a"), ui.Attr("src", wide)), ui.P()))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"of its natural size":                page(``),
		"of its natural size in vertical-lr": page(`html { writing-mode: vertical-lr; }`),
		"of a width alone":                   page(`.a { width: 20px; }`),
		"of a height alone":                  page(`.a { height: 32px; }`),
		"of a max-height":                    page(`.a { max-height: 8px; }`),
		"of an aspect-ratio and a width":     page(`.a { width: 30px; aspect-ratio: 3 / 2; }`),
		"by its insets":                      page(`.a { bottom: 4px; left: 6px; }`),
	})
}

func TestPicturesPlacedInTheirOwnFrameAsChrome(t *testing.T) {
	wide := widePicture(t)
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.v { position: relative; width: 120px; height: 90px; border: 2px solid;
				margin: 10px 20px; writing-mode: vertical-rl; } .v > p { margin: 0; inline-size: 30px; block-size: 20px; }
				.h { writing-mode: horizontal-tb; width: 50px; height: 60px; }
				.a { position: absolute; background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(), ui.Div(ui.Class("h"),
				ui.Img(ui.Class("a"), ui.Attr("src", wide))), ui.P()))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"of its natural size":                page(``),
		"of its natural size in vertical-lr": page(`.v { writing-mode: vertical-lr; }`),
		"of a height alone":                  page(`.a { height: 32px; }`),
		"of a max-width":                     page(`.a { max-width: 20px; }`),
		"by its insets":                      page(`.a { top: 4px; left: 6px; }`),
		"held by a relative horizontal box":  page(`.h { position: relative; }`),
	})
}

func TestPositionedPicturesAsChrome(t *testing.T) {
	wide := widePicture(t)
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.c { position: relative; width: 120px; height: 90px; border: 2px solid;
				margin: 10px 20px; } .c > p { margin: 0; height: 20px; }
				.a { position: absolute; background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("c"), ui.P(), ui.Img(ui.Class("a"), ui.Attr("src", wide)), ui.P()))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"of a max-width":                page(`.a { max-width: 20px; }`),
		"of a max-height":               page(`.a { max-height: 8px; }`),
		"of a min-width":                page(`.a { min-width: 80px; }`),
		"of a min-height":               page(`.a { min-height: 40px; }`),
		"of an aspect-ratio alone":      page(`.a { aspect-ratio: 1; }`),
		"in vertical-rl":                page(`.a { writing-mode: vertical-rl; }`),
		"in vertical-rl of a height":    page(`.a { writing-mode: vertical-rl; height: 32px; }`),
		"in vertical-rl of a max-width": page(`.a { writing-mode: vertical-rl; max-width: 20px; }`),
	})
}

func TestOrthogonalPicturesInFlowAsChrome(t *testing.T) {
	wide := widePicture(t)
	page := func(sheet string) func(t *testing.T) *ui.Document {
		return func(_ *testing.T) *ui.Document {
			d := ui.NewDocument(ui.CSS(`.c { width: 120px; height: 90px; border: 2px solid; margin: 10px 20px; }
				img { background: teal; }` + sheet))
			d.Body().Append(ui.Div(ui.Class("c"), ui.Img(ui.Attr("src", wide)), ui.Text("x"), ui.Img(ui.Attr("src", wide))))
			return d
		}
	}
	matchVertical(t, map[string]func(t *testing.T) *ui.Document{
		"horizontal pictures in a vertical-rl box": page(`.c { writing-mode: vertical-rl; }
			img { writing-mode: horizontal-tb; }`),
		"vertical-rl pictures in a horizontal box": page(`img { writing-mode: vertical-rl; }`),
		"vertical-lr pictures of a max-width in a horizontal box": page(`img { writing-mode: vertical-lr;
			max-width: 20px; }`),
		"horizontal pictures of a max-width in a vertical-lr box": page(`.c { writing-mode: vertical-lr; }
			img { writing-mode: horizontal-tb; max-width: 20px; }`),
		"pictures of an aspect-ratio alone": page(`img { aspect-ratio: 1; }`),
		"pictures of an aspect-ratio alone in a vertical-rl box": page(`.c { writing-mode: vertical-rl; }
			img { aspect-ratio: 2; }`),
		"pictures of an aspect-ratio and padding": page(`img { aspect-ratio: 1; padding: 2px 6px; }`),
		"pictures of an aspect-ratio, padding and border-box": page(`img { aspect-ratio: 1; padding: 2px 6px;
			box-sizing: border-box; }`),
		"pictures of an aspect-ratio auto and a ratio": page(`img { aspect-ratio: auto 1; }`),
	})
}
