package ui

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// fontFiles is a font fetcher that notes what it was asked for and
// answers from files, an error for an address it doesn't hold.
type fontFiles struct {
	mu    sync.Mutex
	urls  []string
	files map[string][]byte
}

func (a *fontFiles) fetch(url, _ string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.urls = append(a.urls, url)
	if b, ok := a.files[url]; ok {
		return b, nil
	}
	return nil, errors.New("404 Not Found")
}

func (a *fontFiles) asked() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string{}, a.urls...)
}

func webFont(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("internal", "woff", "testdata", name))
	require.NoError(t, err)
	return b
}

//nolint:gochecknoglobals // the tests' fixed size; Go has no constant vectors
var page = geom.V2(400, 300)

// landAll lays d out, waits for the fonts on their way and lays it out
// again with them.
func landAll(d *Document) {
	d.Layout(page)
	d.pageFonts.Wait()
	d.Layout(page)
}

// wrapped fits one line of a 150px box in Go Regular, and not in Go
// Bold.
const wrapped = "wide wide wide wide"

func TestAPageFontIsFetchedWhenTextNeedsItAndTheTextWrapsAgain(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"https://fonts.test/heavy.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Heavy; src: url(heavy.eot), url(heavy.ttf) format("truetype"); }
@font-face { font-family: Unused; src: url(unused.woff2); }
p { font-family: Heavy, sans-serif; width: 150px; margin: 0; }
span { font-family: Heavy, sans-serif; }`, From("https://fonts.test/site.css")))
	p, span := P(Text(wrapped)), Span(Text("wide"))
	d.Body().Append(p, Div(span))

	plain := NewDocument(CSS(`p { width: 150px; margin: 0; } p, span { font-family: sans-serif; }`))
	plainP, plainSpan := P(Text(wrapped)), Span(Text("wide"))
	plain.Body().Append(plainP, Div(plainSpan))
	plain.Layout(page)
	bold := NewDocument(CSS(`p { width: 150px; margin: 0; } p, span { font-family: sans-serif; font-weight: bold; }`))
	boldP, boldSpan := P(Text(wrapped)), Span(Text("wide"))
	bold.Body().Append(boldP, Div(boldSpan))
	bold.Layout(page)
	require.NotEqual(t, plainP.Box().H, boldP.Box().H, "the text must wrap elsewhere in the heavier face")

	assert.Empty(t, a.asked(), "no font is asked for before text is laid out")
	d.Layout(page)
	assert.Empty(t, d.Warnings(), "a font on its way is no warning")
	assert.Equal(t, plainSpan.Box().W, span.Box().W, "the next family draws the text meanwhile")
	assert.Equal(t, plainP.Box().H, p.Box().H)
	assert.Equal(t, 1, d.FontsLoading())

	d.pageFonts.Wait()
	d.Layout(page)
	assert.Equal(t, boldSpan.Box().W, span.Box().W, "measured again in the font that landed")
	assert.Equal(t, boldP.Box().H, p.Box().H, "and wrapped again")
	assert.Zero(t, d.FontsLoading())

	d.Layout(page)
	d.pageFonts.Wait()
	assert.Equal(t, []string{"https://fonts.test/heavy.ttf"}, a.asked(),
		"asked once, from the sheet's address, past the .eot; a face no text uses is never asked for")
	assert.Empty(t, d.Warnings())
}

func TestFacesOfOneFileShareItsFetch(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"var.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Var; font-weight: 400; src: url(var.ttf); }
@font-face { font-family: Var; font-weight: 700; src: url(var.ttf); }
p { font-family: Var; } b { font-family: Var; font-weight: 700; }`))
	d.Body().Append(P(Text("light"), B(Text("heavy"))))
	landAll(d)
	assert.Equal(t, []string{"var.ttf"}, a.asked())
	assert.Len(t, d.fonts.Faces(), 5+2, "both faces draw from the one file")
}

