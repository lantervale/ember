package css //nolint:testpackage // holds the unexported test helpers (sheet, page) and reads computed styles off dom

import (
	"reflect"
	"strings"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func sheet(t *testing.T, src string) *Sheet {
	t.Helper()
	s, warns := Parse("t.css", src)
	if len(warns) > 0 {
		t.Fatalf("warnings: %s", warnings(warns))
	}
	return s
}

func styleOf(n *dom.Node) *style.Computed { return dom.Data(n).Style }

func hex(s string) color.Color { return linear(value2(s), color.Color{}) }

func value2(src string) Component { return items(componentsOf(src))[0] }

func page(kids ...*dom.Node) *dom.Node {
	return with(dom.NewElement(dom.HTML), with(dom.NewElement(dom.Body), kids...))
}

func TestDefaultParses(t *testing.T) {
	if _, warns := Parse("ua", Default); len(warns) > 0 {
		t.Fatal(warnings(warns))
	}
}

func TestCascadeOrder(t *testing.T) {
	d := el(dom.Div, "x", "a", "b")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `
#x { color: #00f; }
div.a { color: #0f0; }
.a.b { color: #f00; }
div { background-color: #111 !important; }
#x { background-color: #222; }
.b { opacity: 0.2; }
.a { opacity: 0.4; }
`)})
	s := styleOf(d)
	if s.Color != hex("#00f") {
		t.Errorf("id beats classes: %v", s.Color)
	}
	if s.Background != hex("#111") {
		t.Errorf("!important beats specificity: %v", s.Background)
	}
	if s.Opacity != 0.4 {
		t.Errorf("later rule wins a tie: %v", s.Opacity)
	}
}

func TestInlineAndImportant(t *testing.T) {
	d := el(dom.Div, "x")
	d.SetStyle("color: #0f0; opacity: 0.5 !important; width: 10px")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `#x { color: #f00; opacity: 0.1 !important; width: 20px !important }`)})
	s := styleOf(d)
	if s.Color != hex("#0f0") || s.Opacity != 0.5 || s.Size.Width != style.Px(20) {
		t.Errorf("color %v opacity %v width %v", s.Color, s.Opacity, s.Size.Width)
	}
}

func TestAgentImportantWinsAndAgentLoses(t *testing.T) {
	b := dom.NewElement(dom.Button)
	root := page(b)
	Compute(root, []*Sheet{sheet(t, `button { display: block; padding: 4px }`)})
	s := styleOf(b)
	if s.Display != style.Block || s.Padding.Top != style.Px(4) {
		t.Errorf("author beats agent: %v %v", s.Display, s.Padding.Top)
	}
}

func TestInheritance(t *testing.T) {
	sp := dom.NewElement(dom.Span)
	d := with(el(dom.Div, "", "box"), sp)
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `
.box { color: #f7efdd; font-size: 20px; padding: 3px; line-height: 1.5; border: 2px solid; }
span { border-color: inherit; padding: inherit; }
`)})
	s := styleOf(sp)
	if s.Color != hex("#f7efdd") || s.Font.Size != 20 || s.LineHeight.Factor != 1.5 {
		t.Errorf("inherited text: %v %v %v", s.Color, s.Font.Size, s.LineHeight)
	}
	if s.Padding.Left != style.Px(3) {
		t.Errorf("explicit inherit: %v", s.Padding.Left)
	}
	if s.BorderColor.Top != hex("#f7efdd") {
		t.Errorf("currentcolor border, inherited: %v", s.BorderColor.Top)
	}
	if s.Border.Top != 0 {
		t.Errorf("span has no border style: %v", s.Border.Top)
	}
	if b := styleOf(d).Border; b.Top != 2 || b.Left != 2 {
		t.Errorf("box border: %+v", b)
	}
	if styleOf(sp).Font.Family[0] != "Go" {
		t.Errorf("family from the agent's html: %v", s.Font.Family)
	}
}

func TestEmRemPercentAndCalc(t *testing.T) {
	inner := el(dom.Div, "in")
	outer := with(el(dom.Div, "out"), inner)
	root := page(outer)
	Compute(root, []*Sheet{sheet(t, `
html { font-size: 10px; }
#out { font-size: 2em; width: calc(100% - 2rem); }
#in { font-size: 150%; padding-left: 1em; margin-top: calc(50% + 1em - 4px); letter-spacing: 0.1em }
`)})
	if s := styleOf(outer); s.Font.Size != 20 ||
		s.Size.Width != (style.Length{Kind: style.Fixed, Px: -20, Percent: 100, IsPercent: true}) {
		t.Errorf("outer: %v %v", s.Font.Size, s.Size.Width)
	}
	s := styleOf(inner)
	if s.Font.Size != 30 || s.Padding.Left != style.Px(30) {
		t.Errorf("inner: %v %v", s.Font.Size, s.Padding.Left)
	}
	if s.Margin.Top != (style.Length{Kind: style.Fixed, Px: 26, Percent: 50, IsPercent: true}) {
		t.Errorf("calc: %v", s.Margin.Top)
	}
	if s.LetterGap() != 3 {
		t.Errorf("letter-spacing: %v", s.LetterGap())
	}
}

func TestVars(t *testing.T) {
	a := el(dom.Div, "a")
	b := el(dom.Div, "b")
	c := el(dom.Div, "c")
	root := page(with(el(dom.Div, "", "theme"), a, b, c))
	Compute(root, []*Sheet{sheet(t, `
:root { --ink: #f00; --pad: 4px; --both: var(--pad) var(--pad2, 8px); }
.theme { --ink: #0f0; }
#a { color: var(--ink); padding: var(--both); }
#b { --x: var(--y); --y: var(--x); color: var(--x, #00f); width: var(--missing); }
#c { --pad: 6px; margin: var(--pad) 0 0; border: var(--pad) solid var(--ink) }
`)})
	if s := styleOf(a); s.Color != hex("#0f0") || s.Padding.Top != style.Px(4) || s.Padding.Right != style.Px(8) {
		t.Errorf("a: %v %v %v", s.Color, s.Padding.Top, s.Padding.Right)
	}
	if s := styleOf(b); s.Color != hex("#00f") || s.Size.Width.Kind != style.Auto {
		t.Errorf("b, cycle and missing: %v %v", s.Color, s.Size.Width)
	}
	if s := styleOf(c); s.Margin.Top != style.Px(6) || s.Border.Left != 6 || s.BorderColor.Left != hex("#0f0") {
		t.Errorf("c, shorthand with var: %v %v %v", s.Margin.Top, s.Border.Left, s.BorderColor.Left)
	}
	if v := styleOf(a).Vars["--both"]; v != "4px 8px" {
		t.Errorf("--both is %q", v)
	}
}

func TestInvalidVarIsUnset(t *testing.T) {
	sp := dom.NewElement(dom.Span)
	root := page(with(el(dom.Div, "p"), sp))
	Compute(root, []*Sheet{sheet(t, `
#p { color: #0f0; --w: red; --c: 12px }
span { color: var(--c); width: var(--w); }
`)})
	if s := styleOf(sp); s.Color != hex("#0f0") || s.Size.Width.Kind != style.Auto {
		t.Errorf("unset: %v %v", s.Color, s.Size.Width)
	}
}

