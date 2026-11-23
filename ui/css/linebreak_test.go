package css //nolint:testpackage // reads computed styles off dom

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestLineBreakPicksItsRulesByTheLanguage(t *testing.T) {
	values := []string{"auto", "loose", "normal", "strict", "anywhere"}
	want := map[string][]style.Strictness{
		"": {style.StrictnessNormal, style.StrictnessLoose, style.StrictnessNormal, style.StrictnessStrict,
			style.StrictnessNormal},
		"ja-JP": {style.StrictnessNormal, style.StrictnessLooseCJK, style.StrictnessNormalCJK, style.StrictnessStrict,
			style.StrictnessNormal},
		"zh-Hant": {style.StrictnessNormalCJK, style.StrictnessLooseCJK, style.StrictnessNormalCJK,
			style.StrictnessStrict, style.StrictnessNormal},
		"KO": {style.StrictnessNormal, style.StrictnessLooseCJK, style.StrictnessNormalCJK, style.StrictnessNormal,
			style.StrictnessNormal},
		"yue": {style.StrictnessNormal, style.StrictnessLoose, style.StrictnessNormal, style.StrictnessStrict,
			style.StrictnessNormal},
	}
	for lang, strictness := range want {
		for i, v := range values {
			p := el(dom.P, "p")
			root := with(el(dom.Div, "root"), p)
			if lang != "" {
				root.SetAttr("lang", lang)
			}
			Compute(root, []*Sheet{sheet(t, "#p { line-break: "+v+"; }")})
			assert.Equal(t, strictness[i], styleOf(p).Strictness, "%s in %q", v, lang)
		}
	}
}

func TestALanguageChangedAboveRestylesHowLinesBreak(t *testing.T) {
	p := el(dom.P, "p")
	root := with(el(dom.Div, "root"), p)
	sheets := []*Sheet{sheet(t, "#p { line-break: loose; }")}
	clean := func(n *dom.Node) bool {
		dom.Data(n).Dirty = 0
		return true
	}
	Compute(root, sheets)
	root.Walk(clean)
	root.SetAttr("lang", "ja")
	Compute(root, sheets)
	assert.Equal(t, style.StrictnessLooseCJK, styleOf(p).Strictness)
	root.Walk(clean)
	dom.SetLanguage(root, "zh")
	root.RemoveAttr("lang")
	Compute(root, sheets)
	assert.Equal(t, style.StrictnessLooseCJK, styleOf(p).Strictness, "the Document's language")
}
