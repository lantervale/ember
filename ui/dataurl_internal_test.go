package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestADataURLIsItsOwnPicture(t *testing.T) {
	var file bytes.Buffer
	require.NoError(t, png.Encode(&file, image.NewRGBA(image.Rect(0, 0, 3, 2))))
	fetched := 0
	d := NewDocument(FetchImages(func(string) ([]byte, error) { fetched++; return nil, nil }))

	tex := d.image("data:image/png;base64," + base64.StdEncoding.EncodeToString(file.Bytes()))
	require.NotNil(t, tex, "read from the address itself")
	assert.Equal(t, image.Pt(3, 2), tex.Image.Bounds().Size())
	assert.Zero(t, fetched, "never asked of the fetcher")
	assert.Empty(t, d.Warnings())
}

func TestADataURLIsReadAsItIsWritten(t *testing.T) {
	for url, want := range map[string]string{
		"data:text/plain;base64,SGVyb24=":                "Heron",
		"data:image/svg+xml;utf8,<svg fill=\"%23000\"/>": `<svg fill="#000"/>`,
		"data:image/svg+xml,%3Csvg%2F%3E":                "<svg/>",
		"DATA:,a%20b":                                    "a b",
		"data:image/svg+xml;charset=utf-8;base64,PGc+":   "<g>",
	} {
		got, ok := dataURL(url)
		assert.True(t, ok, url)
		assert.Equal(t, want, string(got), url)
	}
	for _, url := range []string{"data:image/png;base64,***", "data:image/png", "https://x/data:,a", "a.png"} {
		_, ok := dataURL(url)
		assert.False(t, ok, url)
	}
}

func TestADataURLThatIsNoPictureIsAWarning(t *testing.T) {
	d := NewDocument()
	assert.Nil(t, d.image("data:image/png;base64,SGVyb24="))
	assert.Len(t, d.Warnings(), 1)
}
