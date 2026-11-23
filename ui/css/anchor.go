package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // lookup tables
var (
	anchorSides = map[string]style.AnchorSide{
		"top": style.AnchorTop, "right": style.AnchorRight, "bottom": style.AnchorBottom, "left": style.AnchorLeft,
		"start": style.AnchorStart, "end": style.AnchorEnd, "self-start": style.AnchorSelfStart,
		"self-end": style.AnchorSelfEnd, "inside": style.AnchorInside, "outside": style.AnchorOutside,
		"center": style.AnchorCenter,
	}
	anchorSizes = map[string]style.AnchorSide{
		"width": style.AnchorWidth, "height": style.AnchorHeight, "block": style.AnchorBlock,
		"inline": style.AnchorInline, "self-block": style.AnchorSelfBlock, "self-inline": style.AnchorSelfInline,
	}
)

// parseAnchor reads anchor(), an anchor name and a side in either order,
// or anchor-size(), a name and a size, either or both, each with a
// length-percentage after a comma for when the anchor is missing.
func parseAnchor(size bool, args []Component) (Dim, error) {
	name := "anchor"
	if size {
		name = "anchor-size"
	}
	parts := splitOn(args, func(c Component) bool { return c.Kind == Comma })
	if len(parts) > 2 {
		return Dim{}, fmt.Errorf("%s() takes an anchor and a fallback", name)
	}
	fn := style.AnchorFn{Size: size, Side: style.AnchorAxis}
	sided := false
	for _, c := range trim(parts[0]) {
		switch {
		case c.Kind == Whitespace:
		case c.Kind == Ident && strings.HasPrefix(c.Text, "--") && fn.Name == "":
			fn.Name = c.Text
		case !size && !sided && c.Kind == Percentage:
			fn.Side, fn.Percent, sided = style.AnchorPercent, float32(c.Num), true
		case !size && !sided && c.Kind == Function:
			d, err := dimOf(c)
			if err != nil || d.typ != typePercent || len(d.terms) > 0 {
				return Dim{}, fmt.Errorf("anchor() takes a side, found %q", c.String())
			}
			fn.Side, fn.Percent, sided = style.AnchorPercent, float32(d.Pct), true
		case c.Kind == Ident && !sided:
			side, ok := anchorSides[strings.ToLower(c.Text)]
			if size {
				side, ok = anchorSizes[strings.ToLower(c.Text)]
			}
			if !ok {
				return Dim{}, fmt.Errorf("%s() has no %q", name, c.Text)
			}
			fn.Side, sided = side, true
		default:
			return Dim{}, fmt.Errorf("%s(): unexpected %q", name, c.String())
		}
	}
	if !size && !sided {
		return Dim{}, fmt.Errorf("anchor() needs a side")
	}
	t := mathTerm{factor: 1, op: style.CalcAnchor, anchor: fn}
	if size {
		t.op = style.CalcAnchorSize
	}
	out := Dim{typ: typeLength, pct: true}
	if len(parts) == 2 {
		fb, err := parseCalc(parts[1])
		if err != nil {
			return Dim{}, fmt.Errorf("%s(): %w", name, err)
		}
		switch {
		case fb.typ == typeNumber && !fb.pct && len(fb.terms) == 0 && fb.Num == 0:
			fb.typ = typeLength
		case fb.typ != typeLength && fb.typ != typePercent:
			return Dim{}, fmt.Errorf("%s() falls back to a length-percentage, found %s", name, fb.typ)
		}
		t.args = []Dim{fb}
		out.rel = fb.rel
	}
	out.terms = []mathTerm{t}
	return out, nil
}
