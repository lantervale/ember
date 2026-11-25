package css

import (
	"fmt"
	"math"

	"github.com/lantervale/ember/ui/internal/style"
)

const (
	propFlexWrap      = "flex-wrap"
	propFlexLineCount = "flex-line-count"
	kwNowrap          = "nowrap"
	kwWrapReverse     = "wrap-reverse"
	kwBalance         = "balance"
)

// lineCount is a whole number of 1 or more, or a calculation clamped
// to one.
//
//nolint:gochecknoglobals // a grammar term
var lineCount = term(func(c Component) error {
	if c.Kind == Function {
		return wholeNumber(c)
	}
	return positiveInteger(c)
})

// checkFlexWrap takes nowrap alone, or wrap or wrap-reverse and balance
// in either order, each at most once.
func checkFlexWrap(v []Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("expected nowrap, wrap, wrap-reverse or balance")
	}
	wrap, balance := 0, 0
	for _, c := range v {
		switch {
		case isWord(c, kwNowrap) && len(v) == 1:
			wrap++
		case isWord(c, kwWrap, kwWrapReverse):
			wrap++
		case isWord(c, kwBalance):
			balance++
		default:
			return fmt.Errorf("expected wrap, wrap-reverse or balance, found %q", c.String())
		}
	}
	if wrap > 1 || balance > 1 {
		return fmt.Errorf("expected one wrap and one balance at most")
	}
	return nil
}

// flexWrapOf is the flex-wrap its words say, in either order.
func flexWrapOf(_ *ctx, v []Component) style.FlexWrap {
	w, balance := style.Wrap, false
	for _, c := range v {
		switch {
		case isWord(c, kwNowrap):
			return style.NoWrap
		case isWord(c, kwWrapReverse):
			w = style.WrapReverse
		case isWord(c, kwBalance):
			balance = true
		}
	}
	if balance {
		w += style.BalanceWrap - style.Wrap
	}
	return w
}

// lineCountOf is a flex-line-count: a calculation rounded, and one at
// least, NaN's included.
func lineCountOf(x *ctx, v []Component) int {
	f := float64(x.num(v[0]))
	if v[0].Kind == Number {
		f = v[0].Num
	}
	if math.IsNaN(f) || f < 1 {
		return 1
	}
	return int(math.Floor(min(f, math.MaxInt32) + 0.5))
}

// expandFlexFlow takes a flex-direction and a flex-wrap of one or two
// words, the direction before or after the wrap's words, never between.
func expandFlexFlow(v []Component, out map[string][]Component) error {
	if len(v) == 0 || len(v) > 3 {
		return fmt.Errorf("expected 1 to 3 values, found %d", len(v))
	}
	dir := -1
	for i, c := range v {
		if checks["flex-direction"]([]Component{c}) == nil {
			if dir >= 0 || (i > 0 && i < len(v)-1) {
				return fmt.Errorf("%q stands between flex-wrap's words", c.String())
			}
			dir = i
		}
	}
	wrap := v
	if dir >= 0 {
		out["flex-direction"] = []Component{v[dir]}
		wrap = append(append([]Component(nil), v[:dir]...), v[dir+1:]...)
	}
	if len(wrap) == 0 {
		return nil
	}
	if err := checks[propFlexWrap](wrap); err != nil {
		return err
	}
	out[propFlexWrap] = wrap
	return nil
}

// flexWrapText is each flex-wrap as getComputedStyle writes it.
//
//nolint:gochecknoglobals // lookup table
var flexWrapText = map[style.FlexWrap]string{
	style.NoWrap: kwNowrap, style.Wrap: kwWrap, style.WrapReverse: kwWrapReverse,
	style.BalanceWrap: kwBalance, style.BalanceWrapReverse: kwWrapReverse + " " + kwBalance,
}
