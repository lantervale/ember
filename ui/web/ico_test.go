package web_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/ico/icotest"
	"github.com/lantervale/ember/ui/web"
)

// bitmap is a w by h bitmap of 24 bits a pixel, red but for a blue
// first pixel, its last one cut by the AND mask where it is an icon's.
func bitmap(w, h int) icotest.Bitmap {
	return icotest.Bitmap{
		W: w, H: h, BPP: 24,
		At: func(x, y int) uint32 {
			if x == 0 && y == 0 {
				return 0x0000ff
			}
			return 0xff0000
		},
		Hole: func(x, y int) bool { return x == w-1 && y == h-1 },
	}
}

func bmpFile(w, h int) []byte { return bitmap(w, h).File() }

// icoFile is an ICO file of two pictures: one of 16 pixels, and one w
// by h, which a reader shows.
func icoFile(w, h int) []byte { return icotest.ICO(bitmap(16, 16).Entry(), bitmap(w, h).Entry()) }

func TestTheImagePackageReadsICOAndBMPByTheirFirstBytes(t *testing.T) {
	pic, format, err := image.Decode(bytes.NewReader(icoFile(30, 20)))
	require.NoError(t, err)
	assert.Equal(t, "ico", format)
	assert.Equal(t, image.Rect(0, 0, 30, 20), pic.Bounds())
	assert.Equal(t, color.NRGBA{0, 0, 255, 255}, pic.At(0, 0))
	assert.Equal(t, color.NRGBA{255, 0, 0, 255}, pic.At(1, 0))
	assert.Equal(t, color.NRGBA{}, pic.At(29, 19), "cut by the AND mask")

	pic, format, err = image.Decode(bytes.NewReader(bmpFile(30, 20)))
	require.NoError(t, err)
	assert.Equal(t, "bmp", format)
	assert.Equal(t, image.Rect(0, 0, 30, 20), pic.Bounds())
	assert.Equal(t, color.NRGBA{0, 0, 255, 255}, pic.At(0, 0))
	assert.Equal(t, color.NRGBA{255, 0, 0, 255}, pic.At(29, 19))
}

func TestADocumentShowsFetchedICOAndBMPPicturesByWhatTheyHold(t *testing.T) {
	files := map[string][]byte{"/favicon.ico": icoFile(30, 20), "/door": bmpFile(30, 20)}
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	d := ui.NewDocument(ui.FetchImages(web.Images(srv.URL + "/page/")))
	pics := []*ui.Node{ui.Img(ui.Attr("src", "/favicon.ico")), ui.Img(ui.Attr("src", "/door"))}
	d.Body().Append(pics...)
	sized := func() bool {
		d.Layout(geom.V2(400, 200))
		for _, pic := range pics {
			if pic.Box().W != 30 || pic.Box().H != 20 {
				return false
			}
		}
		return true
	}
	assert.Eventually(t, sized, 5*time.Second, 5*time.Millisecond, "the icon and the bitmap land")
	assert.Empty(t, d.Warnings())
	for _, kind := range []string{"image/x-icon", "image/vnd.microsoft.icon", "image/bmp"} {
		assert.Contains(t, accept, kind)
	}
	assert.NotContains(t, accept, "avif")
}

func TestADocumentShowsICOAndBMPFilesOfItsOwn(t *testing.T) {
	d := ui.NewDocument(ui.Images(fstest.MapFS{
		"favicon.ico": {Data: icoFile(24, 18)}, "door.bmp": {Data: bmpFile(12, 8)},
	}))
	icon, door := ui.Img(ui.Attr("src", "favicon.ico")), ui.Img(ui.Attr("src", "door.bmp"))
	d.Body().Append(icon, door)
	d.Layout(geom.V2(400, 200))
	assert.Equal(t, geom.V2(24, 18), geom.V2(icon.Box().W, icon.Box().H))
	assert.Equal(t, geom.V2(12, 8), geom.V2(door.Box().W, door.Box().H))
	assert.Empty(t, d.Warnings())
}

func TestABrokenICOIsAWarning(t *testing.T) {
	data := icoFile(30, 20)
	binary.LittleEndian.PutUint32(data[6+16+12:], uint32(len(data)))
	d := ui.NewDocument(ui.Images(fstest.MapFS{"favicon.ico": {Data: data}}))
	icon := ui.Img(ui.Attr("src", "favicon.ico"))
	d.Body().Append(icon)
	d.Layout(geom.V2(400, 200))
	require.Len(t, d.Warnings(), 1)
	assert.ErrorContains(t, d.Warnings()[0], "ico: a picture is outside the file")
}
