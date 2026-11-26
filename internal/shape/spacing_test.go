package shape_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/internal/shape"
)

func TestLetterSpacingFollowsNoLetterOfACursiveScript(t *testing.T) {
	for _, c := range []struct {
		before, text string
		want         int
	}{
		{"", "abc", 3},
		{"", "áb", 2},
		{"", "بتب", 0},
		{"", "ب ب", 1},
		{"", "ب ب", 1},
		{"", "ب ب", 0},
		{"", "ب.a", 1},
		{"", "a.ب", 2},
		{"", "1ب", 0},
		{"a ", "1ب", 1},
		{"ب ", "1a", 1},
		{"", "ܫܠܡܐ ᠮᠣᠩ ߒߞߏ ࡀࡁࡂ ꡀꡁꡂ 𐴀𐴁𐴂", 5},
		{"", "שלום", 4},
		{"", "𞤀𞤣𞤤", 3},
	} {
		assert.Equal(t, c.want, shape.LetterSpaced(c.before, c.text), "%q after %q", c.text, c.before)
	}
}

func TestLetterSpacingFollowsNoFormatCharacter(t *testing.T) {
	for _, s := range []string{"a\u202eb\u202cc", "a\u2067b\u2069c", "a\u200bb\u200ec", "a\u2060b\ufeffc"} {
		assert.Equal(t, 3, shape.LetterSpaced("", s), "%q: its controls take no spacing", s)
	}
}
