//go:build chrome

package css_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// tree builds elements from lines of tag.class.class#id, each nested in
// the line before it indented less; span elements hold a letter.
func tree(d *ui.Document, src string) {
	type open struct {
		n     *ui.Node
		depth int
	}
	stack := []open{{d.Body(), -1}}
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		depth := len(line) - len(strings.TrimLeft(line, "\t "))
		spec, id, _ := strings.Cut(strings.TrimSpace(line), "#")
		parts := strings.Split(spec, ".")
		var args []ui.Arg
		if len(parts) > 1 {
			args = append(args, ui.Class(parts[1:]...))
		}
		if id != "" {
			args = append(args, ui.ID(id))
		}
		n := ui.Div(args...)
		if parts[0] == "span" {
			n = ui.Span(append(args, ui.Text("x"))...)
		}
		for stack[len(stack)-1].depth >= depth {
			stack = stack[:len(stack)-1]
		}
		stack[len(stack)-1].n.Append(n)
		stack = append(stack, open{n, depth})
	}
}

// colours are the properties the nesting suites compare.
//
//nolint:gochecknoglobals // the properties compared, a test table
var colours = map[string]func(*style.Computed) string{
	"color": compared["color"], "background-color": compared["background-color"],
}

// TestNestedRulesStyleAsChromes styles a page by nested style rules:
// relative selectors, & anywhere in a compound and in functions, & of a
// list weighing as :is(), declarations after nested rules in their
// place, and conditional rules nested in style rules.
func TestNestedRulesStyleAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { color: #020202 }
.a, #b { .zz { color: #f00 } color: #010101; }
.a.c { color: #030303; }
.e, #nothing { & { color: #040404 } }
.e.f { color: #050505 }
.g { color: #060606; .x { } }
.g { color: #070707; }
.h { .x {} color: #080808; }
.h { color: #090909; }
& .j { color: #0b0b0b }
.l { span:first-child { color: #0d0d0d } }
.r, #b2 { @media screen { color: #141414 } }
.r.s { color: #151515 }
.z2 { color: #1b1b1b; & .ok { color: #1c1c1c } }
.z4 { --x: { a: b }; color: #1e1e1e }
.z5 { .z6 { & > .z7 { color: #1f1f1f } } }
.z8 { > .z9 { color: #202020 } .z9 { background-color: #010203 } }
.n1 { :is(&) { color: #121212 } }
.n2 { .q2:not(&) { color: #131313 } }
.n3 { .b3x & { color: #141415 } }
div.n4 { .x& { color: #151516 } }
:scope .n7 { color: #181818 }
@media screen { .n10 { .c10 { color: #1b1b1c } } }
.n11 { :has(&) { background-color: #1c1c1d } }
.n12, .m12 { & + & { color: #1d1d1e } }
.n13 { @media screen { .c13x { color: #1e1e1f } } }
.n14 { @media print { color: #f00 } @media not print { background-color: #0f0 } }
.n15 { &.on { color: #212121 } &:not(.on) { color: #222222 } }
.n16 { & & { color: #232323 } }
`))
	tree(d, `
div.a.c#o1
div.d
	span#o2
div.e.f#o3
div.g#o4
div.h#o5
div.j#o7
div.l
	span#o9
div.r.s#o13
div.z1#o17
div.z2#o18
	span.ok#o19
div.z4#o21
div.z5#o22
	div.z6#o23
		div.z7#o24
div.z8#o25
	div.z9#o26
div.n1#n1
div.q2#q2
div.b3x
	div.n3#n3
div.n4#n4
div.x.n4#n4b
div.n7#n7
div.n10
	div.c10#c10
div.n11
	div#n11a
div.n12#n12
div.m12#m12
div.n13
	div.c13x#c13x
div.n14#n14
div.n15.on#n15a
div.n15#n15b
div.n16
	div.n16#n16b
`)
	matchChromeProps(t, d, colours, 0)
}

// TestScopeRulesStyleAsChromes styles a page by @scope rules: roots and
// ends, :scope and & in them, selectors relative to the root, scopes
// nested in scopes and in style rules, and the nearer root winning a tie
// of specificity over the order.
func TestScopeRulesStyleAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { color: #020202 }
@scope (#s1) { & { color: #010101 } }
@scope (.s2) { .b2 { color: #030303 } }
.b2 { color: #040404 }
@scope (.s3) to (.lim) { .b3 { color: #050505 } }
@scope (.s4) { .outer .b4 { color: #060606 } }
@scope (.s5) { body :scope .b5 { color: #070707 } }
@scope (.s6) { .b6 { & .c6 { color: #080808 } } }
@scope (.s7) { @scope (.b7) { .c7 { color: #090909 } } }
@scope (.s8) { background-color: #0a0a0a }
@scope (.s9) { .s9 { color: #0b0b0b } }
@scope (.s10) to (.l10) { :scope .l10 { color: #0c0c0c } }
@scope (.s11) { :scope { color: #0d0d0d } }
.s11 { color: #0e0e0e }
@scope (#s12) { :scope { background-color: #030303 } }
#s12 { background-color: #040404 }
@scope (.s13) { > .c13 { color: #0d0d0e } }
@scope (.s14) { + .c14 { color: #0e0e0f } }
@scope (.s15) to (.s15) { .c15 { color: #0f0f0f } }
@scope (.s16) to (:scope) { .c16 { color: #101010 } }
@scope (.s17) to (& > .l17) { .c17 { color: #111111 } }
@scope (.o) { .p { color: #121212 } }
@scope (.q) { .p { color: #131313 } }
.t { @scope (.u) { .v { color: #161616 } } }
#n14 { @scope (.c14x) { :scope { color: #171717 } & { background-color: #090909 } } }
@media screen { @scope (.s18) { .c18 { color: #181819 } } }
@scope (.s19, .t19) { :scope > .c19 { color: #191919 } }
@scope (.s20) { .c20:hover, .c20 { color: #202021 } }
.s21 { color: #212122; @scope (.c21) { color: #222223 } }
`))
	tree(d, `
div#s1
div.s2
	div.b2#b2
div.s3
	div.lim#lim3
		div.b3#b3in
	div.b3#b3
div.outer
	div.s4
		div.b4#b4
div.s5
	div.b5#b5
div.s6
	div.b6
		div.c6#c6
div.s7
	div.b7
		div.c7#c7
div.c7#c7out
div.s8#s8
	div#s8kid
div.s9.top#s9a
	div.s9#s9b
div.s10
	div.l10#l10
div.s11#s11
div#s12
div.s13
	div.c13#c13a
		div.c13#c13b
div.s14#s14
div.c14#c14
div.s15
	div.c15#c15
div.s16
	div.c16#c16
div.s17
	div.l17
		div.c17#c17a
	div
		div.l17
			div.c17#c17b
div.q
	div.o
		div.p#p12
div.t
	div.u
		div.v#v
div#n14
	div.c14x#c14x
div.s18
	div.c18#c18
div.t19
	div.c19#c19
div.s20
	div.c20#c20
div.s21#s21
	div.c21#c21
`)
	matchChromeProps(t, d, colours, 0)
}

// TestNestedRulesOfABrowsersSheetStyleAsChromes nests the rules only a
// browser's sheet reads, @supports and @layer, and what such a sheet
// drops without a word, in style rules.
func TestNestedRulesOfABrowsersSheetStyleAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`
div { color: #020202 }
.i { @layer foo { color: #0a0a0a } }
.i2 { color: #0b0b0b; @layer foo { color: #f00 } }
.y { color: #181818 }
.n8 { @supports (display: grid) { color: #191919 } @supports (display: nope) { color: #f00 } }
.n9 { @supports selector(&:has(a)) { background-color: #010101 } }
.w { blink & { color: #f00 } & blink, & > .k { color: #1a1a1a } }
.v { color: #1b1b1b; &:hover { color: #f00 } }
@layer low { .z { .zz { color: #f00 } } }
.z .zz { color: #1c1c1c }
@scope (blink) { .s { color: #f00 } }
`))
	tree(d, `
div.i#i
div.i2#i2
div.y#y
div.n8#n8
div.n9#n9
div.w#w
	div.k#k
div.v#v
div.z
	div.zz#zz
div.s#s
`)
	matchChromeProps(t, d, colours, 0)
}

// TestScopesAndNestedRulesRestyleAsChromes changes the classes that
// make roots, ends and parents of @scope and nested rules, in Chrome by
// a script and in ui through the tree.
func TestScopesAndNestedRulesRestyleAsChromes(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { color: #020202 }
@scope (.root) to (.end) { .kid { color: #010101 } }
.card { &.on .kid { background-color: #030303 } }
@scope (.near) { .kid { color: #040404 } }
`))
	tree(d, `
div#a
	div#b
		div.kid#k1
			div.kid#k2
div.card#c
	div.kid#k3
`)
	script := `document.getElementById("a").classList.add("root");
document.getElementById("k1").classList.add("end");
document.getElementById("b").classList.add("near");
document.getElementById("c").classList.add("on");`
	then := func() {
		for id, class := range map[string]string{"a": "root", "k1": "end", "b": "near", "c": "on"} {
			d.ByID(id).AddClass(class)
		}
		d.Layout(geom.V2(800, 600))
	}
	matchChromeShown(t, d, colours, 0, script, then, func(n *dom.Node) *style.Computed { return dom.Data(n).Style })
}
