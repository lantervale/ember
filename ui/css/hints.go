package css

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// hintedBelow are the attributes that hint an element's descendants, a
// table's cells, keyed as dom.Pipeline.Changed.
//
//nolint:gochecknoglobals // lookup table
var hintedBelow = map[string]bool{"[cellpadding": true, "[valign": true}

// hints are the declarations an element's presentational attributes
// stand for, as HTML maps them and Chrome reads them: the width and
// height of a picture, a table or a cell, bgcolor, align, a cell's
// valign and nowrap, a table's cellspacing and its cellpadding on its
// cells, body's text. What an attribute says that the subset has no
// declaration for hints nothing.
func hints(n *dom.Node) []Declaration {
	var src strings.Builder
	add := func(name, value string) {
		if value != "" {
			src.WriteString(name + ": " + value + ";")
		}
	}
	attr := func(name string) string {
		v, _ := n.Attr(name)
		return v
	}
	switch n.Tag() {
	case dom.Img:
		add(propWidth, hintedSize(attr("width"), true))
		add(propHeight, hintedSize(attr("height"), true))
	case dom.Iframe:
		add(propWidth, hintedSize(attr("width"), true))
		add(propHeight, hintedSize(attr("height"), true))
	case dom.Svg:
		add(propWidth, hintedLength(attr("width")))
		add(propHeight, hintedLength(attr("height")))
	case dom.Body:
		add("background-color", hintedColor(attr("bgcolor")))
		add("color", hintedColor(attr("text")))
	case dom.Table:
		add(propWidth, hintedSize(attr("width"), false))
		add(propHeight, hintedSize(attr("height"), false))
		add("background-color", hintedColor(attr("bgcolor")))
		add("border-spacing", hintedPixels(attr("cellspacing")))
		if strings.EqualFold(attr("align"), "center") {
			add("margin-left", "auto")
			add("margin-right", "auto")
		}
	case dom.Thead, dom.Tbody, dom.Tfoot, dom.Tr:
		add("background-color", hintedColor(attr("bgcolor")))
		add("text-align", hintedAlign(attr("align")))
		add("vertical-align", hintedCellAlign(attr("valign")))
		add(propHeight, hintedRowHeight(n))
	case dom.Td, dom.Th:
		add(propWidth, hintedSize(attr("width"), false))
		add(propHeight, hintedSize(attr("height"), false))
		add("background-color", hintedColor(attr("bgcolor")))
		add("text-align", hintedAlign(attr("align")))
		if _, ok := n.Attr("nowrap"); ok {
			add("white-space", "nowrap")
		}
		add("vertical-align", hintedCellAlign(attr("valign")))
		add("padding", hintedCellPadding(n))
	case dom.Div, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.H5, dom.H6:
		add("text-align", hintedAlign(attr("align")))
	case dom.TextTag, dom.HTML, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside, dom.Article,
		dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button, dom.Input, dom.Select, dom.Option, dom.Textarea,
		dom.Details, dom.Summary, dom.Dialog, dom.Progress, dom.Meter, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre,
		dom.Caption, dom.Ul, dom.Ol, dom.Li, dom.Dl, dom.Dt, dom.Dd, dom.Blockquote, dom.Figure, dom.Figcaption,
		dom.Form, dom.Address, dom.Hgroup, dom.Menu, dom.Search, dom.A, dom.B, dom.I, dom.U, dom.S, dom.Small, dom.Abbr,
		dom.Cite, dom.Mark, dom.Time, dom.Var, dom.Del, dom.Ins, dom.Dfn, dom.Bdi, dom.Bdo, dom.DataElement, dom.Br,
		dom.Center, dom.Big, dom.Font, dom.Sub, dom.Sup, dom.Q, dom.Fieldset, dom.Legend:
		return nil
	}
	if src.Len() == 0 {
		return nil
	}
	decls, _ := ParseInline(src.String())
	return decls
}

// hintedSize is a dimension attribute as a length: a number of pixels or
// a percentage, whatever follows it; zero says nothing but on a picture.
func hintedSize(v string, zero bool) string {
	v = strings.TrimLeft(v, " \t\r\n\f")
	end := 0
	for end < len(v) && (v[end] >= '0' && v[end] <= '9' || v[end] == '.') {
		end++
	}
	f, err := strconv.ParseFloat(v[:end], 32)
	if err != nil || f == 0 && !zero {
		return ""
	}
	unit := "px"
	if strings.HasPrefix(v[end:], "%") {
		unit = "%"
	}
	return strconv.FormatFloat(f, 'f', -1, 32) + unit
}

// hintedLength is the width or height attribute of an svg element as a
// length: a number of pixels, or one with a unit of CSS.
func hintedLength(v string) string {
	v = strings.TrimSpace(v)
	end := 0
	for end < len(v) && (v[end] >= '0' && v[end] <= '9' || v[end] == '.') {
		end++
	}
	if _, err := strconv.ParseFloat(v[:end], 32); err != nil {
		return ""
	}
	switch unit := strings.ToLower(v[end:]); unit {
	case "":
		return v + "px"
	case "px", "em", unitRem, "%", "pt", "pc", "in", "cm", "mm":
		return v[:end] + unit
	}
	return ""
}

// hintedPixels is an attribute that counts pixels, cellspacing or
// cellpadding, as a length.
func hintedPixels(v string) string {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return ""
	}
	return strconv.Itoa(n) + "px"
}

// hintedColor is a colour attribute as a colour: a name, or hexadecimal
// digits with their # or without.
func hintedColor(v string) string {
	v = strings.TrimSpace(v)
	hex := strings.TrimPrefix(v, "#")
	digits, letters := true, hex != ""
	for _, r := range hex {
		digit := r >= '0' && r <= '9'
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		if !digit && !letter {
			return ""
		}
		if !digit && (r|0x20) > 'f' {
			digits = false
		}
		letters = letters && letter
	}
	switch {
	case digits && (len(hex) == 3 || len(hex) == 6):
		return "#" + hex
	case letters && hex == v:
		return v
	}
	return ""
}

// hintedAlign is an align attribute as a text-align: left, right and
// center place blocks too, as Chrome's -webkit- values.
func hintedAlign(v string) string {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case kwCenter, kwMiddle:
		return "-webkit-center"
	case kwLeft, kwRight:
		return "-webkit-" + v
	case "justify":
		return v
	}
	return ""
}

// hintedCellAlign is the vertical-align a valign says, a cell's own or the one its row
// or row group hands down.
func hintedCellAlign(v string) string {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case kwTop, kwBaseline, kwMiddle, kwBottom:
		return v
	}
	return ""
}

// hintedRowHeight is the height a row's height says; a row group's says nothing.
func hintedRowHeight(n *dom.Node) string {
	if n.Tag() != dom.Tr {
		return ""
	}
	v, _ := n.Attr("height")
	return hintedSize(v, false)
}

// hintedCellPadding is the cellpadding of a cell's table.
func hintedCellPadding(cell *dom.Node) string {
	for n, depth := cell.Parent(), 0; n != nil && depth < 3; n, depth = n.Parent(), depth+1 {
		if n.Tag() == dom.Table {
			v, _ := n.Attr("cellpadding")
			return hintedPixels(v)
		}
	}
	return ""
}
