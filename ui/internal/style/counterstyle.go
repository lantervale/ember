package style

import (
	"math"
	"strings"

	"github.com/rivo/uniseg"
)

// CounterSystem is an @counter-style's system: how it writes a value
// with its symbols.
type CounterSystem uint8

const (
	// CounterSymbolic repeats the symbol of the value's place in the
	// list once more each time round it: *, †, **, ††.
	CounterSymbolic CounterSystem = iota
	// CounterCyclic goes round the symbols forever.
	CounterCyclic
	// CounterFixed writes First and the values after it, a symbol each.
	CounterFixed
	// CounterAlphabetic counts as a spreadsheet's columns: a … z, aa.
	CounterAlphabetic
	// CounterNumeric writes the value in the base of its symbols, the
	// first being zero.
	CounterNumeric
	// CounterAdditive adds up weighted symbols, the heaviest first, as
	// roman numerals do.
	CounterAdditive
)

// Infinite is a range's bound written infinite.
const Infinite = math.MaxInt

// AdditiveSymbol is one weight and symbol of additive-symbols.
type AdditiveSymbol struct {
	Weight int
	Symbol string
}

// CounterStyle is a counter style resolved: an @counter-style rule, one
// of the predefined ones or symbols(), with what it extends filled in.
// Fallback is the name of the style a value it can't write is written
// in.
type CounterStyle struct {
	Name     string
	System   CounterSystem
	First    int
	Symbols  []string
	Additive []AdditiveSymbol
	// Negative is the sign around a value below zero; Pad and PadSymbol
	// pad a shorter representation to Pad graphemes.
	Negative       [2]string
	Prefix, Suffix string
	// Ranges are the values it writes, nil for its system's own (auto).
	Ranges    [][2]int
	Pad       int
	PadSymbol string
	Fallback  string
}

// InRange reports whether the style writes n itself, rather than its
// fallback.
func (cs *CounterStyle) InRange(n int) bool {
	if cs.Ranges == nil {
		switch cs.System {
		case CounterAlphabetic, CounterSymbolic:
			return n >= 1
		case CounterAdditive:
			return n >= 0
		case CounterCyclic, CounterFixed, CounterNumeric:
		}
		return true
	}
	for _, r := range cs.Ranges {
		if n >= r[0] && n <= r[1] {
			return true
		}
	}
	return false
}

// signed reports whether the system writes a value below zero with the
// negative sign.
func (cs *CounterStyle) signed() bool {
	return cs.System != CounterCyclic && cs.System != CounterFixed
}

// Represent writes n as the style counts, without its prefix and suffix,
// and reports false when it can't: n is out of its range or its system
// has no way to write it, and its fallback writes it.
func (cs *CounterStyle) Represent(n int) (string, bool) {
	if !cs.InRange(n) {
		return "", false
	}
	negative := n < 0 && cs.signed()
	v := n
	if negative {
		v = -n
	}
	s, ok := cs.algorithm(v)
	if !ok {
		return "", false
	}
	if cs.Pad > 0 {
		short := cs.Pad - uniseg.GraphemeClusterCount(s)
		if negative {
			short -= uniseg.GraphemeClusterCount(cs.Negative[0] + cs.Negative[1])
		}
		if short > 0 {
			s = strings.Repeat(cs.PadSymbol, short) + s
		}
	}
	if negative {
		s = cs.Negative[0] + s + cs.Negative[1]
	}
	return s, true
}

func (cs *CounterStyle) algorithm(n int) (string, bool) {
	syms := cs.Symbols
	switch cs.System {
	case CounterCyclic:
		if len(syms) == 0 {
			return "", false
		}
		k := len(syms)
		return syms[((n-1)%k+k)%k], true
	case CounterFixed:
		if n < cs.First || n-cs.First >= len(syms) {
			return "", false
		}
		return syms[n-cs.First], true
	case CounterSymbolic:
		if n < 1 || len(syms) == 0 {
			return "", false
		}
		k := len(syms)
		if (n+k-1)/k > repeatLimit {
			return "", false
		}
		return strings.Repeat(syms[(n-1)%k], (n+k-1)/k), true
	case CounterAlphabetic:
		if n < 1 || len(syms) < 2 {
			return "", false
		}
		var out []string
		for k := len(syms); n > 0; n = (n - 1) / k {
			out = append(out, syms[(n-1)%k])
		}
		return reversed(out), true
	case CounterNumeric:
		if len(syms) < 2 {
			return "", false
		}
		if n == 0 {
			return syms[0], true
		}
		var out []string
		for k := len(syms); n > 0; n /= k {
			out = append(out, syms[n%k])
		}
		return reversed(out), true
	case CounterAdditive:
		return cs.additive(n)
	}
	return "", false
}

func (cs *CounterStyle) additive(n int) (string, bool) {
	if n == 0 {
		for _, a := range cs.Additive {
			if a.Weight == 0 {
				return a.Symbol, true
			}
		}
		return "", false
	}
	var b strings.Builder
	for _, a := range cs.Additive {
		if a.Weight == 0 {
			break
		}
		if n/a.Weight > repeatLimit {
			return "", false
		}
		for ; n >= a.Weight; n -= a.Weight {
			b.WriteString(a.Symbol)
		}
		if n == 0 {
			return b.String(), true
		}
	}
	return "", false
}

// repeatLimit is how many times a symbolic or an additive style repeats
// a symbol before it leaves the value to its fallback, so a large value
// writes no endless text.
const repeatLimit = 1000

func reversed(parts []string) string {
	var b strings.Builder
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString(parts[i])
	}
	return b.String()
}
