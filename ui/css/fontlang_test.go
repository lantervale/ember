package css //nolint:testpackage // reads computed styles off dom with the package's helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
)

func TestTheFontHoldsTheLanguageOfItsText(t *testing.T) {
	inner := el(dom.Span, "inner")
	outer := with(el(dom.P, "outer"), inner)
	plain := el(dom.P, "plain")
	root := page(outer, plain)
	clean := func(n *dom.Node) bool {
		dom.Data(n).Dirty = 0
		return true
	}
	Compute(root, nil)
	assert.Empty(t, styleOf(inner).Font.Lang, "no language anywhere")

	root.Walk(clean)
	dom.SetLanguage(root, "zh-Hant")
	outer.SetAttr("lang", "ja")
	Compute(root, nil)
	assert.Equal(t, "ja", styleOf(outer).Font.Lang)
	assert.Equal(t, "ja", styleOf(inner).Font.Lang, "inherited from the nearest element that has one")
	assert.Equal(t, "zh-Hant", styleOf(plain).Font.Lang, "the Document's")

	root.Walk(clean)
	inner.SetAttr("lang", "ko")
	Compute(root, nil)
	assert.Equal(t, "ko", styleOf(inner).Font.Lang)
	assert.False(t, styleOf(inner).Font.Equal(styleOf(outer).Font), "fonts of two languages differ")
}
