package css //nolint:testpackage // holds the unexported warnings helper the other tests use

import (
	"strings"
	"testing"
	"testing/fstest"
)

func warnings(ws []Warning) string {
	var s []string
	for _, w := range ws {
		s = append(s, w.Error())
	}
	return strings.Join(s, "\n")
}

func TestParseRules(t *testing.T) {
	sheet, warns := Parse("tool.css", `
.props { display: grid; grid-template-columns: max-content 1fr; gap: 6px 12px; }
.props > *, #main { min-width: 0 !important; COLOR: var(--Ink) ! IMPORTANT }
:root { --Ink: #f7efdd; --empty:; }
`)
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	if len(sheet.Rules) != 3 {
		t.Fatalf("got %d rules", len(sheet.Rules))
	}
	r := sheet.Rules[0]
	if r.Pos != (Pos{2, 1}) || len(r.Selectors) != 1 || r.Selectors[0].String() != ".props" {
		t.Errorf("rule 0: %v %v", r.Pos, r.Selectors)
	}
	var decls []string
	for _, d := range r.Decls {
		decls = append(decls, d.String())
	}
	if got := strings.Join(decls, "; "); got != "display: grid; grid-template-columns: max-content 1fr; gap: 6px 12px" {
		t.Errorf("decls: %s", got)
	}
	r = sheet.Rules[1]
	if len(r.Selectors) != 2 || r.Selectors[0].String() != ".props > *" || r.Selectors[1].String() != "#main" {
		t.Errorf("selectors: %v", r.Selectors)
	}
	if d := r.Decls[0]; !d.Important || d.String() != "min-width: 0 !important" {
		t.Errorf("important: %v", d)
	}
	if d := r.Decls[1]; d.Name != "color" || !d.Important || join(d.Value) != "var(--Ink)" {
		t.Errorf("color: %v", d)
	}
	if d := sheet.Rules[2].Decls[0]; d.Name != "--Ink" || join(d.Value) != "#f7efdd" {
		t.Errorf("custom: %v", d)
	}
}

