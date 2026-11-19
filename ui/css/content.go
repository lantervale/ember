package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// listStyles are list-style-type's words, which counter() and counters()
// take too.
//
//nolint:gochecknoglobals // lookup table
var listStyles = map[string]style.ListStyle{
	"disc": style.ListDisc, kwNone: style.ListNone, "circle": style.ListCircle, "square": style.ListSquare,
	"decimal": style.ListDecimal, "decimal-leading-zero": style.ListDecimalLeadingZero,
	"lower-alpha": style.ListLowerAlpha, "lower-latin": style.ListLowerAlpha,
	"upper-alpha": style.ListUpperAlpha, "upper-latin": style.ListUpperAlpha,
	"lower-roman": style.ListLowerRoman, "upper-roman": style.ListUpperRoman,
	"disclosure-closed": style.ListDisclosureClosed, "disclosure-open": style.ListDisclosureOpen,
}

func checkListStyle(v []Component) error {
	if len(v) != 1 {
		return fmt.Errorf("expected one value, found %d", len(v))
	}
	if v[0].Kind == Ident {
		if _, ok := listStyles[strings.ToLower(v[0].Text)]; ok {
			return nil
		}
	}
	return fmt.Errorf("expected none, disc, circle, square, decimal, decimal-leading-zero, lower-alpha, "+
		"upper-alpha, lower-roman, upper-roman, disclosure-open or disclosure-closed, found %q", v[0].String())
}

//nolint:gochecknoglobals // lookup table
var quoteWords = map[string]style.ContentItemKind{
	"open-quote": style.ContentOpenQuote, "close-quote": style.ContentCloseQuote,
	"no-open-quote": style.ContentNoOpenQuote, "no-close-quote": style.ContentNoCloseQuote,
}

// checkContent checks content: normal, none, or strings, url(), attr(),
// counter(), counters() and the quote words in a row, then, after a /,
// the text that stands for them.
func checkContent(v []Component) error {
	if len(v) == 1 && isWord(v[0], "normal", kwNone) {
		return nil
	}
	alt := false
	seen := 0
	for _, c := range v {
		switch {
		case isSlash(c):
			if alt || seen == 0 {
				return fmt.Errorf("unexpected / in content")
			}
			alt, seen = true, 0
			continue
		case c.Kind == String:
		case c.Kind == Function && strings.EqualFold(c.Text, "attr"):
			if a := items(c.Args); len(a) != 1 || a[0].Kind != Ident {
				return fmt.Errorf("attr() takes one attribute name, found %q", c.String())
			}
		case c.Kind == Function && strings.EqualFold(c.Text, "counter"):
			if err := checkCounterFunction(c, 1); err != nil {
				return err
			}
		case c.Kind == Function && strings.EqualFold(c.Text, "counters") && !alt:
			if err := checkCounterFunction(c, 2); err != nil {
				return err
			}
		case alt:
			return fmt.Errorf("expected a string, attr() or counter() after /, found %q", c.String())
		case c.Kind == URL || c.Kind == Function && strings.EqualFold(c.Text, "url"):
			if err := urlTerm(c); err != nil {
				return err
			}
		case c.Kind == Ident && quoteWords[strings.ToLower(c.Text)] != 0:
		default:
			return fmt.Errorf("expected normal, none, a string, url(), attr(), counter(), counters() or a quote, found %q",
				c.String())
		}
		seen++
	}
	if seen == 0 {
		return fmt.Errorf("content ends with /")
	}
	return nil
}

// checkCounterFunction checks counter(name, style) or, with fixed 2,
// counters(name, string, style); the style may be left out.
func checkCounterFunction(c Component, fixed int) error {
	parts := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	name := strings.ToLower(c.Text)
	if len(parts) < fixed || len(parts) > fixed+1 {
		return fmt.Errorf("%s() takes %d or %d arguments, found %d", name, fixed, fixed+1, len(parts))
	}
	if len(parts[0]) != 1 || parts[0][0].Kind != Ident || isWord(parts[0][0], kwNone) {
		return fmt.Errorf("%s() takes a counter's name first, found %q", name, join(parts[0]))
	}
	if fixed == 2 && (len(parts[1]) != 1 || parts[1][0].Kind != String) {
		return fmt.Errorf("counters() takes a string second, found %q", join(parts[1]))
	}
	if len(parts) > fixed {
		if err := checkListStyle(parts[fixed]); err != nil {
			return fmt.Errorf("%s(): the counter style: %w", name, err)
		}
	}
	return nil
}

// checkCounters checks counter-reset, counter-increment and counter-set:
// none, or names each with a whole number or none.
func checkCounters(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	named := false
	for i, c := range v {
		switch {
		case c.Kind == Ident && !isWord(c, kwNone, "inherit", "initial", kwUnset):
			named = true
		case c.Kind == Number && c.Num == float64(int(c.Num)) && i > 0 && v[i-1].Kind == Ident:
		case c.Kind == Function && strings.EqualFold(c.Text, "reversed"):
			return fmt.Errorf("reversed() is not supported; an ol counts down with its reversed attribute")
		default:
			return fmt.Errorf("expected none, or a counter's name and a whole number, found %q", c.String())
		}
	}
	if !named {
		return fmt.Errorf("expected a counter's name")
	}
	return nil
}

