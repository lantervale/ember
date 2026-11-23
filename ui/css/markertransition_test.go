package css //nolint:testpackage // reads the ::marker's node off dom with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAMarkersTransitionRunsOnItsOwnNode(t *testing.T) {
	li := with(el(dom.Li, "li"), dom.NewText("item"))
	root := page(with(el(dom.Ul, "ul"), li))
	sheets := []*Sheet{sheet(t, `li::marker { color: #000; font-size: 10px;
  transition: color 1s linear, font-size 1s linear }
li.on::marker { color: #fff; font-size: 20px }`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	g := dom.Data(li).MarkerNode
	if g == nil {
		t.Fatal("a styled ::marker has a node")
	}
	li.AddClass("on")
	tr.Compute(root, sheets)
	tr.Step(0.5)
	tr.Compute(root, sheets)
	srgbNear(t, "halfway", shown(g).Color, [4]float32{0.5, 0.5, 0.5, 1})
	if got := dom.Data(li).Marker.Font.Size; !nearf(got, 15) {
		t.Errorf("font-size halfway: %v", got)
	}
	tr.Step(1)
	tr.Compute(root, sheets)
	srgbNear(t, "done", shown(g).Color, [4]float32{1, 1, 1, 1})
	if got := dom.Data(li).Marker.Font.Size; got != 20 {
		t.Errorf("font-size done: %v", got)
	}
}
