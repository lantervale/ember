package srcset_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/srcset"
)

func TestASrcsetIsReadAsHTMLReadsIt(t *testing.T) {
	for _, c := range []struct {
		srcset string
		want   []srcset.Candidate
	}{
		{"a.png", []srcset.Candidate{{URL: "a.png"}}},
		{" a.png 1x ,b.png 2x,", []srcset.Candidate{{URL: "a.png", Density: 1}, {URL: "b.png", Density: 2}}},
		{"a.png 100w, b.png 200w 80h", []srcset.Candidate{{URL: "a.png", Width: 100}, {URL: "b.png", Width: 200}}},
		{"a.png,b.png 2x", []srcset.Candidate{{URL: "a.png,b.png", Density: 2}}},
		{"a.png, b.png 2x", []srcset.Candidate{{URL: "a.png"}, {URL: "b.png", Density: 2}}},
		{"data:image/png;base64,iVBO 2x, c.png", []srcset.Candidate{
			{URL: "data:image/png;base64,iVBO", Density: 2}, {URL: "c.png"}}},
		{"a.png 1x 2x, b.png 100w 2x, c.png 50h, d.png 0w, e.png 1.5x", []srcset.Candidate{
			{URL: "e.png", Density: 1.5}}},
		{"a.png 1x (b, c), d.png 2x", []srcset.Candidate{{URL: "d.png", Density: 2}}},
		{"a.png .5x, b.png +1x, c.png 1e1x, d.png 1.x, e.png -1x", []srcset.Candidate{
			{URL: "a.png", Density: 0.5}, {URL: "c.png", Density: 10}}},
		{"a.png,, b.png", []srcset.Candidate{{URL: "a.png"}, {URL: "b.png"}}},
		{"  ,, ", nil},
	} {
		assert.Equal(t, c.want, srcset.Parse(c.srcset), c.srcset)
	}
}

func TestTheCandidateChosenIsChromes(t *testing.T) {
	xs := srcset.Parse("a 1x, b 2x, d 4x")
	for _, c := range []struct {
		set     []srcset.Candidate
		src     string
		size    float64
		scale   float64
		want    string
		density float64
	}{
		{xs, "", 0, 1, "a", 1},
		{xs, "", 0, 1.5, "b", 2},
		{xs, "", 0, 2, "b", 2},
		{xs, "", 0, 3, "d", 4},
		{xs, "", 0, 9, "d", 4},
		{srcset.Parse("a 1x, c 3x"), "", 0, 2, "c", 3},
		{srcset.Parse("a 1.2x, c 2.5x"), "", 0, 1, "a", 1.2},
		{srcset.Parse("a 1.5x, d 4x"), "", 0, 2, "d", 4},
		{srcset.Parse("a 1.5x, d 4x"), "", 0, 1.5, "a", 1.5},
		{srcset.Parse("a 0.6x, b 0.9x"), "", 0, 1, "b", 0.9},
		{srcset.Parse("a 2x, b 2x"), "", 0, 1, "a", 2},
		{srcset.Parse("b, a 2x"), "", 0, 2, "a", 2},
		{srcset.Parse("b 2x"), "a", 0, 1, "a", 1},
		{srcset.Parse("b 1x"), "a", 0, 1, "b", 1},
		{srcset.Parse("a 100w, b 200w, d 400w"), "", 100, 1, "a", 1},
		{srcset.Parse("a 100w, b 200w, d 400w"), "", 100, 2, "b", 2},
		{srcset.Parse("a 100w, c 300w"), "", 150, 1, "c", 2},
		{srcset.Parse("a 50w"), "s", 100, 1, "a", 0.5},
		{srcset.Parse("a 100w, b 200w"), "", 0, 1, "a", math.Inf(1)},
		{nil, "s", 0, 2, "s", 1},
		{nil, "", 0, 2, "", 0},
	} {
		url, density := srcset.Choose(c.set, c.src, c.size, c.scale, func(string) bool { return false })
		assert.Equal(t, c.want, url, "%v %q at %g", c.set, c.src, c.scale)
		if math.IsInf(c.density, 1) {
			assert.True(t, math.IsInf(density, 1), "a slot of no width")
			continue
		}
		assert.InDelta(t, c.density, density, 1e-9, "%v %q at %g", c.set, c.src, c.scale)
	}
}

func TestADenserCandidateHeldWinsOverTheOneChosen(t *testing.T) {
	set := srcset.Parse("a 1x, b 2x, c 3x, d 4x")
	held := func(urls ...string) func(string) bool {
		return func(url string) bool { return slices.Contains(urls, url) }
	}
	url, density := srcset.Choose(set, "", 0, 1, held("a", "b", "c"))
	assert.Equal(t, "c", url)
	assert.InDelta(t, 3, density, 1e-9)
	url, _ = srcset.Choose(set, "", 0, 3, held("a", "b"))
	assert.Equal(t, "c", url, "none held denser: the one chosen")
}

func TestReplaceChangesTheAddressesAlone(t *testing.T) {
	assert.Equal(t, " A.PNG 1x ,B.PNG 2x, C.PNG, D.PNG 1x (e, f),G.PNG",
		srcset.Replace(" a.png 1x ,b.png 2x, c.png, d.png 1x (e, f),g.png", strings.ToUpper))
	assert.Empty(t, srcset.Replace("", strings.ToUpper))
}
