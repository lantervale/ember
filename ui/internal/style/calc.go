package style

import (
	"encoding/binary"
	"math"
	"unique"
)

// MaxLength is the length an infinite calc() stands for, the largest
// Chrome lays out.
const MaxLength = 1 << 25

// Calc is a length-percentage with math functions over percentages in it,
// as the steps that work it out on a stack, encoded in a string so that
// one value is one Handle and lengths compare as their fields.
type Calc struct{ code string }

// CalcStep is one step of a Calc: CalcPush puts Px and Percent of the
// reference on the stack, CalcScale multiplies its top by Px, and the
// other ops replace its top N values by what the op makes of them.
type CalcStep struct {
	Op          CalcOp
	N           int
	Px, Percent float64
}

// CalcOp is a step of a Calc, and what a math function does to its
// arguments: angles in radians, steps of round() second.
type CalcOp uint8

const (
	CalcPush CalcOp = iota
	CalcScale
	CalcSum
	CalcMul
	CalcDiv
	CalcMin
	CalcMax
	CalcRoundNearest
	CalcRoundUp
	CalcRoundDown
	CalcRoundToZero
	CalcMod
	CalcRem
	CalcAbs
	CalcSign
	CalcHypot
	CalcSin
	CalcCos
	CalcTan
	CalcAsin
	CalcAcos
	CalcAtan
	CalcAtan2
	CalcPow
	CalcSqrt
	CalcExp
	CalcLog
)

// NewCalc is the Calc of steps.
func NewCalc(steps []CalcStep) Calc {
	var b []byte
	for _, s := range steps {
		b = append(b, byte(s.Op))
		if s.Op == CalcPush || s.Op == CalcScale {
			b = binary.LittleEndian.AppendUint64(b, math.Float64bits(s.Px))
		}
		if s.Op == CalcPush {
			b = binary.LittleEndian.AppendUint64(b, math.Float64bits(s.Percent))
		}
		if s.Op != CalcPush && s.Op != CalcScale {
			b = binary.LittleEndian.AppendUint32(b, uint32(s.N)) //nolint:gosec // a count of arguments
		}
	}
	return Calc{code: string(b)}
}

// At is the value against ref, the length 100% stands for: NaN is 0 and
// an infinity the longest length, as at the top of a calc().
func (c *Calc) At(ref float32) float32 {
	var fixed [16]float64
	stack := fixed[:0]
	code := c.code
	f64 := func() float64 {
		v := math.Float64frombits(binary.LittleEndian.Uint64([]byte(code[:8])))
		code = code[8:]
		return v
	}
	for len(code) > 0 {
		op := CalcOp(code[0])
		code = code[1:]
		if op == CalcPush {
			px := f64()
			stack = append(stack, px+f64()*float64(ref)/100)
			continue
		}
		if op == CalcScale {
			stack[len(stack)-1] *= f64()
			continue
		}
		n := int(binary.LittleEndian.Uint32([]byte(code[:4])))
		code = code[4:]
		of := stack[len(stack)-n:]
		v := op.Apply(of)
		stack = append(stack[:len(stack)-n], v)
	}
	return Finite(stack[0], MaxLength)
}

// Finite is v at the top of a calc(): NaN is 0, and an infinity or a
// value past limit is limit, signed.
func Finite(v, limit float64) float32 {
	switch {
	case math.IsNaN(v):
		return 0
	case v > limit:
		return float32(limit)
	case v < -limit:
		return float32(-limit)
	}
	return float32(v)
}

// Apply is what op makes of args, as CSS Values 4 says: NaN through
// everything, infinities and signed zeros as IEEE 754.
func (op CalcOp) Apply(args []float64) float64 {
	a := args[0]
	if len(args) == 1 {
		if f, ok := calcUnary[op]; ok {
			return f(a)
		}
		return a
	}
	if f, ok := calcFolds[op]; ok {
		for _, b := range args[1:] {
			a = f(a, b)
		}
		return a
	}
	return calcBinary[op](a, args[1])
}

//nolint:gochecknoglobals,exhaustive // what the ops of one argument do to it
var calcUnary = map[CalcOp]func(float64) float64{
	CalcAbs: math.Abs, CalcHypot: math.Abs, CalcSign: sign, CalcSin: math.Sin, CalcCos: math.Cos, CalcTan: tan,
	CalcAsin: math.Asin, CalcAcos: math.Acos, CalcAtan: math.Atan, CalcSqrt: math.Sqrt, CalcExp: math.Exp,
	CalcLog:          math.Log,
	CalcRoundNearest: func(a float64) float64 { return round(CalcRoundNearest, a, 1) },
	CalcRoundUp:      func(a float64) float64 { return round(CalcRoundUp, a, 1) },
	CalcRoundDown:    func(a float64) float64 { return round(CalcRoundDown, a, 1) },
	CalcRoundToZero:  func(a float64) float64 { return round(CalcRoundToZero, a, 1) },
}