func TestAVarThatHoldsNothingLeavesTheDeclarationUnset(t *testing.T) {
	sp := dom.NewElement(dom.Span)
	root := page(with(el(dom.Div, "p"), sp))
	Compute(root, []*Sheet{sheet(t, `
#p { color: #0f0; --none: ; }
span { transform: var(--none); box-shadow: var(--none); color: var(--none); width: var(--none) }
`)})
	if s := styleOf(sp); len(s.Transform) != 0 || len(s.Shadows) != 0 || s.Color != hex("#0f0") ||
		s.Size.Width.Kind != style.Auto {
		t.Errorf("unset: %v %v %v %v", s.Transform, s.Shadows, s.Color, s.Size.Width)
	}
}

func TestStatesAndStructure(t *testing.T) {
	b1, b2 := dom.NewElement(dom.Button), dom.NewElement(dom.Button)
	root := page(with(el(dom.Nav, ""), b1, b2))
	sheets := []*Sheet{sheet(t, `
button { opacity: 0.5 }
button:hover:not(:disabled) { opacity: 1 }
button:first-child { order: 1 }
button:last-child { order: 2 }
`)}
	Compute(root, sheets)
	if styleOf(b1).Order != 1 || styleOf(b2).Order != 2 {
		t.Errorf("first/last: %v %v", styleOf(b1).Order, styleOf(b2).Order)
	}
	before := styleOf(b2)
	b1.SetState(dom.Hover, true)
	Compute(root, sheets)
	if styleOf(b1).Opacity != 1 {
		t.Errorf("hover: %v", styleOf(b1).Opacity)
	}
	if styleOf(b2) != before {
		t.Error("b2 restyled though nothing changed for it")
	}
	b1.SetState(dom.Disabled, true)
	Compute(root, sheets)
	if styleOf(b1).Opacity != 0.5 {
		t.Errorf("hover but disabled: %v", styleOf(b1).Opacity)
	}
}

func TestRestyleOnlyWhatChanged(t *testing.T) {
	a, b := el(dom.Div, "a"), el(dom.Div, "b")
	kid := dom.NewElement(dom.Span)
	txt := dom.NewText("hi")
	a.Append(kid)
	kid.Append(txt)
	root := page(a, b)
	sheets := []*Sheet{sheet(t, `.big { font-size: 30px } .big span { color: #f00 }`)}
	Compute(root, sheets)
	if dom.Data(txt).Style != styleOf(kid) {
		t.Error("text takes its parent's style")
	}
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Dirty = 0; return true })
	sb, skid := styleOf(b), styleOf(kid)

	a.AddClass("big")
	Compute(root, sheets)
	if styleOf(b) != sb {
		t.Error("sibling restyled")
	}
	if styleOf(kid) == skid || styleOf(kid).Font.Size != 30 || styleOf(kid).Color != hex("#f00") {
		t.Errorf("child not restyled: %v", styleOf(kid).Font.Size)
	}
	if dom.Data(txt).Style != styleOf(kid) {
		t.Error("text keeps its parent's style")
	}
	if dom.Data(kid).Dirty&dom.DirtyLayout == 0 || dom.Data(b).Dirty&dom.DirtyLayout != 0 {
		t.Error("DirtyLayout on what changed only")
	}
	root.Walk(func(n *dom.Node) bool {
		if dom.Data(n).Dirty&(dom.DirtyStyle|dom.DirtyTree) != 0 {
			t.Errorf("%s still dirty", describe(n))
		}
		return true
	})
}

func TestBlockification(t *testing.T) {
	sp, abs := dom.NewElement(dom.Span), dom.NewElement(dom.Span)
	root := page(with(el(dom.Div, "f"), sp), abs)
	Compute(root,
		[]*Sheet{sheet(t, `#f { display: flex } #f > span { display: inline-flex } span { position: absolute }`)})
	if styleOf(sp).Display != style.FlexDisplay || styleOf(abs).Display != style.Block {
		t.Errorf("%v %v", styleOf(sp).Display, styleOf(abs).Display)
	}
	if styleOf(root).Display != style.Block {
		t.Errorf("root: %v", styleOf(root).Display)
	}
}

func TestFloatsAreBlocksWhereTheyApply(t *testing.T) {
	left, item, abs, root := el(dom.Span, "left"), el(dom.Span, "item"), el(dom.Span, "abs"), el(dom.Div, "root")
	Compute(page(left, with(el(dom.Div, "flex"), item), abs, root), []*Sheet{sheet(t, `
#left { float: left; clear: both }
#flex { display: flex } #item { float: right }
#abs { float: inline-end; position: absolute; clear: inline-start }
#root { display: flow-root }`)})
	if s := styleOf(left); s.Float != style.FloatLeft || s.Clear != style.ClearBoth || s.Display != style.Block {
		t.Errorf("a floated span: float %v clear %v display %v", s.Float, s.Clear, s.Display)
	}
	if s := styleOf(item); s.Float != style.FloatNone {
		t.Errorf("a flex item floats %v", s.Float)
	}
	if s := styleOf(abs); s.Float != style.FloatNone || s.Clear != style.ClearLeft {
		t.Errorf("an absolute span: float %v clear %v", s.Float, s.Clear)
	}
	if s := styleOf(root); s.Display != style.Block || !s.FlowRoot {
		t.Errorf("flow-root: display %v, flow root %v", s.Display, s.FlowRoot)
	}
}

func TestTextIndentInherits(t *testing.T) {
	p, kid := el(dom.P, "p"), el(dom.Span, "kid")
	Compute(page(with(p, kid)), []*Sheet{sheet(t, `#p { font-size: 20px; text-indent: calc(2em - 10%) }`)})
	want := style.Length{Kind: style.Fixed, Px: 40, Percent: -10, IsPercent: true}
	if got := styleOf(p).TextIndent; got != want {
		t.Errorf("text-indent: %+v, want %+v", got, want)
	}
	if got := styleOf(kid).TextIndent; got != want {
		t.Errorf("inherited text-indent: %+v, want %+v", got, want)
	}
}

