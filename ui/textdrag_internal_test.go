package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSmartDeleteTakesASpaceAsChromeOnMacOS holds what Chrome 158 on
// macOS left of a textarea's text once a drag moved its selection out.
func TestSmartDeleteTakesASpaceAsChromeOnMacOS(t *testing.T) {
	for _, c := range []struct {
		text     string
		from, to int
		left     string
	}{
		{"one two three", 4, 7, "one three"},
		{"one two three", 8, 13, "one two"},
		{"one two, three", 4, 7, "one, three"},
		{"one  two three", 5, 8, "one  three"},
		{"one two three", 0, 3, "two three"},
		{"one two three", 3, 7, "one three"},
		{"aa bb", 3, 5, "aa"},
	} {
		r := []rune(c.text)
		from, to := smartDelete(r, c.from, c.to)
		assert.Equal(t, c.left, string(r[:from])+string(r[to:]), c.text)
	}
}