// checkQuotes checks quotes: auto, none, or pairs of strings.
func checkQuotes(v []Component) error {
	if len(v) == 1 && isWord(v[0], "auto", kwNone) {
		return nil
	}
	if len(v) == 0 || len(v)%2 != 0 {
		return fmt.Errorf("quotes takes auto, none or pairs of strings")
	}
	for _, c := range v {
		if c.Kind != String {
			return fmt.Errorf("quotes takes auto, none or pairs of strings, found %q", c.String())
		}
	}
	return nil
}

// content computes a content value; attr() reads the element's attribute.
func (x *ctx) content(v []Component) style.Content {
	switch {
	case len(v) == 1 && isWord(v[0], "normal"):
		return style.Content{}
	case len(v) == 1 && isWord(v[0], kwNone):
		return style.Content{Kind: style.ContentNone}
	}
	out := style.Content{Kind: style.ContentItems}
	text := func(s string) {
		if n := len(out.Items); n > 0 && out.Items[n-1].Kind == style.ContentString {
			out.Items[n-1].Text += s
			return
		}
		out.Items = append(out.Items, style.ContentItem{Kind: style.ContentString, Text: s})
	}
	for _, c := range v {
		if isSlash(c) {
			break
		}
		switch {
		case c.Kind == String:
			text(c.Text)
		case c.Kind == Ident:
			out.Items = append(out.Items, style.ContentItem{Kind: quoteWords[strings.ToLower(c.Text)]})
		case c.Kind == URL || strings.EqualFold(c.Text, "url"):
			out.Items = append(out.Items, style.ContentItem{Kind: style.ContentImage, Text: imageSource(x, []Component{c})})
		case strings.EqualFold(c.Text, "attr"):
			if x.node != nil {
				value, _ := x.node.Attr(strings.ToLower(items(c.Args)[0].Text))
				text(value)
			}
		default:
			out.Items = append(out.Items, counterItem(c))
		}
	}
	return out
}

func counterItem(c Component) style.ContentItem {
	parts := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	it := style.ContentItem{Kind: style.ContentCounter, Name: parts[0][0].Text, Style: style.ListDecimal}
	at := 1
	if strings.EqualFold(c.Text, "counters") {
		it.Kind, it.Text, at = style.ContentCounters, parts[1][0].Text, 2
	}
	if len(parts) > at {
		it.Style = listStyles[strings.ToLower(parts[at][0].Text)]
	}
	return it
}

// counters computes a counter property's list; a name without a number
// gets missing.
func counters(v []Component, missing int) []style.Counter {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	var out []style.Counter
	for _, c := range v {
		if c.Kind == Ident {
			out = append(out, style.Counter{Name: c.Text, Value: missing})
		} else if len(out) > 0 {
			out[len(out)-1].Value = int(c.Num)
		}
	}
	return out
}

func contentProps() map[string]prop {
	list := func(get func(*style.Computed) *[]style.Counter, missing int) prop {
		return field(get, func(_ *ctx, v []Component) []style.Counter { return counters(v, missing) })
	}
	//nolint:goconst // the content properties' table
	return map[string]prop{
		"content":           field(func(c *style.Computed) *style.Content { return &c.Content }, (*ctx).content),
		"counter-reset":     list(func(c *style.Computed) *[]style.Counter { return &c.CounterReset }, 0),
		"counter-increment": list(func(c *style.Computed) *[]style.Counter { return &c.CounterIncrement }, 1),
		"counter-set":       list(func(c *style.Computed) *[]style.Counter { return &c.CounterSet }, 0),
		"quotes": {
			set: func(x *ctx, v []Component) {
				x.c.Quotes, x.c.QuotesNone = nil, isWord(v[0], kwNone)
				for _, c := range v {
					if c.Kind == String {
						x.c.Quotes = append(x.c.Quotes, c.Text)
					}
				}
			},
			inherit: func(dst, src *style.Computed) { dst.Quotes, dst.QuotesNone = src.Quotes, src.QuotesNone },
		},
	}
}

// generates reports whether an element of tag takes ::before and ::after
// boxes: not the ones Chrome gives none (img, br, text fields, select,
// canvas), nor those ui draws whole.
func generates(tag dom.Tag) bool {
	switch tag {
	case dom.Img, dom.Svg, dom.Br, dom.Input, dom.Textarea, dom.Select, dom.Canvas, dom.Iframe, dom.Progress,
		dom.Meter, dom.Col, dom.Colgroup, dom.TextTag:
		return false
	case dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
		dom.Option, dom.Details, dom.Summary, dom.Dialog, dom.Th, dom.Td, dom.Hr, dom.Kbd, dom.Samp, dom.Pre, dom.Caption,
		dom.Ul, dom.Ol, dom.Li, dom.Dl, dom.Dt, dom.Dd, dom.H5, dom.H6, dom.Blockquote, dom.Figure, dom.Figcaption,
		dom.Form, dom.Address, dom.Hgroup, dom.Menu, dom.Search, dom.A, dom.B, dom.I, dom.U, dom.S, dom.Small,
		dom.Abbr, dom.Cite, dom.Mark, dom.Time, dom.Var, dom.Del, dom.Ins, dom.Dfn, dom.Bdi, dom.Bdo, dom.DataElement,
		dom.Center, dom.Big, dom.Font, dom.Sub, dom.Sup, dom.Q, dom.Fieldset, dom.Legend,
		dom.Table, dom.Thead, dom.Tbody, dom.Tfoot, dom.Tr:
	}
	return true
}
