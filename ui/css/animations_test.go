package css //nolint:testpackage // reads Shown and Style off dom with the package's test helpers

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func animationPage(t *testing.T, css string) (*Transitions, *dom.Node, *dom.Node, []*Sheet) {
	t.Helper()
	d := el(dom.Div, "d")
	root := page(d)
	sheets := []*Sheet{sheet(t, css)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	return tr, root, d, sheets
}

func opacityAt(t *testing.T, what string, n *dom.Node, want float32) {
	t.Helper()
	if got := shown(n).Opacity; !nearf(got, want) {
		t.Errorf("%s: opacity %v, want %v", what, got, want)
	}
}

func TestKeyframesParse(t *testing.T) {
	s, warns := Parse("k.css", `
@keyframes pulse { from { opacity: .6 } 50%, 75% { opacity: 1; transform: scale(1.05) } to { opacity: .6 } }
@keyframes "quoted" { 0% { color: red; animation-timing-function: linear } }
@keyframes pulse { to { opacity: 0 } }
@keyframes bad { 120% { opacity: 0 } 10% { opacity: 1 !important; color: red } }
`)
	if len(warns) != 2 || !strings.Contains(warns[0].Msg, "120%") || !strings.Contains(warns[1].Msg, "!important") {
		t.Fatalf("warnings: %s", warnings(warns))
	}
	names := []string{}
	for _, k := range s.Keyframes {
		names = append(names, k.Name)
	}
	if !slices.Equal(names, []string{"pulse", "quoted", "pulse", "bad"}) {
		t.Fatalf("keyframes rules %v", names)
	}
	p := s.Keyframes[0]
	var at []float32
	for _, f := range p.Frames {
		at = append(at, f.At)
	}
	if !slices.Equal(at, []float32{0, 0.5, 0.75, 1}) || len(p.Frames[1].Decls) != 2 {
		t.Errorf("pulse's frames %+v", p.Frames)
	}
	if b := s.Keyframes[3]; len(b.Frames) != 1 || len(b.Frames[0].Decls) != 1 || b.Frames[0].Decls[0].Name != "color" {
		t.Errorf("a bad selector drops its keyframe, !important its declaration: %+v", b.Frames)
	}
	if !strings.Contains(s.String(), `@keyframes pulse {`) || !strings.Contains(s.String(), `@keyframes "quoted" {`) ||
		!strings.Contains(s.String(), "50% {") {
		t.Errorf("String writes the keyframes back:\n%s", s.String())
	}
}

func TestJoinedAndImportedSheetsKeepTheirKeyframes(t *testing.T) {
	a, _ := Parse("a.css", `@keyframes f { from { opacity: 0.1 } } @media (min-width: 9px) { @keyframes g {} }`)
	b, _ := Parse("b.css", `@keyframes f { from { opacity: 0.2 } }`)
	j, _ := Join("j", "print", []*Sheet{a, b})
	names := func(s *Sheet) []string {
		var out []string
		for _, k := range s.Keyframes {
			var media []string
			for _, m := range k.media {
				media = append(media, s.media[m].text)
			}
			out = append(out, k.Name+"@"+strings.Join(media, "&"))
		}
		return out
	}
	if got := names(j); !slices.Equal(got, []string{"f@print", "g@print&(min-width: 9px)", "f@print"}) {
		t.Errorf("joined: %v", got)
	}
	site, _ := Parse("site.css", `@import url(a.css) screen; @keyframes h {}`, Browser(), From("https://x.test/"))
	if got := names(site.Imported([]*Sheet{a})); !slices.Equal(got,
		[]string{"f@screen", "g@screen&(min-width: 9px)", "h@"}) {
		t.Errorf("imported, before the sheet's own: %v", got)
	}
}

func TestAnimationLonghandsAndShorthand(t *testing.T) {
	_, _, d, _ := animationPage(t, `@keyframes a {} @keyframes b {}
div { animation: a 2s ease-in 0.5s 3 alternate both paused, 300ms infinite reverse linear b;
  animation-timing-function: steps(2), ease; }`)
	got := styleOf(d).Animations
	want := []style.Animation{
		{Name: "a", Duration: 2, Delay: 0.5, Easing: "steps(2)", Iterations: 3, Direction: style.AnimateAlternate,
			Fill: style.FillBoth, Paused: true},
		{Name: "b", Duration: 0.3, Easing: "ease", Iterations: float32(math.Inf(1)), Direction: style.AnimateReverse},
	}
	if !slices.Equal(got, want) {
		t.Errorf("animations\n got %+v\nwant %+v", got, want)
	}
	_, _, d, _ = animationPage(t, `div { animation: none }`)
	if styleOf(d).Animations != nil {
		t.Errorf("none: %+v", styleOf(d).Animations)
	}
	if _, warns := Parse("w.css", `div { animation: 1s 2s 3s a } p { animation-iteration-count: -1 }`); len(warns) != 2 {
		t.Errorf("three times and a negative count are refused: %s", warnings(warns))
	}
}

func TestAFadeInFillsForwardsToTheComputedValue(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes in { from { opacity: 0 } }
div { opacity: 0.8; animation: in 1s linear forwards }`)
	opacityAt(t, "start", d, 0)
	if styleOf(d).Opacity != 0.8 {
		t.Error("Style keeps the cascade's value")
	}
	if !tr.Step(0.5) {
		t.Fatal("running")
	}
	opacityAt(t, "half", d, 0.4)
	if tr.Step(0.6) {
		t.Error("over after 1 s")
	}
	opacityAt(t, "filled to the missing 100%: the computed value", d, 0.8)
}

func TestFillModes(t *testing.T) {
	for _, c := range []struct {
		fill          string
		before, after float32
	}{
		{"none", 1, 1},
		{"forwards", 1, 0.2},
		{"backwards", 0.6, 1},
		{"both", 0.6, 0.2},
	} {
		tr, _, d, _ := animationPage(t, `@keyframes f { from { opacity: 0.6 } to { opacity: 0.2 } }
div { animation: f 1s linear 1s `+c.fill+` }`)
		tr.Step(0.5)
		opacityAt(t, c.fill+" in the delay", d, c.before)
		tr.Step(1)
		opacityAt(t, c.fill+" halfway", d, 0.4)
		tr.Step(1)
		opacityAt(t, c.fill+" after", d, c.after)
	}
}

func TestANegativeDelayStartsPartway(t *testing.T) {
	_, _, d, _ := animationPage(t, `@keyframes f { to { opacity: 0 } } div { animation: f 2s linear -1s }`)
	opacityAt(t, "starts halfway", d, 0.5)
}

func TestIterationsAndDirections(t *testing.T) {
	for _, c := range []struct {
		anim string
		at   float32
		want float32
	}{
		{"2 alternate", 1.25, 0.75},
		{"2 alternate-reverse", 1.25, 0.25},
		{"reverse", 0.25, 0.75},
		{"infinite", 5.25, 0.25},
		{"1.5 forwards", 3, 0.5},
		{"2 alternate forwards", 3, 0},
		{"2 forwards", 3, 1},
		{"0 forwards", 1, 0},
	} {
		tr, _, d, _ := animationPage(t, `@keyframes f { from { opacity: 0 } to { opacity: 1 } }
div { animation: f 1s linear `+c.anim+` }`)
		tr.Step(c.at)
		opacityAt(t, c.anim, d, c.want)
	}
}

func TestAnInfiniteAnimationKeepsRunning(t *testing.T) {
	tr, _, _, _ := animationPage(t, `@keyframes spin { to { transform: rotate(360deg) } }
div { animation: spin 1s linear infinite }`)
	for range 5 {
		if !tr.Step(0.7) {
			t.Fatal("an infinite animation never stops")
		}
	}
}

func TestTimingFunctionsApplyPerKeyframe(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes f {
  from { opacity: 0 } 50% { opacity: 0.5; animation-timing-function: steps(2) } to { opacity: 1 } }
div { animation: f 2s linear }`)
	tr.Step(0.5)
	opacityAt(t, "linear interval", d, 0.25)
	tr.Step(0.75)
	opacityAt(t, "steps(2) from 50%", d, 0.5)
	tr.Step(0.5)
	opacityAt(t, "second step", d, 0.75)
	tr2, _, d2, _ := animationPage(t, `@keyframes f { to { opacity: 0 } } div { animation: f 1s steps(4) }`)
	tr2.Step(0.3)
	opacityAt(t, "the element's steps(4)", d2, 0.75)
}

