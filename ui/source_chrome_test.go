//go:build chrome

package ui_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAnImgIsSizedByTheAttributesOfTheSourceItChoseAsChromeSizesIt(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 40, 30))))
	files := fstest.MapFS{"a.png": {Data: b.Bytes()}, "b.png": {Data: b.Bytes()}}
	attrs := func(kv ...string) []ui.Arg {
		var args []ui.Arg
		for i := 0; i < len(kv); i += 2 {
			args = append(args, ui.Attr(kv[i], kv[i+1]))
		}
		return args
	}
	picture := func(source, img []string, media string) *ui.Node {
		s := attrs(append([]string{"srcset", "a.png"}, source...)...)
		if media != "" {
			s = append(s, ui.Attr("media", media))
		}
		return ui.Picture(ui.Source(s...), ui.Img(attrs(append([]string{"src", "b.png"}, img...)...)...))
	}
	d := ui.NewDocument(ui.Images(files), ui.CSS("picture, img { display: block; } source { display: none; }"))
	d.Body().Append(
		picture([]string{"width", "100", "height", "50"}, []string{"width", "20", "height", "20"}, ""),
		picture([]string{"width", "100"}, []string{"width", "20", "height", "30"}, ""),
		picture([]string{"height", "60"}, []string{"width", "70"}, ""),
		picture(nil, []string{"width", "20", "height", "30"}, ""),
		picture([]string{"width", "100", "height", "50"}, []string{"width", "20", "height", "30"}, "(max-width: 10px)"),
		picture([]string{"width", "50%", "height", "12.5"}, nil, ""),
		ui.Img(attrs("src", "b.png", "width", "15", "height", "15")...),
	)
	chrome.Match(t, d, geom.V2(400, 600))
}