func TestComputedValues(t *testing.T) {
	d := el(dom.Div, "d")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `#d {
  display: grid; grid-template-columns: 10px minmax(0, 1fr) repeat(auto-fill, 20px);
  grid-auto-rows: 2em; grid-column: 2 / span 3; grid-auto-flow: column dense;
  gap: 6px 12px; flex: 2; overflow: hidden visible; aspect-ratio: 16 / 9;
  font-weight: bolder; font-style: italic; text-overflow: ellipsis; white-space: nowrap;
  box-shadow: 1px 2px 3px #000, inset 0 0 4px 1px; transform: translate(10px, 50%) rotate(0.5turn) scale(2);
  transition: opacity 200ms, color 1s ease-in 0.5s; z-index: 3; cursor: pointer; visibility: hidden;
  border-radius: 4px 50%; outline: 2px dashed #fff; border-image: url(p.png) 8 4 round repeat;
  font-family: "My Face", monospace;
}`)})
	s := styleOf(d)
	if len(s.Columns) != 3 || s.Columns[0].Tracks[0].Min != style.Px(10) ||
		s.Columns[1].Tracks[0] != (style.Track{Min: style.Px(0), Max: style.Length{Kind: style.Fr, Px: 1}}) ||
		s.Columns[2].Count != style.AutoFill {
		t.Errorf("columns: %+v", s.Columns)
	}
	if s.AutoRows[0].Min != style.Px(32) || s.Column != [2]style.GridLine{{Line: 2}, {Span: 3}} ||
		s.Flow != style.FlowColumnDense {
		t.Errorf("grid: %v %v %v", s.AutoRows, s.Column, s.Flow)
	}
	if s.Gap.Height != style.Px(6) || s.Gap.Width != style.Px(12) || s.Grow != 2 || s.Shrink != 1 || s.Basis != pct(0) {
		t.Errorf("flex: %v %v %v %v", s.Gap, s.Grow, s.Shrink, s.Basis)
	}
	if s.Overflow != (style.Size[style.Overflow]{Width: style.Hidden, Height: style.OverflowAuto}) {
		t.Errorf("overflow: %v", s.Overflow)
	}
	if s.AspectRatio != 16.0/9 || s.Font.Weight != 700 || !s.Font.Italic || !s.Ellipsis ||
		s.WhiteSpace != style.NoWrapSpace {
		t.Errorf("misc: %v %v %v %v %v", s.AspectRatio, s.Font.Weight, s.Font.Italic, s.Ellipsis, s.WhiteSpace)
	}
	lookOf(t, s)
}

// lookOf checks the rest of TestComputedValues' style: shadows,
// transforms, transitions and the box's look.
func lookOf(t *testing.T, s *style.Computed) {
	t.Helper()
	if len(s.Shadows) != 2 || s.Shadows[0].Blur != 3 || !s.Shadows[1].Inset || s.Shadows[1].Spread != 1 ||
		s.Shadows[1].Color != s.Color {
		t.Errorf("shadows: %+v", s.Shadows)
	}
	if len(s.Transform) != 3 || s.Transform[0].Y != pct(50) || s.Transform[1].A[0] < 3.14 || s.Transform[2].A[1] != 2 {
		t.Errorf("transform: %+v", s.Transform)
	}
	want := []style.Transition{{Property: "opacity", Duration: 0.2, Easing: "ease"},
		{Property: "color", Duration: 1, Delay: 0.5, Easing: "ease-in"}}
	if len(s.Transitions) != 2 || s.Transitions[0] != want[0] || s.Transitions[1] != want[1] {
		t.Errorf("transitions: %+v", s.Transitions)
	}
	if s.ZAuto || s.ZIndex != 3 || s.Cursor != "pointer" || s.Visible {
		t.Errorf("z %v %v cursor %q visible %v", s.ZAuto, s.ZIndex, s.Cursor, s.Visible)
	}
	if s.Radius.TopRight.Width != pct(50) || s.OutlineWidth != 2 || s.OutlineStyle != style.BorderDashed {
		t.Errorf("look: %v %v %v", s.Radius, s.OutlineWidth, s.OutlineStyle)
	}
	if s.BorderImage.Source != "p.png" ||
		s.BorderImage.Slice != (style.Sides[float32]{Top: 8, Right: 4, Bottom: 8, Left: 4}) {
		t.Errorf("border-image: %+v", s.BorderImage)
	}
	if s.BorderImageRepeat != [2]style.ImageRepeat{style.RepeatRound, style.RepeatTile} {
		t.Errorf("border-image-repeat: %v", s.BorderImageRepeat)
	}
	if len(s.Font.Family) != 2 || s.Font.Family[0] != "My Face" || s.Font.Family[1] != "monospace" {
		t.Errorf("family: %v", s.Font.Family)
	}
}

func TestInlineStyleWarnings(t *testing.T) {
	d := el(dom.Div, "d", "x")
	d.SetStyle("zoom: 2")
	warns := Compute(page(d), nil)
	if got := warnings(warns); got != "<div#d.x> style:1:1: zoom is not supported; use transform: scale()" {
		t.Errorf("got %s", got)
	}
}

func TestFontFaces(t *testing.T) {
	s, warns := Parse("ui/tool.css", `
@font-face { font-family: "Silk Screen"; src: url(fonts/Silk.ttf); font-weight: bold; font-style: italic }
@font-face { font-family: Pixel; src: url("Pixel.ttf") }
@font-face { src: url(x.ttf) }
`)
	if got := warnings(warns); got != "ui/tool.css:4:1: @font-face needs font-family and src; dropped" {
		t.Errorf("warnings: %s", got)
	}
	got := s.FontFaces()
	if len(got) != 2 {
		t.Fatalf("faces: %+v", got)
	}
	silk, pixel := got[0], got[1]
	if silk.Family != "Silk Screen" || silk.Src != "ui/fonts/Silk.ttf" || silk.Weight != 700 || !silk.Italic {
		t.Errorf("faces: %+v", silk)
	}
	if pixel.Family != "Pixel" || pixel.Src != "ui/Pixel.ttf" || pixel.Weight != 400 || pixel.Italic {
		t.Errorf("faces: %+v", pixel)
	}
}

func pct(v float32) style.Length { return style.Length{Kind: style.Fixed, Percent: v, IsPercent: true} }

func TestScrollbarWidth(t *testing.T) {
	a, b, c := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c")
	Compute(page(a, b, c), []*Sheet{sheet(t, `#a { overflow: auto } #b { overflow-y: scroll } #c { overflow: hidden }`)})
	if styleOf(a).ScrollbarWidth != 15 || styleOf(b).ScrollbarWidth != 15 || styleOf(c).ScrollbarWidth != 0 {
		t.Errorf("%v %v %v", styleOf(a).ScrollbarWidth, styleOf(b).ScrollbarWidth, styleOf(c).ScrollbarWidth)
	}
}

func TestOverlayScrollbarsLeaveNoGutter(t *testing.T) {
	a, b, c, none := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c"), el(dom.Div, "none")
	root := page(a, b, c, none)
	tr := NewTransitions()
	tr.OverlayScrollbars()
	tr.Compute(root, []*Sheet{sheet(t, `#a { overflow: auto } #b { overflow-y: scroll } #c { overflow: hidden }
		#none { overflow: auto; scrollbar-width: none }`)})
	for _, n := range []*dom.Node{a, b} {
		if c := styleOf(n); c.ScrollbarWidth != 0 || !c.ScrollbarOverlay {
			t.Errorf("#%s: gutter %v, overlay %v", n.ID(), c.ScrollbarWidth, c.ScrollbarOverlay)
		}
	}
	if styleOf(c).ScrollbarOverlay || styleOf(none).ScrollbarOverlay {
		t.Errorf("a box with no bars has none to overlay: hidden %v, none %v",
			styleOf(c).ScrollbarOverlay, styleOf(none).ScrollbarOverlay)
	}
}

