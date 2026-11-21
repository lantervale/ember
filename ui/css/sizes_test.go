package css_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/css"
)

func TestASizesAttributeGivesTheSlotOfItsFirstEntryThatHolds(t *testing.T) {
	m := css.Media{Width: 800, Height: 600, Resolution: 2}
	for _, c := range []struct {
		sizes string
		want  float32
	}{
		{"", 800},
		{"100px", 100},
		{"(min-width: 2000px) 10px, calc(10vw - 1px)", 79},
		{"(max-width: 900px) 50vw, 10px", 400},
		{"(min-width: 900px) 50vw", 800},
		{"50%, 5em", 80},
		{"-10px, 2rem", 32},
		{"0", 0},
		{"10, 20px", 20},
		{"attr(x), min(30vw, 300px)", 240},
		{"(width >= 500px) and (orientation: landscape) 25vw, 1px", 200},
		{"(min-resolution: 2dppx) 10px, 1px", 10},
		{"(unknown: 1) 10px, 1px", 1},
		{"screen 10px, 1px", 1},
		{"(min-width: 1px) 10px 20px, 30px", 30},
		{"20px, 30px", 20},
		{"(max-width: 10px) 4px, , 7px", 7},
		{"auto, 40px", 40},
		{"2ex, 1ch", 16},
		{"10vmin", 60},
	} {
		assert.InDelta(t, c.want, css.ParseSizes(c.sizes).Width(m, -1), 1e-4, c.sizes)
	}
}

func TestAutoInSizesIsTheWidthGivenWhereItMayBeUsed(t *testing.T) {
	m := css.Media{Width: 800, Height: 600}
	auto := css.ParseSizes("auto, 40px")
	assert.InDelta(t, 150, auto.Width(m, 150), 1e-4)
	assert.InDelta(t, 40, auto.Width(m, -1), 1e-4, "no auto: the next entry")
	assert.InDelta(t, 800, css.ParseSizes("AUTO").Width(m, -1), 1e-4)
	assert.InDelta(t, 10, css.ParseSizes("(min-width: 1px) 10px, auto").Width(m, 150), 1e-4)
	assert.InDelta(t, 150, css.ParseSizes("(max-width: 1px) 10px, auto").Width(m, 150), 1e-4)
}

func TestTheMediaOfASourceIsAMediaQueryList(t *testing.T) {
	m := css.Media{Width: 800, Height: 600}
	for _, c := range []struct {
		media string
		want  bool
	}{
		{"", true},
		{"all", true},
		{"screen and (min-width: 100px)", true},
		{"print", false},
		{"(max-width: 10px)", false},
		{"print, (orientation: landscape)", true},
		{"(min-width: 30em)", true},
		{"not screen", false},
		{"(min-width:", false},
	} {
		assert.Equal(t, c.want, css.ParseMediaQueries(c.media).Hold(m), c.media)
	}
}
