package reftest

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/lantervale/ember/ui"
	_ "github.com/lantervale/ember/ui/svg" // SVG pictures in an img or a url()
)

// Width and Height are the size WPT's reftests are drawn at.
const (
	Width  = 800
	Height = 600
)

// FontsSheet is the sheet under every page's: the fonts WPT's runners
// install on the system, from the checkout's fonts/ folder, so the pages
// find them by name in ember and in Chrome alike, and glyphs drawn at
// their exact coverage, without the stem darkening of macOS's font
// smoothing, which WPT's references, boxes where the tests draw Ahem's
// squares, leave out (decision 0234).
const FontsSheet = `@font-face { font-family: Ahem; src: url(` + Base + `fonts/Ahem.ttf); }
:root { -webkit-font-smoothing: antialiased; }`

// Fetches counts what a Document of the runner is still fetching.
type Fetches struct{ n atomic.Int32 }

// Busy reports whether a fetch is under way.
func (f *Fetches) Busy() bool { return f.n.Load() > 0 }

var (
	errOutside = errors.New("outside the checkout")
	errNoSheet = errors.New("not served as text/css")
)

// fetcher reads ref, resolved from the address from, from the checkout
// at root.
func (f *Fetches) fetch(root, from, ref string) ([]byte, error) {
	f.n.Add(1)
	defer f.n.Add(-1)
	p := Resolve(from, ref)
	if p == "" {
		return nil, errOutside
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
}

// fetchSheet is fetch for a sheet, given back as UTF-8 behind a byte
// order mark when it is served with a charset, as ui reads one.
func (f *Fetches) fetchSheet(root, from, ref string) ([]byte, error) {
	data, err := f.fetch(root, from, ref)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(root, filepath.FromSlash(Resolve(from, ref)))
	if !servedAsCSS(file) {
		return nil, errNoSheet
	}
	return servedSheet(data, file), nil
}

// Document builds p, read from the checkout at root, as a ui Document:
// its sheets read as a browser's, scrolling as a browser's viewport with
// its scrollbars over it, the WPT fonts under its own, and its pictures,
// fonts and imported sheets read from the checkout, counted in fetches.
func (p *Page) Document(root string, fetches *Fetches) *ui.Document {
	opts := []ui.DocumentOption{
		ui.BrowserSheets(), ui.ScrollingViewport(), ui.OverlayScrollbars(),
		ui.CSS(FontsSheet, ui.From(Base)),
	}
	for _, s := range p.Sheets {
		opts = append(opts, ui.CSS(s.Text, ui.From(s.From), ui.Encoding(s.Encoding)))
	}
	page := p.Address()
	opts = append(opts,
		ui.FetchImages(func(ref string) ([]byte, error) { return fetches.fetch(root, page, ref) }),
		ui.FetchFonts(func(ref, sheet string) ([]byte, error) { return fetches.fetch(root, or(sheet, page), ref) }),
		ui.FetchSheets(func(ref, sheet string) ([]byte, error) { return fetches.fetchSheet(root, or(sheet, page), ref) }),
	)
	d := ui.NewDocument(opts...)
	b := builder{page: page}
	var place func(n *html.Node)
	place = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type != html.ElementNode:
			case c.DataAtom == atom.Html && c.Namespace == "":
				b.attributes(c, d.HTML())
				place(c)
			case c.DataAtom == atom.Body && c.Namespace == "":
				b.attributes(c, d.Body())
				b.children(c, d.Body())
			default:
				place(c)
			}
		}
	}
	place(p.Root)
	return d
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

type builder struct{ page string }

func (b builder) children(n *html.Node, into *ui.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			ui.Text(c.Data).Apply(into)
		case c.Type != html.ElementNode:
		case c.Namespace == nsSVG:
			if c.DataAtom == atom.Svg {
				e := ui.Svg(markup(c))
				b.attributes(c, e)
				into.Append(e)
			}
		case dropped[c.DataAtom]:
		default:
			into.Append(b.element(c))
		}
	}
}

