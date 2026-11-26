package ui_test

import (
	"errors"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// server is a fetcher for sheets by address that notes each ask and
// the sheet that asked.
type server struct {
	mu     sync.Mutex
	sheets map[string]string
	asked  []string
	by     map[string]string
}

func (s *server) fetch(url, sheet string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asked = append(s.asked, url)
	if s.by == nil {
		s.by = map[string]string{}
	}
	s.by[url] = sheet
	text, ok := s.sheets[url]
	if !ok {
		return nil, errors.New("404 Not Found")
	}
	return []byte(text), nil
}

func (s *server) asks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.asked...)
}

// settled lays d out until no sheet is on its way, and gives n's width.
func settled(t *testing.T, d *ui.Document, n *ui.Node, size geom.Vec2) float32 {
	t.Helper()
	d.Layout(size)
	require.Eventually(t, func() bool {
		d.Layout(size)
		return d.SheetsLoading() == 0
	}, 5*time.Second, time.Millisecond)
	d.Layout(size)
	return n.Box().W
}

func TestADocumentFollowsItsSheetsImports(t *testing.T) {
	srv := &server{sheets: map[string]string{
		"https://site.test/css/a.css":       `@import "parts/b.css"; .box { width: 100px; height: 10px; }`,
		"https://site.test/css/parts/b.css": `.box { width: 50px; } #box { height: 20px; }`,
	}}
	box := ui.Div(ui.ID("box"), ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch),
		ui.CSS(`@import url(css/a.css); .box { width: 300px; }`, ui.From("https://site.test/index.css")))
	d.Body().Append(box)
	size := geom.V2(800, 600)
	d.Layout(size)
	assert.InDelta(t, 300, box.Box().W, 0.01, "the sheet shows before what it imports lands")
	assert.InDelta(t, 300, settled(t, d, box, size), 0.01, "a rule after the import wins by order")
	assert.InDelta(t, 20, box.Box().H, 0.01, "an imported rule wins by specificity")
	assert.Equal(t, []string{"https://site.test/css/a.css", "https://site.test/css/parts/b.css"}, srv.asks())
	assert.Equal(t, "https://site.test/css/a.css", srv.by["https://site.test/css/parts/b.css"],
		"a request says the sheet that imports")
	assert.Equal(t, "https://site.test/index.css", srv.by["https://site.test/css/a.css"])
	assert.Empty(t, d.Warnings())
}

func TestAnImportsMediaFollowsTheViewport(t *testing.T) {
	srv := &server{sheets: map[string]string{"https://site.test/wide.css": `.box { width: 400px; }`}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch), ui.BrowserSheets(), ui.CSS(
		`@layer base; @import url(wide.css) layer(wide) (min-width: 700px); @layer base { .box { width: 9px; } }`,
		ui.From("https://site.test/")))
	d.Body().Append(box)
	assert.InDelta(t, 400, settled(t, d, box, geom.V2(800, 600)), 0.01)
	d.Layout(geom.V2(600, 600))
	assert.InDelta(t, 9, box.Box().W, 0.01, "out of its media the import's rules are not")
	assert.Empty(t, d.Warnings())
}

func TestASheetImportedTwiceIsFetchedOnceAndStandsInBothPlaces(t *testing.T) {
	srv := &server{sheets: map[string]string{
		"https://site.test/a.css": `.box { width: 100px; }`,
		"https://site.test/b.css": `.box { width: 200px; }`,
	}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch),
		ui.CSS(`@import url(a.css); @import url(b.css); @import url(a.css);`, ui.From("https://site.test/")))
	d.Body().Append(box)
	assert.InDelta(t, 100, settled(t, d, box, geom.V2(800, 600)), 0.01)
	assert.ElementsMatch(t, []string{"https://site.test/a.css", "https://site.test/b.css"}, srv.asks())
}

func TestACycleOfImportsIsStopped(t *testing.T) {
	srv := &server{sheets: map[string]string{
		"https://site.test/a.css": `@import url(b.css); .box { width: 100px; }`,
		"https://site.test/b.css": `@import url(a.css); @import url(index.css); .box { width: 200px; height: 5px; }`,
	}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch),
		ui.CSS(`@import url(a.css);`, ui.From("https://site.test/index.css")))
	d.Body().Append(box)
	assert.InDelta(t, 100, settled(t, d, box, geom.V2(800, 600)), 0.01)
	assert.InDelta(t, 5, box.Box().H, 0.01)
	assert.ElementsMatch(t, []string{"https://site.test/a.css", "https://site.test/b.css"}, srv.asks())
	assert.Len(t, d.Warnings(), 2)
	for _, w := range d.Warnings() {
		assert.ErrorContains(t, w, "imports itself")
	}
}

func TestAnImportThatFailsIsAWarningAndTheRestShows(t *testing.T) {
	srv := &server{sheets: map[string]string{"https://site.test/b.css": `.box { width: 200px; }`}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch),
		ui.CSS(`@import url(gone.css); @import url(b.css);`, ui.From("https://site.test/")))
	d.Body().Append(box)
	assert.InDelta(t, 200, settled(t, d, box, geom.V2(800, 600)), 0.01)
	require.Len(t, d.Warnings(), 1)
	assert.EqualError(t, d.Warnings()[0], "ui: @import https://site.test/gone.css: 404 Not Found")
}

