package css //nolint:testpackage // reads the package's test helpers and its transitions' state

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestADisplayTransitionKeepsTheBoxToTheEnd(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `div { display: block; opacity: 1;
  transition: display 1s linear allow-discrete, opacity 1s linear } div.on { display: none; opacity: 0 }`)
	d.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.9)
	tr.Compute(root, sheets)
	if styleOf(d).Display != style.Block {
		t.Error("the box stays all along the transition")
	}
	opacityAt(t, "fading", d, 0.1)
	tr.Step(0.2)
	tr.Compute(root, sheets)
	if styleOf(d).Display != style.DisplayNone {
		t.Error("none once it ends")
	}
}

func TestADiscreteValueFlipsHalfwayOnlyWhenAllowed(t *testing.T) {
	for _, c := range []struct {
		behavior string
		mid      style.Float
	}{{"allow-discrete", style.FloatLeft}, {"normal", style.FloatRight}} {
		tr, root, d, sheets := animationPage(t, `div { float: left; transition: float 1s linear `+c.behavior+
			` } div.on { float: right }`)
		d.AddClass("on")
		tr.Compute(root, sheets)
		tr.Step(0.4)
		tr.Compute(root, sheets)
		if styleOf(d).Float != c.mid {
			t.Errorf("%s at 0.4: float %v, want %v", c.behavior, styleOf(d).Float, c.mid)
		}
		tr.Step(0.2)
		tr.Compute(root, sheets)
		if styleOf(d).Float != style.FloatRight {
			t.Errorf("%s at 0.6: the end's", c.behavior)
		}
	}
}

func TestAKeyframeShowsABoxAllAlongFromNone(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes k { from { display: none } to { display: flex } }
div { animation: k 1s linear }`)
	for _, at := range []float32{0.1, 0.8} {
		tr.Step(at - tr.anims[d].list[0].elapsed)
		tr.Compute(root, sheets)
		if styleOf(d).Display != style.FlexDisplay {
			t.Errorf("at %v: display %v", at, styleOf(d).Display)
		}
	}
}

func TestATransitionStartsFromTheStartingStyle(t *testing.T) {
	src := `div { opacity: 1; transition: opacity 1s linear } div.hid { display: none; opacity: 0.2 }
@starting-style { div.first { opacity: 0 } }`
	tr, root, d, sheets := animationPage(t, src)
	opacityAt(t, "no starting style for it", d, 1)

	d.AddClass("hid")
	tr.Compute(root, sheets)
	d.RemoveClass("hid")
	tr.Compute(root, sheets)
	opacityAt(t, "out of display: none, nothing to start from", d, 1)

	first := el(dom.Div, "f", "first")
	root.Children()[0].Append(first)
	tr.Compute(root, sheets)
	tr.Step(0.25)
	tr.Compute(root, sheets)
	opacityAt(t, "a first style starts from the starting style", first, 0.25)
}

func TestStartingStyleRulesApplyToNothingElse(t *testing.T) {
	s := sheet(t, `@starting-style { div { opacity: 0 } }
@media (min-width: 1px) { @starting-style { p { color: red } } }`)
	if !strings.Contains(s.String(), "@starting-style { div { opacity: 0; } }") {
		t.Errorf("written back: %s", s.String())
	}
	tr, _, d, _ := animationPage(t, `@starting-style { div { opacity: 0 } }`)
	opacityAt(t, "styled", d, 1)
	if !tr.starts || rulesOff([]*Sheet{s}, Media{Width: 10}, true)[0][1] {
		t.Error("the starting style reads them, under their @media")
	}
}

func TestANestedStartingStyleStartsItsRulesSelectors(t *testing.T) {
	s := sheet(t, `div.a, p { opacity: 1; @starting-style { opacity: 0.2 } color: red }`)
	if len(s.Rules) != 2 || !s.Rules[1].starting || s.Rules[0].starting || len(s.Rules[0].Decls) != 2 ||
		len(s.Rules[1].Selectors) != 2 {
		t.Fatalf("read as %s", s.String())
	}
	tr, root, _, sheets := animationPage(t,
		`div { opacity: 1; transition: opacity 1s linear; @starting-style { opacity: 0 } }`)
	d := el(dom.Div, "n")
	root.Children()[0].Append(d)
	tr.Compute(root, sheets)
	tr.Step(0.5)
	tr.Compute(root, sheets)
	opacityAt(t, "from the nested starting style", d, 0.5)
}

func TestCompositionAddsToTheElementsOwnValue(t *testing.T) {
	src := `@keyframes w { from { width: 10px } to { width: 20px } }
div { width: 100px; animation: w 1s linear; animation-composition: %s }`
	for comp, want := range map[string]float32{"replace": 15, "add": 115, "accumulate": 115} {
		tr, root, d, sheets := animationPage(t, strings.Replace(src, "%s", comp, 1))
		tr.Step(0.5)
		tr.Compute(root, sheets)
		if got := styleOf(d).Size.Width.Px; !nearf(got, want) {
			t.Errorf("%s: width %v, want %v", comp, got, want)
		}
	}
}

func TestAccumulatedTransformsSumFunctionByFunction(t *testing.T) {
	a := []style.TransformOp{{Kind: style.ScaleOp, A: [6]float32{2, 2}}, {Kind: style.Rotate, A: [6]float32{0.1}}}
	b := []style.TransformOp{{Kind: style.ScaleOp, A: [6]float32{3, 1}}, {Kind: style.Rotate, A: [6]float32{0.2}}}
	sum, _ := addTransforms(a, b, true)
	if sum[0].A[0] != 4 || sum[0].A[1] != 2 || !nearf(sum[1].A[0], 0.3) {
		t.Errorf("accumulated: %+v", sum)
	}
	if added, _ := addTransforms(a, b, false); len(added) != 4 {
		t.Errorf("added: %+v", added)
	}
	if mixed, _ := addTransforms(a, b[:1], true); len(mixed) != 3 {
		t.Errorf("other functions are put after: %+v", mixed)
	}
}

func TestAChildsOwnTransitionRunsToItsParentsEndValue(t *testing.T) {
	kid := el(dom.Div, "k", "kid")
	parent := el(dom.Div, "p")
	parent.Append(kid)
	root := page(parent)
	sheets := []*Sheet{sheet(t, `#p { font-size: 10px; transition: font-size 1s linear } #p.on { font-size: 20px }
.kid { transition: font-size 2s linear }`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	parent.AddClass("on")
	kid.AddClass("on")
	tr.Compute(root, sheets)
	for range 10 {
		tr.Step(0.05)
		tr.Compute(root, sheets)
	}
	if got := shown(parent).Font.Size; !nearf(got, 15) {
		t.Errorf("parent at 0.5: %v", got)
	}
	if got := shown(kid).Font.Size; !nearf(got, 12.5) {
		t.Errorf("child at 0.5 of its 2s to 20: %v", got)
	}
}
