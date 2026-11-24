package ui

import "testing"

func TestSetDeclarationReplacesItsDeclarationAndKeepsTheRest(t *testing.T) {
	for _, c := range []struct{ from, want string }{
		{"", "width: 140px"},
		{"color: red", "color: red; width: 140px"},
		{"WIDTH: 3em; color: red;", "width: 140px; color: red"},
		{`background: url("a;b.png"); width: 1px`, `background: url("a;b.png"); width: 140px`},
		{"font-family: 'x;y'; min-width: 4px", "font-family: 'x;y'; min-width: 4px; width: 140px"},
	} {
		n := Div(Style(c.from))
		if c.from == "" {
			n = Div()
		}
		setDeclaration(n, "width", "140px")
		if got := n.Style(); got != c.want {
			t.Errorf("%q: %q, want %q", c.from, got, c.want)
		}
	}
}
