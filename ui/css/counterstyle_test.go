package css //nolint:testpackage // resolves the unexported counter styles

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterStyleRulesAreReadDescriptorByDescriptor(t *testing.T) {
	s, warns := Parse("t.css", `
@counter-style paren { system: extends lower-alpha; prefix: "("; suffix: ") "; bogus: 1; pad: x; }
@counter-style decimal { system: cyclic; symbols: "x"; }
@counter-style none { system: cyclic; symbols: "x"; }
@media (min-width: 10px) { @counter-style wide { system: fixed 3; symbols: a b; } }
`)
	require.Len(t, s.CounterStyles, 2)
	assert.Equal(t, "paren", s.CounterStyles[0].Name)
	assert.NotContains(t, s.CounterStyles[0].Descs, "pad", "a descriptor that doesn't check is dropped")
	var msgs []string
	for _, w := range warns {
		msgs = append(msgs, w.Msg)
	}
	assert.Equal(t, []string{
		"@counter-style: bogus is no descriptor",
		"@counter-style paren: pad: expected a whole number at least 0 and a symbol",
		"@counter-style decimal: decimal can't be redefined; the rule is dropped",
		`@counter-style: "none" is no counter style name; the rule is dropped`,
	}, msgs)
	text := s.String()
	assert.Contains(t, text, `@counter-style paren { prefix: "("; suffix: ") "; system: extends lower-alpha; }`)
	assert.Contains(t, text, "@media (min-width: 10px) { @counter-style wide { symbols: a b; system: fixed 3; } }")
	again, warns := Parse("again.css", text)
	assert.Empty(t, warns)
	assert.Len(t, again.CounterStyles, 2, "the sheet's text reads back the same")
}

func TestCounterStylesResolveWhatTheyExtendAndFallBack(t *testing.T) {
	s, _ := Parse("t.css", `
@counter-style paren { system: extends lower-alpha; prefix: "("; suffix: ") "; }
@counter-style lower-alpha { system: cyclic; symbols: "L"; }
@counter-style short { system: fixed; symbols: a; fallback: paren; }
@counter-style loop1 { system: extends loop2; }
@counter-style loop2 { system: extends loop1; }
@counter-style few { system: numeric; symbols: "0"; }
`)
	styles := authorStyles([]*Sheet{s}, Media{})
	assert.Equal(t, "(L) ", styles.marker("paren", 2), "what a style extends is the author's where it redefines it")
	assert.Equal(t, "a", styles.represent("short", 1))
	assert.Equal(t, "L", styles.represent("short", 5), "out of its range, its fallback's representation")
	assert.Equal(t, "7. ", styles.marker("loop1", 7), "a loop of extends extends decimal")
	assert.Equal(t, "7", styles.represent("few", 7), "too few symbols define no style: decimal")
	assert.Equal(t, "7", styles.represent("nothing", 7))
	assert.Equal(t, "vii", styles.represent("lower-roman", 7))
	assert.Equal(t, "-7", styles.represent("lower-roman", -7), "lower-roman falls back to decimal below 1")
	assert.Equal(t, "η", styles.represent("lower-greek", 7))
	assert.True(t, strings.HasSuffix(styles.marker("hiragana", 1), "、"))
}
