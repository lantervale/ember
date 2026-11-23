package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTheCaretOfRightToLeftTextStandsWhereItIsShown(t *testing.T) {
	for _, c := range []struct {
		name string
		text string
		rtl  bool
		want []float32
	}{
		{"latin", "abc", false, []float32{0, 10, 20, 30}},
		{"hebrew", "אבג", true, []float32{30, 20, 10, 0}},
		{"hebrew in a left-to-right field", "אבג", false, []float32{30, 20, 10, 0}},
		{"latin in a right-to-left field", "abc", true, []float32{0, 10, 20, 30}},
		{"latin after hebrew", "אב c", true, []float32{40, 30, 20, 0, 10}},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			// each character ten points wide
			xs := make([]float32, len(text)+1)
			for i := range xs {
				xs[i] = float32(10 * i)
			}
			assert.Equal(t, c.want, visualEdges(text, xs, c.rtl))
		})
	}
}
