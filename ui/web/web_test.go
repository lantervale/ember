package web_test

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/web"
)

// art serves a picture of each kind, a page that is none and a file too
// big, and notes the paths it was asked for.
func art(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var (
		mu    sync.Mutex
		paths []string
	)
	pic := image.NewRGBA(image.Rect(0, 0, 30, 20))
	encoded := map[string]*bytes.Buffer{"png": {}, "jpg": {}, "gif": {}}
	require.NoError(t, png.Encode(encoded["png"], pic))
	require.NoError(t, jpeg.Encode(encoded["jpg"], pic, nil))
	require.NoError(t, gif.Encode(encoded["gif"], pic, nil))
	mux := http.NewServeMux()
	for ext, data := range encoded {
		mux.HandleFunc("/art/door."+ext, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data.Bytes()) })
	}
	mux.HandleFunc("/art/huge.png", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, web.MaxBytes+1))
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &paths
}

func TestImagesResolvesAReferenceFromItsBase(t *testing.T) {
	srv, paths := art(t)
	fetch := web.Images(srv.URL + "/pages/crypt.html")
	for _, ref := range []string{"../art/door.png", "/art/door.png", srv.URL + "/art/door.png"} {
		data, err := fetch(ref)
		require.NoError(t, err, ref)
		assert.NotEmpty(t, data, ref)
	}
	assert.Equal(t, []string{"/art/door.png", "/art/door.png", "/art/door.png"}, *paths)
}

func TestImagesSaysWhyThereIsNoPicture(t *testing.T) {
	srv, _ := art(t)
	fetch := web.Images(srv.URL + "/")
	_, err := fetch("art/gone.png")
	assert.ErrorContains(t, err, "404")
	_, err = fetch("art/huge.png")
	assert.ErrorContains(t, err, "over")
	_, err = fetch("file:///etc/passwd")
	assert.ErrorContains(t, err, "no http or https address")
	_, err = web.Images("crypt/pages")("door.png")
	assert.ErrorContains(t, err, "no http or https address")
}

func TestADocumentShowsThePicturesOfTheWeb(t *testing.T) {
	srv, _ := art(t)
	d := ui.NewDocument(ui.FetchImages(web.Images(srv.URL + "/")))
	doors := []*ui.Node{
		ui.Img(ui.Attr("src", "art/door.png")), ui.Img(ui.Attr("src", "art/door.jpg")),
		ui.Img(ui.Attr("src", "art/door.gif")),
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
	assert.Eventually(t, sized, 5*time.Second, 5*time.Millisecond, "png, jpeg and gif land and take their size")
	assert.Empty(t, d.Warnings())
}

func TestAnSVGIsAskedForAndHandedOverAsItIs(t *testing.T) {
	const file = "<?xml version=\"1.0\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"8\" height=\"8\"/>\n"
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(file))
	}))
	t.Cleanup(srv.Close)
	data, err := web.Images(srv.URL)("star")
	require.NoError(t, err)
	assert.Equal(t, file, string(data), "the bytes as served, for the format ui/svg registers")
	assert.Contains(t, accept, "image/svg+xml")
	assert.Contains(t, accept, "image/png")
}
