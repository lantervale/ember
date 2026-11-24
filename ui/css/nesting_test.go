package css //nolint:testpackage // reads the rules a nested sheet is read as, and computed styles off dom

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/ui/internal/dom"
)

func selectorsOf(s *Sheet) []string {
	var out []string
	for _, r := range s.Rules {
		var sels []string
		for _, sel := range r.Selectors {
			sels = append(sels, sel.String())
		}
		var decls []string
		for _, d := range r.Decls {
			decls = append(decls, d.String())
		}
		out = append(out, strings.Join(sels, ", ")+" {"+strings.Join(decls, "; ")+"}")
	}
	return out
}

func TestNestedRulesAreReadRelativeToTheirRule(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{".a { color: red; .b { color: blue } }", []string{".a {color: red}", ".a .b {color: blue}"}},
		{".a { > .b { color: blue } }", []string{".a {}", ".a > .b {color: blue}"}},
		{".a { + .b, ~ .c { color: blue } }", []string{".a {}", ".a + .b, .a ~ .c {color: blue}"}},
		{".a { &:hover { color: blue } }", []string{".a {}", ".a:hover {color: blue}"}},
		{".a { .b & { color: blue } }", []string{".a {}", ".b .a {color: blue}"}},
		{"div { .x& { color: blue } }", []string{"div {}", ".x:is(div) {color: blue}"}},
		{"div { &.x { color: blue } }", []string{"div {}", "div.x {color: blue}"}},
		{".a, #b { .c { color: blue } }", []string{".a, #b {}", ":is(.a, #b) .c {color: blue}"}},
		{".a > .b { &:hover { color: blue } }", []string{".a > .b {}", ":is(.a > .b):hover {color: blue}"}},
		{".a { :not(&) { color: blue } }", []string{".a {}", ":not(.a) {color: blue}"}},
		{".a { .b { & > .c { color: blue } } }", []string{".a {}", ".a .b {}", ":is(.a .b) > .c {color: blue}"}},
		{".a { .b {} color: red; }", []string{".a {}", ".a .b {}", ".a {color: red}"}},
		{".a { color: red; @media screen { color: blue } color: green }",
			[]string{".a {color: red}", ".a {color: blue}", ".a {color: green}"}},
		{".a { span:first-child { color: blue } }", []string{".a {}", ".a span:first-child {color: blue}"}},
		{".a { --x: { b: c }; color: red }", []string{".a {--x: { b: c }; color: red}"}},
		{"& .j { color: red }", []string{":scope .j {color: red}"}},
		{".a { &::before { content: 'x' } }", []string{".a {}", ".a::before {content: \"x\"}"}},
	}
	for _, c := range cases {
		s := sheet(t, c.src)
		if got := selectorsOf(s); strings.Join(got, " | ") != strings.Join(c.want, " | ") {
			t.Errorf("%s\n got %s\nwant %s", c.src, strings.Join(got, " | "), strings.Join(c.want, " | "))
		}
	}
}

func TestNestedRulesDropWhatChromeDrops(t *testing.T) {
	cases := []struct{ src, want string }{
		{".a { color: red; ..b { color: blue } & .c { color: green } }",
			`t.css:1:18: expected a class name after '.'; rule dropped`},
		{".a { color: red .x { } }", `t.css:1:6: ui has no <color> element; rule dropped`},
		{".a { &span { color: blue } }", `t.css:1:7: a type selector must come first in a compound; rule dropped`},
		{".a { @font-face { font-family: x } color: red }", `t.css:1:6: @font-face is not supported inside a style rule`},
		{".a { .b; color: red }", `t.css:1:6: expected a property name, found "."`},
		{".a { color red }", `t.css:1:6: expected ':' after "color"`},
	}
	for _, c := range cases {
		s, warns := Parse("t.css", c.src)
		if got := warnings(warns); got != c.want {
			t.Errorf("%s\n got %s\nwant %s", c.src, got, c.want)
		}
		if len(s.Rules) == 0 || s.Rules[0].Selectors[0].String() != ".a" {
			t.Errorf("%s: the rule itself is kept: %s", c.src, selectorsOf(s))
		}
	}
}

