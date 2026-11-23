package ui_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gomono"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// A word of ASCII its first face lacks draws with the next, whose
// ascent stretches a line-height: normal line as any other text's
// does: Noto Sans Devanagari's line at 24px is 22 + 10 points, Go Mono
// rises 23 above the baseline, so "Hello" in Deva, Mono is 33 tall, as
// in Chrome.
func TestAnASCIIWordItsFirstFaceLacksStretchesTheLineWithItsFallback(t *testing.T) {
	deva, err := os.ReadFile(filepath.Join("..", "internal", "shape", "testdata", "noto-devanagari.ttf"))
	require.NoError(t, err)
	files := map[string][]byte{"deva.ttf": deva, "mono.ttf": gomono.TTF}
	d := ui.NewDocument(ui.FetchFonts(func(url, _ string) ([]byte, error) { return files[url], nil }), ui.CSS(`
@font-face { font-family: Deva; src: url(deva.ttf); }
@font-face { font-family: Mono; src: url(mono.ttf); }
span { display: inline-block; font: 24px Deva, Mono; }`))
	deva1, hello := ui.Span(ui.Text("नमस्ते")), ui.Span(ui.Text("Hello"))
	d.Body().Append(ui.Div(deva1), ui.Div(hello))
	size := geom.V2(400, 300)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.FontsLoading() == 0
	}, 5*time.Second, 5*time.Millisecond)
	d.Layout(size)
	assert.InDelta(t, 32, deva1.Box().H, 0.01, "Devanagari alone")
	assert.InDelta(t, 33, hello.Box().H, 0.01, "Latin from Go Mono")
}
