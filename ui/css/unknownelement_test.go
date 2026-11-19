package css_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/css"
)

func TestABrowsersSheetKeepsARuleNamingAnElementUIHasNot(t *testing.T) {
	sheet, warns := css.Parse("", `html, applet, div, object.wide > p, video:not(.x) { margin: 0 }
video, e-page-transition { width: 1px }
p:not(video) { height: 1px }
p:is(video, div) { top: 1px }`, css.Browser())
	require.Len(t, warns, 1)
	assert.Contains(t, warns[0].Msg, "ui has no <video> element; rule dropped")
	assert.Equal(t, 3, warns[0].Pos.Line)
	require.Len(t, sheet.Rules, 2)
	assert.Len(t, sheet.Rules[0].Selectors, 2)
	assert.Equal(t, "html", sheet.Rules[0].Selectors[0].String())
	assert.Equal(t, "div", sheet.Rules[0].Selectors[1].String())
	assert.Equal(t, 4, sheet.Rules[1].Pos.Line)
}

func TestAScreensSheetRefusesARuleNamingAnElementUIHasNot(t *testing.T) {
	sheet, warns := css.Parse("", `html, applet, div { margin: 0 }`)
	require.Len(t, warns, 1)
	assert.Contains(t, warns[0].Msg, "ui has no <applet> element; rule dropped")
	assert.Empty(t, sheet.Rules)
}
