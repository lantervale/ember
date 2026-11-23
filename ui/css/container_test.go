package css //nolint:testpackage // reads computed styles off dom and the package's unexported test helpers

import (
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// boxes stands for a layout: the content box of each container laid
// out, the others having none.
type boxes map[*dom.Node][2]float32

func (b boxes) size(n *dom.Node) (float32, float32, bool) {
	s, ok := b[n]
	return s[0], s[1], ok
}

// requery answers the containers for b, then restyles what that changed,
// and reports whether it changed something.
func requery(tr *Transitions, root *dom.Node, sheets []*Sheet, b boxes) bool {
	changed := tr.Requery(b.size)
	tr.Compute(root, sheets)
	return changed
}

func TestContainerTypeAndNameAreLonghandsOfContainer(t *testing.T) {
	ids := []string{"size", "inline", "named", "short", "shortTyped", "none", "upper"}
	var kids []*dom.Node
	for _, id := range ids {
		kids = append(kids, el(dom.Div, id))
	}
	Compute(page(kids...), []*Sheet{sheet(t, `
#size { container-type: size; }
#inline { container-type: inline-size; }
#named { container-name: card  side; }
#short { container: card; }
#shortTyped { container: a b / inline-size; }
#none { container: none / size; }
#upper { container-name: Card; }
`)})
	for i, want := range []struct {
		kind  style.ContainerType
		names []string
	}{
		{style.ContainerSize, nil}, {style.ContainerInlineSize, nil}, {style.ContainerNormal, []string{"card", "side"}},
		{style.ContainerNormal, []string{"card"}}, {style.ContainerInlineSize, []string{"a", "b"}},
		{style.ContainerSize, nil}, {style.ContainerNormal, []string{"Card"}},
	} {
		s := styleOf(kids[i])
		if s.ContainerType != want.kind || !slices.Equal(s.ContainerNames, want.names) {
			t.Errorf("%s: %v %q, want %v %q", ids[i], s.ContainerType, s.ContainerNames, want.kind, want.names)
		}
	}
}

func TestWhatContainerCannotTakeIsDropped(t *testing.T) {
	for _, decl := range []string{"container-type: size inline-size", "container-type: scroll-state scroll-state",
		"container-type: normal scroll-state", "container-name: a none", "container-name: and",
		"container-name: not", "container-name: or", "container-name: default", "container-name: 1px",
		"container: / size", "container: a / size / size", "container: a / block"} {
		if _, warns := Parse("t.css", "p { "+decl+" }"); len(warns) != 1 {
			t.Errorf("%s: %s", decl, warnings(warns))
		}
	}
}

func TestContainerRulesAreAnsweredForTheContainersBox(t *testing.T) {
	outer, mid := el(dom.Div, "outer"), el(dom.Div, "mid")
	var kids []*dom.Node
	for i := range 16 {
		kids = append(kids, el(dom.P, "", "n"+string(rune('a'+i))))
	}
	root := page(with(outer, with(mid, kids...)))
	sheets := []*Sheet{sheet(t, `
#outer { container: outer / size; font-size: 10px; --a: x; }
#mid { container-type: inline-size; }
p { width: 1px; }
@container card { .na { width: 11px; } }
@container (width > 100px), (height > 100px) { .nb { width: 12px; } }
@container (height > 250px) { .nc { width: 13px; } }
@container outer (width > 400px) { .nd { width: 14px; } }
@container not (width > 300px) { .ne { width: 15px; } }
@container (min-width: 150px) and (max-width: 250px) { .nf { width: 16px; } }
@container (inline-size > 150px) and (block-size > 0) { .ng { width: 17px; } }
@container (orientation: landscape) { .nh { width: 18px; } }
@container OUTER (width > 0px) { .ni { width: 19px; } }
@container (width > 10ch) and (width < 21em) { .nj { width: 20px; } }
@container (width > calc(100px + 1px)) { .nk { width: 21px; } }
@container (width) { .nl { width: 22px; } }
@container outer { .nm { width: 23px; } }
@container (aspect-ratio: 5/3) { .nn { width: 24px; } }
@container (width > 10cqw) { .no { width: 25px; } }
@container (width > 200px) { @container (height = 300px) { .np { width: 26px; } } }
`)}
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 800, Height: 600})
	tr.Compute(root, sheets)
	for _, k := range kids {
		want := float32(1)
		if k.HasClass("nm") {
			want = 23
		}
		if w := styleOf(k).Size.Width.Px; w != want {
			t.Errorf("%s: %gpx before any layout, where no container has a size yet", k.Classes()[0], w)
		}
	}
	if !requery(tr, root, sheets, boxes{outer: {500, 300}, mid: {200, 18}}) {
		t.Fatal("the containers laid out change what the rules answer")
	}
	want := []float32{1, 12, 13, 14, 15, 16, 17, 18, 1, 20, 21, 22, 23, 24, 25, 1}
	for i, k := range kids {
		if w := styleOf(k).Size.Width.Px; w != want[i] {
			t.Errorf("%s: %gpx, want %g", k.Classes()[0], w, want[i])
		}
	}
	if requery(tr, root, sheets, boxes{outer: {500, 300}, mid: {200, 18}}) {
		t.Error("the same boxes change nothing")
	}
}

