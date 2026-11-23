package fonts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/fonts"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAFacesMetricsTellItsOwnUnderlineAndEmBox(t *testing.T) {
	m := fonts.Builtin().Metrics(style.Font{Family: []string{"Go"}, Size: 40, Weight: 400})
	assert.True(t, m.Underlined)
	assert.InDelta(t, 275.0*40/2048, m.Underline, 1e-4, "the post table's position, the line's top below the baseline")
	assert.InDelta(t, 50.0*40/2048, m.UnderlineThickness, 1e-4)
	assert.InDelta(t, 1579.0/(1579+395), m.EmAscent, 1e-6, "OS/2's typographic ascender over its em")
}
