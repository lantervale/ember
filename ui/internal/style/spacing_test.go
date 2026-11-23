package style_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestWordSpacingFollowsSpacesAndNoBreakSpacesOnly(t *testing.T) {
	for _, r := range []rune{' ', 0xa0} {
		assert.True(t, style.WordSeparator(r), "%U", r)
	}
	for _, r := range []rune{
		'a', '\t', '\n', 0x1361, 0x2003, 0x2009, 0x202f, 0x200b, 0x3000, 0x10100, 0x10101, 0x1039f, 0x1091f,
	} {
		assert.False(t, style.WordSeparator(r), "%U", r)
	}
}

func TestAWordGapIsItsLengthAndItsPercentageOfTheFontSize(t *testing.T) {
	st := style.Initial()
	st.Font.Size = 20
	assert.Zero(t, st.WordGap(), "normal")
	st.WordSpacing = style.Spacing{Px: 3}
	assert.Equal(t, float32(3), st.WordGap())
	st.WordSpacing = style.Spacing{Px: -2, Percent: 50}
	assert.Equal(t, float32(8), st.WordGap())
	st.Font.Size = 40
	assert.Equal(t, float32(18), st.WordGap(), "the percentage is of the element's own font size")
}

func TestSpacingIsLetterSpacingAfterEachCharacterAndWordSpacingAfterEachSeparator(t *testing.T) {
	st := style.Initial()
	st.Font.Size = 10
	assert.Zero(t, st.Spacing("a b\u00a0c"), "normal and normal")
	st.LetterSpacing = style.Spacing{Px: 2}
	assert.Equal(t, float32(10), st.Spacing("a b\u00a0c"))
	st.WordSpacing = style.Spacing{Px: 5}
	assert.Equal(t, float32(20), st.Spacing("a b\u00a0c"))
	assert.Equal(t, float32(14), st.Spacing("  "), "each space of a run")
	assert.Equal(t, float32(6), st.Spacing("a\tb"), "a tab is no separator")
	st.LetterSpacing = style.Spacing{}
	assert.Equal(t, float32(5), st.Spacing("é "), "characters, not bytes")
	st.WordSpacing = style.Spacing{Px: -5}
	assert.Equal(t, float32(-10), st.Spacing("a b c"))
}
