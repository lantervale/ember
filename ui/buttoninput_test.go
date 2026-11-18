package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func TestAnInputOfAButtonTypeIsAButton(t *testing.T) {
	submit := ui.Input(ui.ID("a"), ui.Type("submit"), ui.Value("Search"))
	bare, reset := ui.Input(ui.ID("b"), ui.Type("submit")), ui.Input(ui.ID("c"), ui.Type("reset"))
	plain := ui.Input(ui.ID("d"), ui.Type("button"))
	off := ui.Input(ui.ID("e"), ui.Type("submit"), ui.Value("Off"), ui.Attr("disabled", ""))
	l := liveDoc(t, submit, bare, reset, plain, off)
	labels := []string{"Search", "Submit", "Reset", "", "Off"}
	assert.Equal(t, labels, l.roles(automation.RoleButton), "each shows its value, else what its type says")
	assert.Greater(t, bare.Box().W, plain.Box().W, "a submit without a value is as wide as its word")

	l.clickAt(submit)
	l.typed("x")
	l.frames(1)
	assert.Equal(t, []*ui.Node{submit}, l.clicks, "a click on it is a click")
	assert.Equal(t, labels, l.roles(automation.RoleButton), "what is typed over it changes nothing")

	l.doc.Focus(bare)
	l.frames(1)
	l.key(input.KeyEnter)
	assert.Equal(t, []*ui.Node{submit, bare}, l.clicks, "Enter clicks the one that has the focus")
	l.clickAt(off)
	assert.Equal(t, []*ui.Node{submit, bare}, l.clicks, "a disabled one clicks nothing")
}
