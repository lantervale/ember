package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func gridOf(t *testing.T, decls string) *style.Computed {
	t.Helper()
	d := el(dom.Div, "d")
	Compute(page(d), []*Sheet{sheet(t, "#d { "+decls+" }")})
	return styleOf(d)
}

func TestGridTemplateAreas(t *testing.T) {
	s := gridOf(t, "grid-template-areas: 'head head' 'side main' '... main'")
	want := []style.GridArea{
		{Name: "head", Row: [2]int{1, 2}, Column: [2]int{1, 3}},
		{Name: "side", Row: [2]int{2, 3}, Column: [2]int{1, 2}},
		{Name: "main", Row: [2]int{2, 4}, Column: [2]int{2, 3}},
	}
	if !reflect.DeepEqual(s.Areas, want) {
		t.Errorf("areas %v, want %v", s.Areas, want)
	}
	if s = gridOf(t, "grid-template-areas: 'a'; grid-template-areas: none"); s.Areas != nil {
		t.Errorf("none leaves %v", s.Areas)
	}
	for src, why := range map[string]string{
		"'a a' 'b'":     "as many cells",
		"'a b' 'b a'":   "not a rectangle",
		"'a a' 'a .'":   "not a rectangle",
		"'a' 10px":      "a string a row",
		"''":            "as many cells",
		"'a b' 'c' 'd'": "as many cells",
	} {
		_, warns := Parse("t.css", "div { grid-template-areas: "+src+" }")
		if len(warns) != 1 || !strings.Contains(warns[0].Error(), why) {
			t.Errorf("%s: warnings %v, want one of %q", src, warns, why)
		}
	}
}

func TestGridArea(t *testing.T) {
	name := func(n string) style.GridLine { return style.GridLine{Name: n} }
	line := func(n int) style.GridLine { return style.GridLine{Line: n} }
	cases := map[string][4]style.GridLine{
		"main":             {name("main"), name("main"), name("main"), name("main")},
		"a / b":            {name("a"), name("b"), name("a"), name("b")},
		"2 / side":         {line(2), name("side"), {}, name("side")},
		"1 / 2 / 3 / 4":    {line(1), line(2), line(3), line(4)},
		"auto":             {},
		"2 / 1 / span 2":   {line(2), line(1), {Span: 2}, {}},
		"a / 1 / b / -1":   {name("a"), line(1), name("b"), line(-1)},
		"3":                {line(3), {}, {}, {}},
		"Main / 2 / 3 / x": {name("Main"), line(2), line(3), name("x")},
	}
	for src, want := range cases {
		s := gridOf(t, "grid-area: "+src)
		if got := [4]style.GridLine{s.Row[0], s.Column[0], s.Row[1], s.Column[1]}; got != want {
			t.Errorf("grid-area: %s: %v, want %v", src, got, want)
		}
	}
	if _, warns := Parse("t.css", "div { grid-area: 1 / 2 / 3 / 4 / 5 }"); len(warns) != 1 {
		t.Errorf("five lines: %v", warns)
	}
}

func TestGridTemplate(t *testing.T) {
	s := gridOf(t, "grid-template-areas: 'x'; grid-template: 10px 1fr / 20px minmax(0, 1fr)")
	if len(s.Rows) != 2 || len(s.Columns) != 2 || s.Areas != nil {
		t.Errorf("rows / columns: %v / %v, areas %v", s.Rows, s.Columns, s.Areas)
	}
	s = gridOf(t, "grid-template: 'head head' 40px 'side main' / 100px 1fr")
	if len(s.Rows) != 2 || s.Rows[0].Tracks[0].Min != style.Px(40) || s.Rows[1].Tracks[0].Min.Kind != style.Auto ||
		len(s.Columns) != 2 || len(s.Areas) != 3 {
		t.Errorf("strings: %v / %v, areas %v", s.Rows, s.Columns, s.Areas)
	}
	if s = gridOf(t, "grid-template: 10px / 20px; grid-template: none"); s.Rows != nil || s.Columns != nil {
		t.Errorf("none leaves %v / %v", s.Rows, s.Columns)
	}
	for _, src := range []string{"10px", "10px / 20px / 30px", "'a' 'b b' / 1fr", "/ 10px"} {
		if _, warns := Parse("t.css", "div { grid-template: "+src+" }"); len(warns) != 1 {
			t.Errorf("%s: %v", src, warns)
		}
	}
}
