package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func strictly(s style.Strictness) func(*style.Computed) {
	return func(st *style.Computed) {
		width(50)(st)
		st.Strictness = s
	}
}

// strictLines are the lines of content in a paragraph five characters wide
// broken by s.
func strictLines(s style.Strictness, content string) []string {
	p := el(dom.P, strictly(s), text(content))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), ahemFonts{})
	return lines(tr.Of[p])
}

func TestStrictKeepsSmallKanaOffALinesStart(t *testing.T) {
	for _, s := range []style.Strictness{style.StrictnessNormal, style.StrictnessNormalCJK, style.StrictnessLoose,
		style.StrictnessLooseCJK} {
		assert.Equal(t, []string{"字字字字字", "ぁ字"}, strictLines(s, "字字字字字ぁ字"), "strictness %d", s)
		assert.Equal(t, []string{"字字字字字", "ー字"}, strictLines(s, "字字字字字ー字"), "strictness %d", s)
	}
	assert.Equal(t, []string{"字字字字", "字ぁ字"}, strictLines(style.StrictnessStrict, "字字字字字ぁ字"))
	assert.Equal(t, []string{"字字字字", "「ぁ字"}, strictLines(style.StrictnessNormal, "字字字字「ぁ字"),
		"an opening bracket keeps the kana after it")
}

func TestLooseBreaksBeforeIterationMarksAndBetweenInseparables(t *testing.T) {
	assert.Equal(t, []string{"字字字字", "字々字"}, strictLines(style.StrictnessNormal, "字字字字字々字"))
	assert.Equal(t, []string{"字字字字字", "々字"}, strictLines(style.StrictnessLoose, "字字字字字々字"))
	assert.Equal(t, []string{"字字字", "字……字"}, strictLines(style.StrictnessNormal, "字字字字……字"))
	assert.Equal(t, []string{"字字字字…", "…字"}, strictLines(style.StrictnessLoose, "字字字字……字"))
}

func TestTheCJKRulesBreakBeforeHyphensPunctuationAndPostfixes(t *testing.T) {
	assert.Equal(t, []string{"字字字字", "字〜字"}, strictLines(style.StrictnessNormal, "字字字字字〜字"))
	assert.Equal(t, []string{"字字字字字", "〜字"}, strictLines(style.StrictnessNormalCJK, "字字字字字〜字"))
	assert.Equal(t, []string{"字字字字", "字‐字"}, strictLines(style.StrictnessLoose, "字字字字字‐字"))
	for text, want := range map[string][]string{
		"字字字字字‐字": {"字字字字字", "‐字"},
		"字字字字字！字": {"字字字字字", "！字"},
		"字字字字1％":  {"字字字字1", "％"},
		"字字字字字°":  {"字字字字字", "°"},
		"字字字字1°":  {"字字字字", "1°"},
		"字字字字A＄":  {"字字字字A", "＄"},
		"字字字字＄1":  {"字字字字＄", "1"},
		"字字字字‐A字": {"字字字字", "‐A字"},
	} {
		assert.Equal(t, want, strictLines(style.StrictnessLooseCJK, text), text)
	}
}

func TestTheTextBeforeAnElementBreaksBeforeItsFirstCharacter(t *testing.T) {
	loose := func(st *style.Computed) { st.Strictness = style.StrictnessLoose }
	strict := el(dom.P, strictly(style.StrictnessStrict), text("字字字字字"), el(dom.Span, loose, text("ぁ字")))
	relaxed := el(dom.P, strictly(style.StrictnessLoose), text("字字字字字"),
		el(dom.Span, func(st *style.Computed) { st.Strictness = style.StrictnessStrict }, text("ぁ字")))
	tr := Layout(el(dom.Div, block, strict, relaxed), geom.V2(200, 600), ahemFonts{})
	assert.Equal(t, []string{"字字字字", "字"}, lines(tr.Of[strict]), "the kana stays with the ideograph before it")
	assert.Equal(t, []string{"字字字字字"}, lines(tr.Of[relaxed]), "the kana starts the next line")
}

func TestBreakAllLeavesWhatTheStrictnessKeepsOffALinesStart(t *testing.T) {
	all := func(s style.Strictness) func(*style.Computed) {
		return func(st *style.Computed) {
			strictly(s)(st)
			st.WordBreak = style.BreakAll
		}
	}
	strict := el(dom.P, all(style.StrictnessStrict), text("字字字字字ぁ字"))
	normal := el(dom.P, all(style.StrictnessNormal), text("字字字字字ぁ字"))
	mark := el(dom.P, all(style.StrictnessNormal), text("字字字字字々字"))
	tr := Layout(el(dom.Div, block, strict, normal, mark), geom.V2(200, 600), ahemFonts{})
	assert.Equal(t, []string{"字字字字", "字ぁ字"}, lines(tr.Of[strict]))
	assert.Equal(t, []string{"字字字字字", "ぁ字"}, lines(tr.Of[normal]))
	assert.Equal(t, []string{"字字字字", "字々字"}, lines(tr.Of[mark]))
}

func TestAnywhereBreaksWhereItsTextEnds(t *testing.T) {
	anywhere := func(st *style.Computed) { st.LineBreak = style.LineBreakAnywhere }
	p := el(dom.P, width(50), el(dom.Span, anywhere, text("HHH")), text("HHHH"))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), ahemFonts{})
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "the text after the span starts a second line")
}

// asciiLines are the lines of content in a paragraph five characters
// wide.
func asciiLines(content string) []string {
	p := el(dom.P, width(50), text(content))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), ahemFonts{})
	return lines(tr.Of[p])
}

func TestASCIIPairsBreakByChromesTable(t *testing.T) {
	for content, want := range map[string][]string{
		"ab!cdefg":    {"ab!cdefg"},
		"ab}cdefg":    {"ab}cdefg"},
		"ab|c1234":    {"ab|c1234"},
		"$$$$$$$$":    {"$$$$$$$$"},
		"%%%%%%%%":    {"%%%%%%%%"},
		"abc,(def)":   {"abc,", "(def)"},
		"abcd-12":     {"abcd-", "12"},
		"1234-56":     {"1234-", "56"},
		"!!!!-12":     {"!!!!-12"},
		"what?nowhy":  {"what?", "nowhy"},
		"a/b/c/d/e/f": {"a/b/c/d/e/f"},
		"ab!éfgh":     {"ab!", "éfgh"},
	} {
		assert.Equal(t, want, asciiLines(content), content)
	}
}

func TestBreakAllTakesChromesASCIITable(t *testing.T) {
	p := el(dom.P, breaking(style.BreakAll, style.LineBreakAuto, width(10)), text("ab(cd)"))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), ahemFonts{})
	assert.Equal(t, []string{"a", "b", "(c", "d)"}, lines(tr.Of[p]), "no break after ( nor before )")
}

func TestASCIITableBreaksAcrossElements(t *testing.T) {
	p := el(dom.P, width(50), text("ab!"), el(dom.Span, nil, text("cdefg")))
	tr := Layout(el(dom.Div, block, p), geom.V2(200, 600), ahemFonts{})
	assert.Equal(t, float32(10), tr.Of[p].Size.Y, "no break after ! before the span's letter")
}
