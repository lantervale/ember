package css

import (
	"fmt"
	"math"

	"github.com/lantervale/ember/ui/internal/style"
)

const propInitialLetter = "initial-letter"

// checkInitialLetter takes normal, or a size of one line or more, then
// the line it sinks to, or drop or raise.
func checkInitialLetter(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNormal) {
		return nil
	}
	if len(v) == 0 || len(v) > 2 || v[0].Kind != Number || v[0].Num < 1 {
		return fmt.Errorf("expected normal, or a size of 1 or more and a sink")
	}
	if len(v) == 2 && !isWord(v[1], "drop", "raise") &&
		(v[1].Kind != Number || v[1].Num < 1 || v[1].Num != math.Trunc(v[1].Num)) {
		return fmt.Errorf("a sink is a whole number of lines from 1, drop or raise")
	}
	return nil
}

// initialLetterOf is initial-letter: no sink, or drop, is the size
// rounded down, as Chrome sinks it, and raise is 1.
func initialLetterOf(_ *ctx, v []Component) style.InitialLetter {
	if isWord(v[0], kwNormal) {
		return style.InitialLetter{}
	}
	size := float32(v[0].Num)
	sink := max(int(math.Floor(float64(size))), 1)
	switch {
	case len(v) < 2 || isWord(v[1], "drop"):
	case isWord(v[1], "raise"):
		sink = 1
	default:
		sink = int(v[1].Num)
	}
	return style.InitialLetter{Size: size, Sink: sink}
}

// dropCap lays st, a ::first-letter's style of an initial-letter, out as
// Chrome does in its block of style block: a float at the line's start
// whose capitals reach from the cap height of the first line to the
// baseline of the size-th, its own baseline on that of the sink-th line,
// its content box trimmed to its capitals' height.
func (r *restyler) dropCap(st, block *style.Computed, letter string) {
	para := r.env.fontUnits(block.Font)
	lh := para.Line
	switch l := block.LineHeight; {
	case l.Normal:
	case l.Factor > 0:
		lh = l.Factor * block.Font.Size
	default:
		lh = l.Px
	}
	own := r.env.fontUnits(st.Font)
	if own.Cap <= 0 || para.Cap <= 0 {
		return
	}
	il := st.InitialLetter
	capital := (il.Size-1)*lh + para.Cap
	st.Font.Size *= capital / own.Cap
	own = r.env.fontUnits(st.Font)
	st.Float = style.FloatLeft
	if block.Dir == style.RTL {
		st.Float = style.FloatRight
	}
	st.BoxSizing = style.ContentBox
	st.Size.Height = style.Px(own.Cap)
	st.LineHeight = style.LineHeight{Px: max(2*own.Cap-own.Ascent+own.Descent, 0)}
	if r.env.ink != nil {
		if left, width, ok := r.env.ink(st.Font, letter); ok {
			st.Size.Width = style.Px(width)
			st.TextIndent = style.Px(-left)
		}
	}
	firstBaseline := (lh-para.Ascent-para.Descent)/2 + para.Ascent
	st.InitialLetter.Raise = max(il.Size-float32(il.Sink), 0) * lh
	top := firstBaseline + float32(il.Sink-1)*lh - own.Cap
	st.Margin.Top = style.Px(top - st.Padding.Top.Px - st.Border.Top)
}