func TestNestedRulesWeighAsChromes(t *testing.T) {
	a := el(dom.Div, "o1", "a", "c")
	e := el(dom.Div, "o3", "e", "f")
	h := el(dom.Div, "o5", "h")
	r := el(dom.Div, "o13", "r", "s")
	root := page(a, e, h, r)
	Compute(root, []*Sheet{sheet(t, `
.a, #b { .zz { color: #f00 } color: #010101; }
.a.c { color: #020202; }
.e, #nothing { & { color: #040404 } }
.e.f { color: #050505 }
.h { .x {} color: #080808; }
.h { color: #090909; }
.r, #b2 { @media screen { color: #141414 } }
.r.s { color: #151515 }
`)})
	for _, c := range []struct {
		n    *dom.Node
		want string
		why  string
	}{
		{a, "#020202", "declarations after a nested rule weigh as the rule's own selector, not as :is() of its list"},
		{e, "#040404", "& weighs as :is() of its rule's list"},
		{h, "#090909", "declarations after a nested rule stand there in the order"},
		{r, "#151515", "declarations in a nested @media weigh as the rule's own selector"},
	} {
		if got := styleOf(c.n).Color; got != hex(c.want) {
			t.Errorf("#%s: %s: got %v", c.n.ID(), c.why, got)
		}
	}
}

func TestNestedSheetsWriteBackAsWritten(t *testing.T) {
	src := `@media screen { .a { color: red; .b { color: blue } } }
p { color: red }
@scope (.c) to (.d) { .e { color: red } }`
	s := sheet(t, src)
	want := `@media screen { .a { color: red; .b { color: blue } } }
p { color: red; }
@scope (.c) to (.d) { .e { color: red } }
`
	if got := s.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	joined, _ := Join("t.css", "", []*Sheet{sheet(t, "p { color: red }"), s})
	if got := joined.String(); got != "p { color: red; }\n"+want {
		t.Errorf("joined:\n%s", got)
	}
}

func TestScopeRulesApplyFromTheirRootsToTheirEnds(t *testing.T) {
	b2 := el(dom.Div, "b2", "b2")
	b3 := el(dom.Div, "b3", "b3")
	b3in := el(dom.Div, "b3in", "b3")
	b4 := el(dom.Div, "b4", "b4")
	b5 := el(dom.Div, "b5", "b5")
	c6 := el(dom.Div, "c6", "c6")
	c7 := el(dom.Div, "c7", "c7")
	s9a, s9b := el(dom.Div, "s9a", "s9"), el(dom.Div, "s9b", "s9")
	c13a, c13b := el(dom.Div, "c13a", "c13"), el(dom.Div, "c13b", "c13")
	c15, c16 := el(dom.Div, "c15", "c15"), el(dom.Div, "c16", "c16")
	c17a, c17b := el(dom.Div, "c17a", "c17"), el(dom.Div, "c17b", "c17")
	p12 := el(dom.Div, "p12", "p")
	c14x := el(dom.Div, "c14x", "c14x")
	s1 := el(dom.Div, "s1")
	root := page(
		s1,
		with(el(dom.Div, "", "s2"), b2),
		with(el(dom.Div, "", "s3"), with(el(dom.Div, "", "lim"), b3in), b3),
		with(el(dom.Div, "", "outer"), with(el(dom.Div, "", "s4"), b4)),
		with(el(dom.Div, "", "s5"), b5),
		with(el(dom.Div, "", "s6"), with(el(dom.Div, "", "b6"), c6)),
		with(el(dom.Div, "", "s7"), with(el(dom.Div, "", "b7"), c7)),
		with(s9a, s9b),
		with(el(dom.Div, "", "s13"), with(c13a, c13b)),
		with(el(dom.Div, "", "s15"), c15),
		with(el(dom.Div, "", "s16"), c16),
		with(el(dom.Div, "", "s17"), with(el(dom.Div, "", "l17"), c17a),
			with(el(dom.Div, ""), with(el(dom.Div, "", "l17"), c17b))),
		with(el(dom.Div, "", "q"), with(el(dom.Div, "", "o"), p12)),
		with(el(dom.Div, "n14"), c14x),
	)
	Compute(root, []*Sheet{sheet(t, `
div { color: #020202 }
@scope (#s1) { & { color: #010101 } }
@scope (.s2) { .b2 { color: #030303 } }
.b2 { color: #040404 }
@scope (.s3) to (.lim) { .b3 { color: #050505 } }
@scope (.s4) { .outer .b4 { color: #060606 } }
@scope (.s5) { body :scope .b5 { color: #070707 } }
@scope (.s6) { .b6 { & .c6 { color: #080808 } } }
@scope (.s7) { @scope (.b7) { .c7 { color: #090909 } } }
@scope (.s9) { .s9 { color: #0b0b0b } }
@scope (.s13) { > .c13 { color: #0d0d0d } }
@scope (.s15) to (.s15) { .c15 { color: #0f0f0f } }
@scope (.s16) to (:scope) { .c16 { color: #101010 } }
@scope (.s17) to (& > .l17) { .c17 { color: #111111 } }
@scope (.o) { .p { color: #121212 } }
@scope (.q) { .p { color: #131313 } }
#n14 { @scope (.c14x) { :scope { color: #1f1f1f } } }
`)})
	for _, c := range []struct {
		n    *dom.Node
		want string
	}{
		{s1, "#020202"}, {b2, "#030303"}, {b3, "#050505"}, {b3in, "#020202"}, {b4, "#020202"}, {b5, "#070707"},
		{c6, "#080808"}, {c7, "#090909"}, {s9a, "#020202"}, {s9b, "#0b0b0b"}, {c13a, "#0d0d0d"}, {c13b, "#020202"},
		{c15, "#0f0f0f"}, {c16, "#020202"}, {c17a, "#020202"}, {c17b, "#111111"}, {p12, "#121212"}, {c14x, "#1f1f1f"},
	} {
		if got := styleOf(c.n).Color; got != hex(c.want) {
			t.Errorf("#%s: got %v, want %s", c.n.ID(), got, c.want)
		}
	}
}