func TestAScrollingViewportTakesTheRootsOverflowOrBodys(t *testing.T) {
	auto := style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	hidden := style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	for _, c := range []struct {
		name, css  string
		want       style.Size[style.Overflow]
		from       string
		gutter     float32
		divScrolls bool
	}{
		{"nothing set", ``, auto, "", 15, false},
		{"the root's", `html { overflow: hidden } body { overflow: scroll }`, hidden, "html", 0, false},
		{"body's while the root's is visible", `body { overflow-y: scroll }`,
			style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.Scroll}, "body", 15, false},
		{"clip is hidden there", `html { overflow: clip }`, hidden, "html", 0, false},
		{"no other element's", `div { overflow: auto }`, auto, "", 15, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := el(dom.Div, "d")
			root := page(d)
			body := root.Children()[0]
			tr := NewTransitions()
			tr.ScrollingViewport()
			tr.Compute(root, []*Sheet{sheet(t, c.css)})
			html, bd := styleOf(root), styleOf(body)
			v := html.Viewport
			if bd.OverflowPropagated {
				v = bd.Viewport
			}
			if v == nil || v.Overflow != c.want || v.ScrollbarWidth != c.gutter {
				t.Fatalf("the viewport is %+v, want %v with a gutter of %v", v, c.want, c.gutter)
			}
			if html.OverflowPropagated != (c.from == "html") || bd.OverflowPropagated != (c.from == "body") {
				t.Errorf("html gave its overflow: %v, body: %v, want %q", html.OverflowPropagated, bd.OverflowPropagated, c.from)
			}
			visible := style.Size[style.Overflow]{}
			if html.Overflow != visible || bd.Overflow != visible && c.from != "html" {
				t.Errorf("html keeps %v and body %v", html.Overflow, bd.Overflow)
			}
			if got := styleOf(d).Overflow != visible; got != c.divScrolls || styleOf(d).Viewport != nil {
				t.Errorf("the div scrolls: %v, its viewport %v", got, styleOf(d).Viewport)
			}
		})
	}
}

func TestWithoutAScrollingViewportTheRootKeepsItsOverflow(t *testing.T) {
	root := page()
	Compute(root, []*Sheet{sheet(t, `html { overflow: auto }`)})
	if c := styleOf(root); c.Viewport != nil || c.OverflowPropagated || c.Overflow.Height != style.OverflowAuto {
		t.Errorf("the root has %+v, viewport %v", c.Overflow, c.Viewport)
	}
}

func TestBodyTakesItsOverflowBackOnceTheRootGivesItsOwn(t *testing.T) {
	root := page()
	body := root.Children()[0]
	tr := NewTransitions()
	tr.ScrollingViewport()
	sheets := []*Sheet{sheet(t, `body { overflow: hidden } .own { overflow: scroll }`)}
	tr.Compute(root, sheets)
	if !styleOf(body).OverflowPropagated {
		t.Fatal("body's overflow stayed on body")
	}
	root.AddClass("own")
	tr.Compute(root, sheets)
	if b := styleOf(body); b.OverflowPropagated || b.Overflow.Height != style.Hidden || !styleOf(root).OverflowPropagated {
		t.Errorf("body has %+v, given away: %v", b.Overflow, b.OverflowPropagated)
	}
}

func TestAnOverlayViewportKeepsNoGutter(t *testing.T) {
	root := page()
	tr := NewTransitions()
	tr.OverlayScrollbars()
	tr.ScrollingViewport()
	tr.Compute(root, nil)
	if v := styleOf(root).Viewport; v.ScrollbarWidth != 0 || !v.ScrollbarOverlay {
		t.Errorf("gutter %v, overlay %v", v.ScrollbarWidth, v.ScrollbarOverlay)
	}
}

func TestZeroPercentIsAPercent(t *testing.T) {
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, `#d { flex-basis: 0%; width: 0px; margin-left: calc(0% + 4px) }`)})
	s := styleOf(d)
	if !s.Basis.IsPercent || s.Size.Width.IsPercent || !s.Margin.Left.IsPercent {
		t.Errorf("basis %+v width %+v margin %+v", s.Basis, s.Size.Width, s.Margin.Left)
	}
}

func TestBorderImageWidth(t *testing.T) {
	w := func(k style.ImageWidthKind, v float32) style.ImageWidth { return style.ImageWidth{Kind: k, V: v} }
	all := func(iw style.ImageWidth) style.Sides[style.ImageWidth] {
		return style.Sides[style.ImageWidth]{Top: iw, Right: iw, Bottom: iw, Left: iw}
	}
	a, b, c, d := el(dom.Div, "a"), el(dom.Div, "b"), el(dom.Div, "c"), el(dom.Div, "d")
	Compute(page(a, b, c, d), []*Sheet{sheet(t, `#b { border-image-width: 0 16px }
		#c { font-size: 10px; border-image-width: 2 25% auto 1em }
		#d { border-image: url(p.png) 8 / 4px }`)})
	for _, c := range []struct {
		n    *dom.Node
		want style.Sides[style.ImageWidth]
	}{
		{a, all(w(style.WidthTimes, 1))},
		{b, style.Sides[style.ImageWidth]{Top: w(style.WidthTimes, 0), Right: w(style.WidthLength, 16),
			Bottom: w(style.WidthTimes, 0), Left: w(style.WidthLength, 16)}},
		{c, style.Sides[style.ImageWidth]{Top: w(style.WidthTimes, 2), Right: w(style.WidthPercent, 25),
			Bottom: w(style.WidthAuto, 0), Left: w(style.WidthLength, 10)}},
		{d, all(w(style.WidthLength, 4))},
	} {
		if got := styleOf(c.n).BorderImageWidth; got != c.want {
			t.Errorf("%s: %+v, want %+v", c.n.ID(), got, c.want)
		}
	}
}

func TestOverflowWrap(t *testing.T) {
	a, b := el(dom.Div, "a"), el(dom.Div, "b")
	Compute(page(a, b), []*Sheet{sheet(t, `#a { overflow-wrap: break-word } #b { overflow-wrap: anywhere }`)})
	if s := styleOf(a); !s.BreakWord || s.BreakAnywhere {
		t.Errorf("break-word: %v %v", s.BreakWord, s.BreakAnywhere)
	}
	if s := styleOf(b); !s.BreakWord || !s.BreakAnywhere {
		t.Errorf("anywhere: %v %v", s.BreakWord, s.BreakAnywhere)
	}
}

