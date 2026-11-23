package pattern_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/pattern"
)

// The answers are Chrome's patternMismatch, negated: a pattern it
// cannot compile constrains nothing.
func TestAPatternMatchesAWholeValueAsChromeReadsIt(t *testing.T) {
	for _, c := range []struct {
		pat, value string
		match      bool
	}{
		{"(?=a)a.", "ab", true}, {"(a)\\1", "aa", true}, {"(a)\\1", "ab", false}, {"\\d+", "12", true},
		{"\\p{L}+", "été", true}, {"[\\p{L}--[a-z]]", "É", true}, {"[\\p{L}--[a-z]]", "e", false},
		{"(?<n>a)\\k<n>", "aa", true}, {"[\\q{ab|c}]", "ab", true}, {"[\\q{ab|c}]", "c", true},
		{"[\\q{ab|c}]", "abc", false}, {".", "\U0001F600", true}, {"\\w", "é", false}, {"(?<=a)b", "b", false},
		{"(?<=a)b", "ab", false}, {"a(?<=a)b", "ab", true}, {"a|", "x", false}, {"a|", "", true}, {"[^]", "x", true},
		{"\\u{61}", "a", true}, {"\\x61", "a", true}, {"\\cA", "\x01", true}, {"[]", "x", false}, {"\\/", "/", true},
		{"/", "/", true}, {"(?i:A)", "a", true}, {"(?i:A)", "b", false}, {"(?i:a)A", "aa", false}, {"x*?y", "xxy", true},
		{"[a&&b]", "a", false}, {"[\\d&&[0-4]]", "3", true}, {"[\\d&&[0-4]]", "7", false}, {"\\b", "x", false},
		{"\\bx\\b", "x", true}, {"^a$", "a", true}, {"\\0", "\x00", true}, {"a{2,}", "aaa", true}, {"a{2,}", "a", false},
		{"a{1,2}", "aaa", false}, {"\\p{Script=Greek}", "α", true}, {"\\p{sc=Greek}", "a", false}, {"\\P{Lu}", "a", true},
		{"[[a-c]--b]", "b", false}, {"[[a-c]--b]", "c", true}, {"(?:)", "9", false}, {"(?:)", "", true},
		{"[\\w--\\d]", "5", false}, {"[\\w--\\d]", "a", true}, {"[^\\p{L}]", "a", false}, {".", "\n", false},
		{"(?s:.)", "\n", true}, {"\\s", "\u00a0", true}, {"$", "9", false}, {"a(?=b)", "9", false}, {"[\\(]", "(", true},
		{"[a\\-z]", "-", true}, {"[a\\-z]", "b", false}, {"(?i:[^a])", "A", false},
		{"(?i:\\W)", "ſ", false}, {"(?i:\\w)", "K", true}, {"(a*)*b", "aab", true}, {"(a|ab)(c|bcd)(d*)", "abcd", true},
		{"(?!a)\\w", "b", true}, {"(?!a)\\w", "a", false}, {"(?<!a)b", "b", true},
		{"(?<=\\1(a))b", "aab", false}, {"(.)(?<=\\1\\1)", "x", false}, {"\\u0041", "A", true},
		{"\\uD83D\\uDE00", "\U0001F600", true}, {"[\\q{}]", "", true}, {"(?i:[\\q{AB}])", "ab", true},
		{"[\\p{Lu}&&\\p{Script=Latin}]", "Ω", false}, {"[\\p{Lu}&&\\p{Script=Latin}]", "Q", true}, {"\\p{Any}", "\x00", true},
		{"[0-9]{3}-[0-9]{4}", "555-1234", true}, {"[A-Za-z]{3}", "ab", false},
	} {
		re, err := pattern.Compile(c.pat)
		if err != nil {
			t.Errorf("%q: %v", c.pat, err)
			continue
		}
		if got := re.Match(c.value); got != c.match {
			t.Errorf("%q on %q: %v, want %v", c.pat, c.value, got, c.match)
		}
	}
}

func TestAPatternChromeRefusesIsAnError(t *testing.T) {
	for _, src := range []string{"[a-z-]", "a{", "a)", "\\k<n>", "\\1", "(a)\\2", "\\p{Foo}", "[(]", "[!!]", "[a-]",
		"x{2,1}", "\\M", "(", "[a", "*a", "a**", "(?=a)*", "(?<=a)+", "\\c1", "\\00", "[^\\q{ab}]", "(?ii:a)", "(?-:a)",
		"(?<a>x)(?<a>y)", "[a&&&b]", "[a&&b--c]", "[a-b&&c]", "}", "]", "\\p{gc=Foo}", "\\x1", "\\u{110000}", "(?<1a>x)",
		"[a--]", "[&&a]", "\\", "(?i)a", "\\-"} {
		if _, err := pattern.Compile(src); err == nil {
			t.Errorf("%q: no error", src)
		}
	}
}

func TestAPropertyGoesTablesLackIsUnsupported(t *testing.T) {
	for _, src := range []string{"\\p{RGI_Emoji}", "\\p{Emoji}", "\\p{scx=Latin}", "\\p{sc=Latn}"} {
		if _, err := pattern.Compile(src); !errors.Is(err, pattern.ErrUnsupported) {
			t.Errorf("%q: %v", src, err)
		}
	}
}

func TestABacktrackingBlowUpIsCut(t *testing.T) {
	re, err := pattern.Compile("(a*)*b")
	if err != nil {
		t.Fatal(err)
	}
	if !re.Match(strings.Repeat("a", 40)) {
		t.Error("a match past the budget is taken to match")
	}
}
