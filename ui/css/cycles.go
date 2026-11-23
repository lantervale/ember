package css

import (
	"slices"
	"strings"
)

// unitCycles resolves the custom properties the element's font-size and
// line-height read before the rest, as Chrome does, so that a registered
// one reading a unit of the font, or lh, in what reads it is found in a
// cycle with it.
func (x *ctx) unitCycles(casc cascaded) {
	if x.env.props == nil {
		return
	}
	for _, name := range [...]string{propFontSize, propLineHeight} {
		d := casc.longhands[orderIndex[name]]
		if d == nil || !hasVar(d.Value) {
			continue
		}
		x.stack = append(x.stack, name)
		x.substitute(d.Value)
		x.stack = x.stack[:len(x.stack)-1]
	}
}

// unitCycle finds the cycle a registered custom property of value v
// makes with font-size when v reads a unit of the font, and with
// line-height when it reads lh, should that property be resolving it.
func (x *ctx) unitCycle(v []Component) {
	font, line := fontUnitsIn(v, x.root)
	if font {
		x.cycle(propFontSize)
	}
	if line {
		x.cycle(propLineHeight)
	}
}

// cycle marks name and every property resolving above it as in a cycle
// when name is resolving.
func (x *ctx) cycle(name string) {
	i := slices.Index(x.stack, name)
	if i < 0 {
		return
	}
	if x.cyclic == nil {
		x.cyclic = map[string]bool{}
	}
	for _, n := range x.stack[i:] {
		x.cyclic[n] = true
	}
}

// readFont notes what v reads of the element's font, for fontFirst.
func (x *ctx) readFont(v []Component) {
	font, line := fontUnitsIn(v, x.root)
	x.fontRead = x.fontRead || font
	x.lineRead = x.lineRead || line
}

// fontUnitsIn reports whether v has a unit of the element's font, lh
// among them, and whether it has lh; on the root the root's units count
// too, its font being theirs.
func fontUnitsIn(v []Component, root bool) (bool, bool) {
	font, line := false, false
	for _, c := range v {
		if c.Kind == Dimension {
			u := strings.ToLower(c.Unit)
			r, ok := relativeUnits[u]
			switch {
			case u == "em" || root && u == unitRem:
				font = true
			case ok && 1<<r&fontUnits != 0 && (r < unitRex || root):
				font = true
				line = line || r == unitLh
			}
		}
		if f, l := fontUnitsIn(c.Args, root); f {
			font, line = true, line || l
		}
	}
	return font, line
}
