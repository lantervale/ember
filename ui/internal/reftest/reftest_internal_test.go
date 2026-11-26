package reftest

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html/atom"

	"github.com/lantervale/ember/geom"
)

func TestDocumentBuildsThePageWithItsSheets(t *testing.T) {
	p, err := ReadPage(fixture, "css/css-demo/green.html")
	if err != nil {
		t.Fatal(err)
	}
	d := p.Document(fixture, &Fetches{})
	d.Layout(geom.V2(Width, Height))
	box, ok := d.BoxOf(d.ByID("t"))
	if !ok || box != (geom.Rect{X: 8, Y: 8, W: 100, H: 100}) {
		t.Errorf("the div's box %v (%v)", box, ok)
	}
	if v, _ := d.ComputedValue(d.ByID("t"), "background-color"); v != "green" {
		t.Errorf("background-color %q", v)
	}
	if v, _ := d.ComputedValue(d.ByID("t"), "-webkit-font-smoothing"); v != "antialiased" {
		t.Errorf("-webkit-font-smoothing %q: glyphs at their exact coverage, as the references draw boxes", v)
	}
}

const fixture = "testdata/wpt"

func TestLoadReadsTheManifestsReftestsUnderTheDirectories(t *testing.T) {
	tests, missing, err := Load(fixture, []string{"css/css-demo", "css/css-absent"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, tt := range tests {
		paths = append(paths, tt.Path)
	}
	want := []string{"css/css-demo/driven.html", "css/css-demo/empty.xht", "css/css-demo/green.html"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("tests %v, want %v", paths, want)
	}
	if !reflect.DeepEqual(missing, []string{"css/css-absent"}) {
		t.Errorf("missing %v", missing)
	}
	green := tests[2]
	if len(green.Refs) != 1 || green.Refs[0] != (Ref{"css/css-demo/reference/green-ref.html", Match}) {
		t.Errorf("refs %v", green.Refs)
	}
	if f := FuzzyFor(green.Fuzzy, green.Refs[0].Path); f == nil || f.MaxDifference != (Range{0, 2}) ||
		f.TotalPixels != (Range{0, 10}) {
		t.Errorf("fuzzy %+v", green.Fuzzy)
	}
	if !tests[0].Testdriver || tests[1].Refs[0].Rel != Mismatch {
		t.Errorf("driven %+v, empty %+v", tests[0], tests[1])
	}
}

func TestParseFuzzyReadsNamedAndBareRanges(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Fuzzy
	}{
		{"maxDifference=0-2;totalPixels=0-10", Fuzzy{MaxDifference: Range{0, 2}, TotalPixels: Range{0, 10}}},
		{"3;40-50", Fuzzy{MaxDifference: Range{3, 3}, TotalPixels: Range{40, 50}}},
		{"ref.html:totalPixels=5;maxDifference=1", Fuzzy{Ref: "ref.html", MaxDifference: Range{1, 1},
			TotalPixels: Range{5, 5}}},
	} {
		got, err := ParseFuzzy(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseFuzzy(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
	if _, err := ParseFuzzy("0-2"); err == nil {
		t.Error("a fuzzy of one part is read")
	}
}

func TestPassesDecidesAsWPTsRunner(t *testing.T) {
	f := &Fuzzy{MaxDifference: Range{1, 2}, TotalPixels: Range{1, 10}}
	for _, c := range []struct {
		rel  Rel
		d    Diff
		f    *Fuzzy
		want bool
	}{
		{Match, Diff{}, nil, true},
		{Match, Diff{1, 1}, nil, false},
		{Match, Diff{2, 10}, f, true},
		{Match, Diff{3, 10}, f, false},
		{Match, Diff{}, f, true},
		{Mismatch, Diff{}, nil, false},
		{Mismatch, Diff{255, 1}, nil, true},
		{Mismatch, Diff{2, 5}, f, false},
	} {
		if got := Passes(c.rel, c.d, c.f); got != c.want {
			t.Errorf("Passes(%s, %v, %v) = %v", c.rel, c.d, c.f, got)
		}
	}
}

func TestCompareCountsPixelsAndTheLargestChannelDifference(t *testing.T) {
	a := solid(4, 4, color.NRGBA{0, 128, 0, 255})
	b := solid(4, 4, color.NRGBA{0, 128, 0, 255})
	b.Set(1, 1, color.NRGBA{3, 128, 0, 255})
	b.Set(2, 2, color.NRGBA{0, 100, 0, 255})
	if d := Compare(a, b); d != (Diff{28, 2}) {
		t.Errorf("Compare = %v", d)
	}
	if d := Compare(a, solid(4, 5, color.NRGBA{0, 128, 0, 255})); d != (Diff{255, 4}) {
		t.Errorf("a picture of another size: %v", d)
	}
}

func solid(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestReadPageGathersSheetsReferencesAndFuzzy(t *testing.T) {
	p, err := ReadPage(fixture, "css/css-demo/green.html")
	if err != nil {
		t.Fatal(err)
	}
	if p.Skip != "" {
		t.Errorf("skipped for %s", p.Skip)
	}
	if len(p.Sheets) != 2 || !strings.Contains(p.Sheets[0].Text, "a linked sheet") ||
		p.Sheets[0].From != Base+"css/css-demo/support.css" || !strings.Contains(p.Sheets[1].Text, "#t") {
		t.Errorf("sheets %+v", p.Sheets)
	}
	if !reflect.DeepEqual(p.Refs, []Ref{{"css/css-demo/reference/green-ref.html", Match}}) {
		t.Errorf("refs %v", p.Refs)
	}
	if len(p.Fuzzy) != 1 || p.Fuzzy[0].TotalPixels != (Range{0, 10}) {
		t.Errorf("fuzzy %+v", p.Fuzzy)
	}
}

func TestReadPageKeepsXHTMLsEmptyElementsEmpty(t *testing.T) {
	p, err := ReadPage(fixture, "css/css-demo/empty.xht")
	if err != nil {
		t.Fatal(err)
	}
	h := find(p.Root, atom.Html)
	body := find(h, atom.Body)
	div := find(body, atom.Div)
	if div == nil || div.FirstChild != nil || div.NextSibling == nil || div.NextSibling.DataAtom != atom.P {
		t.Fatalf("the div holds what follows it")
	}
	svg := find(body, atom.Svg)
	if svg == nil || svg.Namespace != nsSVG || svg.FirstChild == nil || svg.FirstChild.Data != "rect" {
		t.Errorf("an svg element is SVG's, known by its atom as html.Parse knows it")
	}
	if p.Skip != "" || len(p.Refs) != 1 || p.Refs[0].Rel != Mismatch {
		t.Errorf("skip %q, refs %v", p.Skip, p.Refs)
	}
}

func TestReadPageDecodesAPageAsABrowser(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	page := func(decl, text string) string {
		return decl + `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>` + text + `</p></body></html>`
	}
	write("ascii.xht", page(`<?xml version="1.0" encoding="us-ascii"?>`, "caf\xe9"))
	write("sjis.xht", page(`<?xml version="1.0" encoding="shift-JIS"?>`, "\x95\xbd\x98\x61"))
	write("plain.xht", page("", "caf\xc3\xa9"))
	write("served.xht", page(`<?xml version="1.0" encoding="utf-8"?>`, "caf\xe9"))
	write("served.xht.headers", "Content-Type: application/xhtml+xml; charset=iso-8859-1\n")
	write("bom.xht", "\xef\xbb\xbf"+page(`<?xml version="1.0" encoding="us-ascii"?>`, "caf\xc3\xa9"))
	write("bom.xht.headers", "Content-Type: application/xhtml+xml; charset=us-ascii\n")
	write("meta.html", `<meta charset="shift_jis"><p>`+"\x95\xbd\x98\x61")
	write("sheets.xht", page(`<?xml version="1.0" encoding="iso-8859-1"?>`,
		`<link rel="stylesheet" href="served.css"/><link rel="stylesheet" href="latin.css"/>`))
	write("served.css", ".\x95\xbd\x98\x61 {}")
	write("served.css.headers", "Content-Type: text/css; charset=Shift_JIS\n")
	write("latin.css", ".caf\xe9 {}")
	write("plain.css", ".red {}")
	write("plain.css.headers", "Content-Type: text/plain\n")
	write("notes.txt", ".red {}")
	write("typed.xht", page("", `<link rel="stylesheet" href="plain.css"/><link rel="stylesheet" href="notes.txt"/>`))
	if typed, err := ReadPage(root, "typed.xht"); err != nil || len(typed.Sheets) != 0 {
		t.Errorf("sheets not served as text/css: %+v (%v)", typed, err)
	}
	p, err := ReadPage(root, "sheets.xht")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sheets) != 2 || p.Sheets[0].Text != ".平和 {}" || p.Sheets[1].Text != ".café {}" ||
		p.Sheets[1].Encoding != "windows-1252" {
		t.Errorf("linked sheets decoded as %+v", p.Sheets)
	}
	write("labelled.xht", page("", `<link rel="stylesheet" href="sjis.css" charset="shift-JIS"/>`))
	write("sjis.css", ".\x95\xbd\x98\x61 {}")
	if p, err = ReadPage(root, "labelled.xht"); err != nil || len(p.Sheets) != 1 || p.Sheets[0].Text != ".平和 {}" {
		t.Errorf("a sheet decoded in its link's charset: %+v (%v)", p, err)
	}
	for _, c := range []struct{ path, text, encoding string }{
		{"ascii.xht", "café", "windows-1252"},
		{"sjis.xht", "平和", "shift_jis"},
		{"plain.xht", "café", "utf-8"},
		{"served.xht", "café", "windows-1252"},
		{"bom.xht", "café", "utf-8"},
		{"meta.html", "平和", "shift_jis"},
	} {
		got, err := ReadPage(root, c.path)
		if err != nil {
			t.Errorf("%s: %v", c.path, err)
			continue
		}
		var text string
		if para := find(find(find(got.Root, atom.Html), atom.Body), atom.P); para != nil && para.FirstChild != nil {
			text = para.FirstChild.Data
		}
		if text != c.text || got.Encoding != c.encoding {
			t.Errorf("%s: %q in %s, want %q in %s", c.path, text, got.Encoding, c.text, c.encoding)
		}
	}
}

func TestReadPageSaysWhyAPageIsSkipped(t *testing.T) {
	for p, why := range map[string]string{
		"css/css-demo/scripted.html": "script",
		"css/css-demo/paged.html":    "print (paged media)",
	} {
		page, err := ReadPage(fixture, p)
		if err != nil {
			t.Fatal(err)
		}
		if page.Skip != why {
			t.Errorf("%s skipped for %q, want %q", p, page.Skip, why)
		}
	}
}

func TestResolveStaysInTheCheckout(t *testing.T) {
	for ref, want := range map[string]string{
		"reference/a.html": "css/x/reference/a.html",
		"../b.css":         "css/b.css",
		"/fonts/Ahem.ttf":  "fonts/Ahem.ttf",
		"http://other/x":   "",
	} {
		if got := Resolve(Base+"css/x/t.html", ref); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestPropertiesAreWhatTheTestUsesBeyondItsReference(t *testing.T) {
	if got := Properties("@media screen { a:hover { Color: red; background:url(x:y) } }\nb{width:1px}"); !reflect.
		DeepEqual(got, []string{"background", "color", "width"}) {
		t.Errorf("Properties = %v", got)
	}
	test, err := ReadPage(fixture, "css/css-demo/green.html")
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReadPage(fixture, "css/css-demo/reference/green-ref.html")
	if err != nil {
		t.Fatal(err)
	}
	if got := PropertiesOf(test, ref); !reflect.DeepEqual(got, []string{"background", "margin", "margin-left"}) {
		t.Errorf("PropertiesOf = %v", got)
	}
}

func TestDecideFollowsAlternativesAndChains(t *testing.T) {
	dir := t.TempDir()
	green, red := solid(2, 2, color.NRGBA{0, 128, 0, 255}), solid(2, 2, color.NRGBA{255, 0, 0, 255})
	pages := map[string]*Page{}
	renders := map[string]Render{}
	add := func(p string, img image.Image, refs ...Ref) {
		pages[p] = &Page{Path: p, Refs: refs}
		renders[p] = Render{Kind: Drawn, Picture: writePNG(t, dir, p, img)}
	}
	add("t", green)
	add("red", red)
	add("green", green, Ref{"chained-red", Match})
	add("chained-red", red)
	add("plain", green)
	add("one-way", green, Ref{"other-way", Match})
	add("other-way", green, Ref{"one-way", Match})
	add("white", solid(2, 2, color.NRGBA{255, 255, 255, 255}))
	cases := []struct {
		refs []Ref
		want Status
	}{
		{[]Ref{{"red", Match}, {"plain", Match}}, Pass},
		{[]Ref{{"green", Match}}, Fail},
		{[]Ref{{"red", Mismatch}}, Pass},
		{[]Ref{{"absent", Match}}, Error},
		{[]Ref{{"one-way", Match}}, Pass},
	}
	for _, c := range cases {
		got := decide(Test{Path: "t", Refs: c.refs}, pages, map[string]error{}, renders)
		if got.Status != c.want {
			t.Errorf("refs %v: %s (%s), want %s", c.refs, got.Status, got.Reason, c.want)
		}
	}
	if got := decide(Test{Path: "white", Refs: []Ref{{Blank, Match}}}, pages, nil, renders); got.Status != Pass {
		t.Errorf("white against about:blank: %s (%s)", got.Status, got.Reason)
	}
	if got := decide(Test{Path: "t", Refs: []Ref{{Blank, Match}}}, pages, nil, renders); got.Status != Fail {
		t.Errorf("green against about:blank: %s (%s)", got.Status, got.Reason)
	}
	pages["skipped"] = &Page{Path: "skipped", Skip: "script"}
	if got := decide(Test{Path: "t", Refs: []Ref{{"skipped", Match}}}, pages, nil, renders); got.Status != Skip {
		t.Errorf("a skipped reference: %s", got.Status)
	}
}

func writePNG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name+".png")
	if err := os.WriteFile(p, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExpectationsRoundTrip(t *testing.T) {
	results := []Result{
		{Path: "a", Status: Pass, Chrome: &ChromeResult{OK: true}},
		{Path: "b", Status: Fail, Chrome: &ChromeResult{OK: true}},
		{Path: "c", Status: Pass},
		{Path: "d", Status: Fail},
	}
	var b bytes.Buffer
	if err := WriteExpectations(&b, "header\nsecond", results); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "expected.txt")
	if err := os.WriteFile(p, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadExpectations(p)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Expectation{"a": {true, true}, "b": {false, true}, "c": {true, false}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expectations %v, want %v", got, want)
	}
}

func TestTheSummaryRanksChromeFirst(t *testing.T) {
	results := []Result{
		{Path: "css/a/1.html", Status: Pass, Properties: []string{"float"},
			Chrome: &ChromeResult{Mean: 0.1, OK: false}},
		{Path: "css/a/2.html", Status: Fail, Reason: "== r", Properties: []string{"width"},
			Chrome: &ChromeResult{OK: true}},
		{Path: "css/b/3.html", Status: Skip, Reason: "script"},
	}
	var b bytes.Buffer
	WriteSummary(&b, Summary{Commit: "abc", Top: 20, ChromeVersion: 158}, results)
	s := b.String()
	chrome, ref := strings.Index(s, "Differing from Chrome"), strings.Index(s, "Failing against their references")
	if chrome < 0 || ref < chrome {
		t.Fatalf("Chrome's ranking isn't first:\n%s", s)
	}
	for _, want := range []string{"| css/a | 2 | 1 | 1 | 0 | 0 | 50.0% | 1 | 1 |", "`float`",
		"Matching the reference but not Chrome (1", "Matching Chrome but not the reference (1", "| script | 1 |"} {
		if !strings.Contains(s, want) {
			t.Errorf("the summary lacks %q:\n%s", want, s)
		}
	}
}
