package layout //nolint:testpackage // tests unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTheTextBesideAPieceIsShapedWithItWithinItsRunAndReadRoundItBeyond(t *testing.T) {
	ps := []piece{
		{kind: word, text: "xy", whole: 1},
		{kind: atomic},
		{kind: word, text: "of", whole: 2},
		{kind: edge, whole: 2},
		{kind: word, text: "fi", whole: 2},
		{kind: gap, text: " ", whole: 2},
		{kind: word, text: "cello", whole: 2},
		{kind: edge},
		{kind: word, text: "ab", whole: 3},
	}
	lead, before := beside(ps, 4, -1, 2)
	assert.Equal(t, "of", lead)
	assert.Equal(t, "xy￼", before, "an atomic inline stands as an object replacement character")
	trail, after := beside(ps, 4, 1, 2)
	assert.Equal(t, " cell", trail, "as much as shaping reads")
	assert.Equal(t, "oab", after)
	trail, after = beside(ps, 6, 1, 2)
	assert.Empty(t, trail, "an edge with room cuts the run")
	assert.Equal(t, "ab", after)
}
