package css //nolint:testpackage // uses the package's test helpers (sheet, page) and reads computed styles off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAHiddenInputHasNoBoxWhateverThePageSays(t *testing.T) {
	hidden := func(kind, class, inline string) *dom.Node {
		in := el(dom.Input, "", class)
		in.SetAttr("type", kind)
		if inline != "" {
			in.SetStyle(inline)
		}
		return in
	}
	cases := map[string]*dom.Node{
		"a rule of the page":           hidden("hidden", "shown", ""),
		"an important rule":            hidden("hidden", "forced", ""),
		"its own style":                hidden("hidden", "", "display: block"),
		"its own important style":      hidden("hidden", "", "display: block !important"),
		"its type in capitals":         hidden("HIDDEN", "shown", ""),
		"its type in mixed case":       hidden("Hidden", "", ""),
		"a selector as Chrome's sheet": hidden("hidden", "typed", ""),
	}
	text := hidden("text", "shown", "")
	nodes := []*dom.Node{text}
	for _, n := range cases {
		nodes = append(nodes, n)
	}
	Compute(page(nodes...), []*Sheet{sheet(t, `.shown { display: inline-flex }
.forced { display: block !important }
input[type="hidden" i].typed { display: inline-block !important }`)})
	for name, n := range cases {
		if got := styleOf(n).Display; got != style.DisplayNone {
			t.Errorf("%s: display %v", name, got)
		}
	}
	if got := styleOf(text).Display; got != style.InlineFlex {
		t.Errorf("a text field takes the page's display: %v", got)
	}
}
