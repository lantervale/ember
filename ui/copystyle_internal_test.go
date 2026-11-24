package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACopiedStyleFoldsTheSidesItHoldsWhole(t *testing.T) {
	b := block{{name: "box-sizing", value: "border-box"}, {name: "margin-top", value: "0px"},
		{name: "margin-right", value: "1px"}, {name: "margin-bottom", value: "0px"}, {name: "margin-left", value: "1px"},
		{name: "padding-block-start", value: "1em"}, {name: "padding-block-end", value: "1em"},
		{name: "padding-top", value: "2px"}, {name: "color", value: "red", important: true}}
	assert.Equal(t, "box-sizing: border-box; margin: 0px 1px; padding-block: 1em; padding-top: 2px; "+
		"color: red !important;", b.text())
	b.set(decl{name: "margin-left", value: "3px"})
	assert.Contains(t, b.text(), "margin: 0px 1px 0px 3px;", "four sides")
	b.set(decl{name: "margin-left", value: "1px", important: true})
	assert.Contains(t, b.text(), "margin-top: 0px; margin-right: 1px; margin-bottom: 0px; margin-left: 1px !important;",
		"sides of two importances stay apart")
}

func TestRemovingAShorthandRemovesItsLonghands(t *testing.T) {
	b := block{{name: "text-decoration-thickness", value: "initial"}, {name: "text-decoration-style", value: "initial"},
		{name: "color", value: "red"}, {name: "margin-top", value: "0px"}}
	b.remove("text-decoration")
	b.remove("margin")
	assert.Equal(t, "color: red;", b.text())
	b = block{{name: "margin", value: "0px"}, {name: "white-space", value: "pre"}}
	b.remove("margin-top")
	b.remove("white-space-collapse")
	assert.Empty(t, b, "a shorthand goes with any of its longhands")
}
