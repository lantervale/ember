package css //nolint:testpackage // tests the unexported interpolations and the longhands' table

import (
	"math"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestEveryLonghandDeclaresHowItAnimates(t *testing.T) {
	for name := range properties {
		p, ok := props[name]
		switch {
		case !ok:
			t.Errorf("%s: no prop", name)
		case p.anim.kind == undeclared:
			t.Errorf("%s: declares no animation type", name)
		case p.anim.moves() && (p.anim.lerp == nil || p.anim.equal == nil):
			t.Errorf("%s: interpolates without lerp and equal", name)
		}
	}
}

func TestWhatAKeyframeAndATransitionTake(t *testing.T) {
	for name, want := range map[string]bool{
		"width": true, "font-size": true, "margin": true, "border": true, "text-align": true,
		"display": true, "transition-behavior": false, "animation-name": false, "animation": false,
		"transition": false, "--x": true,
	} {
		if animates(name) != want {
			t.Errorf("animates(%s) = %v", name, !want)
		}
	}
	for _, name := range []string{"width", "font-size", "margin", "border-radius", "display", "inset"} {
		if err := transitionTarget(word(name)); err != nil {
			t.Errorf("transition-property: %s refused: %v", name, err)
		}
	}
	if transitionTarget(word("nonsense")) == nil {
		t.Error("an unknown property is refused")
	}
}

func TestATransitionOfAShorthandCoversItsLonghands(t *testing.T) {
	list := []style.Transition{{Property: "margin", Duration: 1}, {Property: "border-color", Duration: 2}}
	if tr, ok := transitionFor(list, "margin-left", false); !ok || tr.Duration != 1 {
		t.Error("margin covers margin-left")
	}
	if tr, ok := transitionFor(list, "border-top-color", false); !ok || tr.Duration != 2 {
		t.Error("border-color covers border-top-color")
	}
	if _, ok := transitionFor(list, "border-top-width", false); ok {
		t.Error("border-color doesn't cover border-top-width")
	}
}

func px(v float32) style.Length { return style.Px(v) }

func TestLengthsInterpolate(t *testing.T) {
	cases := []struct {
		name     string
		a, b     style.Length
		t        float32
		nonNeg   bool
		want     style.Length
		lerpable bool
	}{
		{"px", px(10), px(20), 0.25, false, px(12.5), true},
		{"px and % mix", px(100), pct(50), 0.35, false,
			style.Length{Kind: style.Fixed, Px: 65, Percent: 17.5, IsPercent: true}, true},
		{"negative past an end", px(0), px(10), -0.5, false, px(-5), true},
		{"clamped at zero", px(0), px(10), -0.5, true, px(0), true},
		{"fr", style.Length{Kind: style.Fr, Px: 1}, style.Length{Kind: style.Fr, Px: 3}, 0.35, true,
			style.Length{Kind: style.Fr, Px: 1.7}, true},
		{"auto and a length flip", style.Length{}, px(10), 0.5, false, style.Length{}, false},
		{"auto with auto", style.Length{}, style.Length{}, 0.5, false, style.Length{}, true},
		{"none and a length flip", style.Length{Kind: style.None}, px(10), 0.5, false,
			style.Length{Kind: style.None}, false},
	}
	for _, c := range cases {
		got, ok := lerpLength(c.a, c.b, c.t, c.nonNeg)
		if ok != c.lerpable || ok && (got.Kind != c.want.Kind || !nearf(got.Px, c.want.Px) ||
			!nearf(got.Percent, c.want.Percent) || got.IsPercent != c.want.IsPercent) {
			t.Errorf("%s: got %+v %v, want %+v %v", c.name, got, ok, c.want, c.lerpable)
		}
	}
}

func TestACalcOfMinInterpolatesAsASum(t *testing.T) {
	d, _ := parseDim(componentsOf("min(10px, 10%)")[0])
	c := calcOf(d)
	a := style.Length{Kind: style.Fixed, IsPercent: true, Calc: c, Px: 10}
	got, ok := lerpLength(a, px(100), 0.35, false)
	if !ok {
		t.Fatal("a calc() interpolates")
	}
	calc := got.Calc.Value()
	for ref, want := range map[float32]float32{50: 0.65*5 + 35, 400: 0.65*10 + 35} {
		if v := calc.At(ref); !nearf(v, want) {
			t.Errorf("against %v: %v, want %v", ref, v, want)
		}
	}
}

func TestPositionsInterpolateFromTheStartEdge(t *testing.T) {
	right := style.BackgroundPosition{X: px(10), Y: pct(0), FromRight: true}
	got, ok := positions.lerp(right, style.TopLeft, 0.35)
	if !ok || got.FromRight || !nearf(got.X.Percent, 65) || !nearf(got.X.Px, -6.5) || !got.X.IsPercent {
		t.Errorf("right 10px to 0: %+v", got.X)
	}
}

func TestRepeatableListsRepeatToTheirLeastCommonMultiple(t *testing.T) {
	a := []style.BackgroundPosition{style.TopLeft, {X: px(10), Y: px(10)}}
	b := []style.BackgroundPosition{{X: px(100), Y: px(50)}, {X: px(20), Y: px(20)}, {X: px(30), Y: px(30)}}
	got, ok := repeatable(positions, style.TopLeft).lerp(a, b, 0.5)
	if !ok || len(got) != 6 {
		t.Fatalf("got %d positions, %v", len(got), ok)
	}
	if !nearf(got[1].X.Px, 15) || !nearf(got[3].X.Px, 55) {
		t.Errorf("item by item: %+v %+v", got[1].X, got[3].X)
	}
	auto := style.BackgroundSize{Width: px(10)}
	if _, lerped := repeatable(backgroundSizes, style.BackgroundSize{}).lerp(
		[]style.BackgroundSize{auto}, []style.BackgroundSize{{Width: px(10), Height: px(4)}}, 0.5); lerped {
		t.Error("auto against a length flips the list")
	}
	if _, lerped := backgroundSizes.lerp(style.BackgroundSize{Scale: style.ScaleCover}, auto, 0.5); lerped {
		t.Error("cover flips")
	}
}

func TestNumbersAndIntegers(t *testing.T) {
	if v, _ := numbers(0, 1).lerp(0.2, 1, 1.5); v != 1 {
		t.Errorf("opacity past its end: %v", v)
	}
	for _, c := range []struct {
		a, b int
		t    float32
		want int
	}{{0, 3, 0.35, 1}, {0, 3, 0.5, 2}, {0, -3, 0.5, -1}, {0, -3, 0.501, -2}, {400, 700, 0.35, 505}} {
		if v, _ := integers(math.MinInt).lerp(c.a, c.b, c.t); v != c.want {
			t.Errorf("%d to %d at %v: %d, want %d", c.a, c.b, c.t, v, c.want)
		}
	}
	if _, ok := counts.lerp(0, 5, 0.5); ok {
		t.Error("column-count auto flips")
	}
	if v, _ := ratios.lerp(1, 4, 0.35); !nearf(v, 1.6245) {
		t.Errorf("aspect-ratio by its logarithm: %v", v)
	}
	if v, _ := widths.lerp(10, 0, 1.5); v != 0 {
		t.Errorf("a width under none: %v", v)
	}
}

func TestWidthsSnapToDevicePixels(t *testing.T) {
	for _, c := range []struct {
		dppx, v float32
		least   bool
		want    float32
	}{
		{1, 4.7, true, 4}, {2, 4.7, true, 4.5}, {1, 0.2, true, 1}, {2, 0.2, true, 0.5}, {2, 0, true, 0},
		{1, -3.7, false, -3}, {2, -3.7, false, -3.5}, {2, 0.3, false, 0}, {3, 1, true, 1}, {0, 2.5, false, 2},
		{0.8, 2, true, 2},
	} {
		m := &measures{dppx: c.dppx}
		if got := m.snap(c.v, c.least); !nearf(got, c.want) {
			t.Errorf("%v at %v dppx: %v, want %v", c.v, c.dppx, got, c.want)
		}
	}
}

func TestKindsThatOnlyMatchTheirOwn(t *testing.T) {
	if _, ok := lineHeights.lerp(style.LineHeight{Factor: 1}, style.LineHeight{Px: 30}, 0.5); ok {
		t.Error("a number and a length flip")
	}
	if v, ok := lineHeights.lerp(style.LineHeight{Factor: 1}, style.LineHeight{Factor: 3}, 0.35); !ok ||
		!nearf(v.Factor, 1.7) {
		t.Errorf("two numbers: %+v", v)
	}
	if _, ok := lineHeights.lerp(style.LineHeight{Normal: true}, style.LineHeight{Factor: 3}, 0.5); ok {
		t.Error("normal flips")
	}
	number := style.ImageWidth{Kind: style.WidthTimes, V: 1}
	if _, ok := imageWidths.lerp(style.All(number), style.All(style.ImageWidth{Kind: style.WidthLength, V: 3}),
		0.5); ok {
		t.Error("a number and a length flip")
	}
	if v, ok := imageSlices.lerp(style.All[float32](10), style.All[float32](30), 0.35); !ok || !nearf(v.Top, 17) {
		t.Errorf("slices: %+v", v)
	}
	if _, ok := imageSlices.lerp(style.Sides[float32]{}, style.All[float32](30), 0.35); ok {
		t.Error("100% flips against numbers")
	}
}

func TestVisibilityIsVisibleWhileEitherEndIs(t *testing.T) {
	for _, c := range []struct {
		a, b bool
		t    float32
		want bool
	}{{true, false, 0.99, true}, {false, true, 0.01, true}, {false, true, 0, false}, {true, false, 1, false}} {
		if v, _ := visibility.lerp(c.a, c.b, c.t); v != c.want {
			t.Errorf("%v to %v at %v: %v", c.a, c.b, c.t, v)
		}
	}
}

func TestShadowsPadWithTransparentOnes(t *testing.T) {
	red := color.Color{R: 1, A: 1}
	got, ok := shadows.lerp([]style.Shadow{{X: 1, Y: 2, Blur: 3, Color: red}}, nil, 0.35)
	if !ok || len(got) != 1 || !nearf(got[0].X, 0.65) || !nearf(got[0].Color.A, 0.65) || !nearf(got[0].Color.R, 1) {
		t.Errorf("to none: %+v", got)
	}
	if _, lerped := shadows.lerp([]style.Shadow{{Inset: true}}, []style.Shadow{{}}, 0.5); lerped {
		t.Error("an inset and an outer shadow flip")
	}
}

func TestTrackListsOfTheSameShapeInterpolate(t *testing.T) {
	list := func(a, fr float32, name string) []style.TrackRepeat {
		return []style.TrackRepeat{{Count: 1, Tracks: []style.Track{{Min: px(a), Max: px(a)}}, Before: []string{name}},
			{Count: 1, Tracks: []style.Track{{Min: style.Length{Kind: style.Fr, Px: fr},
				Max: style.Length{Kind: style.Fr, Px: fr}}}}}
	}
	got, ok := trackLists.lerp(list(100, 1, "a"), list(200, 3, "c"), 0.35)
	if !ok || !nearf(got[0].Tracks[0].Min.Px, 135) || !nearf(got[1].Tracks[0].Max.Px, 1.7) || got[0].Before[0] != "a" {
		t.Errorf("at 0.35: %+v", got)
	}
	if got, _ = trackLists.lerp(list(100, 1, "a"), list(200, 3, "c"), 0.5); got[0].Before[0] != "c" {
		t.Error("names flip halfway")
	}
	if _, lerped := trackLists.lerp(list(100, 1, "a")[:1], list(200, 3, "c"), 0.5); lerped {
		t.Error("lists of other lengths flip")
	}
}

func TestATransitionRunsOnALayoutProperty(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { width: 100px; font-size: 10px; padding-left: 1em;
  transition: width 1s linear, font-size 1s linear } button.on { width: 200px; font-size: 20px }`)
	child := el(dom.Span, "s")
	b.Append(child)
	tr.Compute(root, sheets)
	b.AddClass("on")
	tr.Compute(root, sheets)
	if w := styleOf(b).Size.Width; !nearf(w.Px, 100) {
		t.Errorf("at the start: %+v", w)
	}
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	if !tr.Step(0.25) || dom.Data(b).Dirty&dom.DirtyStyle == 0 {
		t.Fatal("a transition on Style restyles its element each step")
	}
	tr.Compute(root, sheets)
	if w := styleOf(b).Size.Width; !nearf(w.Px, 125) {
		t.Errorf("width at a quarter: %+v", w)
	}
	if f := styleOf(child).Font.Size; !nearf(f, 12.5) {
		t.Errorf("the child inherits the moving font size: %v", f)
	}
	if p := styleOf(b).Padding.Left; !nearf(p.Px, 12.5) {
		t.Errorf("an em follows the moving font size: %+v", p)
	}
	tr.Step(0.5)
	tr.Compute(root, sheets)
	b.RemoveClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.25)
	tr.Compute(root, sheets)
	if w := styleOf(b).Size.Width; !nearf(w.Px, 150) {
		t.Errorf("a reversal shortened: %+v", w)
	}
	tr.Step(1)
	tr.Compute(root, sheets)
	if w := styleOf(b).Size.Width; !nearf(w.Px, 100) || tr.Running(b) {
		t.Errorf("over: %+v", w)
	}
}

func TestATransitionDoesNotRunBetweenValuesThatDontInterpolate(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { width: 100px; text-align: left; transition: all 1s linear }
button.on { width: auto; text-align: right }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	if s := styleOf(b); s.Size.Width.Kind != style.Auto || s.TextAlign != style.Right || tr.Running(b) {
		t.Errorf("auto and a keyword show at once: %+v %v", s.Size.Width, s.TextAlign)
	}
}

func TestVisibilityTransitionsAsCSSSays(t *testing.T) {
	tr, root, b, sheets := transitionPage(t,
		`button { transition: visibility 1s linear } button.on { visibility: hidden }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.9)
	tr.Compute(root, sheets)
	if !styleOf(b).Visible {
		t.Error("visible until the end")
	}
	tr.Step(0.2)
	tr.Compute(root, sheets)
	if styleOf(b).Visible {
		t.Error("hidden once over")
	}
}

