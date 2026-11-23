package css

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestSizesInterpolateThroughCalcSize(t *testing.T) {
	auto := style.Length{}
	got, ok := lerpSizes(style.Px(0), auto, 0.25, true)
	assert.True(t, ok)
	assert.Equal(t, style.Length{Kind: style.Auto, Sized: true, Scale: 0.25}, got, "0 to auto: a quarter of auto's size")

	_, ok = lerpSizes(style.Px(0), auto, 0.25, false)
	assert.False(t, ok, "without interpolate-size: allow-keywords, a keyword flips")

	half := style.Length{Kind: style.Auto, Sized: true, Scale: 0.5}
	got, ok = lerpSizes(half, style.Px(100), 0.5, false)
	assert.True(t, ok, "calc-size() interpolates whatever interpolate-size says")
	assert.Equal(t, style.Length{Kind: style.Auto, Sized: true, Scale: 0.25, Px: 50}, got)

	_, ok = lerpSizes(auto, style.Length{Kind: style.MinContent}, 0.5, true)
	assert.False(t, ok, "two keywords of their own sizes don't interpolate")
}

func TestCalcSizeComputes(t *testing.T) {
	sheet, warns := Parse("t.css", `p { height: calc-size(calc-size(auto, size + 10px), size * 0.5 + 4px) }`)
	assert.Empty(t, warns)
	v := sheet.Rules[0].Decls[0].Value
	x := &ctx{c: &style.Computed{}}
	assert.Equal(t, style.Length{Kind: style.Auto, Sized: true, Scale: 0.5, Px: 9}, x.length(items(v)[0]))
}
