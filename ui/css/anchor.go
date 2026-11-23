package css

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

const (
	fnAnchorSize = "anchor-size"
	kwOutside    = "outside"
)

//nolint:gochecknoglobals,goconst // lookup tables
var (
	anchorSides = map[string]style.AnchorSide{
		"top": style.AnchorTop, "right": style.AnchorRight, "bottom": style.AnchorBottom, "left": style.AnchorLeft,
		"start": style.AnchorStart, "end": style.AnchorEnd, "self-start": style.AnchorSelfStart,
		"self-end": style.AnchorSelfEnd, "inside": style.AnchorInside, kwOutside: style.AnchorOutside,
		"center": style.AnchorCenter,
	}
	anchorSizes = map[string]style.AnchorSide{
		"width": style.AnchorWidth, "height": style.AnchorHeight, "block": style.AnchorBlock,
		"inline": style.AnchorInline, "self-block": style.AnchorSelfBlock, "self-inline": style.AnchorSelfInline,
	}
)

// anchorFn reads what anchor() or anchor-size() name of anchor: its
// name and side or size, in either order.
func anchorFn(name string, size bool, v []Component) (style.AnchorFn, error) {
	fn := style.AnchorFn{Size: size, Side: style.AnchorAxis}
	sided := false
	for _, c := range trim(v) {
		switch {
		case c.Kind == Whitespace:
		case dashed(c) && fn.Name == "":
			fn.Name = c.Text
		case !size && !sided && (c.Kind == Percentage || c.Kind == Function):
			d, err := dimOf(c)
			if err != nil || d.typ != typePercent || len(d.terms) > 0 {
				return fn, fmt.Errorf("anchor() takes a side, found %q", c.String())
			}
			fn.Side, fn.Percent, sided = style.AnchorPercent, float32(d.Pct), true
		case c.Kind == Ident && !sided:
			side, ok := anchorSides[strings.ToLower(c.Text)]
			if size {
				side, ok = anchorSizes[strings.ToLower(c.Text)]
			}
			if !ok {
				return fn, fmt.Errorf("%s() has no %q", name, c.Text)
			}
			fn.Side, sided = side, true
		default:
			return fn, fmt.Errorf("%s(): unexpected %q", name, c.String())
		}
	}
	if !size && !sided {
		return fn, fmt.Errorf("anchor() needs a side")
	}
	return fn, nil
}

// parseAnchor reads anchor(), an anchor name and a side in either order,
// or anchor-size(), a name and a size, either or both, each with a
// length-percentage after a comma for when the anchor is missing.
func parseAnchor(size bool, args []Component) (Dim, error) {
	name := "anchor"
	if size {
		name = fnAnchorSize
	}
	parts := splitOn(args, func(c Component) bool { return c.Kind == Comma })
	if len(parts) > 2 {
		return Dim{}, fmt.Errorf("%s() takes an anchor and a fallback", name)
	}
	fn, err := anchorFn(name, size, parts[0])
	if err != nil {
		return Dim{}, err
	}
	t := mathTerm{factor: 1, op: style.CalcAnchor, anchor: fn}
	if size {
		t.op = style.CalcAnchorSize
	}
	out := Dim{typ: typeLength, pct: true}
	if len(parts) == 2 {
		fb, fbErr := parseCalc(parts[1])
		if fbErr != nil {
			return Dim{}, fmt.Errorf("%s(): %w", name, fbErr)
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