func (b builder) element(n *html.Node) *ui.Node {
	e := ui.Div()
	if build, ok := elements[n.Data]; ok {
		e = build()
	}
	b.attributes(n, e)
	b.children(n, e)
	return e
}

func (b builder) attributes(n *html.Node, e *ui.Node) {
	for _, a := range n.Attr {
		if a.Namespace != "" || strings.HasPrefix(a.Key, "on") || a.Key == "autofocus" {
			continue
		}
		value := a.Val
		if a.Key == "src" {
			if at := Resolve(b.page, value); at != "" {
				value = Base + at
			}
		}
		e.SetAttr(a.Key, value)
	}
	if !b.xml() {
		return
	}
	for _, a := range n.Attr {
		if a.Key == "xml:lang" && a.Namespace == "" {
			e.SetAttr("lang", a.Val)
		}
	}
}

// xml reports whether the page is XHTML, where xml:lang is an element's
// language, over its lang.
func (b builder) xml() bool {
	switch strings.ToLower(path.Ext(b.page)) {
	case ".xht", ".xhtml", ".xml":
		return true
	}
	return false
}

// markup is what an svg element holds, as SVG.
func markup(n *html.Node) string {
	var s strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&s, c)
	}
	return s.String()
}

// dropped are the elements that build nothing, their children neither.
//
//nolint:gochecknoglobals,exhaustive // the elements left out
var dropped = map[atom.Atom]bool{
	atom.Head: true, atom.Script: true, atom.Style: true, atom.Link: true, atom.Meta: true,
	atom.Title: true, atom.Template: true,
}

// elements are ui's elements by tag; any other is a div.
//
//nolint:gochecknoglobals // the builder's table
var elements = map[string]func(...ui.Arg) *ui.Node{
	"div": ui.Div, "section": ui.Section, "header": ui.Header, "footer": ui.Footer, "nav": ui.Nav,
	"main": ui.Main, "aside": ui.Aside, "article": ui.Article, "p": ui.P, "h1": ui.H1, "h2": ui.H2,
	"h3": ui.H3, "h4": ui.H4, "span": ui.Span, "strong": ui.Strong, "em": ui.Em, "code": ui.Code,
	"kbd": ui.Kbd, "samp": ui.Samp, "pre": ui.Pre, "label": ui.Label, "button": ui.Button,
	"input": ui.Input, "select": ui.Select, "option": ui.Option, "textarea": ui.Textarea,
	"details": ui.Details, "summary": ui.Summary, "table": ui.Table, "thead": ui.Thead,
	"tbody": ui.Tbody, "tfoot": ui.Tfoot, "caption": ui.Caption, "tr": ui.Tr, "th": ui.Th, "td": ui.Td,
	"progress": ui.Progress, "meter": ui.Meter, "img": ui.Img, "hr": ui.Hr, "ul": ui.Ul, "ol": ui.Ol,
	"li": ui.Li, "menu": ui.Menu, "dl": ui.Dl, "dt": ui.Dt, "dd": ui.Dd, "h5": ui.H5, "h6": ui.H6,
	"blockquote": ui.Blockquote, "figure": ui.Figure, "figcaption": ui.Figcaption, "form": ui.Form,
	"fieldset": ui.Fieldset, "legend": ui.Legend, "address": ui.Address, "center": ui.Center,
	"hgroup": ui.Hgroup, "search": ui.Search, "br": ui.Br, "a": ui.A, "b": ui.B, "i": ui.I, "u": ui.U,
	"s": ui.S, "small": ui.Small, "big": ui.Big, "sub": ui.Sub, "sup": ui.Sup, "abbr": ui.Abbr,
	"cite": ui.Cite, "q": ui.Q, "mark": ui.Mark, "time": ui.Time, "var": ui.Var, "font": ui.Font,
	"del": ui.Del, "ins": ui.Ins, "dfn": ui.Dfn, "bdi": ui.Bdi, "bdo": ui.Bdo, "data": ui.Data,
	"picture": ui.Picture, "source": ui.Source, "col": ui.Col, "colgroup": ui.Colgroup,
}
