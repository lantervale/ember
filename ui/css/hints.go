package css

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// hintedBelow are the attributes that hint an element's descendants, a
// table's cells and groups, and lang, whose language picks how their
// lines break and where they are underlined, keyed as
// dom.Pipeline.Changed.
//
//nolint:gochecknoglobals // lookup table
var hintedBelow = map[string]bool{
	"[cellpadding": true, "[valign": true, "[border": true, "[bordercolor": true, "[rules": true, langChanged: true,
}

// hints are the declarations an element's presentational attributes
// stand for, as HTML maps them and Chrome reads them: the width and
// height of a picture (those of the source it chose by, when it has
// them), a table or a cell, bgcolor, background, align, a cell's valign
// and nowrap, a table's border, bordercolor, frame, rules and
// cellspacing, with what its border, rules and cellpadding say of its
// cells and groups, body's text, a picture's border, hspace and vspace,
// br's clear, hr's size, width, color and noshade, font's size, face
// and color, the type of a list and a list item. What an attribute says
// that the subset has no declaration for hints nothing.
//
//nolint:funlen,gocyclo,cyclop // one case per kind of element
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
		dims := dom.DimensionSource(n)
		width, _ := dims.Attr("width")
		height, _ := dims.Attr("height")
		add(propWidth, hintedSize(width, true))
		add(propHeight, hintedSize(height, true))
		replacedAlign(attr("align"), add)
		if v, ok := n.Attr("border"); ok {
			if w, digits := leadingInt(v); digits {
				add("border-width", strconv.Itoa(w)+"px")
				add("border-style", kwSolid)
			}
		}
		if h := hintedSize(attr("hspace"), true); h != "" {
			add("margin-left", h)
			add("margin-right", h)
		}
		if v := hintedSize(attr("vspace"), true); v != "" {
			add("margin-top", v)
			add("margin-bottom", v)
		}
	case dom.Iframe:
		add(propWidth, hintedSize(attr("width"), true))
		add(propHeight, hintedSize(attr("height"), true))
		replacedAlign(attr("align"), add)
	case dom.Svg:
		add(propWidth, hintedLength(attr("width")))
		add(propHeight, hintedLength(attr("height")))
	case dom.Body:
		add("background-color", hintedColor(attr("bgcolor")))
		add("background-image", hintedImage(attr("background")))
		add("color", hintedColor(attr("text")))
	case dom.Table:
		add(propWidth, hintedSize(attr("width"), false))
		add(propHeight, hintedSize(attr("height"), false))
		add("background-color", hintedColor(attr("bgcolor")))
		add("background-image", hintedImage(attr("background")))
		add("border-spacing", hintedPixels(attr("cellspacing")))
		switch align := strings.ToLower(strings.TrimSpace(attr("align"))); align {
		case kwCenter:
			add("margin-left", "auto")
			add("margin-right", "auto")
		case kwLeft, kwRight:
			add("float", align)
		}
		tableBorders(n, add)
	case dom.Thead, dom.Tbody, dom.Tfoot, dom.Tr:
		add("background-color", hintedColor(attr("bgcolor")))
		add("background-image", hintedImage(attr("background")))
		add("text-align", hintedAlign(attr("align")))
		add("vertical-align", hintedCellAlign(attr("valign")))
		add(propHeight, hintedRowHeight(n))
		if n.Tag() != dom.Tr && tableRules(tableOf(n)) == rulesGroups {
			add("border-top", "1px solid")
			add("border-bottom", "1px solid")
		}
	case dom.Col, dom.Colgroup:
		add(propWidth, hintedSize(attr("width"), false))
		add("background-color", hintedColor(attr("bgcolor")))
		add("background-image", hintedImage(attr("background")))
		add("text-align", hintedAlign(attr("align")))
		add("vertical-align", hintedCellAlign(attr("valign")))
		if n.Tag() == dom.Colgroup && tableRules(tableOf(n)) == rulesGroups {
			add("border-left", "1px solid")
			add("border-right", "1px solid")
		}
	case dom.Td, dom.Th:
		add(propWidth, hintedSize(attr("width"), false))
		add(propHeight, hintedSize(attr("height"), false))
		add("background-color", hintedColor(attr("bgcolor")))
		add("background-image", hintedImage(attr("background")))
		add("text-align", hintedAlign(attr("align")))
		if _, ok := n.Attr("nowrap"); ok {
			add("white-space", "nowrap")
		}
		add("vertical-align", hintedCellAlign(attr("valign")))
		if table := tableOf(n); table != nil {
			v, _ := table.Attr("cellpadding")
			add("padding", hintedPixels(v))
			cellBorders(table, add)
		}
	case dom.Br:
		switch v := strings.ToLower(strings.TrimSpace(attr("clear"))); v {
		case kwAll, kwBoth:
			add("clear", kwBoth)
		case kwLeft, kwRight, kwNone:
			add("clear", v)
		}
	case dom.Hr:
		ruleHints(n, add)
	case dom.Font:
		add("font-size", hintedFontSize(attr("size")))
		add("font-family", hintedFamily(attr("face")))
		add("color", hintedColor(attr("color")))
	case dom.Div, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.H5, dom.H6:
		add("text-align", hintedAlign(attr("align")))
	case dom.Ol:
		add("list-style-type", hintedListType(attr("type"), true, false))
	case dom.Ul:
		add("list-style-type", hintedListType(attr("type"), false, true))
	case dom.Li:
		add("list-style-type", hintedListType(attr("type"), true, true))
	case dom.TextTag, dom.HTML, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside, dom.Article,
		dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button, dom.Input, dom.Select, dom.Option, dom.Textarea,
		dom.Details, dom.Summary, dom.Dialog, dom.Progress, dom.Meter, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre,
		dom.Caption, dom.Dl, dom.Dt, dom.Dd, dom.Blockquote, dom.Figure, dom.Figcaption,
		dom.Form, dom.Address, dom.Hgroup, dom.Menu, dom.Search, dom.A, dom.B, dom.I, dom.U, dom.S, dom.Small, dom.Abbr,
		dom.Cite, dom.Mark, dom.Time, dom.Var, dom.Del, dom.Ins, dom.Dfn, dom.Bdi, dom.Bdo, dom.DataElement,
		dom.Center, dom.Big, dom.Sub, dom.Sup, dom.Q, dom.Fieldset, dom.Legend,
		dom.PictureElement, dom.Source:
		return nil
	}
	if src.Len() == 0 {
		return nil
	}
	decls, _ := ParseInline(src.String())
	return decls
}

