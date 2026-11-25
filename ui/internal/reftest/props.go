package reftest

import (
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

var (
	comments = regexp.MustCompile(`(?s)/\*.*?\*/`)
	property = regexp.MustCompile(`^-?[a-zA-Z][a-zA-Z0-9-]*$`)
)

// descriptors are the names of at-rules' descriptors (@font-face,
// @property), which are no properties.
//
//nolint:gochecknoglobals // the table
var descriptors = []string{
	"src", "unicode-range", "font-display", "size-adjust", "ascent-override", "descent-override",
	"line-gap-override", "syntax", "inherits", "initial-value",
}

// Properties are the CSS properties the declarations of css name,
// lowercase, without custom properties nor at-rules' descriptors.
func Properties(css string) []string {
	css = comments.ReplaceAllString(css, "")
	var props []string
	for _, block := range blocks(css) {
		for _, decl := range strings.Split(block, ";") {
			name, _, ok := strings.Cut(decl, ":")
			p := strings.ToLower(strings.TrimSpace(name))
			if ok && property.MatchString(p) && !slices.Contains(descriptors, p) && !slices.Contains(props, p) {
				props = append(props, p)
			}
		}
	}
	slices.Sort(props)
	return props
}

// blocks are the innermost {…} of css, where its declarations are, or
// css itself when it has none (a style attribute).
func blocks(css string) []string {
	if !strings.Contains(css, "{") {
		return []string{css}
	}
	var out []string
	start := -1
	for i, r := range css {
		switch r {
		case '{':
			start = i + 1
		case '}':
			if start >= 0 {
				out = append(out, css[start:i])
			}
			start = -1
		}
	}
	return out
}

// pageProperties are the properties of p's sheets and style attributes.
func pageProperties(p *Page) []string {
	var css strings.Builder
	for _, s := range p.Sheets {
		if s.From == Base {
			continue
		}
		css.WriteString(s.Text)
		css.WriteString("\n")
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if style := attr(n, "style"); style != "" {
				css.WriteString("x{" + style + "}\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	if p.Root != nil {
		walk(p.Root)
	}
	return Properties(css.String())
}

// PropertiesOf are the properties test uses that ref doesn't, what the
// test is about rather than what both draw with; all test uses when ref
// is nil or uses them all.
func PropertiesOf(test, ref *Page) []string {
	all := pageProperties(test)
	if ref == nil {
		return all
	}
	theirs := pageProperties(ref)
	var own []string
	for _, p := range all {
		if !slices.Contains(theirs, p) {
			own = append(own, p)
		}
	}
	if len(own) == 0 {
		return all
	}
	return own
}