func TestEachPropertyRunsOnItsOwnKeyframes(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes f {
  0% { opacity: 0 } 50% { transform: translateX(10px) } 100% { opacity: 1 } }
div { animation: f 1s linear }`)
	tr.Step(0.25)
	opacityAt(t, "opacity skips the 50% that doesn't set it", d, 0.25)
	if ops := shown(d).Transform; len(ops) != 1 || !nearf(ops[0].X.Px, 5) {
		t.Errorf("transform from none to the 50%%: %+v", ops)
	}
}

func TestPlayStatePausesTheClock(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { to { opacity: 0 } }
div { animation: f 1s linear } .paused { animation-play-state: paused }`)
	tr.Step(0.25)
	d.AddClass("paused")
	tr.Compute(root, sheets)
	if tr.Step(1) {
		t.Error("a paused animation doesn't run")
	}
	opacityAt(t, "paused", d, 0.75)
	d.RemoveClass("paused")
	tr.Compute(root, sheets)
	tr.Step(0.25)
	opacityAt(t, "resumed where it paused", d, 0.5)
}

func TestImportantBeatsTheAnimation(t *testing.T) {
	_, _, d, _ := animationPage(t, `@keyframes f { from { opacity: 0; color: #fff } to { opacity: 0 } }
div { opacity: 0.5 !important; color: #000; animation: f 1s }`)
	opacityAt(t, "an !important longhand", d, 0.5)
	srgbNear(t, "the rest animates", shown(d).Color, [4]float32{1, 1, 1, 1})
}