// replacedAlign is the align of a picture or a frame: left and right
// float it at the top of its line, the rest stand it on its line.
func replacedAlign(v string, add func(name, value string)) {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case kwLeft, kwRight:
		add("float", v)
		add("vertical-align", kwTop)
	case kwTop, kwBaseline:
		add("vertical-align", v)
	case kwBottom:
		add("vertical-align", kwBaseline)
	case "texttop":
		add("vertical-align", "text-top")
	case kwCenter, kwMiddle:
		add("vertical-align", "-webkit-baseline-middle")
	case "absmiddle", "abscenter":
		add("vertical-align", kwMiddle)
	case "absbottom":
		add("vertical-align", kwBottom)
	}
}

// ruleHints are a rule's: align its margins, width, size its
// thickness, and color or noshade a flat colour for its bevel.
func ruleHints(n *dom.Node, add func(name, value string)) {
	attr := func(name string) string {
		v, _ := n.Attr(name)
		return v
	}
	if v, ok := n.Attr("align"); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case kwLeft:
			add("margin-left", "0")
			add("margin-right", "auto")
		case kwRight:
			add("margin-left", "auto")
			add("margin-right", "0")
		default:
			add("margin-left", "auto")
			add("margin-right", "auto")
		}
	}
	if w, ok := n.Attr("width"); ok {
		if v, digits := leadingInt(w); digits && v == 0 && !strings.Contains(w, "%") {
			add(propWidth, "1px")
		} else {
			add(propWidth, hintedSize(w, false))
		}
	}
	if c := hintedColor(attr("color")); c != "" {
		add("border-style", kwSolid)
		add("border-color", c)
		add("background-color", c)
	} else if _, ok := n.Attr("noshade"); ok {
		add("border-style", kwSolid)
		add("border-color", "#808080")
		add("background-color", "#808080")
	}
	if v, ok := n.Attr("size"); ok {
		if size, _ := leadingInt(v); size > 1 {
			add(propHeight, strconv.Itoa(size-2)+"px")
		} else {
			add("border-bottom-width", "0")
		}
	}
}

// tableRule is what a table's rules attribute says.
type tableRule uint8

const (
	rulesUnset tableRule = iota
	rulesNone
	rulesGroups
	rulesRows
	rulesCols
	rulesAll
)