func TestAFaceIsFetchedOnlyForTheCharactersOfItsRange(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"latin.ttf": gobold.TTF, "greek.ttf": gomono.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Split; src: url(greek.ttf); unicode-range: U+0370-03FF; }
@font-face { font-family: Split; src: url(latin.ttf); unicode-range: U+0000-00FF; }
span { font-family: Split, sans-serif; }`))
	latin, greek := Span(Text("abc")), Span()
	d.Body().Append(Div(latin), Div(greek))
	landAll(d)
	assert.Equal(t, []string{"latin.ttf"}, a.asked(), "no text needs a Greek letter")

	f := style.Font{Family: []string{"Split", "sans-serif"}, Size: 16, Weight: 400}
	assert.Equal(t, d.fonts.Advance(style.Font{Family: []string{"Go"}, Size: 16, Weight: 700}, "abc"),
		d.fonts.Advance(f, "abc"), "the Latin face measures Latin text")

	greek.SetText("λ")
	landAll(d)
	assert.Equal(t, []string{"latin.ttf", "greek.ttf"}, a.asked(), "the Greek face comes with the first Greek letter")
	assert.InDelta(t, float32(1229)/2048*16, d.fonts.Advance(f, "λ"), 0.0001,
		"and measures it: every glyph of Go Mono is 1229 units of 2048 wide")
	assert.True(t, d.face(f).Draws("aλ"), "twod draws each character with its face")
}

func TestAFaceIsNotUsedOutsideItsRange(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"amp.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Amp; src: url(amp.ttf); unicode-range: U+26; }
span { font-family: Amp, sans-serif; }`))
	d.Body().Append(Span(Text("a & b")))
	landAll(d)
	f := style.Font{Family: []string{"Amp", "sans-serif"}, Size: 16, Weight: 400}
	regular := style.Font{Family: []string{"Go"}, Size: 16, Weight: 400}
	heavy := style.Font{Family: []string{"Go"}, Size: 16, Weight: 700}
	assert.Equal(t, d.fonts.Advance(regular, "ab"), d.fonts.Advance(f, "ab"),
		"the face has these letters, and its range has not: the next family draws them")
	assert.Equal(t, d.fonts.Advance(heavy, "&"), d.fonts.Advance(f, "&"))
}

func TestAFontThatFailsIsAWarningNamingItsAddressAndTheTextStays(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{
		"https://fonts.test/broken.woff2": []byte("wOF2 and then nothing a font holds"),
		"https://fonts.test/page.woff":    []byte("<html>404</html>"),
	}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Gone; src: url(https://fonts.test/gone.woff2); }
@font-face { font-family: Broken; src: url(https://fonts.test/broken.woff2), url(https://fonts.test/page.woff); }
.gone { font-family: Gone, sans-serif; } .broken { font-family: Broken, sans-serif; }`))
	gone, broken := Span(Class("gone"), Text("text")), Span(Class("broken"), Text("text"))
	d.Body().Append(Div(gone), Div(broken))
	d.Layout(page)
	before := gone.Box().W
	d.pageFonts.Wait()
	d.Layout(page)
	d.Layout(page)
	d.pageFonts.Wait()

	var warns []string
	for _, w := range d.Warnings() {
		warns = append(warns, w.Error())
	}
	assert.ElementsMatch(t, []string{
		"ui: font https://fonts.test/gone.woff2: 404 Not Found",
		"ui: font https://fonts.test/broken.woff2: woff: the file ends early",
		"ui: font https://fonts.test/page.woff: woff: not a font file",
	}, warns)
	assert.Equal(t, before, gone.Box().W, "the text stays in the next family")
	assert.Equal(t, before, broken.Box().W)
	assert.Len(t, a.asked(), 3, "each address is asked for once, the second of a list when the first fails")
	assert.Zero(t, d.FontsLoading())
}

func TestWOFFAndWOFF2FontsAreRead(t *testing.T) {
	ttf := webFont(t, "go-regular-latin.ttf")
	for _, name := range []string{"go-regular-latin.woff", "go-regular-latin.woff2"} {
		a := &fontFiles{files: map[string][]byte{"web": webFont(t, name), "plain": ttf}}
		d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Web; src: url(web); }
@font-face { font-family: Plain; src: url(plain); }
.web { font-family: Web, monospace; } .plain { font-family: Plain, monospace; } .mono { font-family: monospace; }`))
		web, plain, mono := Span(Class("web"), Text(wrapped)), Span(Class("plain"), Text(wrapped)),
			Span(Class("mono"), Text(wrapped))
		d.Body().Append(Div(web), Div(plain), Div(mono))
		landAll(d)
		assert.Empty(t, d.Warnings(), name)
		assert.Equal(t, plain.Box().W, web.Box().W, "%s measures as the font it packs", name)
		assert.NotEqual(t, mono.Box().W, web.Box().W, "%s: the text is still in the next family", name)
	}
}

