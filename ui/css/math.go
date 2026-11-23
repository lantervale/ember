package css

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// mathConstants are the numbers a math function may name.
//
//nolint:gochecknoglobals // lookup table
var mathConstants = map[string]float64{
	"e": math.E, "pi": math.Pi, "infinity": math.Inf(1), "-infinity": math.Inf(-1), "nan": math.NaN(),
}

// angleUnits are the angles in degrees.
//
//nolint:gochecknoglobals // lookup table
var angleUnits = map[string]float64{"deg": 1, "rad": 180 / math.Pi, "grad": 0.9, "turn": 360}

// roundings are round()'s strategies.
//
//nolint:gochecknoglobals // lookup table
var roundings = map[string]style.CalcOp{
	"nearest": style.CalcRoundNearest, "up": style.CalcRoundUp, "down": style.CalcRoundDown,
	"to-zero": style.CalcRoundToZero,
}

//nolint:gochecknoglobals // lookup table
var mathOps = map[string]style.CalcOp{
	"min": style.CalcMin, "max": style.CalcMax, "hypot": style.CalcHypot, "mod": style.CalcMod,
	"rem": style.CalcRem, "abs": style.CalcAbs, "sign": style.CalcSign, "sin": style.CalcSin, "cos": style.CalcCos,
	"tan": style.CalcTan, "asin": style.CalcAsin, "acos": style.CalcAcos, "atan": style.CalcAtan,
	"atan2": style.CalcAtan2, "pow": style.CalcPow, "sqrt": style.CalcSqrt, "exp": style.CalcExp,
	"log": style.CalcLog,
}

// parseMath reads the math function name of args: calc(), the
// comparisons min(), max() and clamp(), the stepped round(), mod() and
// rem(), the trigonometric sin() to atan2(), the exponential pow() to
// exp(), and abs() and sign(), typed as CSS Values 4 types them.
func parseMath(name string, args []Component) (Dim, error) {
	if name == fnCalc {
		return parseCalc(args)
	}
	parts := splitOn(args, func(c Component) bool { return c.Kind == Comma })
	op, ok := mathOps[name]
	switch {
	case name == "clamp":
		return parseClamp(parts)
	case name == "round":
		op = style.CalcRoundNearest
		if len(parts) > 0 {
			if p := trim(parts[0]); len(p) == 1 && p[0].Kind == Ident {
				if op, ok = roundings[strings.ToLower(p[0].Text)]; !ok {
					return Dim{}, fmt.Errorf("round() has no strategy %q", p[0].Text)
				}
				parts = parts[1:]
			}
		}
	case !ok:
		return Dim{}, fmt.Errorf("%s() is not supported here", name)
	}
	ds := make([]Dim, len(parts))
	for i, p := range parts {
		d, err := parseCalc(p)
		if err != nil {
			return Dim{}, fmt.Errorf("%s(): %w", name, err)
		}
		ds[i] = d
	}
	lo, hi := mathArity(op)
	if len(ds) < lo || hi > 0 && len(ds) > hi {
		return Dim{}, fmt.Errorf("%s() takes %s", name, arity(lo, hi))
	}
	typ := ds[0].typ
	for _, d := range ds[1:] {
		if typ, ok = unify(typ, d.typ); !ok {
			return Dim{}, fmt.Errorf("%s() takes values of one type, found %s and %s", name, ds[0].typ, d.typ)
		}
	}
	return typed(name, op, typ, ds)
}

// typed is op over ds, whose type is typ, typed as op makes it: a sign is
// a number, trigonometry takes angles in radians and gives degrees.
func typed(name string, op style.CalcOp, typ dimType, ds []Dim) (Dim, error) {
	switch op {
	case style.CalcRoundNearest, style.CalcRoundUp, style.CalcRoundDown, style.CalcRoundToZero:
		if len(ds) == 1 && typ != typeNumber {
			return Dim{}, fmt.Errorf("round() of %s needs a step", typ)
		}
	case style.CalcSign:
		typ = typeNumber
	case style.CalcSin, style.CalcCos, style.CalcTan:
		switch typ {
		case typeAngle:
			ds[0] = ds[0].scale(math.Pi / 180)
		case typeNumber:
		default:
			return Dim{}, fmt.Errorf("%s() takes an angle or a number, found %s", name, typ)
		}
		typ = typeNumber
	case style.CalcAsin, style.CalcAcos, style.CalcAtan:
		if typ != typeNumber {
			return Dim{}, fmt.Errorf("%s() takes a number, found %s", name, typ)
		}
		return node(op, typeAngle, ds...).scale(180 / math.Pi), nil
	case style.CalcAtan2:
		return node(op, typeAngle, ds...).scale(180 / math.Pi), nil
	case style.CalcPow, style.CalcSqrt, style.CalcExp, style.CalcLog:
		if typ != typeNumber {
			return Dim{}, fmt.Errorf("%s() takes numbers, found %s", name, typ)
		}
	case style.CalcPush, style.CalcScale, style.CalcSum, style.CalcMul, style.CalcDiv, style.CalcMin, style.CalcMax,
		style.CalcMod, style.CalcRem, style.CalcAbs, style.CalcHypot:
	}
	return node(op, typ, ds...), nil
}