func TestADiscretePropertyFlipsHalfway(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f {
  from { text-align: right; z-index: 3 } to { text-align: center } }
div { position: relative; animation: f 1s linear }`)
	if s := styleOf(d); s.TextAlign != style.Right || s.ZIndex != 3 {
		t.Errorf("at the start: align %v z %v", s.TextAlign, s.ZIndex)
	}
	tr.Step(0.49)
	tr.Compute(root, sheets)
	if styleOf(d).TextAlign != style.Right {
		t.Error("before half: the from value")
	}
	tr.Step(0.02)
	tr.Compute(root, sheets)
	if s := styleOf(d); s.TextAlign != style.Center || !s.ZAuto {
		t.Errorf("past half: align %v, z-index back to auto %v", s.TextAlign, s.ZAuto)
	}
	tr.Step(1)
	tr.Compute(root, sheets)
	if styleOf(d).TextAlign != style.Start {
		t.Error("over: the cascade's value")
	}
}

func TestVisibilityShowsWhileEitherEndShows(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes hide { to { visibility: hidden } }
div { animation: hide 1s linear forwards }`)
	tr.Step(0.9)
	tr.Compute(root, sheets)
	if !styleOf(d).Visible {
		t.Error("visible all the way to the end")
	}
	tr.Step(0.2)
	tr.Compute(root, sheets)
	if styleOf(d).Visible {
		t.Error("hidden once over")
	}
}

