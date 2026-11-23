package css //nolint:testpackage // tests the unexported alignment grammar

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestAlignmentValuesCarryTheirBaselineAndOverflowPosition(t *testing.T) {
	cases := []struct {
		src  string
		want style.Align
	}{
		{"last baseline", style.AlignLastBaseline},
		{"first baseline", style.AlignBaseline},
		{"safe center", style.AlignCenter | style.AlignSafe},
		{"unsafe flex-end", style.AlignFlexEnd | style.AlignUnsafe},
		{"safe anchor-center", style.AlignAnchorCenter},
	}
	for _, c := range cases {
		v := items(componentsOf(c.src))
		assert.NoError(t, justifySelf.check(v), c.src)
		assert.Equal(t, c.want, alignOf(v), c.src)
	}
	for _, src := range []string{"safe baseline", "baseline last", "center safe", "safe normal", "first"} {
		assert.Error(t, alignSelf.check(items(componentsOf(src))), src)
	}
	assert.Error(t, alignContent.check(items(componentsOf("last baseline"))))
	assert.Error(t, justifyContent.check(items(componentsOf("baseline"))))
}
