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

func TestComplexPredefinedStylesWriteByTheirAlgorithm(t *testing.T) {
	for _, c := range []struct {
		name string
		n    int
		want string
	}{
		{"japanese-informal", 1111, "千百十一"},
		{"japanese-informal", 1001, "千一"},
		{"japanese-formal", 1011, "壱阡壱拾壱"},
		{"japanese-formal", 10000, "一〇〇〇〇"},
		{"japanese-informal", -12, "マイナス十二"},
		{"simp-chinese-informal", 12, "十二"},
		{"simp-chinese-informal", 112, "一百一十二"},
		{"simp-chinese-informal", 1010, "一千零一十"},
		{"simp-chinese-informal", 100000, "十万"},
		{"simp-chinese-informal", 10000001, "一千万零一"},
		{"simp-chinese-formal", 2025, "贰仟零贰拾伍"},
		{"trad-chinese-informal", 100000000, "一億"},
		{"cjk-ideographic", 20, "二十"},
		{"korean-hangul-formal", 1234, "일천이백삼십사"},
		{"korean-hangul-formal", 12345, "일만 이천삼백사십오"},
		{"korean-hanja-informal", 11111, "萬 千百十一"},
		{"korean-hanja-informal", -3, "마이너스 三"},
		{"ethiopic-numeric", 1, "፩"},
		{"ethiopic-numeric", 100, "፻"},
		{"ethiopic-numeric", 78010092, "፸፰፻፩፼፺፪"},
		{"ethiopic-numeric", 0, "0"},
		{"cjk-decimal", 2025, "二〇二五"},
	} {
		assert.Equal(t, c.want, predefined().represent(c.name, c.n), "%s %d", c.name, c.n)
	}
}

func TestSymbolsMakesAnAnonymousCounterStyle(t *testing.T) {
	ref := func(text string) (string, error) {
		v := items(componentsOf(text))
		if err := checkCounterStyleRef(v); err != nil {
			return "", err
		}
		return counterStyleRef(v[0]), nil
	}
	name, err := ref(`SYMBOLS(Cyclic '*' "+")`)
	require.NoError(t, err)
	assert.Equal(t, `symbols(cyclic "*" "+")`, name, "written as getComputedStyle writes it")
	assert.Equal(t, "+", predefined().represent(name, 4))
	assert.Equal(t, "+ ", predefined().marker(name, 4), "no prefix, a space after")
	name, err = ref(`symbols("*")`)
	require.NoError(t, err)
	assert.Equal(t, "**", predefined().represent(name, 2), "symbolic when no system is given")
	assert.Equal(t, "-1", predefined().represent(name, -1), "decimal where it can't write")
	for _, bad := range []string{`symbols(alphabetic "x")`, `symbols()`, `symbols(cyclic x)`, `symbols(url(a.png))`,
		`symbols(extends "a")`} {
		_, err = ref(bad)
		assert.Error(t, err, bad)
	}
}