func TestALengthInterpolatesAndRelaysOut(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes grow { from { width: 0 } to { width: 10em } }
div { font-size: 10px; width: auto; animation: grow 1s linear }`)
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	tr.Step(0.25)
	if dom.Data(d).Dirty&dom.DirtyStyle == 0 || dom.Data(root).Dirty&dom.DirtyDescendant == 0 {
		t.Fatal("a length that moves restyles the element")
	}
	tr.Compute(root, sheets)
	if w := styleOf(d).Size.Width; w.Kind != style.Fixed || !nearf(w.Px, 25) {
		t.Errorf("width at a quarter: %+v", w)
	}
	if dom.Data(d).Dirty&dom.DirtyLayout == 0 {
		t.Error("and lays it out again")
	}
	tr.Step(1)
	tr.Compute(root, sheets)
	if styleOf(d).Size.Width.Kind != style.Auto {
		t.Error("over: auto again")
	}
}

func TestKeyframesFollowTheElement(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { to { transform: translateX(2em) } }
div { font-size: 10px; animation: f 1s linear } .big { font-size: 20px }`)
	tr.Step(0.5)
	if ops := shown(d).Transform; len(ops) != 1 || !nearf(ops[0].X.Px, 10) {
		t.Fatalf("em resolves on the element: %+v", ops)
	}
	d.AddClass("big")
	tr.Compute(root, sheets)
	if ops := shown(d).Transform; len(ops) != 1 || !nearf(ops[0].X.Px, 20) {
		t.Errorf("and again when its style changes: %+v", ops)
	}
}

func TestANameEnteringStartsAndLeavingCancels(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { from { opacity: 0 } }
div { opacity: 1 } .a { animation: f 1s linear } .long { animation-duration: 2s }`)
	opacityAt(t, "no name", d, 1)
	d.AddClass("a")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	opacityAt(t, "started", d, 0.5)
	d.AddClass("long")
	tr.Compute(root, sheets)
	opacityAt(t, "a new duration keeps its time", d, 0.25)
	d.RemoveClass("a")
	tr.Compute(root, sheets)
	opacityAt(t, "cancelled", d, 1)
	if tr.Step(0.1) {
		t.Error("nothing runs")
	}
	d.AddClass("a")
	tr.Compute(root, sheets)
	opacityAt(t, "restarts from the beginning", d, 0)
}

func TestAFinishedAnimationStaysFinished(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { from { opacity: 0 } }
div { animation: f 1s linear } .x { color: #f00 }`)
	tr.Step(2)
	d.AddClass("x")
	tr.Compute(root, sheets)
	opacityAt(t, "another change doesn't restart it", d, 1)
}