func TestAnImportedSheetsWarningsAndFontsAreTheDocuments(t *testing.T) {
	srv := &server{sheets: map[string]string{"https://site.test/a.css": `.box { colour: red; }
@font-face { font-family: Gone; src: url(gone.woff2); }`}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch), ui.CSS(`@import url(a.css);`, ui.From("https://site.test/")))
	d.Body().Append(box)
	settled(t, d, box, geom.V2(800, 600))
	require.Len(t, d.Warnings(), 2)
	assert.EqualError(t, d.Warnings()[0], `https://site.test/a.css:1:8: unknown property "colour"`)
	assert.ErrorContains(t, d.Warnings()[1], `@font-face "Gone"`)
}

func TestWithoutAFetcherAnImportIsAWarning(t *testing.T) {
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.CSS(`@import url(a.css); .box { width: 100px; }`))
	d.Body().Append(box)
	d.Layout(geom.V2(800, 600))
	assert.InDelta(t, 100, box.Box().W, 0.01)
	assert.Zero(t, d.SheetsLoading())
	require.Len(t, d.Warnings(), 1)
	assert.EqualError(t, d.Warnings()[0], `ui: @import a.css is not followed; fetch it with FetchSheets`)
}

func TestAnImportedSheetIsDecodedAsCSSSyntaxSays(t *testing.T) {
	srv := &server{sheets: map[string]string{
		"https://site.test/latin.css":    "@import \"inherits.css\"; .caf\xe9 { width: 10px; }",
		"https://site.test/inherits.css": ".th\xe9 { width: 20px; }",
		"https://site.test/own.css":      "@charset \"shift_jis\"; .\x95\xbd\x98\x61 { width: 30px; }",
		"https://site.test/marked.css":   "\xef\xbb\xbf.caf\xc3\xa9-bom { width: 40px; }",
	}}
	cafe, the, peace, bom := ui.Div(ui.Class("café")), ui.Div(ui.Class("thé")), ui.Div(ui.Class("平和")),
		ui.Div(ui.Class("café-bom"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch),
		ui.CSS(`@import "latin.css"; @import "own.css"; @import "marked.css";`,
			ui.From("https://site.test/page.html"), ui.Encoding("iso-8859-1")))
	d.Body().Append(cafe, the, peace, bom)
	settled(t, d, cafe, geom.V2(800, 600))
	require.Eventually(t, func() bool { return settled(t, d, the, geom.V2(800, 600)) == 20 },
		5*time.Second, time.Millisecond,
		"the sheet it imports falls back to the importing sheet's encoding")
	assert.InDelta(t, 10, cafe.Box().W, 0.01, "decoded in the encoding of the sheet that imports it")
	assert.InDelta(t, 30, peace.Box().W, 0.01, "its own @charset first")
	assert.InDelta(t, 40, bom.Box().W, 0.01, "a byte order mark before all")
	assert.Empty(t, d.Warnings())
}

func TestAStylesheetIsDecodedAsCSSSyntaxSays(t *testing.T) {
	files := fstest.MapFS{"ui/screen.css": {Data: []byte("\xef\xbb\xbf.box { width: 50px; }")}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.Stylesheet(files, "ui/screen.css"))
	d.Body().Append(box)
	d.Layout(geom.V2(800, 600))
	assert.InDelta(t, 50, box.Box().W, 0.01, "its byte order mark is no part of its first rule")
}

func TestAStylesheetImportsFromItsFileSystem(t *testing.T) {
	files := fstest.MapFS{
		"ui/screen.css":    {Data: []byte(`@import "parts/hud.css"; .box { height: 10px; }`)},
		"ui/parts/hud.css": {Data: []byte(`@import "../theme.css"; .box { width: 120px; }`)},
		"ui/theme.css":     {Data: []byte(`.box { width: 60px; height: 60px; }`)},
	}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.Stylesheet(files, "ui/screen.css"))
	d.Body().Append(box)
	d.Layout(geom.V2(800, 600))
	assert.InDelta(t, 120, box.Box().W, 0.01, "read at once, beside the sheet that imports")
	assert.InDelta(t, 10, box.Box().H, 0.01)
	assert.Empty(t, d.Warnings())
}

func TestASheetAddedWhileTheDocumentShowsFollowsItsImports(t *testing.T) {
	srv := &server{sheets: map[string]string{"https://site.test/a.css": `.box { width: 70px; }`}}
	d, box := sheetsDoc(ui.FetchSheets(srv.fetch))
	require.InDelta(t, 100, width(d, box), 0.01)
	s := d.AddSheet(`@import url(a.css);`, ui.From("https://site.test/"))
	assert.InDelta(t, 70, settled(t, d, box, geom.V2(800, 600)), 0.01)
	s.SetMedia("(max-width: 500px)")
	assert.InDelta(t, 100, width(d, box), 0.01, "the sheet's media holds what it imports")
	s.Remove()
	assert.InDelta(t, 100, width(d, box), 0.01)
	assert.Empty(t, d.Warnings())
}

func TestAnImportedRuleIsMatchedInTheSheetItCameFrom(t *testing.T) {
	srv := &server{sheets: map[string]string{"https://site.test/a.css": "\n.box { width: 70px; }"}}
	box := ui.Div(ui.Class("box"))
	d := ui.NewDocument(ui.FetchSheets(srv.fetch), ui.BrowserSheets(),
		ui.CSS(`@import url(a.css) layer(theme) (min-width: 10px);`, ui.From("https://site.test/")))
	d.Body().Append(box)
	settled(t, d, box, geom.V2(800, 600))
	var found bool
	for _, m := range d.MatchedRules(box) {
		if m.Selector != ".box" {
			continue
		}
		found = true
		assert.Equal(t, "https://site.test/a.css", m.Sheet.From())
		assert.Equal(t, 2, m.Pos.Line)
		assert.Equal(t, []string{"(min-width: 10px)"}, m.Media)
		assert.Equal(t, "theme", m.Layer)
	}
	assert.True(t, found)
}
