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

func TestTheTextBeforeAPieceShownRightToLeftIsOnItsRight(t *testing.T) {
	// בד גה ו read in that order, shown from the left.
	shown := []piece{
		{kind: word, text: "ו", whole: 1},
		{kind: gap, text: " ", whole: 1},
		{kind: word, text: "גה", whole: 1},
		{kind: edge, whole: 1},
		{kind: word, text: "בד", whole: 1},
	}
	lead, _ := besideShown(shown, 2, -1, 1, true)
	assert.Equal(t, "בד", lead)
	trail, _ := besideShown(shown, 2, 1, 1, true)
	assert.Equal(t, " ו", trail, "in the order it is read")
	lead, _ = besideShown(shown, 2, -1, 1, false)
	assert.Equal(t, "ו ", lead, "left to right, the text on its left")
}

func TestALinesTailFollowsThePieceHoldingItsLastWord(t *testing.T) {
	tail := lineTail{whole: 1, run: 2, at: 4}
	assert.True(t, tail.in([]piece{{kind: word, run: 1, at: 4}, {kind: word, run: 2, at: 4}}))
	assert.False(t, tail.in([]piece{{kind: gap, run: 2, at: 4}, {kind: word, run: 2, at: 0}}))
}