func TestAKeyframeInterpolatesTheBoxsLooks(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f {
  from { font-size: 10px; border-radius: 0; background-position: right 10px top; letter-spacing: normal;
    z-index: 0; line-height: 1 }
  to { font-size: 20px; border-radius: 10px; background-position: 0 0; letter-spacing: 10px; z-index: 3;
    line-height: 3 } }
div { position: relative; padding-top: 2em; animation: f 1s linear }`)
	tr.Step(0.35)
	tr.Compute(root, sheets)
	s := styleOf(d)
	if !nearf(s.Font.Size, 13.5) || !nearf(s.Radius.TopLeft.Width.Px, 3.5) || !nearf(s.LetterGap(), 3.5) ||
		s.ZIndex != 1 || !nearf(s.LineHeight.Factor, 1.7) {
		t.Errorf("at 0.35: size %v radius %+v spacing %v z %v line %+v", s.Font.Size, s.Radius.TopLeft,
			s.LetterGap(), s.ZIndex, s.LineHeight)
	}
	if !nearf(s.Padding.Top.Px, 27) {
		t.Errorf("an em follows the animated font size: %+v", s.Padding.Top)
	}
	if p := s.BackgroundLayer(0).Position.X; !nearf(p.Percent, 65) || !nearf(p.Px, -6.5) {
		t.Errorf("background-position: %+v", p)
	}
}

func TestASubgridFlips(t *testing.T) {
	tr, root, d, sheets := animationPage(t, `@keyframes f { from { grid-template-columns: 10px 20px }
  to { grid-template-columns: subgrid } }
@keyframes g { from { grid-template-rows: 10px 20px } to { grid-template-rows: 30px 40px } }
div { display: grid; animation: f 1s linear, g 1s linear }`)
	tr.Step(0.35)
	tr.Compute(root, sheets)
	s := styleOf(d)
	if s.Subgrid.Width || len(s.Columns) != 2 || !nearf(s.Columns[0].Tracks[0].Min.Px, 10) {
		t.Errorf("before half, the tracks: %+v %v", s.Columns, s.Subgrid.Width)
	}
	if !nearf(s.Rows[0].Tracks[0].Min.Px, 17) {
		t.Errorf("two track lists interpolate: %+v", s.Rows)
	}
	tr.Step(0.3)
	tr.Compute(root, sheets)
	if !styleOf(d).Subgrid.Width {
		t.Error("past half, subgrid")
	}
}

func TestALightDarkColourMovesBetweenWhatItResolvesTo(t *testing.T) {
	tr, root, b, sheets := transitionPage(t, `button { color-scheme: light; background-color: light-dark(#000, #f00);
  transition: background-color 1s linear } button.on { background-color: light-dark(#fff, #00f) }`)
	b.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	tr.Compute(root, sheets)
	srgbNear(t, "halfway in the light scheme", shown(b).Background, [4]float32{0.5, 0.5, 0.5, 1})
}
