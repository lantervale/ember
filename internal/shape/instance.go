package shape

import (
	"math"
	"strconv"
	"strings"

	"github.com/lantervale/ember/internal/varfont"
)

// coordsKey is c as a key of a map, "" for the default instance.
func coordsKey(c varfont.Coords) string {
	var b strings.Builder
	for _, v := range c {
		b.WriteString(strconv.FormatInt(int64(math.Round(float64(v)*16384)), 36))
		b.WriteByte(',')
	}
	return b.String()
}

// substitution is the FeatureTableSubstitution of the first of l's
// feature variations whose conditions hold at c (nil the default
// instance), as HarfBuzz finds it; nil for none.
func (l *layout) substitution(c varfont.Coords) data {
	v := l.featVars
	if v.u16(0) != 1 {
		return nil
	}
	for i := range int(v.u32(4)) {
		rec := 8 + 8*i
		if holds(v.at(int(v.u32(rec))), c) {
			return v.at(int(v.u32(rec + 4)))
		}
	}
	return nil
}

// holds reports whether every condition of set holds at c: an axis's
// coordinate within a range, the only kind there is, as a set of none
// holds.
func holds(set data, c varfont.Coords) bool {
	for k := range int(set.u16(0)) {
		cond := set.at(int(set.u32(2 + 4*k)))
		if cond.u16(0) != 1 {
			return false
		}
		var v float32
		if axis := int(cond.u16(2)); axis < len(c) {
			v = c[axis]
		}
		if v < f2dot14(cond, 4) || v > f2dot14(cond, 6) {
			return false
		}
	}
	return true
}

func f2dot14(d data, at int) float32 {
	return float32(int16(d.u16(at))) / 16384 //nolint:gosec // a signed 2.14 number
}

// feature is feature i of l's list, or the one subst puts in its place.
func (l *layout) feature(i int, subst data) data {
	for k := range int(subst.u16(4)) {
		rec := 6 + 6*k
		if int(subst.u16(rec)) == i {
			return subst.at(int(subst.u32(rec + 2)))
		}
	}
	return l.feats.sub(2 + 6*i + 4)
}

// delta is what device table d adds at the plan's instance, in font
// units: a variation index's delta in GDEF's store, rounded as HarfBuzz
// rounds it; hinting devices, for sizes in pixels, add nothing.
func (b *buffer) delta(d data) int32 {
	if d == nil || b.plan.coords == nil || b.f.varStore == nil || d.u16(4) != 0x8000 {
		return 0
	}
	return int32(math.Round(float64(b.f.varStore.Delta(int(d.u16(0)), int(d.u16(2)), b.plan.coords))))
}

// anchor is the point, x then y, of anchor table a at the plan's
// instance, false for none.
func (b *buffer) anchor(a data) (int32, int32, bool) {
	f := a.u16(0)
	if f < 1 || f > 3 {
		return 0, 0, false
	}
	x, y := int32(int16(a.u16(2))), int32(int16(a.u16(4))) //nolint:gosec // signed 16-bit fields
	if f == 3 {
		x += b.delta(a.sub(6))
		y += b.delta(a.sub(8))
	}
	return x, y, true
}

// advance is glyph id's advance in font units at the plan's instance.
func (b *buffer) advance(id uint16) (int32, bool) {
	if v := b.plan.vf; v != nil && b.plan.coords != nil {
		return int32(math.Round(float64(v.Advance(id, b.plan.coords)))), true
	}
	return b.f.defaultAdvance(id)
}
