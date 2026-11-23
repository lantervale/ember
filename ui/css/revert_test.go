package css //nolint:testpackage // uses the unexported test helpers (sheet, page) and reads computed styles off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestRevertRollsAnAuthorsDeclarationBackToTheUserAgents(t *testing.T) {
	p, div, short := el(dom.P, "p"), el(dom.Div, "div"), el(dom.P, "short")
	Compute(page(p, div, short), []*Sheet{sheet(t, `
p, div { display: inline; margin-top: 3px; }
#p { display: revert; margin-top: revert; }
#div { display: revert; margin-top: revert; }
#short { margin: revert; }
`)})
	if s := styleOf(p); s.Display != style.Block || s.Margin.Top.Px != 16 {
		t.Errorf("a p reverts to the user agent's block and margin: %v %+v", s.Display, s.Margin.Top)
	}
	if s := styleOf(div); s.Display != style.Block || s.Margin.Top.Px != 0 {
		t.Errorf("a div reverts to the user agent's block and no margin: %v %+v", s.Display, s.Margin.Top)
	}
	if s := styleOf(short); s.Margin.Top.Px != 16 {
		t.Errorf("a shorthand reverts each of its longhands: %+v", s.Margin.Top)
	}
}

func TestRevertOfAnInheritedPropertyTheUserAgentLeavesOutInherits(t *testing.T) {
	p := el(dom.P, "p")
	Compute(page(with(el(dom.Div, "out"), p)), []*Sheet{sheet(t, `
#out { color: rgb(255, 0, 0); }
p { color: rgb(0, 0, 255); }
#p { color: revert; }
`)})
	if c := styleOf(p).Color; c != styleOf(p.Parent()).Color {
		t.Errorf("revert with nothing to roll back to is unset: %v", c)
	}
}

func TestRevertLayerRollsBackToTheLayerBelow(t *testing.T) {
	up, top, only := el(dom.P, "up"), el(dom.P, "top"), el(dom.Ul, "only")
	s, warns := asBrowser(`
@layer base, theme;
@layer base { p { margin-top: 7px; } }
@layer theme { p { margin-top: 3px; } #up { margin-top: revert-layer; } }
#top { margin-top: revert-layer; }
@layer base { #only { padding-left: revert-layer; } }
`)
	if warns != "" {
		t.Fatal(warns)
	}
	Compute(page(up, top, only), []*Sheet{s})
	if s := styleOf(up); s.Margin.Top.Px != 7 {
		t.Errorf("the theme's revert-layer leaves the base's: %+v", s.Margin.Top)
	}
	if s := styleOf(top); s.Margin.Top.Px != 3 {
		t.Errorf("an unlayered revert-layer leaves the last layer's: %+v", s.Margin.Top)
	}
	if s := styleOf(only); s.Padding.Left.Px != 40 {
		t.Errorf("revert-layer in the first layer is revert: %+v", s.Padding.Left)
	}
}
