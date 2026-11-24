package ui

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// The properties and keywords a copied style names.
const (
	cssTextDecoration       = "text-decoration"
	cssWhiteSpace           = "white-space"
	cssFontStyle            = "font-style"
	cssFontWeight           = "font-weight"
	cssFontFamily           = "font-family"
	cssFontSize             = "font-size"
	cssBackgroundColor      = "background-color"
	cssNormal               = "normal"
	cssColor                = "color"
	cssTextAlign            = "text-align"
	cssNone                 = "none"
	cssInitial              = "initial"
	cssFontVariantLigatures = "font-variant-ligatures"
	cssFontVariantCaps      = "font-variant-caps"
)

// decl is one property of a copied element's style attribute.
type decl struct {
	name, value string
	important   bool
	// percent is a value of one percentage, written as the computed one.
	percent bool
}

// block is a style attribute as Chrome's copy builds it: each property
// once, in the order first set.
type block []decl

func (b block) clone() block { return slices.Clone(b) }

// set gives d's property d's value, in place where b has it, else last.
func (b *block) set(d decl) {
	for i := range *b {
		if (*b)[i].name == d.name {
			(*b)[i] = d
			return
		}
	}
	*b = append(*b, d)
}

// remove drops the properties name sets.
func (b *block) remove(name string) {
	covered := covers(name)
	*b = slices.DeleteFunc(*b, func(d decl) bool {
		return slices.Contains(covered, d.name) || slices.Contains(covers(d.name), name)
	})
}

// covers are the properties name sets: a shorthand's longhands with it.
func covers(name string) []string {
	if long, ok := css.SideShorthands()[name]; ok {
		return append([]string{name}, long...)
	}
	switch name {
	case cssTextDecoration:
		return []string{name, "text-decoration-line", "text-decoration-thickness", "text-decoration-style",
			"text-decoration-color"}
	case cssWhiteSpace:
		return []string{name, "white-space-collapse", "text-wrap-mode"}
	case "font":
		return []string{name, cssFontStyle, cssFontVariantLigatures, cssFontVariantCaps, "font-variant-numeric",
			"font-variant-east-asian", "font-variant-alternates", "font-variant-position", cssFontWeight, "font-stretch",
			cssFontSize, "line-height", cssFontFamily, "font-kerning", "font-size-adjust", "font-optical-sizing"}
	case "background":
		return []string{name, cssBackgroundColor, "background-image", "background-position", "background-size",
			"background-repeat", "background-attachment", "background-origin", "background-clip"}
	}
	return []string{name}
}

// text is b as a style attribute's value, the sides of a shorthand that
// b holds whole folded into it, as Chrome writes them.
func (b block) text() string {
	sideOf := map[string]string{}
	for short, long := range css.SideShorthands() {
		for _, l := range long {
			sideOf[l] = short
		}
	}
	done := map[string]bool{}
	var parts []string
	for _, d := range b {
		if done[d.name] {
			continue
		}
		name, value := d.name, d.value
		if short, side := sideOf[d.name]; side {
			if v, ok := b.fold(css.SideShorthands()[short], d.important); ok {
				name, value = short, v
				for _, l := range css.SideShorthands()[short] {
					done[l] = true
				}
			}
		}
		if d.important {
			value += " !important"
		}
		parts = append(parts, name+": "+value+";")
	}
	return strings.Join(parts, " ")
}

// fold is the shorthand value of the sides long when b holds each of
// them at the same importance.
func (b block) fold(long []string, important bool) (string, bool) {
	v := make([]string, len(long))
	for i, l := range long {
		j := slices.IndexFunc(b, func(d decl) bool { return d.name == l })
		if j < 0 || b[j].important != important {
			return "", false
		}
		v[i] = b[j].value
	}
	if len(v) == 2 {
		if v[0] == v[1] {
			return v[0], true
		}
		return v[0] + " " + v[1], true
	}
	switch {
	case v[1] == v[3] && v[0] == v[2] && v[0] == v[1]:
		return v[0], true
	case v[1] == v[3] && v[0] == v[2]:
		return v[0] + " " + v[1], true
	case v[1] == v[3]:
		return v[0] + " " + v[1] + " " + v[2], true
	}
	return strings.Join(v, " "), true
}

