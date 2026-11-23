package web_test

import (
	"bytes"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/web"
)

// webps serves the WebP files of testdata without saying what they are,
// under names that say nothing either, and notes the Accept header it
// was last asked with.
func webps(t *testing.T) (*httptest.Server, *string) {
	t.Helper()
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		data, err := os.ReadFile("testdata" + r.URL.Path + ".webp")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv, &accept
}

func TestADocumentShowsWebPPicturesByWhatTheyHold(t *testing.T) {
	srv, accept := webps(t)
	d := ui.NewDocument(ui.FetchImages(web.Images(srv.URL + "/")))
	doors := []*ui.Node{
		ui.Img(ui.Attr("src", "lossy")), ui.Img(ui.Attr("src", "lossless-alpha")),
		ui.Img(ui.Attr("src", "lossy-alpha")),
	}
	d.Body().Append(doors...)
	sized := func() bool {
		d.Layout(geom.V2(400, 200))
		for _, door := range doors {
			if door.Box().W != 30 || door.Box().H != 20 {
				return false
			}
		}
		return true
	}
	assert.Eventually(t, sized, 5*time.Second, 5*time.Millisecond, "lossy, lossless and lossy with alpha land")
	assert.Empty(t, d.Warnings())
	assert.Contains(t, *accept, "image/webp")
	assert.NotContains(t, *accept, "avif")
}

func TestAWebPKeepsItsAlpha(t *testing.T) {
	for _, name := range []string{"lossless-alpha", "lossy-alpha"} {
		data, err := os.ReadFile("testdata/" + name + ".webp")
		require.NoError(t, err)
		pic, format, err := image.Decode(bytes.NewReader(data))
		require.NoError(t, err, name)
		assert.Equal(t, "webp", format)
		_, _, _, solid := pic.At(5, 10).RGBA()
		_, _, _, through := pic.At(25, 10).RGBA()
		assert.Equal(t, uint32(0xffff), solid, name)
		assert.Zero(t, through, name)
	}
}

func TestAFetchedAnimatedWebPLandsAsAPicture(t *testing.T) {
	srv, _ := webps(t)
	d := ui.NewDocument(ui.FetchImages(web.Images(srv.URL + "/")))
	img := ui.Img(ui.Attr("src", "animated"))
	d.Body().Append(img)
	landed := func() bool {
		d.Layout(geom.V2(400, 200))
		return img.Box().W > 16
	}
	require.Eventually(t, landed, 5*time.Second, 5*time.Millisecond)
	assert.Empty(t, d.Warnings())
}
