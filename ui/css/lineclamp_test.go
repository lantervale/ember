package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func clampOfDecls(t *testing.T, decls string) *style.Computed {
	t.Helper()
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, "#d { "+decls+" }")})
	return styleOf(d)
}

func TestLineClampComputesAsChromeLaysItOut(t *testing.T) {
	cases := []struct {
		decls    string
		display  style.Display
		flowRoot bool
		lines    int
		keeps    bool
	}{
		{"line-clamp: 3", style.Block, false, 3, false},
		{"line-clamp: none", style.Block, false, 0, false},
		{"display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2", style.Block, true, 2, true},
		{"display: -webkit-inline-box; -webkit-box-orient: block-axis; -webkit-line-clamp: 1", style.InlineBlock,
			true, 1, true},
		{"display: -webkit-box; -webkit-line-clamp: 2", style.FlexDisplay, false, 0, false},
		{"display: -webkit-box; -webkit-box-orient: vertical", style.FlexDisplay, false, 0, false},
		{"display: block; -webkit-box-orient: vertical; -webkit-line-clamp: 2", style.Block, false, 0, false},
		{"display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; line-clamp: 4", style.Block, true,
			4, false},
		{"line-clamp: auto", style.Block, false, style.ClampByHeight, false},
		{"line-clamp: no-ellipsis", style.Block, false, style.ClampByHeight, false},
		{"line-clamp: no-ellipsis auto", style.Block, false, style.ClampByHeight, false},
		{"line-clamp: 2 auto", style.Block, false, 2, false},
		{"line-clamp: 2 -webkit-legacy", style.Block, false, 0, false},
		{"display: -webkit-box; -webkit-box-orient: vertical; line-clamp: 2 -webkit-legacy", style.Block, true, 2, true},
		{"display: -webkit-box; -webkit-box-orient: vertical; line-clamp: auto -webkit-legacy", style.Block, true,
			style.ClampByHeight, true},
	}
	for _, c := range cases {
		s := clampOfDecls(t, "display: block; "+c.decls)
		lines, keeps := s.Clamp()
		if s.Display != c.display || s.FlowRoot != c.flowRoot || lines != c.lines || keeps != c.keeps {
			t.Errorf("%s: display %v flow-root %v, clamp %d keeping %v; want %v %v, %d %v", c.decls, s.Display,
				s.FlowRoot, lines, keeps, c.display, c.flowRoot, c.lines, c.keeps)
		}
	}
	if s := clampOfDecls(t, "display: -webkit-box; -webkit-box-orient: vertical"); s.Direction != style.Column {
		t.Errorf("a vertical -webkit-box is a column: %v", s.Direction)
	}
}

func TestLineClampsAreCheckedAsIntegers(t *testing.T) {
	for _, decl := range []string{"line-clamp: 0", "line-clamp: 1.5", "-webkit-line-clamp: -1",
		"-webkit-box-orient: diagonal", "line-clamp: 2 2", "line-clamp: none no-ellipsis", "line-clamp: 2 auto no-ellipsis",
		"line-clamp: -webkit-legacy", "line-clamp: -webkit-legacy 2", "line-clamp: 2 -webkit-legacy no-ellipsis",
		"line-clamp: none -webkit-legacy", "line-clamp: auto auto", "line-clamp: \"x\"",
		"max-lines: 2", "block-ellipsis: auto", "continue: discard"} {
		if got := read(decl); got == decl+" | " {
			t.Errorf("%s read", decl)
		}
	}
}

func TestNoEllipsisLeavesTheClampsEllipsisOut(t *testing.T) {
	for _, decls := range []string{"line-clamp: 2 no-ellipsis", "line-clamp: no-ellipsis 2"} {
		s := clampOfDecls(t, decls)
		if lines, _ := s.Clamp(); lines != 2 || !s.NoEllipsis {
			t.Errorf("%s: clamp %d, no ellipsis %v", decls, lines, s.NoEllipsis)
		}
	}
	if s := clampOfDecls(t, "line-clamp: 2 no-ellipsis; line-clamp: 3"); s.NoEllipsis {
		t.Error("a later line-clamp without it has its ellipsis")
	}
}