func TestDeeplyNestedScopesMatchInLinearTime(t *testing.T) {
	const depth = 90
	span := el(dom.Span, "leaf")
	src := "span { color: #0f0 }"
	n := with(el(dom.Div, "", "s"+strconv.Itoa(depth-1)), span)
	for i := depth - 1; i >= 0; i-- {
		src = "@scope (.s" + strconv.Itoa(i) + ") { " + src + " }"
		if i > 0 {
			n = with(el(dom.Div, "", "s"+strconv.Itoa(i-1)), n)
		}
	}
	done := make(chan struct{})
	go func() {
		Compute(page(n), []*Sheet{sheet(t, "span { color: #000 } "+src)})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("90 nested @scope rules did not match within 5 s")
	}
	if got := styleOf(span).Color; got != hex("#0f0") {
		t.Errorf("the innermost scope's span: %v", got)
	}
}

func TestScopeFollowsItsRootsClasses(t *testing.T) {
	kid := el(dom.Div, "kid", "kid")
	box := with(el(dom.Div, "box"), kid)
	root := page(box)
	sheets := []*Sheet{sheet(t, `div { color: #000 } @scope (.on) { .kid { color: #0f0 } }`)}
	Compute(root, sheets)
	if got := styleOf(kid).Color; got != hex("#000") {
		t.Fatalf("out of scope: %v", got)
	}
	box.AddClass("on")
	Compute(root, sheets)
	if got := styleOf(kid).Color; got != hex("#0f0") {
		t.Errorf("a class making its parent a root brings the rule in: %v", got)
	}
}

func TestABrowsersNestedRulesAreReadAsChromes(t *testing.T) {
	cases := []struct{ src, rules, warns string }{
		{".a { @layer x { color: red } }", ".a {} | .a {color: red}", ""},
		{".a { @layer x; color: red }", ".a {} | .a {color: red}",
			"t.css:1:6: @layer without a block is dropped inside a style rule, as in Chrome"},
		{".a { @supports (display: grid) { color: red } @supports (display: nope) { color: blue } }",
			".a {} | .a {color: red}", ""},
		{"@supports selector(&:hover) { .a { color: red } }", ".a {color: red}", ""},
		{".a { blink & { color: red } & blink, & > .k { color: blue } }", ".a {} | .a > .k {color: blue}", ""},
		{".a { @-webkit-keyframes x { from { color: red } } color: blue }", ".a {color: blue}",
			"t.css:1:6: @-webkit-keyframes is not supported inside a style rule"},
	}
	for _, c := range cases {
		s, warns := asBrowser(c.src)
		if got := strings.Join(selectorsOf(s), " | "); got != c.rules || warns != c.warns {
			t.Errorf("%s\n got %s %q\nwant %s %q", c.src, got, warns, c.rules, c.warns)
		}
	}
}
