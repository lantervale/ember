//go:build chrome

package layout_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// widePNG is a 400×200 picture, wider than the tracks it is put in.
func widePNG() []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 400, 200))); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func TestPicturesSizedByTheirContainerMatchChrome(t *testing.T) {
	const sheet = `
.grid { display: grid; grid-template-columns: repeat(6, 1fr); column-gap: 10px; width: 600px; margin-bottom: 8px }
.lead { grid-column: 1 / span 4 } .side { grid-column: 5 / span 2 }
.full { width: 100% } .most { max-width: 100% } .half { width: 50% }
.card { position: relative; line-height: 0 } .card img { aspect-ratio: 16 / 9; object-fit: cover }
.cover { position: absolute; width: 100%; object-fit: contain }
.row { display: flex; width: 300px; margin-bottom: 8px } .row div { flex: 1 }
.float { float: left; margin-bottom: 8px } .clear { clear: both }
.inline { display: inline-block }
.fit { display: grid; grid-template-columns: repeat(2, 1fr); width: 100%; min-width: fit-content }
.fit > div { grid-column: 1 / span 2; display: grid; grid-template-columns: 1fr }`
	cases := map[string]func(src ui.Arg) []*ui.Node{
		"a full width picture in fr tracks": func(src ui.Arg) []*ui.Node {
			return []*ui.Node{ui.Div(ui.Class("grid"),
				ui.Div(ui.Class("lead"), ui.Text("lead")),
				ui.Div(ui.Class("side"), ui.Div(ui.Img(src, ui.Class("full"))), ui.Text("side")))}
		},
		"a placeholder over a picture": func(src ui.Arg) []*ui.Node {
			return []*ui.Node{ui.Div(ui.Class("grid"),
				ui.Div(ui.Class("lead"), ui.Div(ui.Class("card"), ui.Img(src, ui.Class("cover")), ui.Img(src, ui.Class("full")))),
				ui.Div(ui.Class("side"), ui.Div(ui.Class("card"), ui.Img(src, ui.Class("cover")), ui.Img(src, ui.Class("full")))))}
		},
		"in a grid no narrower than its content": func(src ui.Arg) []*ui.Node {
			card := func() *ui.Node {
				return ui.Div(ui.Class("fit"), ui.Div(ui.Div(ui.Class("card"), ui.Img(src, ui.Class("cover")),
					ui.Img(src, ui.Class("full"))), ui.Text("a headline under the picture")))
			}
			return []*ui.Node{ui.Div(ui.Class("grid"), ui.Div(ui.Class("lead"), card()), ui.Div(ui.Class("side"), card()))}
		},
		"a picture no wider than its track": func(src ui.Arg) []*ui.Node {
			return []*ui.Node{ui.Div(ui.Class("grid"),
				ui.Div(ui.Class("lead"), ui.Img(src, ui.Class("half"))),
				ui.Div(ui.Class("side"), ui.Img(src, ui.Class("most"))))}
		},
		"in flex items, a float and an inline block": func(src ui.Arg) []*ui.Node {
			return []*ui.Node{
				ui.Div(ui.Class("row"), ui.Div(ui.Img(src, ui.Class("full"))), ui.Div(ui.Img(src, ui.Class("most")))),
				ui.Div(ui.Class("float"), ui.Img(src, ui.Class("full"))),
				ui.Div(ui.Class("clear")),
				ui.Div(ui.Class("inline"), ui.Img(src, ui.Class("half"))),
			}
		},
	}
	for name, nodes := range cases {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte(sheet)}, "wide.png": {Data: widePNG()}},
				"s.css"))
			d.Body().Append(nodes(ui.Attr("src", "wide.png"))...)
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}
