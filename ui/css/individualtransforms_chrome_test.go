//go:build chrome

package css_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// translateCSS, rotateCSS and scaleCSS write translate, rotate and scale
// as getComputedStyle does: y left out where it is 0px, y of a scale
// where it is x's.
func translateCSS(s *style.Computed) string {
	t := s.Translate
	if !t.Set {
		return "none"
	}
	if t.Y == style.Px(0) {
		return lengthCSS(t.X)
	}
	return lengthCSS(t.X) + " " + lengthCSS(t.Y)
}

func rotateCSS(s *style.Computed) string {
	if !s.Rotate.Set {
		return "none"
	}
	return num(s.Rotate.Deg) + "deg"
}

func scaleCSS(s *style.Computed) string {
	c := s.Scale
	switch {
	case !c.Set:
		return "none"
	case c.X == c.Y:
		return num(c.X)
	}
	return num(c.X) + " " + num(c.Y)
}

//nolint:gochecknoglobals // a test table
var individualTransforms = map[string]func(*style.Computed) string{
	"translate": translateCSS, "rotate": rotateCSS, "scale": scaleCSS,
	"transform": func(s *style.Computed) string { return matrixCSS(s.Transform) },
}

// TestIndividualTransformsMatchChrome computes translate, rotate and
// scale in each of their forms, their axes and units, and transform
// beside them, which they leave as it is.
func TestIndividualTransformsMatchChrome(t *testing.T) {
	src := `div { width: 100px; height: 20px; font-size: 10px }
.a { translate: 0 } .b { translate: 10px 0 } .c { translate: 0 10px } .d { translate: 50% 2em 0 }
.e { translate: 10% 0px } .f { translate: 10px 0% } .g { translate: 1em 0px 0px; rotate: 0.5turn }
.h { rotate: z 45deg } .i { rotate: 0 0 1 45deg } .j { rotate: 0 0 -2 45deg } .k { rotate: 30deg z; scale: 2 2 }
.l { rotate: -90deg; scale: 2 3 1 } .m { rotate: 400grad; scale: 150% 2 } .n { scale: 1 1 100%; translate: none }
.o { scale: 2 1; rotate: none } .p { scale: -1 } .q { scale: 0 }
.r { translate: 10px 5px; rotate: 30deg; scale: 2; transform: translate(3px) }
.s { translate: calc(10px + 1em) calc(2 * 4px) } .u { rotate: calc(1rad + 10deg) } .v { scale: calc(1 / 4) }`
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(src)}}, "t.css"))
	for _, c := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q",
		"r", "s", "u", "v", "none"} {
		d.Body().Append(ui.Div(ui.Class(c)))
	}
	matchChromeProps(t, d, individualTransforms, 2)
}

// TestIndividualTransformTransitionsMatchChrome moves translate, rotate
// and scale each as its own type: from and to none, lengths with
// percentages, angles past a turn, factors per axis.
func TestIndividualTransformTransitionsMatchChrome(t *testing.T) {
	src := `div { width: 100px; height: 20px; font-size: 10px; transition: translate 1s linear, rotate 1s linear,
  scale 1s linear }
.t0.on { translate: 40px 10px; rotate: 90deg; scale: 3 }
.t1 { translate: 10px; rotate: 30deg; scale: 2 0.5 } .t1.on { translate: none; rotate: none; scale: none }
.t2 { translate: 0 50%; rotate: -720deg; scale: 1 } .t2.on { translate: 50% 10px; rotate: 1turn; scale: 0.5 4 }
.t3 { translate: 2em; rotate: 0deg; scale: 100% } .t3.on { translate: calc(10px + 20%) -1em; rotate: 45deg z;
  scale: 2 }`
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			matchMovedAt(t, src, divs("t0", "t1", "t2", "t3"), individualTransforms, at, 1)
		})
	}
}

// TestIndividualTransformAnimationsMatchChrome runs keyframes of
// translate, rotate and scale replacing, adding to and accumulating on
// the element's own values and on none, and one keyframe missing an end.
func TestIndividualTransformAnimationsMatchChrome(t *testing.T) {
	src := `@keyframes w { from { translate: 10px 5px; rotate: 10deg; scale: 2 }
  to { translate: 20px 50%; rotate: 100deg; scale: 3 1 } }
@keyframes h { to { translate: 30px; rotate: 90deg; scale: 0.5 } }
@keyframes k { to { translate: 30px; scale: 0.5 } }
div { width: 100px; height: 20px; translate: 5px; rotate: 5deg; scale: 2 }
.none { translate: none; rotate: none; scale: none }
.r { animation: w 1s linear } .a { animation: w 1s linear; animation-composition: add }
.c { animation: w 1s linear; animation-composition: accumulate } .h { animation: h 1s linear }
.m { animation: w 1s linear, k 1s linear; animation-composition: replace, add }`
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			build := func(body *dom.Node) {
				for _, c := range []string{"r", "a", "c", "h", "m"} {
					body.Append(ui.Div(ui.Class(c)), ui.Div(ui.Class(c, "none")))
				}
			}
			matchMovedAt(t, src, build, individualTransforms, at, 1)
		})
	}
}

// TestAdditiveTransformAnimationsMissingAKeyframeMatchChrome adds to and
// accumulates on the element's own translate, rotate, scale and
// transform from keyframes that leave out the first or the last, so
// lists of different lengths meet, as they do between keyframes that
// share only their first functions.
func TestAdditiveTransformAnimationsMissingAKeyframeMatchChrome(t *testing.T) {
	src := `@keyframes to { to { translate: 30px 10px; rotate: 90deg; scale: 0.5 3;
  transform: rotate(20deg) translate(10px) } }
@keyframes from { from { translate: 30px 10px; rotate: 90deg; scale: 0.5 3;
  transform: rotate(20deg) translate(10px) } }
@keyframes p { from { transform: rotate(10deg) translate(5px) }
  to { transform: rotate(40deg) skewX(20deg) scale(2) } }
@keyframes mid { 50% { translate: 30px 10px; rotate: 90deg; scale: 0.5 3;
  transform: rotate(20deg) translate(10px) } }
div { width: 100px; height: 20px; translate: 5px; rotate: 30deg; scale: 2; transform: rotate(10deg) }
.none { translate: none; rotate: none; scale: none; transform: none }
.ta { animation: to 1s linear; animation-composition: add }
.tc { animation: to 1s linear; animation-composition: accumulate }
.fa { animation: from 1s linear; animation-composition: add }
.fc { animation: from 1s linear; animation-composition: accumulate }
.ma { animation: mid 1s linear; animation-composition: add } .p { animation: p 1s linear }`
	for _, at := range []float32{0.1, 0.35, 0.65, 0.9} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			build := func(body *dom.Node) {
				for _, c := range []string{"ta", "tc", "fa", "fc", "ma", "p"} {
					body.Append(ui.Div(ui.Class(c)), ui.Div(ui.Class(c, "none")))
				}
			}
			matchMovedAt(t, src, build, individualTransforms, at, 1)
		})
	}
}