// tableRules is what table's rules attribute says, unset for none or a
// word it does not know.
func tableRules(table *dom.Node) tableRule {
	if table == nil {
		return rulesUnset
	}
	v, _ := table.Attr("rules")
	switch strings.ToLower(strings.TrimSpace(v)) {
	case kwNone:
		return rulesNone
	case "groups":
		return rulesGroups
	case "rows":
		return rulesRows
	case "cols":
		return rulesCols
	case kwAll:
		return rulesAll
	}
	return rulesUnset
}

// frameSides are the sides, top, right, bottom and left, a table's
// frame attribute draws.
//
//nolint:gochecknoglobals // lookup table
var frameSides = map[string][4]bool{
	"void": {}, "above": {true, false, false, false}, "below": {false, false, true, false},
	"hsides": {true, false, true, false}, "lhs": {false, false, false, true}, "rhs": {false, true, false, false},
	"vsides": {false, true, false, true}, "box": {true, true, true, true}, kwBorder: {true, true, true, true},
}

// tableBorders is what a table's border, bordercolor, frame and rules
// say of its own border: border its width, outset, solid with a
// bordercolor; frame a thin solid border on its sides, hidden on the
// others; rules a collapsed border, hidden where nothing else draws it.
// Of border and frame the later sets the width, as in Chrome.
func tableBorders(table *dom.Node, add func(name, value string)) {
	color, coloured := table.Attr("bordercolor")
	if coloured {
		add("border-color", hintedColor(color))
	}
	var width string
	var sides [4]bool
	bordered, framed := false, false
	for _, a := range table.Attrs() {
		switch strings.ToLower(a.Name) {
		case "border":
			w := hintedBorder(a.Value)
			width, bordered = strconv.Itoa(w)+"px", w > 0
		case "frame":
			if s, ok := frameSides[strings.ToLower(strings.TrimSpace(a.Value))]; ok {
				width, sides, framed = "1px", s, true
			}
		}
	}
	add("border-width", width)
	rules := tableRules(table)
	switch {
	case framed:
		words := make([]string, len(sides))
		for i, on := range sides {
			words[i] = map[bool]string{true: kwSolid, false: kwHidden}[on]
		}
		add("border-style", strings.Join(words, " "))
	case bordered && coloured:
		add("border-style", kwSolid)
	case bordered:
		add("border-style", "outset")
	case rules != rulesUnset && !coloured:
		add("border-style", kwHidden)
	}
	if rules != rulesUnset {
		add("border-collapse", "collapse")
	}
}

// cellBorders is what a cell's table says of its borders: rules=all, or
// a border with a bordercolor, a solid border, a border alone an inset
// one, rows and cols solid lines between rows or columns; each a pixel
// wide, in the colour its row hands down.
func cellBorders(table *dom.Node, add func(name, value string)) {
	v, bordered := table.Attr("border")
	bordered = bordered && hintedBorder(v) > 0
	_, coloured := table.Attr("bordercolor")
	switch tableRules(table) {
	case rulesNone, rulesGroups:
		return
	case rulesAll:
		add("border", "1px solid")
	case rulesRows:
		add("border-top", "1px solid")
		add("border-bottom", "1px solid")
	case rulesCols:
		add("border-left", "1px solid")
		add("border-right", "1px solid")
	case rulesUnset:
		switch {
		case bordered && coloured:
			add("border", "1px solid")
		case bordered:
			add("border", "1px inset")
		default:
			return
		}
	}
	add("border-color", "inherit")
}

// hintedBorder is a table's border attribute in pixels: its leading
// digits, 1 when it has none.
func hintedBorder(v string) int {
	n, ok := leadingInt(v)
	if !ok {
		return 1
	}
	return n
}

// leadingInt is the non-negative integer v starts with past white space,
// and whether it starts with one.
func leadingInt(v string) (int, bool) {
	v = strings.TrimLeft(v, " \t\r\n\f")
	end := 0
	for end < len(v) && v[end] >= '0' && v[end] <= '9' {
		end++
	}
	n, err := strconv.Atoi(v[:end])
	return n, err == nil
}

// fontElementSizes are the font sizes of a font element's size, from 1.
//
//nolint:gochecknoglobals // lookup table
var fontElementSizes = [...]string{"x-small", "small", kwMedium, "large", "x-large", "xx-large", "xxx-large"}

