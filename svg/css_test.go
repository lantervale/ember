package svg //nolint:testpackage // white-box: reads the parsed model's unexported parts

import (
	"slices"
	"testing"
)

func sheet(t *testing.T, css string) ([]rule, []string) {
	t.Helper()
	var dropped []string
	rules := parseSheet(css, nil, 100, func(what string) { dropped = append(dropped, what) })
	return rules, dropped
}

func TestStyleSheetsReadSimpleSelectors(t *testing.T) {
	rules, dropped := sheet(t, `
		/* a comment { fill: no } */
		path { fill: red }
		.a, #b { stroke: blue; stroke-width: 2 !important }
		g.a.c > rect#d { fill: green }
		* { opacity: .5 }
		svg g path { fill: black }
	`)
	if len(dropped) != 0 {
		t.Errorf("dropped %v", dropped)
	}
	if len(rules) != 6 {
		t.Fatalf("%d rules, want 6", len(rules))
	}
	// Sorted by how specific they are: *, path, svg g path, .a, g.a.c > rect#d… and #b.
	var got []int
	for _, r := range rules {
		got = append(got, r.specificity)
	}
	if !slices.IsSorted(got) || got[0] != 0 || got[1] != 1 || got[2] != 3 {
		t.Errorf("specificities %v", got)
	}
	if rules[3].decl["stroke-width"] != "2" {
		t.Errorf("!important is kept in the value: %q", rules[3].decl["stroke-width"])
	}
}

func TestStyleSheetsDropWhatTheyCannotMatch(t *testing.T) {
	rules, dropped := sheet(t, `
		@import url(x.css);
		@media (min-width: 100px) { path { fill: red } }
		path:hover { fill: red }
		[fill] { fill: red }
		a + b { fill: red }
		> a { fill: red }
		rect { fill: blue }
	`)
	if len(rules) != 1 || rules[0].steps[0].tag != "rect" {
		t.Errorf("rules %+v, want only rect's", rules)
	}
	want := []string{"@import", "@media", "style selector", "style selector", "style selector", "style selector"}
	if !slices.Equal(dropped, want) {
		t.Errorf("dropped %v, want %v", dropped, want)
	}
}

func TestSelectorsMatchThroughAncestors(t *testing.T) {
	root := &node{name: "svg"}
	g := &node{name: "g", parent: root, classes: []string{"a", "c"}, attr: map[string]string{}}
	inner := &node{name: "g", parent: g, attr: map[string]string{}}
	rect := &node{name: "rect", parent: inner, attr: map[string]string{"id": "d"}, classes: []string{"x"}}
	root.attr = map[string]string{}
	for sel, want := range map[string]bool{
		"rect": true, "path": false, "#d": true, ".x": true, "rect.x#d": true, "rect.y": false,
		"g rect": true, "svg rect": true, ".a rect": true, ".a.c rect": true, ".a.b rect": false,
		"g > rect": true, ".a > rect": false, "svg > g > g > rect": true, "svg > rect": false,
		".a g > rect": true, "*": true, "g *": true,
	} {
		r, ok := parseSelector(sel)
		if !ok {
			t.Errorf("%q is not read", sel)
			continue
		}
		if got := r.matches(rect); got != want {
			t.Errorf("%q matches: %v, want %v", sel, got, want)
		}
	}
}

func TestStyleSheetsStopAtTheirLimit(t *testing.T) {
	if rules := parseSheet("a{x:y} b{x:y} c{x:y}", nil, 2, func(string) {}); len(rules) != 2 {
		t.Errorf("%d rules, want 2", len(rules))
	}
}