func TestStyleQueriesReadTheContainersCustomProperties(t *testing.T) {
	var kids []*dom.Node
	for i := range 9 {
		kids = append(kids, el(dom.P, "", "s"+string(rune('a'+i))))
	}
	card := el(dom.Div, "card")
	card.SetStyle("container-name: card; --a: x; --c: 1; --d: 1px 2px")
	sized := el(dom.Div, "sized")
	sized.SetStyle("container-type: inline-size; --a: x")
	in := el(dom.P, "", "si")
	root := page(with(card, kids...), with(sized, in))
	sheets := []*Sheet{sheet(t, `
p { width: 1px; }
@container card { .sa { width: 11px; } }
@container style(--a) { .sb { width: 12px; } }
@container style(--b) { .sc { width: 13px; } }
@container card style(--a: x) { .sd { width: 14px; } }
@container style((--a: x) and (--c: 1)) { .se { width: 15px; } }
@container style(not (--a: y)) { .sf { width: 16px; } }
@container style(--a:x) { .sg { width: 17px; } }
@container style(--d: 1px 2px) { .sh { width: 18px; } }
@container other style(--a) { .si { width: 19px; } }
@container (width > 100px) and style(--a: x) { .si { width: 20px; } }
`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	requery(tr, root, sheets, boxes{sized: {300, 18}})
	want := []float32{11, 12, 1, 14, 15, 16, 17, 18, 1}
	for i, k := range kids {
		if w := styleOf(k).Size.Width.Px; w != want[i] {
			t.Errorf("%s: %gpx, want %g", k.Classes()[0], w, want[i])
		}
	}
	if w := styleOf(in).Size.Width.Px; w != 20 {
		t.Errorf("a size feature and a style feature ask one container: %gpx", w)
	}
}

func TestWhatAContainerQueryCannotAnswerIsAWarning(t *testing.T) {
	for prelude, want := range map[string]string{
		"(unknown-thing > 1px)":        "@container: the unknown-thing feature is not supported in a range",
		"scroll-state(scrolled: up)":   "@container: scrolled:up is no scroll-state feature ui answers",
		"style(color: red)":            "@container: style() reads custom properties alone",
		"(width > 10%)":                "@container: width can't take 10%",
		"(prefers-color-scheme: dark)": "@container: the prefers-color-scheme feature is not supported",
	} {
		_, warns := Parse("t.css", "@container "+prelude+" { p { color: #f00 } }")
		if got := warnings(warns); !strings.Contains(got, want) {
			t.Errorf("@container %s: %q, want %q", prelude, got, want)
		}
	}
	for _, prelude := range []string{"none (width > 0)", "and (width > 0)", "", "card card", "(width > 0) card",
		"not card"} {
		s, warns := Parse("t.css", "@container "+prelude+" { p { color: #f00 } } a { color: #0f0 }")
		if len(warns) != 1 || len(s.Rules) != 1 || s.Rules[0].Selectors[0].String() != "a" {
			t.Errorf("@container %s: dropped whole, the rule after it kept: %s %d", prelude, warnings(warns),
				len(s.Rules))
		}
	}
}

func TestContainerUnitsMeasureTheNearestContainerOnTheirAxis(t *testing.T) {
	sized, inline, plain := el(dom.Div, "sized"), el(dom.Div, "inline"), el(dom.Div, "plain")
	in, deep, out := el(dom.P, "in"), el(dom.P, "deep"), el(dom.P, "out")
	root := page(with(sized, with(inline, with(plain, deep), in)), out)
	sheets := []*Sheet{sheet(t, `
#sized { container-type: size; width: 10cqw; }
#inline { container-type: inline-size; }
p { width: 10cqw; height: 10cqh; margin-left: 10cqmin; margin-right: 10cqmax; padding-left: 10cqi;
  padding-right: 10cqb; margin-top: calc(50cqw - 10px); }
#sized::before { content: ""; width: 10cqw; }
`)}
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 800, Height: 600})
	tr.Compute(root, sheets)
	check := func(when string, n *dom.Node, w, h, mn, mx, top float32) {
		t.Helper()
		s := styleOf(n)
		got := []float32{s.Size.Width.Px, s.Size.Height.Px, s.Margin.Left.Px, s.Margin.Right.Px, s.Padding.Left.Px,
			s.Padding.Right.Px, s.Margin.Top.Px}
		if want := []float32{w, h, mn, mx, w, h, top}; !slices.Equal(got, want) {
			t.Errorf("%s, %s: %v, want %v", when, n.ID(), got, want)
		}
	}
	check("before a layout, the small viewport", deep, 80, 60, 60, 80, 390)
	requery(tr, root, sheets, boxes{sized: {500, 100}, inline: {300, 40}})
	check("laid out", deep, 30, 10, 10, 30, 140)
	check("laid out", in, 30, 10, 10, 30, 140)
	check("laid out", out, 80, 60, 60, 80, 390)
	if b := dom.Data(sized).Before; b == nil || styleOf(b).Size.Width.Px != 50 {
		t.Error("a pseudo-element's container may be its element")
	}
	if w := styleOf(sized).Size.Width.Px; w != 80 {
		t.Errorf("a container's own units are its container's: %gpx", w)
	}
	if !requery(tr, root, sheets, boxes{sized: {500, 100}, inline: {200, 40}}) {
		t.Fatal("a length reads the container resized")
	}
	check("resized", deep, 20, 10, 10, 20, 90)
}