func TestDisplayNoneCancelsAndShowingRestarts(t *testing.T) {
	kid := el(dom.Span, "kid")
	d := with(el(dom.Div, "d"), kid)
	root := page(d)
	sheets := []*Sheet{sheet(t, `@keyframes f { from { opacity: 0 } }
span, div { animation: f 1s linear } .gone { display: none }`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	tr.Step(0.5)
	d.AddClass("gone")
	tr.Compute(root, sheets)
	if tr.Step(0.1) {
		t.Error("display: none cancels its animations and its descendants'")
	}
	d.RemoveClass("gone")
	tr.Compute(root, sheets)
	opacityAt(t, "shown again: restarted", d, 0)
	opacityAt(t, "and its child's", kid, 0)
}

func TestARemovedNodesAnimationStops(t *testing.T) {
	tr, _, d, _ := animationPage(t, `@keyframes f { to { opacity: 0 } } div { animation: f 1s infinite }`)
	d.Parent().Remove(d)
	if tr.Step(0.1) {
		t.Error("a removed node's animation stops")
	}
}

func TestANameWithNoKeyframesRunsNothing(t *testing.T) {
	tr, root, d, _ := animationPage(t, `div { animation: nowhere 1s }`)
	if tr.Step(0.1) || shown(d) != styleOf(d) {
		t.Error("no @keyframes: no animation")
	}
	sheets := []*Sheet{sheet(t, `@keyframes nowhere { from { opacity: 0 } } div { animation: nowhere 1s linear }`)}
	tr.Compute(root, sheets)
	opacityAt(t, "the rule arriving starts it", d, 0)
}

func TestTheLastKeyframesOfANameWins(t *testing.T) {
	_, _, d, _ := animationPage(t, `@keyframes f { from { opacity: 0.1 } }
@media (max-width: 1px) { @keyframes f { from { opacity: 0.9 } } }
@keyframes f { from { opacity: 0.3 } }
@media (min-width: 0) { @keyframes f { from { opacity: 0.5 } } }
div { animation: f 1s }`)
	opacityAt(t, "the last that applies", d, 0.5)
}

func TestLaterAnimationsWinAProperty(t *testing.T) {
	_, _, d, _ := animationPage(t, `@keyframes a { from { opacity: 0.2 } } @keyframes b { from { opacity: 0.4 } }
div { animation: a 1s, b 1s }`)
	opacityAt(t, "b, last in the list", d, 0.4)
}

func TestAnimatedColorReachesTheChildren(t *testing.T) {
	kid := el(dom.Span, "kid")
	d := with(el(dom.Div, "d"), kid)
	root := page(d)
	sheets := []*Sheet{sheet(t, `@keyframes ink { from { color: #000 } to { color: #fff } }
div { color: #f00; border: 1px solid; animation: ink 1s linear }`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	tr.Step(0.5)
	srgbNear(t, "div", shown(d).Color, [4]float32{0.5, 0.5, 0.5, 1})
	srgbNear(t, "its currentColor border", shown(d).BorderColor.Top, [4]float32{0.5, 0.5, 0.5, 1})
	srgbNear(t, "the span inheriting it", shown(kid).Color, [4]float32{0.5, 0.5, 0.5, 1})
}

func TestAnAnimationStartsNoTransition(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { from { opacity: 0 } to { opacity: 0 } }
div { transition: opacity 1s; animation: f 1s linear } .x { color: #f00 }`)
	d.AddClass("x")
	tr.Compute(root, sheets)
	if tr.Running(d) && len(tr.nodes[d].runs) > 0 {
		t.Error("the animated value is no change to transition from")
	}
	tr.Step(1.1)
	opacityAt(t, "over: the cascade's", d, 1)
}

func TestAnimationEvents(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { to { opacity: 0 } }
div { animation: f 1s linear 0.5s 2 } .off { animation: none }`)
	if ev := tr.Events(); len(ev) != 0 {
		t.Errorf("nothing before the delay: %+v", ev)
	}
	tr.Step(0.6)
	tr.Step(1)
	tr.Step(1)
	want := []AnimationEvent{
		{Node: d, Name: "f", Kind: AnimationStart, Elapsed: 0},
		{Node: d, Name: "f", Kind: AnimationIteration, Elapsed: 1},
		{Node: d, Name: "f", Kind: AnimationEnd, Elapsed: 2},
	}
	if got := tr.Events(); !slices.Equal(got, want) {
		t.Errorf("events\n got %+v\nwant %+v", got, want)
	}
	if tr.Events() != nil {
		t.Error("Events drains the queue")
	}
	d.AddClass("off")
	tr.Compute(root, sheets)
	d.RemoveClass("off")
	tr.Compute(root, sheets)
	tr.Step(0.6)
	d.AddClass("off")
	tr.Compute(root, sheets)
	if got := tr.Events(); len(got) != 2 || got[1].Kind != AnimationCancel {
		t.Errorf("start, then cancel: %+v", got)
	}
}

func TestAPseudoElementAnimates(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes spin { to { transform: rotate(180deg) } }
div::before { content: ""; width: 0; animation: spin 1s linear infinite }
@keyframes grow { to { width: 10px } } .w::before { animation: grow 1s linear }`)
	tr.Step(0.5)
	if ops := shown(dom.Data(d).Before).Transform; len(ops) != 1 || !nearf(ops[0].A[0], math.Pi/2) {
		t.Errorf("::before spins: %+v", ops)
	}
	d.AddClass("w")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	tr.Compute(root, sheets)
	if w := styleOf(dom.Data(d).Before).Size.Width; !nearf(w.Px, 5) {
		t.Errorf("and grows: %+v", w)
	}
}