func TestSwitchLook(t *testing.T) {
	sw, box := el(dom.Input, "sw"), el(dom.Input, "box")
	sw.SetAttr("type", "checkbox")
	sw.SetAttr("switch", "")
	box.SetAttr("type", "checkbox")
	root := page(sw, box)
	Compute(root, nil)
	if s := styleOf(sw); s.Size.Width != style.Px(32) || s.Background != hex("#efefef") || s.Border.Top != 1 ||
		s.Color != hex("#767676") {
		t.Errorf("switch off: %v %v %v %v", s.Size.Width, s.Background, s.Border.Top, s.Color)
	}
	sw.SetState(dom.Checked, true)
	box.SetState(dom.Checked, true)
	Compute(root, nil)
	if s := styleOf(sw); s.Background != hex("#0075ff") || s.Color != hex("#ffffff") {
		t.Errorf("switch on: %v %v", s.Background, s.Color)
	}
	if s := styleOf(box); s.Size.Width != style.Px(13) || s.Background != hex("#0075ff") {
		t.Errorf("checkbox on: %v %v", s.Size.Width, s.Background)
	}
	sw.SetAttr("disabled", "")
	box.SetAttr("disabled", "")
	sw.SetState(dom.Disabled, true)
	box.SetState(dom.Disabled, true)
	Compute(root, nil)
	if s := styleOf(sw); s.Background != hex("#d1d1d1") || s.Color != hex("#f8f8f8") {
		t.Errorf("switch on, disabled: %v %v", s.Background, s.Color)
	}
	if s := styleOf(box); s.Background != hex("#d1d1d1") || s.Color != hex("#ededed") {
		t.Errorf("checkbox on, disabled (native Chrome's): %v %v", s.Background, s.Color)
	}
}

func TestSummaryDisclosure(t *testing.T) {
	closed, open, plain := el(dom.Summary, "closed"), el(dom.Summary, "open"), el(dom.Summary, "plain")
	d1, d2, d3 := with(el(dom.Details, ""), closed), with(el(dom.Details, ""), open), with(el(dom.Details, ""), plain)
	d2.SetAttr("open", "")
	Compute(page(d1, d2, d3), []*Sheet{sheet(t, `#plain { list-style: none }`)})
	if s := styleOf(closed); s.ListStyle != style.ListDisclosureClosed || s.Display != style.Block {
		t.Errorf("closed: %v %v", s.ListStyle, s.Display)
	}
	if styleOf(open).ListStyle != style.ListDisclosureOpen || styleOf(plain).ListStyle != style.ListNone {
		t.Errorf("open %v plain %v", styleOf(open).ListStyle, styleOf(plain).ListStyle)
	}
}

func TestPlaceholder(t *testing.T) {
	in, area, div := el(dom.Input, "in"), el(dom.Textarea, "area"), el(dom.Div, "d")
	in.SetAttr("placeholder", "Name")
	area.SetAttr("placeholder", "Notes")
	Compute(page(in, area, div), []*Sheet{sheet(t, `
#area { color: #123456; --ph: #abcdef; }
textarea::placeholder { color: var(--ph); }
.form ::placeholder { color: currentColor; }
`)})
	if s := styleOf(in); !s.HasPlaceholder || s.PlaceholderColor != hex("#757575") {
		t.Errorf("UA placeholder: %v %v", s.HasPlaceholder, s.PlaceholderColor)
	}
	if s := styleOf(area); s.PlaceholderColor != hex("#abcdef") {
		t.Errorf("author placeholder with var: %v", s.PlaceholderColor)
	}
	if styleOf(in).Color == styleOf(in).PlaceholderColor {
		t.Error("the element's own colour is untouched")
	}
	if styleOf(div).HasPlaceholder {
		t.Error("an element without a placeholder attribute shows none to style")
	}
}

func TestAPseudoElementsRuleSaysWhatItDoesNotTake(t *testing.T) {
	_, warns := Parse("t.css",
		"input::placeholder { color: #f00; width: 5px } p, input::placeholder { color: #f00; width: 5px }")
	if got, want := warnings(warns), "t.css:1:35: ::placeholder does not take width"; got != want {
		t.Errorf("warnings:\n got %s\nwant %s", got, want)
	}
}

