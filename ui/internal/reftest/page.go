package reftest

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The namespaces html gives SVG and MathML elements.
const (
	nsSVG  = "svg"
	nsMath = "math"
)

// Base is the address the pages are read at: a page's address is Base
// and its path, which its references to other files resolve from.
const Base = "http://wpt.test/"

// Sheet is a style element's text or a linked sheet's, and the address
// its url() resolve from.
type Sheet struct {
	Text, From string
}

// Page is a WPT page read for ui: its tree, its sheets in document order,
// the references and fuzzy rules it names itself (those of a reference
// chained to another) and why ui can't show it, when it can't.
type Page struct {
	Path   string
	Root   *html.Node
	Sheets []Sheet
	Refs   []Ref
	Fuzzy  []Fuzzy
	// Skip is why the page is not drawn, "" when it is.
	Skip string
}

// Address is p's address, which its relative references resolve from.
func (p *Page) Address() string { return Base + p.Path }

// unsupported are the elements whose pages are skipped: ui has no element
// for them, or they need what the runner doesn't give (another document,
// a script, a player).
//
//nolint:gochecknoglobals,exhaustive // the table
var unsupported = map[atom.Atom]string{
	atom.Iframe: "iframe", atom.Frame: "frame", atom.Frameset: "frameset", atom.Object: "object",
	atom.Embed: "embed", atom.Video: "video", atom.Audio: "audio", atom.Canvas: "canvas",
	atom.Dialog: "dialog", atom.Math: "MathML",
}

// skippedFlags are the WPT meta flags whose tests need what the runner
// can't do.
//
//nolint:gochecknoglobals // the table
var skippedFlags = map[string]string{
	"paged":     "print (paged media)",
	"interact":  "interaction",
	"animated":  "animation",
	"userstyle": "user style sheet",
	"speech":    "speech synthesis",
}

// ReadPage reads the page at p, a path from the checkout at root.
func ReadPage(root, p string) (*Page, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return nil, err
	}
	page := &Page{Path: p}
	switch strings.ToLower(path.Ext(p)) {
	case ".svg":
		page.Skip = "SVG document"
		return page, nil
	case ".xht", ".xhtml", ".xml":
		page.Root, err = parseXML(data)
	default:
		page.Root, err = html.Parse(bytes.NewReader(data))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	if strings.Contains(path.Base(p), "-print") || strings.Contains(p, "/printing/") {
		page.Skip = "print"
	}
	page.read(root)
	return page, nil
}

// read gathers p's sheets, references and fuzzy rules from its tree, and
// what makes it unshowable.
func (p *Page) read(root string) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			p.element(root, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(p.Root)
}

func (p *Page) element(root string, n *html.Node) {
	skip := func(why string) {
		if p.Skip == "" {
			p.Skip = why
		}
	}
	if n.Namespace == nsMath {
		skip("MathML")
		return
	}
	if n.Namespace != "" {
		return
	}
	if why, ok := unsupported[n.DataAtom]; ok {
		skip("element " + why)
	}
	switch n.DataAtom { //nolint:exhaustive // the elements a page is read by
	case atom.Script:
		skip("script")
	case atom.Html:
		if slices.Contains(strings.Fields(attr(n, "class")), "reftest-wait") {
			skip("script")
		}
	case atom.Style:
		if printOnly(attr(n, "media")) {
			return
		}
		var text strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			text.WriteString(c.Data)
		}
		p.Sheets = append(p.Sheets, Sheet{Text: media(text.String(), attr(n, "media")), From: p.Address()})
	case atom.Link:
		p.link(root, n)
	case atom.Meta:
		p.meta(n)
	}
}

func (p *Page) link(root string, n *html.Node) {
	href := attr(n, "href")
	rels := strings.Fields(strings.ToLower(attr(n, "rel")))
	switch {
	case slices.Contains(rels, "match"):
		p.Refs = append(p.Refs, Ref{Path: Resolve(p.Address(), href), Rel: Match})
	case slices.Contains(rels, "mismatch"):
		p.Refs = append(p.Refs, Ref{Path: Resolve(p.Address(), href), Rel: Mismatch})
	case slices.Contains(rels, "stylesheet") && !slices.Contains(rels, "alternate"):
		if printOnly(attr(n, "media")) {
			return
		}
		at := Resolve(p.Address(), href)
		if at == "" {
			return
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(at)))
		if err != nil {
			return
		}
		p.Sheets = append(p.Sheets, Sheet{Text: media(string(data), attr(n, "media")), From: Base + at})
	}
}

func (p *Page) meta(n *html.Node) {
	content := attr(n, "content")
	switch strings.ToLower(attr(n, "name")) {
	case "flags":
		for _, flag := range strings.Fields(content) {
			if why, ok := skippedFlags[flag]; ok && p.Skip == "" {
				p.Skip = why
			}
		}
	case "fuzzy":
		if f, err := ParseFuzzy(content); err == nil {
			if f.Ref != "" {
				f.Ref = Resolve(p.Address(), f.Ref)
			}
			p.Fuzzy = append(p.Fuzzy, f)
		}
	}
}

