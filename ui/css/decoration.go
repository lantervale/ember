package css

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals,goconst // lookup tables and grammar terms, built once
var (
	decorationLines = map[string]style.DecorationLine{
		"underline": style.Underline, "overline": style.Overline, "line-through": style.LineThrough,
	}
	decorationStyles = map[string]style.DecorationStyle{
		"solid": style.DecorationSolid, "double": style.DecorationDouble,
		"dotted": style.DecorationDotted, "dashed": style.DecorationDashed,
	}
	decorationThickness = either(dimTerm(allowPercent|nonNegative), words("auto", "from-font"))
)

func decorationStyle(c Component) error {
	if isWord(c, "wavy") {
		return fmt.Errorf("wavy lines are not drawn; use solid, double, dotted or dashed")
	}
	return words("solid", "double", "dotted", "dashed")(c)
}

// checkDecorationLine takes none, or underline, overline and line-through,
// each once in any order.
func checkDecorationLine(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	if len(v) == 0 {
		return fmt.Errorf("expected none, underline, overline or line-through")
	}
	var seen style.DecorationLine
	for _, c := range v {
		l, ok := decorationLines[strings.ToLower(c.Text)]
		switch {
		case isWord(c, kwNone):
			return fmt.Errorf("none stands alone")
		case isWord(c, "blink"):
			return fmt.Errorf("blink is not drawn")
		case c.Kind != Ident || !ok:
			return fmt.Errorf("expected none, underline, overline or line-through, found %q", c.String())
		case seen&l != 0:
			return fmt.Errorf("%s is given twice", c.String())
		}
		seen |= l
	}
	return nil
}

func decorationLine(_ *ctx, v []Component) style.DecorationLine {
	var l style.DecorationLine
	for _, c := range v {
		l |= decorationLines[strings.ToLower(c.Text)]
	}
	return l
}

// expandTextDecoration splits text-decoration: its lines, a style, a
// colour and a thickness, in any order.
func expandTextDecoration(v []Component, out map[string][]Component) error {
	if len(v) == 0 {
		return fmt.Errorf("expected a line, such as underline or none")
	}
	var lines []Component
	for _, c := range v {
		var name string
		switch {
		case isWord(c, kwNone, "blink") || c.Kind == Ident && decorationLines[strings.ToLower(c.Text)] != 0:
			lines = append(lines, c)
			continue
		case isWord(c, "wavy"):
			return decorationStyle(c)
		case decorationStyle(c) == nil:
			name = propTextDecorationStyle
		case decorationThickness(c) == nil:
			name = propTextDecorationThickness
		case colorTerm(c) == nil:
			name = propTextDecorationColor
		default:
			return fmt.Errorf("%q is no line, style, colour or thickness of a text-decoration", c.String())
		}
		if _, taken := out[name]; taken {
			return fmt.Errorf("%q is a second %s", c.String(), name)
		}
		out[name] = []Component{c}
	}
	if lines == nil {
		return nil
	}
	out["text-decoration-line"] = lines
	return checkDecorationLine(lines)
}

// decorations are the decorations drawn on the element's text: its
// parent's, which a float, a box out of the flow and an atomic inline
// leave behind, then its own. A box that is not inline starts lines of
// its own, whose root inline box decorates them, so those it takes are
// drawn in its font. The parent's slice is kept where nothing changes.
func (x *ctx) decorations() []style.Decoration {
	c := x.c
	var out []style.Decoration
	if !stopsDecorations(c) {
		out = x.parent.Decorations
		if c.Display != style.Inline && c.Display != style.Contents {
			out = decoratedIn(out, c.Font)
		}
	}
	if c.TextDecoration.Line == 0 {
		return out
	}
	return append(slices.Clip(out), style.Decoration{TextDecoration: c.TextDecoration, Font: c.Font})
}

func stopsDecorations(c *style.Computed) bool {
	switch c.Display {
	case style.InlineBlock, style.InlineFlex, style.InlineGrid:
		return true
	case style.Inline, style.Block, style.FlexDisplay, style.GridDisplay, style.DisplayNone, style.Contents:
		// in the flow of its parent's text
	}
	return c.Float != style.FloatNone || c.Position == style.Absolute || c.Position == style.FixedPosition
}

// decoratedIn is ds with f as each one's font, in a copy when one changes.
func decoratedIn(ds []style.Decoration, f style.Font) []style.Decoration {
	var out []style.Decoration
	for i := range ds {
		if ds[i].Font.Equal(f) {
			continue
		}
		if out == nil {
			out = slices.Clone(ds)
		}
		out[i].Font = f
	}
	if out == nil {
		return ds
	}
	return out
}
