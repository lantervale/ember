package css

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// calc-size() (decision 0177): a size keyword's size, the size a box would take
// with it, worked on by a calculation in which size stands for it. ui reads the
// calculations whose size is a sum of a multiple of size and a length-percentage,
// and lays it out in height.

func isCalcSize(c Component) bool { return c.Kind == Function && strings.EqualFold(c.Text, "calc-size") }

// calcSizeArgs is calc-size()'s basis and its calculation, without whitespace.
func calcSizeArgs(c Component) (Component, []Component, bool) {
	args := items(c.Args)
	for i, a := range args {
		if a.Kind == Comma {
			if i != 1 || i+1 >= len(args) {
				return Component{}, nil, false
			}
			return args[0], c.Args[indexAfterComma(c.Args):], true
		}
	}
	return Component{}, nil, false
}

// indexAfterComma is where the arguments after their first comma start.
func indexAfterComma(args []Component) int {
	for i, a := range args {
		if a.Kind == Comma {
			return i + 1
		}
	}
	return len(args)
}

// withSize is a calculation as a calc() with size the length px.
func withSize(calc []Component, px float64) Component {
	return Component{Token: Token{Kind: Function, Text: "calc"}, Args: substituteSize(calc, px)}
}

func substituteSize(cs []Component, px float64) []Component {
	out := make([]Component, len(cs))
	for i, c := range cs {
		switch {
		case isWord(c, "size"):
			out[i] = Component{Token: Token{Kind: Dimension, Num: px, Unit: "px",
				Text: strconv.FormatFloat(px, 'g', -1, 64)}}
		case len(c.Args) > 0:
			c.Args = substituteSize(c.Args, px)
			out[i] = c
		default:
			out[i] = c
		}
	}
	return out
}

func mentionsSize(cs []Component) bool {
	for _, c := range cs {
		if isWord(c, "size") || mentionsSize(c.Args) {
			return true
		}
	}
	return false
}

// calcSizeTerm checks calc-size(): a basis (a sizing keyword, any, a
// length-percentage or another calc-size()) and a calculation of lengths and
// percentages in which size stands for the basis's size, as Chrome 157 reads it.
func calcSizeTerm(c Component) error {
	if !isCalcSize(c) {
		return fmt.Errorf("expected calc-size(), found %q", c.String())
	}
	basis, calc, ok := calcSizeArgs(c)
	if !ok {
		return fmt.Errorf("calc-size() takes a basis and a calculation")
	}
	switch {
	case isWord(basis, "auto", kwMinContent, "max-content", kwFitContent, kwStretch):
	case isWord(basis, "any"):
		if mentionsSize(calc) {
			return fmt.Errorf("calc-size(any, …) has no size to use")
		}
	case isCalcSize(basis):
		if err := calcSizeTerm(basis); err != nil {
			return err
		}
	default:
		if err := sizeLP(basis); err != nil {
			return err
		}
	}
	return lengthPercent(withSize(calc, 1))
}

// calcSize is calc-size()'s value: its calculation's multiple of size and the
// rest, from the calculation worked with size 0 and 1.
func (x *ctx) calcSize(c Component) style.Length {
	basis, calc, ok := calcSizeArgs(c)
	if !ok {
		return style.Length{}
	}
	zero, one := x.length(withSize(calc, 0)), x.length(withSize(calc, 1))
	scale := one.Px - zero.Px
	rest := style.Length{Kind: style.Fixed, Px: zero.Px, Percent: zero.Percent, IsPercent: zero.IsPercent}
	var b style.Length
	switch {
	case isWord(basis, "any"):
		return rest
	case basis.Kind == Ident:
		b = x.length(basis)
		b.Sized, b.Scale = true, 1
	default:
		b = x.length(basis)
	}
	if !b.Sized {
		// a length-percentage basis is that size
		return style.Length{Kind: style.Fixed, Px: scale*b.Px + rest.Px, Percent: scale*b.Percent + rest.Percent,
			IsPercent: b.IsPercent || rest.IsPercent}
	}
	return style.Length{Kind: b.Kind, Sized: true, Scale: scale * b.Scale, Px: scale*b.Px + rest.Px,
		Percent: scale*b.Percent + rest.Percent, IsPercent: b.IsPercent || rest.IsPercent}
}

// sizingKeyword reports whether l is a sizing keyword calc-size() takes as its
// basis, or calc-size() of one.
func sizingKeyword(l style.Length) bool {
	switch l.Kind {
	case style.Auto, style.MinContent, style.MaxContent, style.Stretch:
		return true
	case style.FitContent:
		return l.Sized || l.Px == 0 && l.Percent == 0
	case style.Fixed, style.Fr, style.None:
	}
	return false
}

// asSized is l as calc-size() of basis: a keyword its own size when keywords
// interpolate, a length-percentage no multiple of size; false where it isn't
// one of either.
func asSized(l style.Length, basis style.LengthKind, keywords bool) (style.Length, bool) {
	switch {
	case l.Sized:
		return l, l.Kind == basis
	case sizingKeyword(l):
		return style.Length{Kind: l.Kind, Sized: true, Scale: 1}, keywords && l.Kind == basis
	case l.Kind == style.Fixed && l.Calc == style.Length{}.Calc:
		return style.Length{Kind: basis, Sized: true, Px: l.Px, Percent: l.Percent, IsPercent: l.IsPercent}, true
	}
	return l, false
}

// lerpSizes interpolates two sizes, through calc-size() where either is one or,
// keywords allowing, a sizing keyword: both of one basis, or one a
// length-percentage.
func lerpSizes(a, b style.Length, t float32, keywords bool) (style.Length, bool) {
	if !a.Sized && !b.Sized {
		if l, ok := lerpLength(a, b, t, true); ok || !keywords {
			return l, ok
		}
	}
	basis := a.Kind
	if !a.Sized && !sizingKeyword(a) {
		basis = b.Kind
	}
	sa, oka := asSized(a, basis, keywords)
	sb, okb := asSized(b, basis, keywords)
	if !oka || !okb || !sa.Sized && !sb.Sized {
		return a, false
	}
	return style.Length{Kind: basis, Sized: true, Scale: max(lerpf(sa.Scale, sb.Scale, t), 0),
		Px: lerpf(sa.Px, sb.Px, t), Percent: lerpf(sa.Percent, sb.Percent, t),
		IsPercent: sa.IsPercent || sb.IsPercent}, true
}

// sizeProp is a size longhand that interpolates through calc-size().
func sizeProp(get func(*style.Computed) *style.Length) prop {
	p := lengthField(get, sizeLength)
	p.anim.lerp = func(dst, a, b *style.Computed, t float32) bool {
		v, ok := lerpSizes(*get(a), *get(b), t, a.KeywordSizes || b.KeywordSizes)
		if ok {
			*get(dst) = v
		}
		return ok
	}
	return p
}
