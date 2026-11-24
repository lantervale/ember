package css //nolint:testpackage // tests the unexported serialisation of a specified value

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestASpecifiedValueIsWrittenAsChromeWritesIt(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"margin-inline", "0", "0px"},
		{"line-height", "0", "0"},
		{"color", "LinkText", "linktext"},
		{"color", "#ffe", "rgb(255, 255, 238)"},
		{"color", "rgba(0,0,0,.5)", "rgba(0, 0, 0, 0.5)"},
		{"font-size", "0.81250EM", "0.8125em"},
		{"font-family", `"Go Mono" ,monospace`, `"Go Mono", monospace`},
		{"font-family", `"Go", Go  Mono, serif`, `Go, "Go Mono", serif`},
		{"transition", "color  1s,opacity 2s", "color 1s, opacity 2s"},
		{"content", `'a"b'`, `"a\"b"`},
	} {
		assert.Equal(t, c.want, specified(c.name, componentsOf(c.src)), "%s: %s", c.name, c.src)
	}
}

func TestASideShorthandIsDeclaredAsItsSides(t *testing.T) {
	sheet, _ := Parse("t", "p { margin: 0 1px 2px; padding-block: 3px; border: 1px solid; }")
	var got []Declared
	for i := range sheet.Rules[0].Decls {
		got = declare(got, &sheet.Rules[0].Decls[i])
	}
	assert.Equal(t, []Declared{
		{Name: "margin-top", Value: "0px"}, {Name: "margin-right", Value: "1px"},
		{Name: "margin-bottom", Value: "2px"}, {Name: "margin-left", Value: "1px"},
		{Name: "padding-block-start", Value: "3px"}, {Name: "padding-block-end", Value: "3px"},
		{Name: "border", Value: "1px solid"},
	}, got)
}

func TestALaterDeclarationTakesTheFirstOnesPlace(t *testing.T) {
	sheet, _ := Parse("t", "p { color: red; font-size: 50%; color: blue !important; }")
	var got []Declared
	for i := range sheet.Rules[0].Decls {
		got = declare(got, &sheet.Rules[0].Decls[i])
	}
	assert.Equal(t, []Declared{{Name: "color", Value: "blue", Important: true},
		{Name: "font-size", Value: "50%", Percent: true}}, got)
}