// style is what a copy writes in e's style attribute, as Chrome's
// StyledMarkupSerializer builds it: the wrapping styles where e stands
// beside the highest node written, but those Chrome's own sheet gives e
// and those e's rules and style attribute set; then what its style
// attribute declares; the whole merged over what its rules declare, a
// percentage written as the computed value.
func (s *serializer) style(e *Node) block {
	var mine block
	if s.wraps(e) {
		mine = s.wrapping.clone()
		for _, name := range chromeDefaults(e) {
			mine.remove(name)
		}
		for _, name := range s.d.ownStyle(e) {
			mine.remove(name)
		}
	}
	rules, inline := s.d.trans.Declared(e)
	for _, x := range inline {
		mine.set(fromDeclared(x))
	}
	out := make(block, 0, len(rules)+len(mine))
	for _, x := range rules {
		out.set(fromDeclared(x))
	}
	for _, x := range mine {
		out.set(x)
	}
	st := dom.Data(e).Style
	for i, x := range out {
		if !x.percent || st == nil {
			continue
		}
		switch x.name {
		case cssFontSize:
			out[i].value = px(st.Font.Size)
		case "line-height":
			if !st.LineHeight.Normal {
				out[i].value = px(st.LineHeight.Px + st.LineHeight.Factor*st.Font.Size)
			}
		}
	}
	return out
}

func fromDeclared(x css.Declared) decl {
	return decl{name: x.Name, value: x.Value, important: x.Important, percent: x.Percent}
}

// ownStyle is the properties e's rules and style attribute set to other
// than its parent's do, which Chrome takes out of the wrapping styles.
func (d *Document) ownStyle(e *Node) []string {
	mine := func(n *Node) map[string]string {
		out := map[string]string{}
		if n == nil {
			return out
		}
		rules, inline := d.trans.Declared(n)
		for _, x := range append(rules, inline...) {
			out[x.Name] = x.Value
		}
		return out
	}
	own, parent := mine(e), mine(e.Parent())
	var names []string
	for name, v := range own {
		if pv, ok := parent[name]; !ok || pv != v {
			names = append(names, name)
		}
	}
	return names
}

// editingStyle is the text styles e shows its text in, as Chrome's copy
// writes them round what it wraps: e's computed inherited text styles,
// the background of the nearest element that has one, the decorations
// its text takes from its ancestors.
func (d *Document) editingStyle(e *Node) block {
	st := dom.Data(e).Style
	if st == nil {
		return nil
	}
	italic := map[bool]string{true: "italic", false: cssNormal}[st.Font.Italic]
	b := block{
		{name: cssColor, value: rgb(st.Color)},
		{name: cssFontFamily, value: css.FontFamily(st.Font.Family)},
		{name: cssFontSize, value: px(st.Font.Size)},
		{name: cssFontStyle, value: italic},
		{name: cssFontVariantLigatures, value: orNormal(st.FontVariant.Ligatures)},
		{name: cssFontVariantCaps, value: [...]string{cssNormal, "small-caps", "all-small-caps", "petite-caps",
			"all-petite-caps", "unicase", "titling-caps"}[st.Font.Caps]},
		{name: cssFontWeight, value: num(st.Font.Weight)},
		{name: "letter-spacing", value: spacingText(st.LetterSpacing, cssNormal)},
		{name: "orphans", value: strconv.Itoa(st.Orphans)},
		{name: cssTextAlign, value: [...]string{"start", "end", "left", "right", "center", "justify", "-webkit-left",
			"-webkit-right", "-webkit-center"}[st.TextAlign]},
		{name: "text-indent", value: indent(st.TextIndent)},
		{name: "text-transform", value: [...]string{cssNone, "uppercase", "lowercase", "capitalize"}[st.TextTransform]},
		{name: "widows", value: strconv.Itoa(st.Widows)},
		{name: "word-spacing", value: spacingText(st.WordSpacing, "0px")},
		{name: "-webkit-text-stroke-width", value: px(st.TextStroke.Width)},
		{name: cssWhiteSpace, value: [...]string{cssNormal, "nowrap", "pre", "pre-wrap", "pre-line",
			"preserve-breaks nowrap", "break-spaces", "break-spaces nowrap"}[st.WhiteSpace]},
	}
	for a := e; a != nil; a = a.Parent() {
		if as := dom.Data(a).Style; as != nil && as.Background.A > 0 {
			b = append(b, decl{name: cssBackgroundColor, value: rgb(as.Background)})
			break
		}
	}
	if lines := decorationLines(st); lines != "" {
		return append(b, decl{name: cssTextDecoration, value: lines})
	}
	return append(b, decl{name: "text-decoration-thickness", value: cssInitial},
		decl{name: "text-decoration-style", value: cssInitial}, decl{name: "text-decoration-color", value: cssInitial})
}