func TestParseWarnings(t *testing.T) {
	cases := []struct{ src, want string }{
		{"p { zoom: 2; color: #f00 }", `t.css:1:5: zoom is not supported; use transform: scale()`},
		{"p {\n  colour: red;\n}", `t.css:2:3: unknown property "colour"`},
		{"p { -webkit-transform: none }", `t.css:1:5: vendor-prefixed property "-webkit-transform" is not supported`},
		{"p { word-wrap: break-word }", `t.css:1:5: word-wrap is the old name of overflow-wrap`},
		{"@import url(a.css);", `t.css:1:1: @import is not supported`},
		{"@supports (display: grid) { p { color: #f00 } }", `t.css:1:1: @supports is not supported`},
		{"@layer base { p { color: #f00 } }", `t.css:1:1: @layer is not supported`},
		{"@-moz-document url-prefix() { p { color: #f00 } }", `t.css:1:1: @-moz-document is not supported`},
		{"p { -moz-transform: none }", `t.css:1:5: vendor-prefixed property "-moz-transform" is not supported`},
		{"p { display: -webkit-flex }", `t.css:1:14: display: expected block, inline, inline-block, flex, ` +
			`inline-flex, grid, inline-grid, none, contents, list-item, flow-root, found "-webkit-flex"`},
		{"p::-webkit-scrollbar { color: #f00 }",
			`t.css:1:4: the vendor-prefixed pseudo-element ::-webkit-scrollbar is not supported; rule dropped`},
		{"a:-webkit-any-link { color: #f00 }",
			`t.css:1:3: the :-webkit-any-link pseudo-class is not supported; rule dropped`},
		{"p { color red }", `t.css:1:5: expected ':' after "color"`},
		{"p { color: }", `t.css:1:5: color has no value`},
		{"p { 12px: red }", `t.css:1:5: expected a property name, found "12px"`},
		{"p[href^=x] { color: #f00 }", `t.css:1:7: only [attr] and [attr="value"] are supported, not ^=; rule dropped`},
		{"p + span { color: #f00 }", `t.css:1:3: the "+" combinator is not supported; rule dropped`},
		{"input:checked ~ label { color: #f00 }", `t.css:1:15: the "~" combinator is not supported; rule dropped`},
		{"p::nope { color: #f00 }", `t.css:1:4: unknown pseudo-element ::nope; rule dropped`},
		{"p::before:hover { color: #f00 }",
			`t.css:1:3: ::before must end its selector: nothing may follow a pseudo-element; rule dropped`},
		{"p:after span { color: #f00 }",
			`t.css:1:3: :after must end its selector: nothing may follow a pseudo-element; rule dropped`},
		{"p::first-line { color: #f00 }",
			`t.css:1:4: ::first-line is not supported: ui styles no line of a block apart; rule dropped`},
		{"p:first-letter { color: #f00 }",
			`t.css:1:3: ::first-letter is not supported: ui styles no letter of a block apart; rule dropped`},
		{"input::file-selector-button { color: #f00 }",
			`t.css:1:8: ::file-selector-button is not supported: ui has no file input; rule dropped`},
		{"p::part { color: #f00 }", `t.css:1:4: ::part takes an argument in ( ); rule dropped`},
		{"p::before(x) { color: #f00 }", `t.css:1:4: ::before takes no ( ); rule dropped`},
		{"p::marker { margin: 0 }", `t.css:1:13: ::marker does not take margin`},
		{"p::before { content: counter(c, lower-greek) }",
			`t.css:1:22: content: counter(): the counter style: expected none, disc, circle, square, decimal, ` +
				`decimal-leading-zero, lower-alpha, upper-alpha, lower-roman, upper-roman, disclosure-open or ` +
				`disclosure-closed, found "lower-greek"`},
		{"p { counter-reset: reversed(c) }",
			`t.css:1:20: counter-reset: reversed() is not supported; an ol counts down with its reversed attribute`},
		{"p { quotes: \"a\" }", `t.css:1:13: quotes: quotes takes auto, none or pairs of strings`},
		{"p::before { content: \"a\" / }", `t.css:1:22: content: content ends with /`},
		{"p:nth-of-type(2) { color: #f00 }", `t.css:1:3: the :nth-of-type() pseudo-class is not supported; rule dropped`},
		{"p:target { color: #f00 }", `t.css:1:3: the :target pseudo-class is not supported; rule dropped`},
		{"p, { color: #f00 }", `t.css:1:2: empty selector; rule dropped`},
		{"p > { color: #f00 }", `t.css:1:3: selector ends with a combinator; rule dropped`},
		{"p", `t.css:1:1: rule has no { block }`},
	}
	for _, c := range cases {
		_, warns := Parse("t.css", c.src)
		if got := warnings(warns); got != c.want {
			t.Errorf("Parse(%q)\n got %s\nwant %s", c.src, got, c.want)
		}
	}
}

func TestParseKeepsGoodDeclarations(t *testing.T) {
	sheet, warns := Parse("t.css", "p { zoom: 2; color: #f00; } span { margin: 0 }")
	if len(warns) != 1 {
		t.Fatalf("warnings: %s", warnings(warns))
	}
	if len(sheet.Rules) != 2 || len(sheet.Rules[0].Decls) != 1 || sheet.Rules[0].Decls[0].Name != "color" {
		t.Errorf("rules: %+v", sheet.Rules)
	}
}

func TestParseInline(t *testing.T) {
	decls, warns := ParseInline("width: 50%; ; color: #f00 !important; zoom: 2")
	if got := warnings(warns); got != `style:1:39: zoom is not supported; use transform: scale()` {
		t.Errorf("warnings: %s", got)
	}
	if len(decls) != 2 || decls[0].String() != "width: 50%" || !decls[1].Important {
		t.Errorf("decls: %v", decls)
	}
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{"ui/tool.css": {Data: []byte("p {\n  zoom: 2;\n}")}}
	sheet, warns, err := Load(fsys, "ui/tool.css")
	if err != nil {
		t.Fatal(err)
	}
	if sheet.Name != "ui/tool.css" || warnings(warns) != "ui/tool.css:2:3: zoom is not supported; use transform: scale()" {
		t.Errorf("got %q, %s", sheet.Name, warnings(warns))
	}
	if _, _, err = Load(fsys, "missing.css"); err == nil {
		t.Error("missing file: no error")
	}
}

func TestPropertiesForChromeTravelInTheSheet(t *testing.T) {
	sheet, warns := Parse("t.css", "textarea { resize: none; appearance: textfield }")
	if len(warns) > 0 {
		t.Fatalf("warnings:\n%s", warnings(warns))
	}
	if got := sheet.String(); got != "textarea { resize: none; appearance: textfield; }\n" {
		t.Errorf("sheet: %q", got)
	}
}
