//go:build chrome

package ui_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// foundry is a font fetcher answering from the fixtures of the WOFF
// readers and Go's fonts.
func foundry(t *testing.T) ui.FontFetcher {
	t.Helper()
	files := map[string][]byte{"heavy.ttf": gobold.TTF, "mono.ttf": gomono.TTF}
	for _, name := range []string{"go-regular-latin.woff2", "go-regular-latin.woff", "go-icons.woff2"} {
		b, err := os.ReadFile(filepath.Join("internal", "woff", "testdata", name))
		require.NoError(t, err)
		files[name] = b
	}
	kerned, err := os.ReadFile(filepath.Join("..", "internal", "draw", "testdata", "go-kerned.ttf"))
	require.NoError(t, err)
	files["kerned.ttf"] = kerned
	featured, err := os.ReadFile(filepath.Join("..", "internal", "shape", "testdata", "go-features.ttf"))
	require.NoError(t, err)
	files["features.ttf"] = featured
	return func(url, _ string) ([]byte, error) {
		if b, ok := files[url]; ok {
			return b, nil
		}
		return nil, errors.New("404 Not Found")
	}
}

// landed lays d out until the fonts its text needs have landed, so
// Chrome is given them with the page.
func landed(t *testing.T, d *ui.Document, size geom.Vec2) {
	t.Helper()
	d.Layout(size)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.FontsLoading() == 0
	}, 5*time.Second, 5*time.Millisecond, "the page's fonts never landed")
	d.Layout(size)
}

const fetchedProse = "A page's own font, fetched while the page shows, wraps its lines where Chrome " +
	"wraps them with the same file: every word is measured in it, and kerned by it."

func TestAFetchedWebFontWrapsAndMeasuresAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Packed2; src: url(go-regular-latin.eot), url(go-regular-latin.woff2) format("woff2"); }
@font-face { font-family: Packed; src: url(go-regular-latin.woff) format("woff"); }
@font-face { font-family: Heavy; src: url(heavy.ttf); font-weight: 400; }
p { margin: 0 0 8px; width: 260px; }
.woff2 { font: 15px/1.4 Packed2, monospace; }
.woff { font: 13px Packed, monospace; width: 180px; }
.heavy { font: 17px Heavy, monospace; }
span { display: inline-block; }`))
	d.Body().Append(
		ui.P(ui.Class("woff2"), ui.Text(fetchedProse)),
		ui.P(ui.Class("woff"), ui.Text(fetchedProse)),
		ui.P(ui.Class("heavy"), ui.Text(fetchedProse)),
		ui.Div(ui.Span(ui.Class("woff2"), ui.Text("AVATAR Type, kerned"))),
		ui.Div(ui.Span(ui.Class("woff"), ui.Text("AVATAR Type, kerned"))),
		ui.Div(ui.Span(ui.Class("heavy"), ui.Text("AVATAR Type, kerned"))),
	)
	size := geom.V2(600, 700)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestACharacterAFetchedFontLacksIsTheNextFamilysAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Latin; src: url(go-regular-latin.woff2); }
@font-face { font-family: Split; src: url(mono.ttf); unicode-range: U+30-39; }
@font-face { font-family: Split; src: url(heavy.ttf); unicode-range: U+41-5A, U+61-7A; }
span { display: inline-block; font-size: 18px; }
.latin { font-family: Latin, Go; }
.split { font-family: Split, Go; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.Class("latin"), ui.Text("Latin, then Ωμέγα and Привет"))),
		ui.Div(ui.Span(ui.Class("split"), ui.Text("Letters 0123456789, the rest: (regular)"))),
	)
	size := geom.V2(600, 300)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAnIconFontsGlyphInThePrivateUseAreaIsSizedAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Icons; src: url(go-icons.woff2) format("woff2"); font-display: block; }
.icon { display: inline-block; font: normal 24px/1 Icons; }
.search::before { content: "\f002"; }
.home::before { content: "\f015"; }
.at::before { content: "\e900"; }
button { font-size: 14px; }`))
	d.Body().Append(
		ui.Div(ui.I(ui.Class("icon", "search")), ui.I(ui.Class("icon", "home")), ui.I(ui.Class("icon", "at"))),
		ui.Div(ui.Button(ui.I(ui.Class("icon", "search")), ui.Text(" Search"))),
	)
	size := geom.V2(400, 300)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestAFetchedFontKernedByGPOSIsMeasuredAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Kerned; src: url(kerned.ttf); }
span { display: inline-block; font: 40px Kerned, monospace; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.ID("kerned"), ui.Text("AVAVAV"))),
		ui.Div(ui.Span(ui.ID("plain"), ui.Text("VAVAVA"))),
		ui.Div(ui.Span(ui.Text("A")), ui.Span(ui.Text("V"))),
	)
	size := geom.V2(600, 300)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	assert.InDelta(t, d.ByID("plain").Box().W-200.0/2048*40, d.ByID("kerned").Box().W, 0.01,
		"one more pair than the other, 200 units of 2048 closer")
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
