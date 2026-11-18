package css //nolint:testpackage // reads the restyle count and the selectors' analysis

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// menuPage is a page of two menus, what follows the first, and a details:
//
//	body
//	  div#menu.menu[data-x]   span#label.label  div#row.row(span#item.item(b#deep))  a#link
//	  p#next.after            span#in.item
//	  p#last                  input#box  span#tail.item
//	  div#other.menu          span#olabel.label  span#oitem.item
//	  details#more            summary#sum  div#body(span#inner.item)
func menuPage() (*dom.Node, map[string]*dom.Node) {
	menu := with(el(dom.Div, "menu", "menu"),
		el(dom.Span, "label", "label"),
		with(el(dom.Div, "row", "row"), with(el(dom.Span, "item", "item"), el(dom.B, "deep"))),
		el(dom.A, "link"))
	menu.SetAttr("data-x", "1")
	root := page(menu,
		with(el(dom.P, "next", "after"), el(dom.Span, "in", "item")),
		with(el(dom.P, "last"), el(dom.Input, "box"), el(dom.Span, "tail", "item")),
		with(el(dom.Div, "other", "menu"), el(dom.Span, "olabel", "label"), el(dom.Span, "oitem", "item")),
		with(el(dom.Details, "more"), el(dom.Summary, "sum"), with(el(dom.Div, "body"), el(dom.Span, "inner", "item"))))
	ids := map[string]*dom.Node{}
	root.Walk(func(n *dom.Node) bool {
		if !n.IsText() && n.ID() != "" {
			ids[n.ID()] = n
		}
		return true
	})
	return root, ids
}

// styled is what css keeps of an element: its style and those of its
// generated boxes.
type styled struct {
	own, before, after style.Computed
	has                [2]bool
}

func styles(root *dom.Node) map[*dom.Node]styled {
	out := map[*dom.Node]styled{}
	root.Walk(func(n *dom.Node) bool {
		if n.IsText() {
			return true
		}
		d := dom.Data(n)
		s := styled{own: *d.Style}
		if d.Before != nil {
			s.before, s.has[0] = *dom.Data(d.Before).Style, true
		}
		if d.After != nil {
			s.after, s.has[1] = *dom.Data(d.After).Style, true
		}
		out[n] = s
		return true
	})
	return out
}

// stale lists the elements whose kept style is not the one a restyle of
// the whole tree gives them.
func stale(root *dom.Node, sheets []*Sheet) []string {
	kept := styles(root)
	root.Walk(func(n *dom.Node) bool { dom.Data(n).Style = nil; return true })
	Compute(root, sheets)
	fresh := styles(root)
	var out []string
	root.Walk(func(n *dom.Node) bool {
		if !n.IsText() && !reflect.DeepEqual(kept[n], fresh[n]) {
			out = append(out, describe(n))
		}
		return true
	})
	return out
}

