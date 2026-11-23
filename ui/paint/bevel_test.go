package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestShadesAreChromesForAnInsetOrOutsetSide(t *testing.T) {
	near := func(want [3]float32, got color.Color, msg string) {
		t.Helper()
		s := got.SRGB()
		for k := range 3 {
			assert.InDelta(t, want[k], s[k]*255, 1, msg)
		}
	}
	for _, c := range []struct {
		name          string
		c             color.Color
		darker, light [3]float32
	}{
		{"gray", color.RGB8(128, 128, 128), [3]float32{44, 44, 44}, [3]float32{212, 212, 212}},
		{"red", color.RGB8(224, 60, 49), [3]float32{140, 37, 30}, [3]float32{255, 68, 55}},
		{"black", color.RGB8(0, 0, 0), [3]float32{84, 84, 84}, [3]float32{168, 168, 168}},
		{"an iframe's grey", color.RGB8(238, 238, 238), [3]float32{154, 154, 154}, [3]float32{238, 238, 238}},
		{"just darker than #ebebeb", color.RGB8(234, 234, 234), [3]float32{150, 150, 150}, [3]float32{255, 255, 255}},
	} {
		d, l := shades(c.c)
		near(c.darker, d, c.name+": darker, as Chrome draws it")
		near(c.light, l, c.name+": lighter")
	}
}

func TestInsetDarkensTopAndLeftAndOutsetTheOthers(t *testing.T) {
	gray := color.RGB8(128, 128, 128)
	darker, lighter := shades(gray)
	for _, c := range []struct {
		s         style.BorderStyle
		darkSides []int
		name      string
	}{{style.BorderInset, []int{0, 3}, "inset"}, {style.BorderOutset, []int{1, 2}, "outset"}} {
		b := el(10, 10, 40, 30, func(s *style.Computed) {
			s.BorderStyle = style.All(c.s)
			s.BorderColor = style.All(gray)
			s.Border = style.All[float32](4)
		})
		b.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
		r := newRecorder()
		Paint(r, tree(b))
		sides := r.sides()
		if !assert.Len(t, sides, 4, "%s: four mitered sides", c.name) {
			continue
		}
		for i, o := range sides {
			want := lighter
			for _, d := range c.darkSides {
				if d == i {
					want = darker
				}
			}
			assert.Equal(t, want, o.Color, "%s side %d (top, right, bottom, left)", c.name, i)
		}
	}
}

func TestGrooveIsAnInsetHalfRoundAnOutsetHalfAndRidgeTheOtherWay(t *testing.T) {
	gray := color.RGB8(128, 128, 128)
	darker, lighter := shades(gray)
	for _, c := range []struct {
		s     style.BorderStyle
		outer color.Color
		name  string
	}{{style.BorderGroove, darker, "groove"}, {style.BorderRidge, lighter, "ridge"}} {
		b := el(10, 10, 40, 30, func(s *style.Computed) {
			s.BorderStyle = style.All(c.s)
			s.BorderColor = style.All(gray)
			s.Border = style.All[float32](4)
		})
		b.Border = box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}
		r := newRecorder()
		Paint(r, tree(b))
		sides := r.sides()
		if !assert.Len(t, sides, 8, "%s: an outer and an inner half per side", c.name) {
			continue
		}
		inner := lighter
		if c.outer == lighter {
			inner = darker
		}
		assert.Equal(t, c.outer, sides[0].Color, "%s: the top's outer half", c.name)
		assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 40, H: 2}, sides[0].Rect)
		assert.Equal(t, inner, sides[4].Color, "%s: the top's inner half", c.name)
		assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 36, H: 2}, sides[4].Rect)
		assert.Equal(t, inner, sides[2].Color, "%s: the bottom's outer half", c.name)
	}
}
