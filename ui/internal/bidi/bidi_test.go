package bidi_test

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/ui/internal/bidi"
)

const (
	rle = string(rune(0x202b))
	lro = string(rune(0x202d))
	rlo = string(rune(0x202e))
	pdf = string(rune(0x202c))
	lri = string(rune(0x2066))
	rli = string(rune(0x2067))
	fsi = string(rune(0x2068))
	pdi = string(rune(0x2069))
)

func TestLevels(t *testing.T) {
	for _, c := range []struct {
		name string
		text string
		base bidi.Level
		want []bidi.Level
		// paragraph is the level Auto finds, when base is Auto.
		paragraph bidi.Level
	}{
		{"latin", "abc", 0, []bidi.Level{0, 0, 0}, 0},
		{"hebrew alone", "אבג", bidi.Auto, []bidi.Level{1, 1, 1}, 1},
		{"auto past an isolate", rli + "א" + pdi + "b", bidi.Auto, []bidi.Level{0, 1, 0, 0}, 0},
		{"no strong letter", "123", bidi.Auto, []bidi.Level{0, 0, 0}, 0},
		{"hebrew in latin", "ab אב cd", 0, []bidi.Level{0, 0, 0, 1, 1, 0, 0, 0}, 0},
		{"numbers after hebrew, right to left", "אב 12", 1, []bidi.Level{1, 1, 1, 2, 2}, 1},
		{"numbers after hebrew, left to right", "אב 12", 0, []bidi.Level{1, 1, 1, 2, 2}, 0},
		{"numbers after latin", "ab 12", 0, []bidi.Level{0, 0, 0, 0, 0}, 0},
		{"latin in right to left", "ab cd", 1, []bidi.Level{2, 2, 2, 2, 2}, 1},
		{"a number's separator", "א 1,5", 1, []bidi.Level{1, 1, 2, 2, 2}, 1},
		{"a percent sign after a number", "א 5%", 1, []bidi.Level{1, 1, 2, 2}, 1},
		{"arabic digits after arabic letters", "ا 12", 0, []bidi.Level{1, 1, 2, 2}, 0},
		{"a bracket pair round latin", "ab (cd)", 1, []bidi.Level{2, 2, 2, 2, 2, 2, 2}, 1},
		{"a bracket pair round hebrew in latin", "a (א) b", 0, []bidi.Level{0, 0, 0, 1, 0, 0, 0}, 0},
		{"a bracket pair, context before", "א (b) c", 1, []bidi.Level{1, 1, 1, 2, 1, 1, 2}, 1},
		{"an override", rlo + "abc" + pdf, 0, []bidi.Level{0, 1, 1, 1, 1}, 0},
		{"a left to right override in right to left", lro + "אב" + pdf, 1, []bidi.Level{1, 2, 2, 2}, 1},
		{"an embedding", rle + "ab" + pdf + "c", 0, []bidi.Level{0, 2, 2, 2, 0}, 0},
		{"an isolate", "a" + rli + "b" + pdi + "c", 0, []bidi.Level{0, 0, 2, 0, 0}, 0},
		{"an isolate of hebrew between hebrew", "א" + lri + "ב" + pdi + "ג", 0, []bidi.Level{1, 1, 3, 1, 1}, 0},
		{"a first strong isolate", "a" + fsi + "א b" + pdi, 0, []bidi.Level{0, 0, 1, 1, 2, 0}, 0},
		{"a paragraph separator and the spaces before it", "ab \n", 1, []bidi.Level{2, 2, 1, 1}, 1},
		{"a tab", "א\tב", 0, []bidi.Level{1, 0, 1}, 0},
		{"an unmatched pdi", "a" + pdi + "b", 0, []bidi.Level{0, 0, 0}, 0},
		{"a mark after a letter", "אְb", 0, []bidi.Level{1, 1, 0}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := bidi.Levels([]rune(c.text), c.base)
			var got []bidi.Level
			for i := range []rune(c.text) {
				got = append(got, p.Level(i))
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("%q: %v, want %v", c.text, got, c.want)
			}
			if p.Base() != c.paragraph {
				t.Errorf("%q: paragraph level %d, want %d", c.text, p.Base(), c.paragraph)
			}
		})
	}
}

func TestReorder(t *testing.T) {
	for _, c := range []struct {
		levels []bidi.Level
		want   []int
	}{
		{[]bidi.Level{0, 0, 0}, []int{0, 1, 2}},
		{[]bidi.Level{1, 1, 1}, []int{2, 1, 0}},
		{[]bidi.Level{0, 0, 1, 1, 1, 0}, []int{0, 1, 4, 3, 2, 5}},
		{[]bidi.Level{1, 1, 2, 2, 1}, []int{4, 2, 3, 1, 0}},
		{[]bidi.Level{0, 1, 2, 2, 1, 0}, []int{0, 4, 2, 3, 1, 5}},
		{[]bidi.Level{}, []int{}},
	} {
		if got := bidi.Reorder(c.levels); !slices.Equal(got, c.want) {
			t.Errorf("%v: %v, want %v", c.levels, got, c.want)
		}
	}
}