// marked are the ids of the elements with a red background, sorted.
func marked(ids map[string]*dom.Node) string {
	var out []string
	for id, n := range ids {
		if styleOf(n).Background == hex("#f00") {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return strings.Join(out, " ")
}

func TestAStateRestylesWhatItsSelectorsReach(t *testing.T) {
	for _, c := range []struct {
		name, src, on string
		state         dom.State
		want          string
	}{
		{"a descendant", ".menu:hover .item { background-color: #f00 }", "menu", dom.Hover, "item"},
		{"a child", ".menu:hover > .label { background-color: #f00 }", "menu", dom.Hover, "label"},
		{"a child, not deeper", ".menu:hover > .item { background-color: #f00 }", "menu", dom.Hover, ""},
		{"the next sibling", ".menu:hover + .after { background-color: #f00 }", "menu", dom.Hover, "next"},
		{"under later siblings", ".menu:hover ~ p .item { background-color: #f00 }", "menu", dom.Hover, "in tail"},
		{"a sibling below", ".menu:hover .label + .row > .item { background-color: #f00 }", "menu", dom.Hover, "item"},
		{"a sibling of a descendant", ".menu:hover .row ~ a { background-color: #f00 }", "menu", dom.Hover, "link"},
		{":not()", ".item { background-color: #f00 } .menu:not(:hover) .item { background-color: #00f }",
			"menu", dom.Hover, "in inner item tail"},
		{":is()", ":is(.menu:hover, .none) .item { background-color: #f00 }", "menu", dom.Hover, "item"},
		{":where()", ":where(.menu:hover) > .label { background-color: #f00 }", "menu", dom.Hover, "label"},
		{":is() of a complex selector", ".item:is(.menu:hover .row *) { background-color: #f00 }", "menu", dom.Hover, "item"},
		{":not() of a complex selector",
			".item { background-color: #f00 } .item:not(.menu:hover .item) { background-color: #00f }",
			"menu", dom.Hover, "item"},
		{":is() of siblings", ":is(.menu:hover + p) span { background-color: #f00 }", "menu", dom.Hover, "in"},
		{":is() before a sibling", ".row:is(.menu:hover *) + a { background-color: #f00 }", "menu", dom.Hover, "link"},
		{"every descendant", ".menu:hover * { background-color: #f00 }", "menu", dom.Hover, "deep item label link row"},
		{"a type above", "div:hover b { background-color: #f00 }", "row", dom.Hover, "deep"},
		{"an id above", "#menu:hover b { background-color: #f00 }", "menu", dom.Hover, "deep"},
		{"an attribute above", "[data-x]:hover .item { background-color: #f00 }", "menu", dom.Hover, "item"},
		{"any element above", ":hover > .label { background-color: #f00 }", "other", dom.Hover, "olabel"},
		{"two ancestors, one hovered", ".menu:hover .row:hover .item { background-color: #f00 }", "menu", dom.Hover, ""},
		{"focus", ".menu:focus .item { background-color: #f00 }", "menu", dom.Focus, "item"},
		{"focus-visible", ".menu:focus-visible > .label { background-color: #f00 }", "other", dom.FocusVisible, "olabel"},
		{"active", ".row:active b { background-color: #f00 }", "row", dom.Active, "deep"},
		{"checked", "input:checked ~ span { background-color: #f00 }", "box", dom.Checked, "tail"},
		{"disabled", "input:disabled + .item { background-color: #f00 }", "box", dom.Disabled, "tail"},
		{"enabled", ".item { background-color: #f00 } input:enabled + .item { background-color: #00f }",
			"box", dom.Disabled, "in inner item oitem tail"},
		{"an open details", "details:open .item { background-color: #f00 }", "more", dom.Open, "inner"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, ids := menuPage()
			sheets := []*Sheet{sheet(t, c.src)}
			Compute(root, sheets)
			off := marked(ids)
			if off == c.want && c.want != "" {
				t.Fatalf("%s is marked before the state", off)
			}
			ids[c.on].SetState(c.state, true)
			Compute(root, sheets)
			if got := marked(ids); got != c.want {
				t.Errorf("on: marked %q, want %q", got, c.want)
			}
			if bad := stale(root, sheets); len(bad) > 0 {
				t.Errorf("on: stale styles on %v", bad)
			}
			ids[c.on].SetState(c.state, false)
			Compute(root, sheets)
			if got := marked(ids); got != off {
				t.Errorf("off: marked %q, want %q", got, off)
			}
			if bad := stale(root, sheets); len(bad) > 0 {
				t.Errorf("off: stale styles on %v", bad)
			}
		})
	}
}

// Whatever element takes or loses whatever state, under sheets reading
// states every way the subset has, the styles kept are those a restyle
// of the whole tree gives.
func TestNoStateLeavesAStaleStyle(t *testing.T) {
	sheets := []*Sheet{sheet(t, `
.menu:hover .item { background-color: #f00 }
.menu:hover > .label { color: #0f0 }
.menu:hover { color: #00f; font-size: 20px }
.row:hover { --ink: #123 }
.item { border-top-color: var(--ink, #000) }
.menu:hover + p { padding-left: 4px }
.menu:active ~ p span { margin-left: 2px }
.label:hover ~ .row b { opacity: 0.5 }
:is(.menu:focus, .row:hover) > :first-child { top: 1px }
.item:not(.menu:hover .item) { left: 3px }
:not(:hover) > b { bottom: 2px }
.menu:hover .item::before { content: "x"; color: #f00 }
.row:hover::after { content: "y" }
p:hover > input:checked + span::after { content: "z" }
input:disabled ~ * { background-color: #333 }
input:enabled + span { right: 1px }
details:open > div span { color: #f0f }
body:hover > :last-child { margin-top: 1px }
.menu:hover .row:hover .item:hover b { text-decoration: underline }
:focus-visible ~ * b { letter-spacing: 1px }
`)}
	root, ids := menuPage()
	Compute(root, sheets)
	check := func(what string) {
		t.Helper()
		Compute(root, sheets)
		if bad := stale(root, sheets); len(bad) > 0 {
			t.Errorf("%s: stale styles on %v", what, bad)
		}
	}
	states := []dom.State{dom.Hover, dom.Focus, dom.FocusVisible, dom.Active, dom.Disabled, dom.Checked, dom.Open}
	for id, n := range ids {
		for _, s := range states {
			n.SetState(s, true)
			check(fmt.Sprint(id, " takes ", s))
			n.SetState(s, false)
			check(fmt.Sprint(id, " loses ", s))
		}
	}
	// The pointer's way: an element and its ancestors hovered at once,
	// then another chain the next frame.
	chain := func(n *dom.Node, on bool) {
		for ; n != nil; n = n.Parent() {
			n.SetState(dom.Hover, on)
		}
	}
	var was *dom.Node
	for _, id := range []string{"deep", "label", "in", "box", "oitem", "inner", "row", "menu", "deep"} {
		chain(was, false)
		chain(ids[id], true)
		check("the pointer over " + id)
		ids["box"].SetState(dom.Checked, id == "in" || id == "row")
		ids["menu"].SetState(dom.Active, id == "label")
		check("with a press, over " + id)
		was = ids[id]
	}
	chain(was, false)
	check("the pointer gone")
}

func TestAnInheritedPropertyOfAStateReachesTheDescendants(t *testing.T) {
	root, ids := menuPage()
	sheets := []*Sheet{sheet(t, ".menu:hover { color: #f00; --gap: 4px } b { padding-left: var(--gap, 0) } "+
		".other:hover .item { background-color: #00f }")}
	Compute(root, sheets)
	ids["menu"].SetState(dom.Hover, true)
	Compute(root, sheets)
	if styleOf(ids["deep"]).Color != hex("#f00") || styleOf(ids["deep"]).Padding.Left.Px != 4 {
		t.Errorf("three levels down: color %v, padding %v", styleOf(ids["deep"]).Color, styleOf(ids["deep"]).Padding.Left)
	}
	ids["menu"].SetState(dom.Hover, false)
	Compute(root, sheets)
	if styleOf(ids["deep"]).Color == hex("#f00") || styleOf(ids["deep"]).Padding.Left.Px != 0 {
		t.Error("and back when the pointer leaves")
	}
}

func TestAStateStartsTheTransitionsOfWhatItReaches(t *testing.T) {
	root, ids := menuPage()
	sheets := []*Sheet{sheet(t, ".item { background-color: #000; transition: background-color 1s } "+
		".menu:hover .item { background-color: #f00 }")}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	ids["menu"].SetState(dom.Hover, true)
	tr.Compute(root, sheets)
	if !tr.Running(ids["item"]) {
		t.Fatal("the item under the hovered menu starts its transition")
	}
	if tr.Running(ids["oitem"]) || tr.Running(ids["in"]) {
		t.Error("the items elsewhere start none")
	}
	tr.Step(0.5)
	if got := dom.Data(ids["item"]).Shown.Background; got == hex("#000") || got == hex("#f00") {
		t.Errorf("half way: %v", got)
	}
}

func TestAStateReadUnderAMediaRuleFollowsIt(t *testing.T) {
	root, ids := menuPage()
	sheets := []*Sheet{sheet(t, "@media (min-width: 500px) { .menu:hover .item { background-color: #f00 } }")}
	tr := NewTransitions()
	tr.SetMedia(Media{Width: 400, Height: 300})
	tr.Compute(root, sheets)
	hover := func(on bool) int {
		was := tr.computed
		ids["menu"].SetState(dom.Hover, on)
		tr.Compute(root, sheets)
		return tr.computed - was
	}
	if n := hover(true); marked(ids) != "" || n != 1 {
		t.Errorf("a narrow viewport: marked %q, %d restyled, want none and the menu alone", marked(ids), n)
	}
	tr.SetMedia(Media{Width: 600, Height: 300})
	tr.Compute(root, sheets)
	if marked(ids) != "item" {
		t.Errorf("a wide one, the menu hovered: marked %q", marked(ids))
	}
	if n := hover(false); marked(ids) != "" || n != 2 {
		t.Errorf("the pointer gone: marked %q, %d restyled, want none, the menu and its item", marked(ids), n)
	}
	hover(true)
	tr.SetMedia(Media{Width: 400, Height: 300})
	tr.Compute(root, sheets)
	if marked(ids) != "" {
		t.Errorf("narrow again: marked %q", marked(ids))
	}
}

func TestAnotherSheetsStatesAreRead(t *testing.T) {
	root, ids := menuPage()
	plain := sheet(t, ".menu:hover { outline-width: 1px }")
	tr := NewTransitions()
	tr.Compute(root, []*Sheet{plain})
	reads := []*Sheet{plain, sheet(t, ".menu:hover .item, .menu:hover + p { background-color: #f00 }")}
	tr.Compute(root, reads)
	ids["menu"].SetState(dom.Hover, true)
	tr.Compute(root, reads)
	if marked(ids) != "item next" {
		t.Errorf("a sheet added: marked %q", marked(ids))
	}
	tr.Compute(root, []*Sheet{plain})
	if marked(ids) != "" {
		t.Errorf("the sheet gone, the menu still hovered: marked %q", marked(ids))
	}
	was := tr.computed
	ids["menu"].SetState(dom.Hover, false)
	tr.Compute(root, []*Sheet{plain})
	if tr.computed-was != 1 {
		t.Errorf("no sheet reads the state above now: %d restyled, want the menu alone", tr.computed-was)
	}
}

// wide is a menu of n rows, each a div holding an item and a label,
// between a div and a paragraph holding an item each, in a page of its
// own.
func wide(n int) (*dom.Node, *dom.Node) {
	menu := el(dom.Div, "menu", "menu")
	for range n {
		menu.Append(with(el(dom.Div, "", "row"), el(dom.Span, "", "item"), el(dom.Span, "", "label")))
	}
	return page(with(el(dom.Div, "", "side"), el(dom.Span, "", "item")), menu,
		with(el(dom.P, "last"), el(dom.Span, "", "item"))), menu
}

func TestAStateRestylesOnlyWhatReadsIt(t *testing.T) {
	const rows = 300
	for _, c := range []struct {
		name, src string
		want      int
	}{
		{"no state read above", ".menu:hover { background-color: #111 } .row { color: #fff }", 1},
		{"on another element's class", ".side:hover .item, .side:hover ~ p { color: #f00 }", 1},
		{"on another type", "p:hover span, section:hover > * { color: #f00 }", 1},
		{"its items", ".menu:hover .item { color: #f00 }", 1 + rows},
		{"its rows", ".menu:hover > div { background-color: #f00 }", 1 + rows},
		{"its rows, and what they hand down", ".menu:hover > div { color: #f00 }", 1 + 3*rows},
		{"what follows", ".menu:hover + p { background-color: #f00 }", 2},
		{"under what follows", ".menu:hover ~ p .item { color: #f00 }", 2},
		{"what follows, of another type", ".menu:hover + section, .menu:hover ~ .label { color: #f00 }", 1},
		{"another state", ".menu:focus * { color: #f00 } .menu:active ~ * { color: #f00 }", 1},
		{"a state another class does not have", ".side:not(:hover) .item, li:is(:hover, :focus) > .row { color: #f00 }", 1},
		{"every descendant", ".menu:hover * { opacity: 0.5 }", 1 + 3*rows},
	} {
		root, menu := wide(rows)
		tr := NewTransitions()
		sheets := []*Sheet{sheet(t, c.src)}
		tr.Compute(root, sheets)
		for _, on := range []bool{true, false} {
			was := tr.computed
			menu.SetState(dom.Hover, on)
			tr.Compute(root, sheets)
			if got := tr.computed - was; got != c.want {
				t.Errorf("%s, hover %v: %d elements restyled, want %d", c.name, on, got, c.want)
			}
		}
	}
}

func TestThePointerOverABigPageRestylesItsChainAlone(t *testing.T) {
	root, menu := wide(2000)
	leaf := menu.Children()[1000].Children()[0]
	sheets := []*Sheet{sheet(t, `
a:hover { color: #f00 }
.side:hover .item { color: #f00 }
.tab:hover > span, .tab:hover + .tab { color: #f00 }
li:hover ul { display: block }
`)}
	tr := NewTransitions()
	tr.Compute(root, sheets)
	for _, on := range []bool{true, false} {
		was := tr.computed
		for n := leaf; n != nil; n = n.Parent() {
			n.SetState(dom.Hover, on)
		}
		tr.Compute(root, sheets)
		if got := tr.computed - was; got != 5 {
			t.Errorf("hover %v: %d of 6,000 elements restyled, want the 5 under the pointer", on, got)
		}
	}
}

func TestAClosedDetailsStillHidesItsContentWhenItsStateMoves(t *testing.T) {
	root, ids := menuPage()
	Compute(root, nil)
	if styleOf(ids["body"]).Display != style.DisplayNone {
		t.Fatal("closed: its content shows")
	}
	ids["more"].SetState(dom.Open, true)
	Compute(root, nil)
	if styleOf(ids["body"]).Display == style.DisplayNone || styleOf(ids["sum"]).Display == style.DisplayNone {
		t.Error("open: its content is hidden")
	}
	ids["more"].SetState(dom.Open, false)
	Compute(root, nil)
	if styleOf(ids["body"]).Display != style.DisplayNone {
		t.Error("closed again: its content shows")
	}
}

func TestTheStatesASelectorReadsAwayFromItsSubject(t *testing.T) {
	const below, beside = "below", "beside"
	for _, c := range []struct {
		src  string
		want []string
	}{
		{"a:hover", nil},
		{"a:hover::before", nil},
		{".a:hover:not(:focus)", nil},
		{":is(a:hover, b:focus)", nil},
		{".menu:hover .item", []string{".menu:hover 1 below"}},
		{".menu:hover > .item::after", []string{".menu:hover 1 below"}},
		{"tr:hover > td:hover", []string{"tr:hover 1 below"}},
		{".a:hover + .b", []string{".a:hover 1 beside"}},
		{".a:checked ~ .b .c", []string{".a:checked 32 beside"}},
		{".a:hover .b + .c", []string{".a:hover 1 below"}},
		{".x .a:focus-visible + .b:hover > .c:active", []string{".a:focus-visible 4 beside", ".b:hover 1 below"}},
		{".a:hover:focus .b", []string{".a:hover:focus 3 below"}},
		{".a:not(:hover) .b", []string{".a:not(:hover)|:hover 1 below"}},
		{":is(.a:hover, .b:active) > .c",
			[]string{":is(.a:hover, .b:active)|.a:hover 1 below", ":is(.a:hover, .b:active)|.b:active 8 below"}},
		{"li:where(.a:hover) ~ .c", []string{"li:where(.a:hover)|.a:hover 1 beside"}},
		{".c:is(.a:hover .b)", []string{".a:hover 1 below"}},
		{".c:not(.a:hover + .b, .d:focus)", []string{".a:hover 1 beside"}},
		{".q:is(.a:hover .b) + .c", []string{".a:hover 1 below"}},
		{".q:is(.a .b:hover) + .c", []string{".q:is(.a .b:hover)|.b:hover 1 beside"}},
		{":is(.a:hover + .b) .c", []string{".a:hover 1 beside"}},
		{".c:is(:not(.a:hover .b) > .d)", []string{".a:hover 1 below"}},
		{"button:enabled + p", []string{"button:enabled 16 beside"}},
		{"details:open > summary", []string{"details:open 64 below"}},
	} {
		var got []string
		mustSelector(t, c.src).stateReads(func(on []*Compound, states dom.State, sibling bool) {
			var texts []string
			for _, c := range on {
				texts = append(texts, c.text)
			}
			way := below
			if sibling {
				way = beside
			}
			got = append(got, fmt.Sprint(strings.Join(texts, "|"), " ", uint16(states), " ", way))
		})
		slices.Sort(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: reads %q, want %q", c.src, got, c.want)
		}
	}
}

func TestASheetIndexesItsStateReads(t *testing.T) {
	x := sheet(t, `
a:hover, .b:focus { color: #f00 }
.menu:hover .item, .tab:active + .tab { color: #f00 }
@media (min-width: 10px) { input:checked ~ label span { color: #f00 } }
`).indexed()
	if x.aboveStates != dom.Hover || x.besideStates != dom.Active|dom.Checked {
		t.Errorf("above %b, beside %b", x.aboveStates, x.besideStates)
	}
	var got []string
	for _, r := range x.reads {
		got = append(got, fmt.Sprint(r.on[0].text, " -> ", r.subject.text, " rule ", r.rule, " beside ", r.beside))
	}
	want := []string{
		".menu:hover -> .item rule 1 beside false",
		".tab:active -> .tab rule 1 beside true",
		"input:checked -> span rule 2 beside true",
	}
	if !slices.Equal(got, want) {
		t.Errorf("reads %q, want %q", got, want)
	}
	if ua := userAgent().indexed(); ua.aboveStates&dom.Hover != 0 || ua.besideStates != 0 {
		t.Errorf("the user agent's sheet reads hover above (%b) or a state beside (%b): every Document would pay for it",
			ua.aboveStates, ua.besideStates)
	}
}
