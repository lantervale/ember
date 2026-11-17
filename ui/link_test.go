package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func TestALinkWithAnHrefIsClickedByThePointerAndEnter(t *testing.T) {
	clicks := 0
	link := ui.A(ui.Attr("href", "/vaults"), ui.Text("The vaults"), ui.OnClick(func() { clicks++ }))
	plain := ui.A(ui.Text("No address"), ui.OnClick(func() { clicks += 100 }))
	l := liveDoc(t, ui.P(link, ui.Text(" and "), plain))
	l.clickAt(link)
	assert.Equal(t, 1, clicks, "a click on the link")
	l.clickAt(plain)
	assert.Equal(t, 1, clicks, "an a without an href is plain text")

	l.doc.Focus(link)
	assert.True(t, l.doc.Focused(link), "a link takes the focus")
	l.key(input.KeyEnter)
	assert.Equal(t, 2, clicks, "Enter on the focused link")
	assert.Contains(t, l.roles(automation.RoleButton), "The vaults", "automation presses it")
	l.doc.Focus(plain)
	assert.False(t, l.doc.Focused(plain))
}