// printOnly reports whether a media attribute keeps a sheet from the
// screen.
func printOnly(m string) bool {
	m = strings.TrimSpace(strings.ToLower(m))
	return m == "print" || m == "speech"
}

// media is a sheet under its media attribute.
func media(text, m string) string {
	m = strings.TrimSpace(m)
	if m == "" || strings.EqualFold(m, "all") || strings.EqualFold(m, "screen") {
		return text
	}
	return "@media " + m + " {\n" + text + "\n}"
}

// Resolve is ref resolved from the page or sheet address from, as a path
// from the checkout's root; "" for what is outside it.
func Resolve(from, ref string) string {
	base, err := url.Parse(from)
	if err != nil {
		return ""
	}
	u, err := base.Parse(strings.TrimSpace(ref))
	if err != nil || u.Host != "wpt.test" {
		return ""
	}
	return strings.TrimPrefix(path.Clean(u.Path), "/")
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name && a.Namespace == "" {
			return a.Val
		}
	}
	return ""
}

var fuzzyRange = regexp.MustCompile(`^\s*(?:(maxDifference|totalPixels)\s*=\s*)?(\d+)\s*(?:-\s*(\d+))?\s*$`)

// ParseFuzzy reads a fuzzy meta's content: "[ref:]maxDifference;
// totalPixels", each a number or an inclusive range, and each maybe
// named (maxDifference=0-2;totalPixels=0-10).
func ParseFuzzy(content string) (Fuzzy, error) {
	var f Fuzzy
	if i := strings.Index(content, ":"); i >= 0 {
		f.Ref, content = strings.TrimSpace(content[:i]), content[i+1:]
	}
	parts := strings.Split(content, ";")
	if len(parts) != 2 {
		return f, fmt.Errorf("fuzzy %q has %d parts", content, len(parts))
	}
	for i, part := range parts {
		m := fuzzyRange.FindStringSubmatch(part)
		if m == nil {
			return f, fmt.Errorf("fuzzy %q: %q is no range", content, part)
		}
		lo, _ := strconv.Atoi(m[2])
		hi := lo
		if m[3] != "" {
			hi, _ = strconv.Atoi(m[3])
		}
		r := Range{lo, hi}
		which := i
		switch m[1] {
		case "maxDifference":
			which = 0
		case "totalPixels":
			which = 1
		}
		if which == 0 {
			f.MaxDifference = r
		} else {
			f.TotalPixels = r
		}
	}
	return f, nil
}

// parseXML reads an XHTML page into the tree html.Parse makes, so its
// empty elements (<div/>) stay empty: elements of the XHTML namespace
// are HTML's, others keep theirs ("svg", "math").
func parseXML(data []byte) (*html.Node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	d.AutoClose = xml.HTMLAutoClose
	doc := &html.Node{Type: html.DocumentNode}
	at := doc
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &html.Node{Type: html.ElementNode, Data: t.Name.Local, Namespace: namespace(t.Name.Space)}
			if n.Namespace == "" {
				n.Data = strings.ToLower(n.Data)
			}
			n.DataAtom = atom.Lookup([]byte(n.Data))
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				key := a.Name.Local
				if a.Name.Space == "http://www.w3.org/XML/1998/namespace" || a.Name.Space == "xml" {
					key = "xml:" + key
				}
				n.Attr = append(n.Attr, html.Attribute{Key: key, Val: a.Value})
			}
			at.AppendChild(n)
			at = n
		case xml.EndElement:
			if at.Parent != nil {
				at = at.Parent
			}
		case xml.CharData:
			if at.Type == html.DocumentNode {
				continue
			}
			if last := at.LastChild; last != nil && last.Type == html.TextNode {
				last.Data += string(t)
				continue
			}
			at.AppendChild(&html.Node{Type: html.TextNode, Data: string(t)})
		}
	}
	ensureBody(doc)
	return doc, nil
}

// namespace is the html package's name of an XML namespace: "" for
// XHTML's.
func namespace(space string) string {
	switch space {
	case "http://www.w3.org/2000/svg", nsSVG:
		return nsSVG
	case "http://www.w3.org/1998/Math/MathML", nsMath:
		return nsMath
	}
	return ""
}

// ensureBody gives an XHTML page without a body (an html holding its
// content directly) none: html.Parse always has one, and the builder
// looks for it, so its other children go into a new body.
func ensureBody(doc *html.Node) {
	h := find(doc, atom.Html)
	if h == nil || find(h, atom.Body) != nil {
		return
	}
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	for c := h.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type != html.ElementNode || c.DataAtom != atom.Head {
			h.RemoveChild(c)
			body.AppendChild(c)
		}
		c = next
	}
	h.AppendChild(body)
}

func find(n *html.Node, a atom.Atom) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.DataAtom == a {
			return c
		}
	}
	return nil
}
