//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// featured lays out spans of features.ttf, Go with ligatures, small
// capitals, figures, fractions and class kerning, each with its
// declarations, against Chrome.
func featured(t *testing.T, sheet string, spans ...[2]string) {
	t.Helper()
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Featured; src: url(features.ttf); }
div { margin: 0 0 4px; }
span { display: inline-block; font: 24px Featured, monospace; }
`+sheet))
	for _, s := range spans {
		d.Body().Append(ui.Div(ui.Span(ui.Style(s[0]), ui.Text(s[1]))))
	}
	size := geom.V2(700, 900)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestLigaturesAndKerningAreChromes(t *testing.T) {
	featured(t, "",
		[2]string{"", "office affinity flow"},
		[2]string{"", "AVATAR LTd WAY"},
		[2]string{"", "To, Too"},
		[2]string{"font-variant-ligatures: none", "office affinity"},
		[2]string{"font-variant-ligatures: no-common-ligatures", "office affinity"},
		[2]string{"font-variant-ligatures: discretionary-ligatures", "act office"},
		[2]string{"font-variant-ligatures: common-ligatures no-discretionary-ligatures", "act office"},
		[2]string{`font-feature-settings: "liga" 0`, "office affinity"},
		[2]string{`font-feature-settings: "dlig", "liga" off`, "act office"},
		[2]string{"font-kerning: none", "AVATAR To"},
		[2]string{"font-kerning: normal", "AVATAR To"},
		[2]string{`font-feature-settings: "kern" 0`, "AVATAR To"},
		[2]string{"letter-spacing: 2px", "office AVATAR"},
	)
}

func TestNumericFiguresAreChromes(t *testing.T) {
	featured(t, "",
		[2]string{"font-variant-numeric: oldstyle-nums", "0123456789"},
		[2]string{"font-variant-numeric: lining-nums", "0123456789"},
		[2]string{"font-variant-numeric: tabular-nums", "0123456789"},
		[2]string{"font-variant-numeric: proportional-nums", "0123456789"},
		[2]string{"font-variant-numeric: slashed-zero", "100 2020"},
		[2]string{"font-variant-numeric: diagonal-fractions", "1/2 3/4 and 7/8"},
		[2]string{"font-variant-numeric: tabular-nums slashed-zero", "1000"},
		[2]string{"font-variant: oldstyle-nums diagonal-fractions", "1/2 2024"},
		[2]string{`font-feature-settings: "tnum", "onum" 1`, "0123"},
	)
}

func TestSmallCapitalsComeFromTheFaceWhenItHasThemAsInChrome(t *testing.T) {
	featured(t, ".go { font-family: Go, monospace; }",
		[2]string{"font-variant-caps: small-caps", "Small Capitals 42"},
		[2]string{"font-variant-caps: all-small-caps", "Small Capitals 42"},
		[2]string{"font-variant-caps: petite-caps", "Petite Capitals"},
		[2]string{"font-variant-caps: all-petite-caps", "Petite Capitals"},
		[2]string{"font-variant-caps: unicase", "Unicase Text"},
		[2]string{"font-variant-caps: titling-caps", "Titling Capitals"},
		[2]string{"font: small-caps 24px Featured, monospace", "By the shorthand"},
		[2]string{`font-variant-caps: small-caps; font-feature-settings: "smcp" 0`, "Turned back off"},
		[2]string{"font-variant-caps: small-caps; font-family: Go, monospace", "Made up in Go"},
		[2]string{"font-variant-caps: all-small-caps; font-family: Go, monospace", "Made up in Go"},
	)
}

func TestAFacesFeatureSettingsShapeItsTextAsInChrome(t *testing.T) {
	featured(t, `
@font-face { font-family: Capitals; src: url(features.ttf); font-feature-settings: "smcp", "liga" 0; }
@font-face { font-family: Figures; src: url(features.ttf); font-feature-settings: "tnum"; }
.capitals { font-family: Capitals, monospace; }
.figures { font-family: Figures, monospace; }`,
		[2]string{"font-family: Capitals, monospace", "office Capitals"},
		[2]string{"font-family: Figures, monospace", "0123 office"},
		[2]string{`font-family: Figures, monospace; font-feature-settings: "tnum" 0`, "0123 by the text"},
		[2]string{"font-family: Capitals, monospace; font-variant-ligatures: common-ligatures", "office"},
	)
}

func TestAVariableFaceIsShapedAtItsInstanceAsInChrome(t *testing.T) {
	featured(t, `
@font-face { font-family: Varied; src: url(features-variable.ttf); font-weight: 100 900; font-stretch: 50% 150%; }
.varied { font-family: Varied, monospace; }`,
		[2]string{"font-family: Varied, monospace; font-weight: 800", "office affinity AVATAR To"},
		[2]string{"font-family: Varied, monospace; font-weight: 300; font-stretch: 130%", "office AVATAR"},
		[2]string{`font-family: Varied, monospace; font-variation-settings: "wght" 650`, "Small Caps office"},
		[2]string{"font-family: Varied, monospace; font-weight: 700; font-variant-caps: all-small-caps",
			"All Small 1/2"},
		[2]string{"font-family: Varied, monospace; font-weight: 700; font-variant-numeric: tabular-nums",
			"0123 1010"},
	)
}

func TestRealSmallCapitalsIgnoreFontSynthesisAsInChrome(t *testing.T) {
	featured(t, "",
		[2]string{"font-variant-caps: small-caps; font-synthesis: none", "Real Small Caps"},
		[2]string{"font-variant-caps: all-small-caps; font-synthesis: weight style", "Real Small Caps"},
		[2]string{"font-variant-caps: unicase; font-synthesis: none", "Unicase Text"},
		[2]string{"font-variant-caps: small-caps; font-synthesis: none; font-family: Go, monospace", "Left As It Is"},
	)
}
