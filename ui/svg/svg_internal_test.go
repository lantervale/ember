package svg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rootsvg "github.com/lantervale/ember/svg"
)

func TestAnSVGIsKnownByItsNameOrByItsBytes(t *testing.T) {
	for name, want := range map[string]bool{
		"star.svg": true, "art/STAR.SVG": true, "https://a.test/star.svg?v=2#top": true,
		"star.png": false, "svg": false, "star.svgz": false, "": false,
	} {
		assert.Equal(t, want, is(name, nil), name)
	}
	for file, want := range map[string]bool{
		`<svg xmlns="http://www.w3.org/2000/svg"/>`: true,
		"<svg>":                             true,
		"\xef\xbb\xbf  \n<svg\n width='1'>": true,
		`<?xml version="1.0"?><svg/>`:       true,
		"<?xml version='1.0'?>\n<!-- made by hand -->\n<!-- twice -->\n<svg viewBox=''>":                     true,
		`<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/svg11.dtd"><svg>`:                 true,
		`<?xml version="1.0"?><!doctype svg [ <!ENTITY st0 "fill:#2a7;"> <!ENTITY b ">"> ]> <svg x="&st0;">`: true,
		`<?xml version="1.0"?><rss version="2.0"><channel/></rss>`:                                           false,
		`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><body><svg/></body></html>`:         false,
		`<!DOCTYPE html><html><svg></svg></html>`:                                                            false,
		"<svgs>": false, "<svg": false, "<?xml never closed <svg>": false, "<!-- never closed <svg>": false,
		"": false, "\x89PNG\r\n\x1a\n": false, "svg": false,
		"\xff\xfe<\x00s\x00v\x00g\x00>\x00": true, "\xfe\xff\x00<\x00s\x00v\x00g\x00>": true,
		"<\x00s\x00v\x00g\x00 \x00": true, "\x00<\x00?\x00x\x00?\x00>\x00<\x00s\x00v\x00g\x00/": true,
		"\xff\xfe<\x00r\x00s\x00s\x00>\x00": false, "\xff\xfe<\x00s\x01v\x00g\x00>\x00": false,
	} {
		assert.Equal(t, want, is("picture", []byte(file)), file)
	}
}

func TestEverySampleOfTheReaderIsKnownByItsBytes(t *testing.T) {
	files, err := filepath.Glob("../../svg/testdata/*.svg")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		data, bad := os.ReadFile(f)
		require.NoError(t, bad)
		assert.True(t, sniff(data), f)
	}
}

func TestAPictureIsDrawnAtTheSizeAskedUpToTheReadersLimits(t *testing.T) {
	p, err := parse([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2 1"><rect width="2" height="1"/></svg>`))
	require.NoError(t, err)
	img, err := p.Rasterise(40, 20, 1)
	require.NoError(t, err)
	assert.Equal(t, 40, img.Rect.Dx())
	assert.Equal(t, 20, img.Rect.Dy())
	assert.Equal(t, uint8(255), img.RGBAAt(20, 10).A)

	w, h, scale := fit(4*rootsvg.MaxSide, 8, 2)
	assert.Equal(t, [2]int{rootsvg.MaxSide, 2}, [2]int{w, h}, "the longest side drawn, the shape kept")
	assert.InDelta(t, 0.5, scale, 1e-9, "and its lengths in pixels as much smaller")
	w, h, _ = fit(rootsvg.MaxSide, rootsvg.MaxSide, 1)
	assert.LessOrEqual(t, w*h, rootsvg.MaxPixels)
	assert.Equal(t, w, h)
	_, err = p.Rasterise(4*rootsvg.MaxSide, 8, 1)
	require.NoError(t, err, "past them a smaller picture, never an error")

	bare, err := parse([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10"/></svg>`))
	require.NoError(t, err)
	img, err = bare.Rasterise(40, 40, 2)
	require.NoError(t, err)
	assert.Equal(t, uint8(255), img.RGBAAt(15, 15).A, "a file with no view box is drawn a unit a CSS pixel")
	assert.Equal(t, uint8(0), img.RGBAAt(25, 25).A)

	_, err = parse([]byte("<svg"))
	var format *rootsvg.FormatError
	assert.ErrorAs(t, err, &format)
}
