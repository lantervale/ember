package style_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func written(cs *style.CounterStyle, values ...int) []string {
	out := make([]string, len(values))
	for i, n := range values {
		s, ok := cs.Represent(n)
		if !ok {
			s = "fallback"
		}
		out[i] = s
	}
	return out
}

func TestEachCounterSystemWritesAValueAsCSSCounterStylesHasIt(t *testing.T) {
	abc := []string{"a", "b", "c"}
	cyclic := &style.CounterStyle{System: style.CounterCyclic, Symbols: abc}
	assert.Equal(t, []string{"c", "a", "b", "c", "a", "b"}, written(cyclic, 0, 1, 2, 3, 4, -1))

	fixed := &style.CounterStyle{System: style.CounterFixed, First: 5, Symbols: abc}
	assert.Equal(t, []string{"fallback", "a", "c", "fallback"}, written(fixed, 4, 5, 7, 8))

	symbolic := &style.CounterStyle{System: style.CounterSymbolic, Symbols: []string{"*", "†"}}
	assert.Equal(t, []string{"fallback", "*", "†", "**", "††", "***"}, written(symbolic, 0, 1, 2, 3, 4, 5))

	alpha := &style.CounterStyle{System: style.CounterAlphabetic, Symbols: abc}
	assert.Equal(t, []string{"fallback", "a", "c", "aa", "ac", "ba", "aaa"}, written(alpha, 0, 1, 3, 4, 6, 7, 13))

	binary := &style.CounterStyle{System: style.CounterNumeric, Symbols: []string{"0", "1"}, Negative: [2]string{"-"}}
	assert.Equal(t, []string{"0", "1", "10", "101", "-11"}, written(binary, 0, 1, 2, 5, -3))

	roman := &style.CounterStyle{System: style.CounterAdditive, Additive: []style.AdditiveSymbol{
		{Weight: 10, Symbol: "X"}, {Weight: 9, Symbol: "IX"}, {Weight: 5, Symbol: "V"}, {Weight: 4, Symbol: "IV"},
		{Weight: 1, Symbol: "I"},
	}}
	assert.Equal(t, []string{"fallback", "XIV", "XXIX", "fallback"}, written(roman, 0, 14, 29, -4),
		"an additive style counts from 0 unless a range says otherwise")
	tens := &style.CounterStyle{System: style.CounterAdditive, Additive: []style.AdditiveSymbol{
		{Weight: 10, Symbol: "X"}, {Weight: 0, Symbol: "zero"},
	}}
	assert.Equal(t, []string{"zero", "XX", "fallback"}, written(tens, 0, 20, 21), "what no symbol adds up to falls back")
}

func TestARangeNegativeAndPadShapeTheRepresentation(t *testing.T) {
	digits := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	cs := &style.CounterStyle{System: style.CounterNumeric, Symbols: digits, Negative: [2]string{"(", ")"},
		Pad: 4, PadSymbol: "0"}
	assert.Equal(t, []string{"0007", "(07)", "1234", "(1234)"}, written(cs, 7, -7, 1234, -1234),
		"the negative sign takes its length off the pad")

	ranged := &style.CounterStyle{System: style.CounterNumeric, Symbols: digits,
		Ranges: [][2]int{{1, 3}, {10, style.Infinite}}}
	assert.Equal(t, []string{"fallback", "2", "fallback", "12"}, written(ranged, 0, 2, 5, 12))

	alpha := &style.CounterStyle{System: style.CounterAlphabetic, Symbols: []string{"a", "b"},
		Ranges: [][2]int{{-style.Infinite, style.Infinite}}, Negative: [2]string{"-"}}
	assert.Equal(t, []string{"-b"}, written(alpha, -2), "an explicit range lets an alphabetic style write below 1")
}