// parseClamp reads clamp(): a minimum, a value and a maximum, the bounds
// none for no bound.
func parseClamp(parts [][]Component) (Dim, error) {
	if len(parts) != 3 {
		return Dim{}, fmt.Errorf("clamp() takes a minimum, a value and a maximum")
	}
	var ds [3]Dim
	none := [3]bool{}
	typ := dimType{}
	for i, p := range parts {
		if t := trim(p); i != 1 && len(t) == 1 && isWord(t[0], kwNone) {
			none[i] = true
			continue
		}
		d, err := parseCalc(p)
		if err != nil {
			return Dim{}, fmt.Errorf("clamp(): %w", err)
		}
		ds[i] = d
		if i == 1 {
			typ = d.typ
		}
	}
	for i := range ds {
		if none[i] || i == 1 {
			continue
		}
		var ok bool
		if typ, ok = unify(typ, ds[i].typ); !ok {
			return Dim{}, fmt.Errorf("clamp() takes values of one type, found %s and %s", ds[1].typ, ds[i].typ)
		}
	}
	v := ds[1]
	if !none[2] {
		v = node(style.CalcMin, typ, v, ds[2])
	}
	if !none[0] {
		v = node(style.CalcMax, typ, ds[0], v)
	}
	return v, nil
}

// mathArity is how many arguments op takes, hi 0 for any number.
func mathArity(op style.CalcOp) (int, int) {
	switch op {
	case style.CalcMin, style.CalcMax, style.CalcHypot:
		return 1, 0
	case style.CalcRoundNearest, style.CalcRoundUp, style.CalcRoundDown, style.CalcRoundToZero, style.CalcLog:
		return 1, 2
	case style.CalcMod, style.CalcRem, style.CalcAtan2, style.CalcPow:
		return 2, 2
	case style.CalcPush, style.CalcScale, style.CalcSum, style.CalcMul, style.CalcDiv, style.CalcAbs,
		style.CalcSign, style.CalcSin, style.CalcCos, style.CalcTan, style.CalcAsin, style.CalcAcos, style.CalcAtan,
		style.CalcSqrt, style.CalcExp:
	}
	return 1, 1
}

func arity(lo, hi int) string {
	switch {
	case hi == 0:
		return fmt.Sprintf("%d argument or more", lo)
	case lo == 1 && hi == 1:
		return "one argument"
	case lo == hi:
		return fmt.Sprintf("%d arguments", lo)
	}
	return fmt.Sprintf("%d or %d arguments", lo, hi)
}

// signPercents is d with the percentages inside its sign()s read as
// pixels, a percentage of 100 pixels: Chrome's line-height reads them so,
// where the rest of its percentages are of the font size.
func signPercents(d Dim) Dim {
	if len(d.terms) == 0 {
		return d
	}
	d.terms = slices.Clone(d.terms)
	d.pct = d.Pct != 0
	for i, t := range d.terms {
		args := make([]Dim, len(t.args))
		for k, a := range t.args {
			if t.op == style.CalcSign {
				args[k] = percentsAsPixels(a)
			} else {
				args[k] = signPercents(a)
			}
			d.pct = d.pct || args[k].pct
		}
		d.terms[i].args = args
	}
	return d
}

func percentsAsPixels(d Dim) Dim {
	d.Px, d.Pct, d.pct = d.Px+d.Pct, 0, false
	if d.typ == typePercent {
		d.typ = typeLength
	}
	if len(d.terms) > 0 {
		d.terms = slices.Clone(d.terms)
		for i, t := range d.terms {
			args := make([]Dim, len(t.args))
			for k, a := range t.args {
				args[k] = percentsAsPixels(a)
			}
			d.terms[i].args = args
		}
	}
	return d
}
