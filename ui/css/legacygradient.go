package css

import (
	"slices"
	"strconv"
	"strings"
)

// legacyGradient rewrites c, Chrome's -webkit-linear-gradient() or
// -webkit-radial-gradient() of rest's name, as the plain gradient it
// draws, and reports whether it did. Their first part says where the
// gradient starts, not where it goes: a side is the one opposite to's,
// and an angle turns the other way from east (90deg - a). A radial one
// gives its position first, then its shape and size, contain and cover
// being closest-side and farthest-corner.
func legacyGradient(c *Component, rest string) bool {
	if c.Kind != Function {
		return false
	}
	parts := splitOn(c.Args, func(c Component) bool { return c.Kind == Comma })
	var head []Component
	var dropped int
	switch rest {
	case "linear-gradient":
		var ok bool
		if head, dropped, ok = legacyLinear(items(parts[0])); !ok {
			return false
		}
	case "radial-gradient":
		head, dropped = legacyRadial(parts)
	default:
		return false
	}
	var args []Component
	if len(head) > 0 {
		args = append(head, Component{Token: Token{Kind: Comma}})
	}
	for i, p := range parts[dropped:] {
		if i > 0 {
			args = append(args, Component{Token: Token{Kind: Comma}})
		}
		args = append(args, p...)
	}
	c.Text, c.Args = rest, args
	return true
}

func blank() Component { return Component{Token: Token{Kind: Whitespace, Text: " "}} }

// spaced is cs with a space between each two.
func spaced(cs ...Component) []Component {
	var out []Component
	for i, c := range cs {
		if i > 0 {
			out = append(out, blank())
		}
		out = append(out, c)
	}
	return out
}

//nolint:gochecknoglobals // lookup table
var oppositeSides = map[string]string{"top": "bottom", "bottom": "top", "left": "right", "right": "left"}

// legacyLinear is the plain first part of a legacy linear gradient's
// first part toks, and how many parts it stands for: none when the
// gradient begins with its stops. It is false for a corner or a centre,
// whose line Chrome draws from the box's own corner or point, which no
// plain direction says.
func legacyLinear(toks []Component) ([]Component, int, bool) {
	if len(toks) != 1 {
		return nil, 0, !slices.ContainsFunc(toks, func(t Component) bool { return isWord(t, "center") || isSide(t) })
	}
	if a, err := angle(toks[0]); err == nil && (toks[0].Kind != Number || toks[0].Num == 0) {
		deg := strconv.FormatFloat(90-a, 'f', -1, 64)
		return []Component{{Token: Token{Kind: Dimension, Num: 90 - a, Unit: "deg", Text: deg}}}, 1, true
	}
	if isWord(toks[0], "center") {
		return nil, 0, false
	}
	if !isSide(toks[0]) {
		return nil, 0, true
	}
	return spaced(word("to"), word(oppositeSides[strings.ToLower(toks[0].Text)])), 1, true
}

func isSide(t Component) bool { return isWord(t, kwTop, kwBottom, kwLeft, kwRight) }

//nolint:gochecknoglobals // lookup table
var legacySizes = map[string]string{"contain": "closest-side", "cover": "farthest-corner"}

// legacyRadial is the plain first part of a legacy radial gradient's
// first parts, its position and its shape and size, and how many parts
// they are.
func legacyRadial(parts [][]Component) ([]Component, int) {
	isShape := func(p []Component) bool {
		toks := items(p)
		for _, t := range toks {
			if !isWord(t, "circle", "ellipse", "closest-side", "closest-corner", "farthest-side",
				"farthest-corner", "contain", "cover") {
				return false
			}
		}
		return len(toks) > 0
	}
	var pos, shape []Component
	n := 0
	if len(parts) > 0 && !isShape(parts[0]) && checkPosition(items(parts[0])) == nil {
		pos, n = items(parts[0]), 1
	}
	if len(parts) > n && isShape(parts[n]) {
		for _, t := range items(parts[n]) {
			if as, ok := legacySizes[strings.ToLower(t.Text)]; ok {
				t = word(as)
			}
			shape = append(shape, t)
		}
		n++
	}
	head := spaced(shape...)
	if len(pos) > 0 {
		if len(head) > 0 {
			head = append(head, blank())
		}
		head = append(head, spaced(append([]Component{word("at")}, pos...)...)...)
	}
	return head, n
}
