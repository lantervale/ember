package css

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestScrollStateQueriesAnswerTheContainersState(t *testing.T) {
	sticky, scroller, sized := el(dom.Div, "sticky"), el(dom.Div, "scroller"), el(dom.Div, "sized")
	var kids []*dom.Node
	for i := range 10 {
		kids = append(kids, el(dom.P, "", "n"+string(rune('a'+i))))
	}
	root := page(with(sticky, kids[:5]...), with(scroller, kids[5:8]...), with(sized, kids[8:]...))
	sheets := []*Sheet{sheet(t, `
#sticky { container-type: scroll-state; }
#scroller { container: list / scroll-state; direction: rtl; }
#sized { container-type: inline-size; }
p { width: 1px; }
@container scroll-state(stuck: top) { .na { width: 11px; } }
@container scroll-state(stuck) { .nb { width: 12px; } }
@container scroll-state(stuck: none) { .nc { width: 13px; } }
@container scroll-state((stuck: bottom) or (stuck: block-start)) { .nd { width: 14px; } }
@container scroll-state(snapped: none) { .ne { width: 15px; } }
@container list scroll-state(scrollable: inline-start) { .nf { width: 16px; } }
@container scroll-state(scrollable: y) { .ng { width: 17px; } }
@container scroll-state(not (scrollable)) { .nh { width: 18px; } }
@container scroll-state(stuck) { .ni { width: 19px; } }
@container scroll-state(stuck: none) { .nj { width: 20px; } }
`)}
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 800, Height: 600})
	tr.Compute(root, sheets)
	check := func(want []float32) {
		t.Helper()
		for i, k := range kids {
			if w := styleOf(k).Size.Width.Px; w != want[i] {
				t.Errorf("%s: %gpx, want %g", k.Classes()[0], w, want[i])
			}
		}
	}
	state := map[*dom.Node]ScrollState{sticky: {Stuck: SideTop}, scroller: {Scrollable: SideRight}}
	of := func(n *dom.Node) ScrollState { return state[n] }
	if !tr.RequeryScroll(of) {
		t.Fatal("a stuck box and a scrolled one change what the rules answer")
	}
	tr.Compute(root, sheets)
	check([]float32{11, 12, 1, 14, 15, 16, 1, 1, 1, 1})
	if tr.RequeryScroll(of) {
		t.Error("the same state changes nothing")
	}
	state[sticky], state[scroller] = ScrollState{}, ScrollState{}
	if !tr.RequeryScroll(of) {
		t.Fatal("unstuck, and scrolled to its start")
	}
	tr.Compute(root, sheets)
	check([]float32{1, 1, 13, 1, 15, 1, 1, 18, 1, 1})
}
