package css //nolint:testpackage // reads the sheets' rules as the cascade does

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/dom"
)

const ruleByRule = `/* a comment */
.a\:b { color: #f00 }
@media (min-width: 600px) { .wide { width: 10px } @media (min-height: 10px) { .tall { height: 1px } } }
@layer base, theme;
@layer theme { p { margin: 1px } }
@layer { p { margin: 2px } }
@font-face { font-family: "Mine"; src: url(mine.ttf) }
p::before { content: "} {" }
`

func TestParseRulesGivesEachTopLevelRuleItsOwnSheetAndText(t *testing.T) {
	parts, warns := ParseRules("site.css", ruleByRule, Browser())
	assert.Empty(t, warns)
	var texts []string
	for _, p := range parts {
		texts = append(texts, p.Text())
	}
	assert.Equal(t, []string{
		`.a\:b { color: #f00 }`,
		`@media (min-width: 600px) { .wide { width: 10px } @media (min-height: 10px) { .tall { height: 1px } } }`,
		`@layer base, theme;`,
		`@layer theme { p { margin: 1px } }`,
		`@layer { p { margin: 2px } }`,
		`@font-face { font-family: "Mine"; src: url(mine.ttf) }`,
		`p::before { content: "} {" }`,
	}, texts)
	require.Len(t, parts[1].Rules, 2)
	assert.Equal(t, Pos{3, 29}, parts[1].Rules[0].Pos, "a rule stands where it does in the whole text")
}

func TestJoinedRulesCascadeAsTheWholeSheetDoes(t *testing.T) {
	whole, _ := Parse("site.css", ruleByRule, Browser())
	parts, _ := ParseRules("site.css", ruleByRule, Browser())
	joined, warns := Join("site.css", "", parts)
	assert.Empty(t, warns)
	require.Len(t, joined.Rules, len(whole.Rules))
	for i, r := range whole.Rules {
		got := joined.Rules[i]
		assert.Equal(t, r.Pos, got.Pos)
		assert.Len(t, got.media, len(r.media), "rule %d", i)
		for j := range r.media {
			assert.Equal(t, whole.media[r.media[j]].text, joined.media[got.media[j]].text)
		}
	}
	assert.Equal(t, layerRanks([]*Sheet{whole}), layerRanks([]*Sheet{joined}))
	assert.Equal(t, whole.FontFaces(), joined.FontFaces())
	assert.Len(t, joined.Rules, 6)
}

func TestJoinPutsEveryRuleInTheMediaItIsGiven(t *testing.T) {
	parts, _ := ParseRules("", `p { color: #f00 } @media (min-height: 5px) { p { width: 3px } }`)
	joined, warns := Join("style", "(min-width: 600px)", parts)
	assert.Empty(t, warns)
	p := dom.NewElement(dom.P)
	narrow := mediaOff([]*Sheet{joined}, Media{Width: 500, Height: 500})
	wide := mediaOff([]*Sheet{joined}, Media{Width: 700, Height: 500})
	assert.Equal(t, [][]bool{{true, true}}, narrow)
	assert.Equal(t, [][]bool{{false, false}}, wide)
	assert.True(t, joined.Rules[0].Selectors[0].Matches(p))
	assert.Contains(t, joined.String(), "@media (min-width: 600px) { p {")

	_, warns = Join("style", "(what", parts)
	assert.NotEmpty(t, warns)
}

func TestTwoJoinedSheetsNameOneLayerAndKeepTheirNamelessOnesApart(t *testing.T) {
	a, _ := ParseRules("", `@layer one { p { color: #f00 } } @layer { p { color: #0f0 } }`, Browser())
	b, _ := ParseRules("", `@layer { p { color: #00f } } @layer one { p { width: 1px } }`, Browser())
	joined, _ := Join("style", "", append(a, b...))
	require.Len(t, joined.Rules, 4)
	assert.Equal(t, joined.Rules[0].layer, joined.Rules[3].layer)
	assert.NotEqual(t, joined.Rules[1].layer, joined.Rules[2].layer)
	assert.Len(t, joined.layers, 3)
}