// decorationLines are the lines st's text takes from its ancestors and
// its own text-decoration, as text-decoration-line writes them.
func decorationLines(st *style.Computed) string {
	var on style.DecorationLine
	for _, dec := range st.Decorations {
		on |= dec.Line
	}
	on |= st.TextDecoration.Line
	var names []string
	for _, l := range []struct {
		line style.DecorationLine
		name string
	}{{style.Underline, "underline"}, {style.Overline, "overline"}, {style.LineThrough, "line-through"}} {
		if on&l.line != 0 {
			names = append(names, l.name)
		}
	}
	return strings.Join(names, " ")
}

// chromeDefaults are the text styles Chrome's own sheet gives e, which
// its copy leaves out of the wrapping styles it writes on e: a paste
// gives them again.
func chromeDefaults(e *Node) []string {
	font := []string{cssFontStyle, cssFontVariantLigatures, cssFontVariantCaps, cssFontWeight, cssFontSize,
		cssFontFamily}
	switch e.Tag() { //nolint:exhaustive // the elements Chrome's sheet styles the text of
	case dom.B, dom.Strong, dom.H4:
		return []string{cssFontWeight}
	case dom.I, dom.Em, dom.Cite, dom.Var, dom.Dfn, dom.Address:
		return []string{cssFontStyle}
	case dom.Code, dom.Kbd, dom.Samp:
		return []string{cssFontFamily}
	case dom.Pre:
		return []string{cssFontFamily, cssWhiteSpace}
	case dom.H1, dom.H2, dom.H3, dom.H5, dom.H6:
		return []string{cssFontSize, cssFontWeight}
	case dom.Th:
		return []string{cssFontWeight, cssTextAlign}
	case dom.Caption, dom.Center:
		return []string{cssTextAlign}
	case dom.Table:
		return []string{"text-indent"}
	case dom.Small, dom.Big, dom.Sub, dom.Sup:
		return []string{cssFontSize}
	case dom.U, dom.Ins, dom.S, dom.Del:
		return []string{cssTextDecoration}
	case dom.A:
		if _, ok := e.Attr("href"); ok {
			return []string{cssColor, cssTextDecoration}
		}
	case dom.Abbr:
		if _, ok := e.Attr("title"); ok {
			return []string{cssTextDecoration}
		}
	case dom.Mark, dom.Dialog:
		return []string{cssBackgroundColor, cssColor}
	case dom.Hr:
		return []string{cssColor}
	case dom.Input, dom.Select, dom.Button:
		return append(font, cssColor, "letter-spacing", "word-spacing", "text-transform", "text-indent", cssTextAlign)
	case dom.Textarea:
		return append(font, cssColor, "letter-spacing", "word-spacing", "text-transform", "text-indent", cssTextAlign,
			cssWhiteSpace)
	}
	return nil
}

func rgb(c color.Color) string {
	s := c.SRGB()
	ch := func(v float32) int { return int(math.Round(float64(v) * 255)) }
	if c.A >= 1 {
		return fmt.Sprintf("rgb(%d, %d, %d)", ch(s[0]), ch(s[1]), ch(s[2]))
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", ch(s[0]), ch(s[1]), ch(s[2]), num(c.A))
}

func num(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }

func orNormal(s string) string {
	if s == "" {
		return cssNormal
	}
	return s
}

// spacingText writes a letter-spacing or a word-spacing, zero as none.
func spacingText(s style.Spacing, none string) string {
	switch {
	case s.Px == 0 && s.Percent == 0:
		return none
	case s.Px == 0:
		return num(s.Percent) + "%"
	}
	return px(s.Px)
}

func indent(l style.Length) string {
	if l.IsPercent && l.Px == 0 {
		return num(l.Percent) + "%"
	}
	return px(l.Px)
}
