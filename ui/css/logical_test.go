package css_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/css"
)

func TestLogicalPropertiesAreThePhysicalOnesOfLeftToRightText(t *testing.T) {
	sheet, warns := css.Parse("", `p { margin-inline-start: 1px; margin-block-end: 2px; padding-inline-end: 3px;
inset-block-start: 4px; inset-inline-end: 5px; inline-size: 6px; max-block-size: 7px; min-inline-size: 8px;
border-inline-start: 1px solid red; border-block-end-width: 2px; border-start-end-radius: 3px;
margin-inline: auto; padding-block: 1px 2px; inset-inline: 0 }`)
	assert.Empty(t, warns)
	require.Len(t, sheet.Rules, 1)
	var names []string
	for _, d := range sheet.Rules[0].Decls {
		names = append(names, d.Name)
	}
	assert.Equal(t, []string{"margin-left", "margin-bottom", "padding-right", "top", "right", "width", "max-height",
		"min-width", "border-left", "border-bottom-width", "border-top-right-radius",
		"margin-inline", "padding-block", "inset-inline"}, names)

	_, warns = css.Parse("", `p { margin-inline: 1px 2px 3px; inset-block: red }`)
	assert.Len(t, warns, 2)
}