func TestAnIconShowsAsNothingUntilItsFontLands(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"icons.woff2": webFont(t, "go-icons.woff2")}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Icons; src: url(icons.woff2) format("woff2"); font-display: block; }
i { font-family: Icons; font-style: normal; }
i::before { content: "\f002"; }`))
	icon := I()
	d.Body().Append(Div(icon))
	f := style.Font{Family: []string{"Icons"}, Size: 16, Weight: 400}

	d.Layout(page)
	assert.Zero(t, icon.Box().W, "a character only a font on its way may draw takes no room")
	assert.False(t, d.face(f).Draws(""))
	waiting := d.face(f)

	d.pageFonts.Wait()
	d.Layout(page)
	assert.Empty(t, d.Warnings())
	assert.InDelta(t, d.fonts.Advance(style.Font{Family: []string{"Go"}, Size: 16, Weight: 400}, "?"),
		icon.Box().W, 0.01, "the icon is the font's glyph: the fixture has Go's question mark there")
	assert.True(t, d.face(f).Draws(""), "twod draws it with the font that landed")
	assert.NotSame(t, waiting, d.face(f), "the face made before the font landed is not drawn with again")
}

func TestSeveralFontsLandingLayTheDocumentOutOnce(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"a.ttf": gobold.TTF, "b.ttf": gomono.TTF, "c.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: A; src: url(a.ttf); }
@font-face { font-family: B; src: url(b.ttf); }
@font-face { font-family: C; src: url(c.ttf); }
.a { font-family: A; } .b { font-family: B; } .c { font-family: C; }`))
	d.Body().Append(P(Class("a"), Text("one")), P(Class("b"), Text("two")), P(Class("c"), Text("three")))
	d.Layout(page)
	d.pageFonts.Wait()
	before := d.placed
	d.Layout(page)
	once := d.placed - before
	require.Positive(t, once, "the fonts that landed lay the Document out again")
	d.Layout(page)
	d.Layout(page)
	assert.Equal(t, before+once, d.placed, "once for the three of them, and not again")
}

func TestAFontInADataURLNeedsNoFetcher(t *testing.T) {
	src := "data:font/woff2;base64," + base64.StdEncoding.EncodeToString(webFont(t, "go-icons.woff2"))
	d := NewDocument(CSS(`
@font-face { font-family: Icons; src: url("` + src + `"); }
i { font-family: Icons; }`))
	icon := I(Text(""))
	d.Body().Append(Div(icon))
	landAll(d)
	assert.Empty(t, d.Warnings())
	assert.Positive(t, icon.Box().W)
}

func TestAFontFaceWithNoFetcherIsAWarning(t *testing.T) {
	d := NewDocument(CSS(`@font-face { font-family: Web; src: url(web.woff2); } p { font-family: Web; }`))
	d.Body().Append(P(Text("text")))
	landAll(d)
	require.Len(t, d.Warnings(), 1)
	assert.EqualError(t, d.Warnings()[0],
		`css: @font-face "Web" in CSS has no file to read; load it with Stylesheet or fetch it with FetchFonts`)
}

func TestAStylesheetsFontsAreReadFromItsFilesInAnyFormat(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/site.css": {Data: []byte(`
@font-face { font-family: Packed; src: url(fonts/packed.eot), url(fonts/packed.woff2) format("woff2"); }
@font-face { font-family: Ranged; src: url(fonts/heavy.ttf); unicode-range: U+41-5A; }
@font-face { font-family: Elsewhere; src: url(fonts/elsewhere.ttf); }
.packed { font-family: Packed, monospace; } .ranged { font-family: Ranged, sans-serif; }
.elsewhere { font-family: Elsewhere, sans-serif; }`)},
		"ui/fonts/packed.woff2": {Data: webFont(t, "go-regular-latin.woff2")},
		"ui/fonts/heavy.ttf":    {Data: gobold.TTF},
	}
	a := &fontFiles{files: map[string][]byte{"ui/fonts/elsewhere.ttf": gobold.TTF}}
	d := NewDocument(Stylesheet(fsys, "ui/site.css"), FetchFonts(a.fetch))
	packed, ranged, elsewhere := Span(Class("packed"), Text("text")), Span(Class("ranged"), Text("Aa")),
		Span(Class("elsewhere"), Text("text"))
	d.Body().Append(Div(packed), Div(ranged), Div(elsewhere))
	d.Layout(page)
	assert.Empty(t, d.Warnings())
	regular := style.Font{Family: []string{"Go"}, Size: 16, Weight: 400}
	heavy := style.Font{Family: []string{"Go"}, Size: 16, Weight: 700}
	assert.InDelta(t, d.fonts.Advance(regular, "text"), packed.Box().W, 0.02,
		"a WOFF2 file of the sheet's files is read at once")
	assert.InDelta(t, d.fonts.Advance(heavy, "A")+d.fonts.Advance(regular, "a"), ranged.Box().W, 0.02,
		"its unicode-range holds for a file too")
	d.pageFonts.Wait()
	d.Layout(page)
	assert.Equal(t, []string{"ui/fonts/elsewhere.ttf"}, a.asked(), "a file the sheet's files lack is fetched")
	assert.InDelta(t, d.fonts.Advance(heavy, "text"), elsewhere.Box().W, 0.02)
}

func TestATextFieldIsMeasuredAgainInTheFontThatLanded(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"heavy.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Heavy; src: url(heavy.ttf); }
input { font-family: Heavy, sans-serif; }`))
	field := Input(Attr("value", "some text in a field"))
	d.Body().Append(field)
	d.Layout(page)
	d.layText()
	before, ok := textOf(d.machine(field)).Layout.(*line)
	require.True(t, ok)
	d.pageFonts.Wait()
	d.Layout(page)
	d.layText()
	after, ok := textOf(d.machine(field)).Layout.(*line)
	require.True(t, ok)
	assert.NotSame(t, before, after, "the field kept what it measured in the face before")
}