func TestBackgroundPosition(t *testing.T) {
	cases := map[string]style.BackgroundPosition{
		"right 6px center": {X: style.Px(6), Y: pct(50), FromRight: true},
		"center":           {X: pct(50), Y: pct(50)},
		"top":              {X: pct(50), Y: pct(0)},
		"10px 20%":         {X: style.Px(10), Y: pct(20)},
		"bottom 4px left":  {X: pct(0), Y: style.Px(4), FromBottom: true},
		"left 10% bottom":  {X: pct(10), Y: pct(100)},
	}
	for src, want := range cases {
		d := el(dom.Div, "d")
		Compute(page(d), []*Sheet{sheet(t, "#d { background-position: "+src+" }")})
		if got := styleOf(d).BackgroundLayer(0).Position; got != want {
			t.Errorf("%s: got %+v, want %+v", src, got, want)
		}
	}
	for _, bad := range []string{"left right", "center 4px left", "1px 2px 3px", "top top"} {
		if err := validate("background-position", componentsOf(bad)); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestSelectArrowAndRelativeURLs(t *testing.T) {
	s, b := el(dom.Select, "s"), el(dom.Div, "b")
	s2, _ := Parse("ui/theme/tool.css", "#b { background: url(img/panel.png) no-repeat }")
	Compute(page(s, b), []*Sheet{s2})
	l := styleOf(s).BackgroundLayer(0)
	if l.Image.Source != "ember:select-arrow" || l.Repeat != style.NoRepeat || !l.Position.FromRight {
		t.Errorf("select: %+v %v %+v", l.Image, l.Repeat, l.Position)
	}
	if _, ok := DefaultImages()["ember:select-arrow"]; !ok {
		t.Error("the arrow's PNG")
	}
	if l = styleOf(b).BackgroundLayer(0); l.Image.Source != "ui/theme/img/panel.png" || l.Repeat != style.NoRepeat {
		t.Errorf("relative to the sheet: %q", l.Image.Source)
	}
}

func TestObjectFitAndPosition(t *testing.T) {
	a, b := el(dom.Img, "a"), el(dom.Img, "b")
	Compute(page(a, b),
		[]*Sheet{sheet(t, `#a { object-fit: cover; object-position: left top } #b { object-fit: scale-down }`)})
	if s := styleOf(a); s.ObjectFit != style.FitCover ||
		s.ObjectPosition != (style.BackgroundPosition{X: pct(0), Y: pct(0)}) {
		t.Errorf("a: %v %+v", s.ObjectFit, s.ObjectPosition)
	}
	if s := styleOf(b); s.ObjectFit != style.FitScaleDown ||
		s.ObjectPosition != (style.BackgroundPosition{X: pct(50), Y: pct(50)}) {
		t.Errorf("b, initial position: %v %+v", s.ObjectFit, s.ObjectPosition)
	}
}

func TestTextShadowRenderingAccentCaretSelection(t *testing.T) {
	a := el(dom.Div, "a")
	kid := el(dom.Span, "kid")
	a.Append(kid)
	sel := el(dom.P, "sel")
	Compute(page(a, sel), []*Sheet{sheet(t, `
#a { color: #f00; text-shadow: 1px 2px 3px #000, 0 0 4px; image-rendering: pixelated; caret-color: #00f; }
#kid { color: #0f0; }
#sel::selection { background-color: rgba(255, 207, 118, 0.4); }
`)})
	s := styleOf(a)
	if len(s.TextShadows) != 2 || s.TextShadows[0].Blur != 3 || s.TextShadows[1].Color != hex("#f00") {
		t.Errorf("text-shadow: %+v", s.TextShadows)
	}
	if s.ImageRendering != style.RenderPixelated || !s.AccentAuto || s.AccentColor != hex("#f00") || s.CaretAuto ||
		s.CaretColor != hex("#00f") {
		t.Errorf("a: %v %v %v %v %v", s.ImageRendering, s.AccentAuto, s.AccentColor, s.CaretAuto, s.CaretColor)
	}
	k := styleOf(kid)
	if k.ImageRendering != style.RenderPixelated || len(k.TextShadows) != 2 {
		t.Errorf("inherited: %v %v", k.ImageRendering, k.TextShadows)
	}
	if !k.AccentAuto || k.AccentColor != hex("#0f0") {
		t.Errorf("an inherited auto follows the child's own color: %v", k.AccentColor)
	}
	if k.CaretColor != hex("#00f") {
		t.Errorf("an inherited colour stays: %v", k.CaretColor)
	}
	if p := styleOf(sel); !p.HasSelection || p.SelectionColor.A < 0.39 || p.SelectionColor.A > 0.41 {
		t.Errorf("::selection: %v %v", p.HasSelection, p.SelectionColor)
	}
	if styleOf(a).HasSelection {
		t.Error("no ::selection rule for #a")
	}
	_, warns := Parse("t.css", "p::selection { color: red } p::selection, p::placeholder { font-size: 2px }")
	if got, want := warnings(warns), "t.css:1:60: ::selection does not take font-size"; got != want {
		t.Errorf("warnings:\n got %s\nwant %s", got, want)
	}
}

// A ::selection takes color and background-color, each inherited from the
// parent's ::selection on its own, as Chrome's highlight pseudo-elements.
func TestSelectionTakesColorAndInheritsFromTheParentsSelection(t *testing.T) {
	kid, deep, own := el(dom.B, "kid"), el(dom.I, "deep"), el(dom.Span, "own")
	p, q := with(el(dom.P, "p"), with(kid, deep), own), el(dom.P, "q")
	Compute(page(p, q), []*Sheet{sheet(t, `
#p::selection { background-color: #f00; color: #0f0; }
#own::selection { color: #00f; }
`)})
	for _, n := range []*dom.Node{p, kid, deep} {
		if s := styleOf(n); !s.HasSelection || s.SelectionColor != hex("#f00") || !s.HasSelectionText ||
			s.SelectionText != hex("#0f0") {
			t.Errorf("%s: %v %v %v %v", n.ID(), s.HasSelection, s.SelectionColor, s.HasSelectionText, s.SelectionText)
		}
	}
	if s := styleOf(own); s.SelectionColor != hex("#f00") || s.SelectionText != hex("#00f") {
		t.Errorf("own: %v %v", s.SelectionColor, s.SelectionText)
	}
	if s := styleOf(q); s.HasSelection || s.HasSelectionText {
		t.Error("no ::selection rule for #q")
	}
}

// user-select is used as Chrome resolves it: auto is the parent's, text on
// the root; ember's own sheet starts the root at none, a browser's at auto.
func TestUserSelectIsInheritedAndAutoIsText(t *testing.T) {
	text, none, all := el(dom.Div, "text"), el(dom.Div, "none"), el(dom.Div, "all")
	inText, inNone, inAll, back := el(dom.Span, "in-text"), el(dom.Span, "in-none"), el(dom.Span, "in-all"),
		el(dom.Span, "back")
	plain := el(dom.P, "plain")
	root := page(plain, with(text, inText), with(none, inNone, back), with(all, inAll))
	rules := sheet(t, `#text, #back { user-select: text } #none { user-select: none } #all { user-select: all }
#in-all { user-select: auto }`)
	want := map[*dom.Node]style.UserSelect{
		text: style.SelectText, inText: style.SelectText, none: style.SelectNone, inNone: style.SelectNone,
		back: style.SelectText, all: style.SelectAll, inAll: style.SelectText,
	}
	check := func(name string, rest style.UserSelect) {
		t.Helper()
		for n, w := range want {
			if got := styleOf(n).UserSelect; got != w {
				t.Errorf("%s: #%s is %v, want %v", name, n.ID(), got, w)
			}
		}
		for _, n := range []*dom.Node{root, plain} {
			if got := styleOf(n).UserSelect; got != rest {
				t.Errorf("%s: %s is %v, want %v", name, describe(n), got, rest)
			}
		}
	}
	Compute(root, []*Sheet{rules})
	check("ember's sheet", style.SelectNone)
	browser := NewTransitions()
	browser.BrowserSheets()
	dom.Data(root).Dirty |= dom.DirtyStyle
	browser.Compute(root, []*Sheet{rules})
	check("a browser's sheet", style.SelectText)
}

func TestUserSelectReadsChromesValues(t *testing.T) {
	_, warns := Parse("t.css", "p { user-select: text; user-select: none; user-select: all; user-select: auto }")
	if len(warns) != 0 {
		t.Errorf("warnings: %s", warnings(warns))
	}
	_, warns = Parse("t.css", "p { user-select: contain }")
	if len(warns) != 1 {
		t.Errorf("contain, which Chrome refuses, warns once: %s", warnings(warns))
	}
	s, warns := Parse("t.css", "p { -webkit-user-select: none }", Browser())
	if len(warns) != 0 || len(s.Rules) != 1 || s.Rules[0].Decls[0].Name != "user-select" {
		t.Errorf("-webkit-user-select: %s", warnings(warns))
	}
}

func TestListItemIsBlockWithAMarkerFlag(t *testing.T) {
	a, b := el(dom.Summary, "a"), el(dom.Summary, "b")
	Compute(page(with(el(dom.Details, ""), a), with(el(dom.Details, ""), b)), []*Sheet{sheet(t, `#b { display: block }`)})
	if s := styleOf(a); s.Display != style.Block || !s.ListItem {
		t.Errorf("UA summary: %v %v", s.Display, s.ListItem)
	}
	if s := styleOf(b); s.Display != style.Block || s.ListItem {
		t.Errorf("display: block: %v %v", s.Display, s.ListItem)
	}
}

func TestTruncateStartCutsTheStart(t *testing.T) {
	cut, plain := el(dom.P, "", "truncate-start"), el(dom.P, "")
	Compute(page(cut, plain), nil)
	if !styleOf(cut).EllipsisStart || styleOf(plain).EllipsisStart {
		t.Errorf("class truncate-start: %v, without: %v", styleOf(cut).EllipsisStart, styleOf(plain).EllipsisStart)
	}
}

func TestScrollbarWidthNoneLeavesNoGutter(t *testing.T) {
	none, thin, auto := el(dom.Div, "", "none"), el(dom.Div, "", "thin"), el(dom.Div, "", "auto")
	s, warns := Parse("t.css", `div { overflow: auto; } .none { scrollbar-width: none; } .thin { scrollbar-width: thin; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(none, thin, auto), []*Sheet{s})
	if c := styleOf(none); c.ScrollbarWidth != 0 || !c.ScrollbarNone {
		t.Errorf("none: gutter %v, none %v", c.ScrollbarWidth, c.ScrollbarNone)
	}
	if c := styleOf(thin); c.ScrollbarWidth != 0 || !c.ScrollbarOverlay {
		t.Errorf("thin asks for the system's overlay bars: gutter %v, overlay %v", c.ScrollbarWidth, c.ScrollbarOverlay)
	}
	if styleOf(auto).ScrollbarWidth != 15 {
		t.Errorf("auto keeps the 15-point gutter: %v", styleOf(auto).ScrollbarWidth)
	}
	if !strings.Contains(s.String(), "scrollbar-width: none") {
		t.Errorf("it travels to Chrome: %s", s.String())
	}
}

func TestBorderSpacing(t *testing.T) {
	plain, tight, two := el(dom.Table, ""), el(dom.Table, "", "tight"), el(dom.Table, "", "two")
	cell := el(dom.Td, "")
	tight.Append(cell)
	s, warns := Parse("t.css", `.tight { border-spacing: 0; } .two { border-spacing: 4px 6px; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(plain, tight, two), []*Sheet{s})
	for n, want := range map[*dom.Node][2]float32{plain: {2, 2}, tight: {0, 0}, two: {4, 6}, cell: {0, 0}} {
		if got := styleOf(n).BorderSpacing; got != want {
			t.Errorf("%s: %v, want %v", describe(n), got, want)
		}
	}
}

func TestCaptionSideInheritsAndTableLayoutDoesNot(t *testing.T) {
	plain, below := el(dom.Table, ""), el(dom.Table, "", "below")
	caption := el(dom.Caption, "")
	below.Append(caption)
	s, warns := Parse("t.css", `.below { caption-side: bottom; table-layout: fixed; }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(plain, below), []*Sheet{s})
	if got := styleOf(plain); got.CaptionSide != style.CaptionTop || got.TableLayout != style.TableAuto {
		t.Errorf("top and auto at first: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(below); got.CaptionSide != style.CaptionBottom || got.TableLayout != style.TableFixed {
		t.Errorf("the table's own: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(caption); got.CaptionSide != style.CaptionBottom || got.TableLayout != style.TableAuto {
		t.Errorf("its caption inherits caption-side alone: %v %v", got.CaptionSide, got.TableLayout)
	}
	if got := styleOf(caption).TextAlign; got != style.Center {
		t.Errorf("a caption is centred: %v", got)
	}
}

func TestTextTransform(t *testing.T) {
	up, low, capd := el(dom.P, "", "up"), el(dom.P, "", "low"), el(dom.P, "", "cap")
	none, kid := el(dom.P, ""), el(dom.Span, "")
	up.Append(kid)
	s, warns := Parse("t.css",
		`.up { text-transform: uppercase } .low { text-transform: lowercase } .cap { text-transform: capitalize }`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	Compute(page(up, low, capd, none), []*Sheet{s})
	for n, want := range map[*dom.Node]style.TextTransform{up: style.Uppercase, low: style.Lowercase,
		capd: style.Capitalize, none: style.NoTransform, kid: style.Uppercase} {
		if got := styleOf(n).TextTransform; got != want {
			t.Errorf("%s: %v, want %v", describe(n), got, want)
		}
	}
}

func TestBackgroundSize(t *testing.T) {
	auto := style.Length{}
	cases := map[string]style.BackgroundSize{
		"8px 4px":   {Width: style.Px(8), Height: style.Px(4)},
		"1em":       {Width: style.Px(16), Height: auto},
		"auto":      {},
		"auto auto": {},
		"50%":       {Width: pct(50), Height: auto},
		"auto 25%":  {Width: auto, Height: pct(25)},
		"100% 12px": {Width: pct(100), Height: style.Px(12)},
		"calc(50% - 4px) 0": {
			Width:  style.Length{Kind: style.Fixed, Px: -4, Percent: 50, IsPercent: true},
			Height: style.Px(0),
		},
		"cover":   {Scale: style.ScaleCover},
		"contain": {Scale: style.ScaleContain},
		"COVER":   {Scale: style.ScaleCover},
	}
	for src, want := range cases {
		d := el(dom.Div, "d")
		Compute(page(d), []*Sheet{sheet(t, "#d { background-size: "+src+" }")})
		if got := styleOf(d).BackgroundLayer(0).Size; got != want {
			t.Errorf("%s: got %+v, want %+v", src, got, want)
		}
	}
	plain, kid := el(dom.Div, "p"), el(dom.Div, "k")
	plain.Append(kid)
	Compute(page(plain), []*Sheet{sheet(t, "#p { background-size: cover } #k { background-size: inherit }")})
	if got := styleOf(kid).BackgroundLayer(0).Size; got != (style.BackgroundSize{Scale: style.ScaleCover}) {
		t.Errorf("inherit: got %+v", got)
	}
	for _, bad := range []string{"cover contain", "cover 10px", "10px cover", "-1px", "10px -5%", "1px 2px 3px", "fill",
		"12", "auto auto auto", "1fr"} {
		if err := validate("background-size", componentsOf(bad)); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestTheBackgroundShorthandSetsAndResetsTheSize(t *testing.T) {
	cover, reset := el(dom.Div, "c"), el(dom.Div, "r")
	Compute(page(cover, reset), []*Sheet{sheet(t, `#c { background: url(a.png) no-repeat center / cover }
#r { background-size: 10px; background: url(a.png) }`)})
	l := styleOf(cover).BackgroundLayer(0)
	if l.Size != (style.BackgroundSize{Scale: style.ScaleCover}) || l.Repeat != style.NoRepeat ||
		l.Position != (style.BackgroundPosition{X: pct(50), Y: pct(50)}) {
		t.Errorf("center / cover: %+v %v %+v", l.Size, l.Repeat, l.Position)
	}
	if got := styleOf(reset).BackgroundLayer(0).Size; got != (style.BackgroundSize{}) {
		t.Errorf("the shorthand resets background-size: got %+v", got)
	}
}

func TestAnAttributeChangeRestylesTheWholeSubtreeUnderIt(t *testing.T) {
	deep := el(dom.Span, "deep")
	mid := el(dom.Div, "mid")
	mid.Append(deep)
	root := page(mid)
	sheets := []*Sheet{sheet(t, `[data-mode="dark"] #deep { color: #ff0000 }`)}
	Compute(root, sheets)
	if styleOf(deep).Color == color.Hex("#ff0000") {
		t.Fatal("red before the attribute")
	}
	root.SetAttr("data-mode", "dark")
	Compute(root, sheets)
	if styleOf(deep).Color != color.Hex("#ff0000") {
		t.Error("two levels under the change, the span restyles")
	}
}

func TestARadioIsChromesRoundThirteen(t *testing.T) {
	radio := el(dom.Input, "r")
	radio.SetAttr("type", "radio")
	root := page(radio)
	Compute(root, nil)
	if s := styleOf(radio); s.Size.Width != style.Px(13) || s.Size.Height != style.Px(13) ||
		s.BorderColor.Top != hex("#767676") {
		t.Errorf("radio: %v %v %v", s.Size.Width, s.Size.Height, s.BorderColor.Top)
	}
	radio.SetState(dom.Checked, true)
	Compute(root, nil)
	if s := styleOf(radio); s.BorderColor.Top != hex("#0075ff") || s.Color != hex("#0075ff") {
		t.Errorf("checked: %v %v", s.BorderColor.Top, s.Color)
	}
	radio.SetAttr("disabled", "")
	radio.SetState(dom.Disabled, true)
	Compute(root, nil)
	if s := styleOf(radio); s.Color != hex("#d1d1d1") || s.Background != hex("#f8f8f8") {
		t.Errorf("checked, disabled (native Chrome's): %v %v", s.Color, s.Background)
	}
}

func TestAStateChangeRestylesEveryDescendant(t *testing.T) {
	for _, c := range []struct {
		name  string
		state dom.State
	}{{"focus", dom.Focus}, {"hover", dom.Hover}} {
		t.Run(c.name, func(t *testing.T) {
			child, grand, deep := el(dom.Span, "", "child"), el(dom.Span, "", "grand"), el(dom.Span, "", "deep")
			lb := with(el(dom.Div, "", "lb"), child, with(el(dom.Div, "", "b"), grand, with(el(dom.Div, ""), deep)))
			root := page(lb)
			sheets := []*Sheet{sheet(t, `
.child, .grand, .deep { display: none }
.lb:`+c.name+` > .child { display: block }
.lb:`+c.name+` > .b > .grand { display: block }
.lb:`+c.name+` .deep { display: block }
`)}
			Compute(root, sheets)
			shown := func() [3]bool {
				return [3]bool{
					styleOf(child).Display == style.Block,
					styleOf(grand).Display == style.Block,
					styleOf(deep).Display == style.Block,
				}
			}
			if got := shown(); got != [3]bool{} {
				t.Fatalf("before: %v", got)
			}
			lb.SetState(c.state, true)
			Compute(root, sheets)
			if got := shown(); got != [3]bool{true, true, true} {
				t.Errorf("on: child, grandchild, descendant shown %v", got)
			}
			lb.SetState(c.state, false)
			Compute(root, sheets)
			if got := shown(); got != [3]bool{} {
				t.Errorf("off: %v", got)
			}
		})
	}
}

func TestAnImgsWidthAndHeightAttributesSizeItUnderEveryRule(t *testing.T) {
	a, b, c, d := el(dom.Img, "a"), el(dom.Img, "b"), el(dom.Img, "c"), el(dom.Div, "d")
	a.SetAttr("width", "100")
	a.SetAttr("height", "30")
	b.SetAttr("width", "50%")
	b.SetAttr("height", "x")
	c.SetAttr("width", "100")
	c.SetAttr("height", " 12.5px")
	d.SetAttr("width", "100")
	Compute(page(a, b, c, d), []*Sheet{sheet(t, `#b, #c { height: 7px } * { min-width: 0 } #c { width: 40px }`)})
	if s := styleOf(a).Size; s.Width != style.Px(100) || s.Height != style.Px(30) {
		t.Errorf("a: %+v", s)
	}
	if s := styleOf(b).Size; s.Width != pct(50) || s.Height != style.Px(7) {
		t.Errorf("b, a percentage, and a height that is no number left to the sheet: %+v", s)
	}
	if s := styleOf(c).Size; s.Width != style.Px(40) || s.Height != style.Px(7) {
		t.Errorf("c, any rule wins over an attribute: %+v", s)
	}
	if s := styleOf(d).Size; s.Width.Kind != style.Auto {
		t.Errorf("d, an img's attributes only: %+v", s)
	}
}

func TestAnImgsWidthAndHeightAttributesSizeItWithoutAnySheet(t *testing.T) {
	a := el(dom.Img, "a")
	a.SetAttr("width", "100")
	a.SetAttr("height", "30")
	Compute(page(a), nil)
	if s := styleOf(a).Size; s.Width != style.Px(100) || s.Height != style.Px(30) {
		t.Errorf("a: %+v", s)
	}
}

func TestEveryLonghandIsAppliedFromItsOwnTables(t *testing.T) {
	if len(longhands) != len(order) {
		t.Fatalf("%d longhands for %d in order", len(longhands), len(order))
	}
	same := func(a, b any) bool { return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() }
	for i, l := range longhands {
		name := order[i]
		p, known := props[name]
		switch {
		case l.name != name:
			t.Fatalf("longhand %d is %s, not %s", i, l.name, name)
		case !known || !same(p.set, l.prop.set) || !same(p.inherit, l.prop.inherit):
			t.Errorf("%s is not set and inherited as its property", name)
		case l.inherited != properties[name].inherited:
			t.Errorf("%s inherited: %v", name, l.inherited)
		case l.current != (properties[name].initial == kwCurrentColor):
			t.Errorf("%s starts as currentcolor: %v", name, l.current)
		}
	}
}

func TestABorderStyleAloneIsAMediumBorder(t *testing.T) {
	styled, plain, side := el(dom.Div, "styled"), el(dom.Div, "plain"), el(dom.Div, "side")
	Compute(page(styled, plain, side),
		[]*Sheet{sheet(t, "#styled { border-style: solid } #side { border-left-style: dashed }")})
	if got, want := styleOf(styled).Border, (style.Sides[float32]{Top: 3, Right: 3, Bottom: 3, Left: 3}); got != want {
		t.Errorf("a style and no width: %v, want %v", got, want)
	}
	if got := styleOf(plain).Border; got != (style.Sides[float32]{}) {
		t.Errorf("no style: %v, want none", got)
	}
	if got, want := styleOf(side).Border, (style.Sides[float32]{Left: 3}); got != want {
		t.Errorf("one side's style: %v, want %v", got, want)
	}
}

func TestABorderWidthInheritsWhatBorderStyleNoneLeavesUnused(t *testing.T) {
	child, parent := el(dom.P, "child"), el(dom.Div, "parent")
	parent.Append(child)
	Compute(page(parent), []*Sheet{sheet(t,
		"#parent { border-width: 2em; font-size: 10px } #child { border-width: inherit; border-style: solid }")})
	if got := styleOf(parent).Border; got != (style.Sides[float32]{}) {
		t.Errorf("no style draws no border: %v", got)
	}
	if got, want := styleOf(child).Border, (style.Sides[float32]{Top: 20, Right: 20, Bottom: 20, Left: 20}); got != want {
		t.Errorf("inherited: %v, want the parent's computed %v, as Chrome", got, want)
	}
}

func TestOutlineStyleAutoIsTheFocusRing(t *testing.T) {
	d := el(dom.Div, "d")
	root := page(d)
	Compute(root, []*Sheet{sheet(t, `#d { outline: auto 3px; }`)})
	s := styleOf(d)
	if s.OutlineStyle != style.BorderAuto || s.OutlineWidth != 3 {
		t.Errorf("outline: %v %v", s.OutlineStyle, s.OutlineWidth)
	}
}