//nolint:gochecknoglobals,exhaustive // the ops that fold any number of arguments
var calcFolds = map[CalcOp]func(a, b float64) float64{
	CalcSum:   func(a, b float64) float64 { return a + b },
	CalcMul:   func(a, b float64) float64 { return a * b },
	CalcDiv:   func(a, b float64) float64 { return a / b },
	CalcMin:   func(a, b float64) float64 { return min(a, b) },
	CalcMax:   func(a, b float64) float64 { return max(a, b) },
	CalcHypot: math.Hypot,
}

//nolint:gochecknoglobals,exhaustive // what the ops of two arguments do to them
var calcBinary = map[CalcOp]func(a, b float64) float64{
	CalcRoundNearest: func(a, b float64) float64 { return round(CalcRoundNearest, a, b) },
	CalcRoundUp:      func(a, b float64) float64 { return round(CalcRoundUp, a, b) },
	CalcRoundDown:    func(a, b float64) float64 { return round(CalcRoundDown, a, b) },
	CalcRoundToZero:  func(a, b float64) float64 { return round(CalcRoundToZero, a, b) },
	CalcMod:          mod, CalcRem: math.Mod, CalcAtan2: math.Atan2, CalcPow: math.Pow,
	CalcLog: func(a, b float64) float64 { return math.Log(a) / math.Log(b) },
}

func sign(a float64) float64 {
	if a == 0 || math.IsNaN(a) {
		return a
	}
	return math.Copysign(1, a)
}

// tan is infinite at a quarter turn and its odd multiples, where the
// radians miss the pole by a rounding.
func tan(a float64) float64 {
	if math.IsInf(a, 0) || math.IsNaN(a) {
		return math.NaN()
	}
	deg := math.Mod(a*180/math.Pi, 360)
	switch {
	case math.Abs(deg-90) < 1e-9 || math.Abs(deg+270) < 1e-9:
		return math.Inf(1)
	case math.Abs(deg+90) < 1e-9 || math.Abs(deg-270) < 1e-9:
		return math.Inf(-1)
	}
	return math.Tan(a)
}

// round is a rounded to a multiple of b, halfway up for nearest, a zero
// signed as a was.
func round(op CalcOp, a, b float64) float64 {
	switch {
	case b == 0 || math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) && math.IsInf(b, 0):
		return math.NaN()
	case math.IsInf(a, 0):
		return a
	case math.IsInf(b, 0):
		switch {
		case op == CalcRoundUp && a > 0:
			return math.Inf(1)
		case op == CalcRoundDown && a < 0:
			return math.Inf(-1)
		}
		return math.Copysign(0, a)
	}
	b = math.Abs(b)
	lo := math.Floor(a/b) * b
	if lo == a {
		return a
	}
	hi := lo + b
	var r float64
	switch op { //nolint:exhaustive // the rounding ops
	case CalcRoundUp:
		r = hi
	case CalcRoundDown:
		r = lo
	case CalcRoundToZero:
		r = lo
		if a < 0 {
			r = hi
		}
	default:
		r = hi
		if a-lo < hi-a {
			r = lo
		}
	}
	if r == 0 {
		return math.Copysign(0, a)
	}
	return r
}

// mod is a modulo b with b's sign, where rem keeps a's.
func mod(a, b float64) float64 {
	if math.IsInf(b, 0) && !math.IsInf(a, 0) && !math.IsNaN(a) {
		if math.Signbit(a) != math.Signbit(b) {
			return math.NaN()
		}
		return a
	}
	r := math.Mod(a, b)
	if r != 0 && math.Signbit(r) != math.Signbit(b) {
		r += b
	}
	if r == 0 {
		return math.Copysign(0, b)
	}
	return r
}

// Less is l less o, a Calc when either holds one.
func (l Length) Less(o Length) Length {
	none := unique.Handle[Calc]{}
	if l.Calc == none && o.Calc == none {
		return Length{Kind: Fixed, Px: l.Px - o.Px, Percent: l.Percent - o.Percent, IsPercent: l.IsPercent || o.IsPercent}
	}
	c := Calc{code: l.code() + o.code() + NewCalc([]CalcStep{{Op: CalcScale, Px: -1}, {Op: CalcSum, N: 2}}).code}
	return Length{Kind: Fixed, Px: c.At(0), IsPercent: true, Calc: unique.Make(c)}
}

// code is the steps that leave l on a Calc's stack.
func (l Length) code() string {
	if l.Calc != (unique.Handle[Calc]{}) {
		return l.Calc.Value().code
	}
	return NewCalc([]CalcStep{{Op: CalcPush, Px: float64(l.Px), Percent: float64(l.Percent)}}).code
}
