package css //nolint:testpackage // reads computed styles off dom with the package's helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestTextIsShapedWithTheFeaturesItsPropertiesAskFor(t *testing.T) {
	for _, c := range []struct{ decls, want string }{
		{"", ""},
		{"font-variant-ligatures: none", "-liga,-clig,-dlig,-hlig,-calt|"},
		{"font-variant-ligatures: no-common-ligatures discretionary-ligatures", "-liga,-clig,dlig|"},
		{"font-variant-numeric: oldstyle-nums tabular-nums slashed-zero diagonal-fractions", "onum,tnum,frac,zero|"},
		{"font-variant: historical-forms full-width", "fwid,hist|"},
		{"font-variant-caps: small-caps", ""},
		{"font-kerning: none", "-kern|"},
		{"letter-spacing: 1px", "-liga,-clig,-dlig,-hlig,-calt|"},
		{`font-feature-settings: "tnum", "liga" 0, "salt" 3; font-variant-numeric: proportional-nums`,
			"pnum|-liga,salt=3,tnum"},
		{`font-feature-settings: "kern" 1; font-kerning: none`, "-kern|kern"},
	} {
		d := dom.NewElement(dom.Div)
		d.SetStyle(c.decls)
		Compute(page(d), nil)
		assert.Equal(t, c.want, styleOf(d).Font.Features, c.decls)
	}
}

func TestFeatureSettingsAreReadAsCSSWritesThem(t *testing.T) {
	for _, bad := range []string{`"toolong"`, `"liga" -1`, `liga`, `"liga" 1.5`, `"liga",`} {
		_, warns := Parse("t.css", "p { font-feature-settings: "+bad+"; }")
		assert.NotEmpty(t, warns, bad)
	}
	_, warns := Parse("t.css", "p { font-kerning: sometimes; }")
	assert.NotEmpty(t, warns)
}