// hintedFontSize is a font element's size: 1 to 7, or a step from 3
// signed, clamped to the seven.
func hintedFontSize(v string) string {
	v = strings.TrimLeft(v, " \t\r\n\f")
	step := 0
	switch {
	case strings.HasPrefix(v, "+"):
		step, v = 1, v[1:]
	case strings.HasPrefix(v, "-"):
		step, v = -1, v[1:]
	}
	n, ok := leadingInt(v)
	if !ok || v != strings.TrimLeft(v, " \t\r\n\f") {
		return ""
	}
	if step != 0 {
		n = 3 + step*n
	}
	return fontElementSizes[min(max(n, 1), len(fontElementSizes))-1]
}

// hintedFamily is a font element's face as a font-family: its names as
// they are written, quoted but those of plain words.
func hintedFamily(v string) string {
	var names []string
	for name := range strings.SplitSeq(v, ",") {
		name = strings.Join(strings.Fields(name), " ")
		switch {
		case name == "":
		case plainWords(name):
			names = append(names, name)
		default:
			names = append(names, cssString(name))
		}
	}
	return strings.Join(names, ", ")
}

// plainWords reports whether s is identifiers apart by spaces, a
// family's name CSS reads unquoted.
func plainWords(s string) bool {
	for w := range strings.SplitSeq(s, " ") {
		for i, r := range w {
			letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r == '-' || r >= 0x80
			if !letter && (i == 0 || r < '0' || r > '9') {
				return false
			}
		}
	}
	return true
}

// cssString is s as a CSS string.
func cssString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\a `).Replace(s) + `"`
}

// hintedImage is a background attribute as a background-image.
func hintedImage(v string) string {
	if v = strings.TrimSpace(v); v == "" {
		return ""
	}
	return "url(" + cssString(v) + ")"
}

// tableOf is the table a cell, a row, a row group, a column or a column
// group belongs to.
func tableOf(n *dom.Node) *dom.Node {
	for p, depth := n.Parent(), 0; p != nil && depth < 3; p, depth = p.Parent(), depth+1 {
		if p.Tag() == dom.Table {
			return p
		}
	}
	return nil
}

// hintedListType is a list's or a list item's type attribute as the
// list-style-type it stands for: an ol's numbering, its case kept apart,
// a ul's bullet whatever its case, either for an li.
func hintedListType(v string, numbers, bullets bool) string {
	numbering := map[string]string{"1": kwDecimal, "a": "lower-alpha", "A": "upper-alpha", "i": "lower-roman",
		"I": "upper-roman"}
	if t, ok := numbering[v]; ok && numbers {
		return t
	}
	switch l := strings.ToLower(v); l {
	case kwDisc, kwCircle, kwSquare, kwNone:
		if bullets {
			return l
		}
	}
	return ""
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

// hintedColor is a colour attribute as a colour, by HTML's rules for
// parsing a legacy colour value: a named colour, #rgb, or else any text
// read as three runs of hexadecimal digits, what is not one counting 0.
func hintedColor(v string) string {
	v = strings.Trim(v, " \t\r\n\f")
	if v == "" || strings.EqualFold(v, "transparent") {
		return ""
	}
	if _, ok := named[strings.ToLower(v)]; ok {
		return strings.ToLower(v)
	}
	if len(v) == 4 && v[0] == '#' && strings.IndexFunc(v[1:], func(r rune) bool { return !isHex(r) }) < 0 {
		return v
	}
	var digits []byte
	for _, r := range v {
		switch {
		case r > 0xffff:
			digits = append(digits, '0', '0')
		case isHex(r):
			digits = append(digits, byte(r)) //nolint:gosec // a hexadecimal digit, ASCII
		default:
			digits = append(digits, '0')
		}
	}
	if len(digits) > 128 {
		digits = digits[:128]
	}
	if len(digits) > 0 && v[0] == '#' {
		digits = digits[1:]
	}
	for len(digits) == 0 || len(digits)%3 != 0 {
		digits = append(digits, '0')
	}
	size := len(digits) / 3
	parts := [3][]byte{digits[:size], digits[size : 2*size], digits[2*size:]}
	skip := max(size-8, 0)
	size -= skip
	for size > 2 && parts[0][skip] == '0' && parts[1][skip] == '0' && parts[2][skip] == '0' {
		skip++
		size--
	}
	var rgb [3]uint64
	for i, p := range parts {
		p = p[skip:]
		if len(p) > 2 {
			p = p[:2]
		}
		rgb[i], _ = strconv.ParseUint(string(p), 16, 8)
	}
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
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