func TestFontsReadyAndLoadFontsAnswerForTextNotLaidOut(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"latin.ttf": gobold.TTF, "greek.ttf": gomono.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Split; src: url(greek.ttf); unicode-range: U+0370-03FF; }
@font-face { font-family: Split; src: url(latin.ttf); unicode-range: U+0000-00FF; }`))
	d.Layout(page)
	split := []string{"Split", "sans-serif"}
	assert.True(t, d.FontsReady([]string{"sans-serif"}, 400, false, "abc"),
		"a family of the Document's own is always there")
	assert.False(t, d.FontsReady(split, 400, false, "abc"), "its face has not come")
	assert.Empty(t, a.asked(), "asking whether a font is there fetches nothing")

	assert.False(t, d.LoadFonts(split, 400, false, "abc"))
	assert.Equal(t, 1, d.FontsLoading())
	d.pageFonts.Wait()
	d.Layout(page)
	assert.Equal(t, []string{"latin.ttf"}, a.asked(), "only the face that draws the text is fetched")
	assert.True(t, d.FontsReady(split, 400, false, "abc"))
	assert.True(t, d.LoadFonts(split, 400, false, "abc"), "a font that is there needs no fetch")
	assert.False(t, d.FontsReady(split, 400, false, "λ"), "another character, another face")
}

func TestAFontIsFetchedWithTheAddressOfItsSheet(t *testing.T) {
	var sheets []string
	var mu sync.Mutex
	d := NewDocument(FetchFonts(func(url, sheet string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		sheets = append(sheets, url+" from "+sheet)
		return gobold.TTF, nil
	}), CSS(`@font-face { font-family: Web; src: url(../fonts/web.ttf); } p { font-family: Web; }`,
		From("https://site.test/css/site.css")))
	d.Body().Append(P(Text("text")))
	landAll(d)
	assert.Equal(t, []string{"https://site.test/fonts/web.ttf from https://site.test/css/site.css"}, sheets)
}

func TestATreeStyledAgainDrawsWithTheFacesItHad(t *testing.T) {
	a := &fontFiles{files: map[string][]byte{"heavy.ttf": gobold.TTF}}
	d := NewDocument(FetchFonts(a.fetch), CSS(`
@font-face { font-family: Heavy; src: url(heavy.ttf); }
h1 { font-family: Heavy, sans-serif; font-size: 40px; }
p { font-family: Heavy, serif; }
@media (min-width: 500px) { h1 { font-size: 72px; } p { color: red; } }`))
	h1, p := H1(Text("Title")), P(Text("words"))
	d.Body().Append(h1, p)
	landAll(d)
	faces := func() [2]*twod.Face {
		return [2]*twod.Face{d.face(dom.Data(h1).Style.Font), d.face(dom.Data(p).Style.Font)}
	}
	narrow := faces()
	for range 20 {
		d.Layout(geom.V2(600, 300))
		wide := faces()
		d.Layout(page)
		require.True(t, narrow == wide && narrow == faces(),
			"each crossing of the @media boundary styles the tree again")
	}
	for width := float32(300); width < 700; width += 7 {
		d.Layout(geom.V2(width, 300))
		faces()
	}
	assert.Len(t, d.pageFonts.chained, 2, "one face a list of families, however often it is computed")
}