func TestARequeryRestylesOnlyWhereAnAnswerMoved(t *testing.T) {
	box, other := el(dom.Div, "box"), el(dom.Div, "other")
	var kids []*dom.Node
	for range 10 {
		kids = append(kids, el(dom.P, ""))
	}
	root := page(with(box, kids...), with(other, el(dom.P, ""), el(dom.P, "")))
	sheets := []*Sheet{sheet(t, `
#box, #other { container-type: inline-size; }
@container (width > 300px) { p { color: #f00; } }
`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	requery(tr, root, sheets, boxes{box: {400, 10}, other: {400, 10}})
	if styleOf(kids[0]).Color != hex("#f00") {
		t.Fatal("400 is over 300")
	}
	before := tr.Restyled()
	if requery(tr, root, sheets, boxes{box: {350, 10}, other: {400, 20}}) || tr.Restyled() != before {
		t.Errorf("no answer moved: %d restyled", tr.Restyled()-before)
	}
	before = tr.Restyled()
	if !requery(tr, root, sheets, boxes{box: {250, 10}, other: {400, 20}}) {
		t.Fatal("box crossed 300")
	}
	if n := tr.Restyled() - before; n != len(kids)+1 {
		t.Errorf("box and what it holds restyle, not other's: %d", n)
	}
	if styleOf(kids[0]).Color == hex("#f00") {
		t.Error("250 is under 300")
	}
	plain := []*Sheet{sheet(t, `#box { container-type: inline-size; }`)}
	tr = NewTransitions()
	tr.Compute(root, plain)
	if requery(tr, root, plain, boxes{box: {400, 10}}) {
		t.Error("a sheet with no @container rule and no container unit has nothing to answer")
	}
}

func TestASheetWritesItsContainerRulesForChrome(t *testing.T) {
	s := sheet(t, `@container card (min-width: 400px) , (height > 0) { p { color: #100; }
@media (hover) { @container style(--a: 1) { a { color: #200; } } } }`)
	want := `@container card (min-width: 400px) , (height > 0) { p { color: #100; } }
@media (hover) { @container card (min-width: 400px) , (height > 0) { @container style(--a: 1) { a { color: #200; } } } }
`
	if got := s.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if again, warns := Parse("t.css", s.String()); len(warns) > 0 || again.String() != want {
		t.Errorf("read again: %s\n%s", warnings(warns), again.String())
	}
}
