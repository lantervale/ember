package shape_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
)

func features(t *testing.T) (*sfnt.Font, []byte) {
	t.Helper()
	ttf, err := os.ReadFile("testdata/go-features.ttf")
	require.NoError(t, err)
	f, err := sfnt.Parse(ttf)
	require.NoError(t, err)
	return f, ttf
}

// run is text's glyphs as the font's cmap has them, each of its rune's
// cluster.
func run(t *testing.T, f *sfnt.Font, text string) []shape.Glyph {
	t.Helper()
	var buf sfnt.Buffer
	var out []shape.Glyph
	for i, r := range []rune(text) {
		g, err := f.GlyphIndex(&buf, r)
		require.NoError(t, err)
		out = append(out, shape.Glyph{ID: uint16(g), Cluster: int32(i)})
	}
	return out
}

// hb writes glyphs as hb-shape does.
func hb(glyphs []shape.Glyph) string {
	parts := make([]string, len(glyphs))
	for i, g := range glyphs {
		at := ""
		if g.XOffset != 0 || g.YOffset != 0 {
			at = fmt.Sprintf("@%d,%d", g.XOffset, g.YOffset)
		}
		parts[i] = fmt.Sprintf("gid%d=%d%s+%d", g.ID, g.Cluster, at, g.Advance)
	}
	return "[" + strings.Join(parts, "|") + "]"
}

func TestARunIsShapedAsHarfBuzzShapesIt(t *testing.T) {
	src, ttf := features(t)
	f := shape.Parse(src, ttf)
	// What hb-shape --font-size=2048 --features=… testdata/go-features.ttf prints.
	for _, c := range []struct{ features, text, want string }{
		{"", "office AVATAR To 0123 1/2 act", "[gid80=0+1139|gid98=1+1383|gid68=4+1024|gid70=5+1139|gid1=6+459|" +
			"gid34=7+1206|gid55=8+1366|gid34=9+1206|gid53=10+1251|gid34=11+1366|gid51=12+1479|gid1=13+569|" +
			"gid53=14+1161|gid80=15@30,0+1139|gid1=16+569|gid17=17+1139|gid18=18+1139|gid19=19+1139|gid20=20+1139|" +
			"gid1=21+569|gid18=22+1139|gid16=23+569|gid19=24+1139|gid1=25+569|gid66=26+1139|gid68=27+1024|gid85=28+579]"},
		{"smcp", "Small Caps", "[gid52=0+1366|gid112=1+1320|gid100=2+1064|gid111=3+894|gid111=4+894|gid1=5+569|" +
			"gid36=6+1479|gid100=7+1064|gid115=8+1064|gid118=9+1064]"},
		{"smcp,c2sc", "Small Caps", "[gid118=0+1064|gid112=1+1320|gid100=2+1064|gid111=3+894|gid111=4+894|gid1=5+569|" +
			"gid102=6+1149|gid100=7+1064|gid115=8+1064|gid118=9+1064]"},
		{"onum", "0123", "[gid126=0+1139|gid132=1+1139|gid138=2+1139|gid144=3+1139]"},
		{"lnum", "0123", "[gid129=0+1199|gid135=1+1199|gid141=2+1199|gid147=3+1199]"},
		{"tnum", "0123", "[gid127=0+1339|gid133=1+1339|gid139=2+1339|gid145=3+1339]"},
		{"pnum", "0123", "[gid128=0+1059|gid134=1+1059|gid140=2+1059|gid146=3+1059]"},
		{"zero", "10203", "[gid18=0+1139|gid187=1+1239|gid19=2+1139|gid187=3+1239|gid20=4+1139]"},
		{"frac", "1/2 3/4", "[gid136=0+626|gid186=1+369|gid143=2+626|gid1=3+569|gid148=4+626|gid186=5+369|gid155=6+626]"},
		{"dlig", "act", "[gid66=0+1139|gid99=1+1803]"},
		{"-liga", "office", "[gid80=0+1139|gid71=1+569|gid71=2+569|gid74=3+505|gid68=4+1024|gid70=5+1139]"},
		{"-kern", "AVATAR To", "[gid34=0+1366|gid55=1+1366|gid34=2+1366|gid53=3+1251|gid34=4+1366|gid51=5+1479|" +
			"gid1=6+569|gid53=7+1251|gid80=8+1139]"},
		{"liga=0,dlig", "office act", "[gid80=0+1139|gid71=1+569|gid71=2+569|gid74=3+505|gid68=4+1024|gid70=5+1139|" +
			"gid1=6+569|gid66=7+1139|gid99=8+1803]"},
		{"tnum,zero", "0", "[gid127=0+1339]"},
	} {
		p := f.Plan(shape.Script(c.text), c.features)
		assert.Equal(t, c.want, hb(p.Shape(run(t, src, c.text))), "%q in %q", c.text, c.features)
	}
}

func TestAFontWithoutLayoutTouchesNothing(t *testing.T) {
	src, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	p := shape.Parse(src, goregular.TTF).Plan(shape.MakeTag("latn"), "smcp,tnum")
	glyphs := run(t, src, "office")
	assert.False(t, p.Touches(glyphs))
	assert.False(t, p.KernTable())
	assert.Len(t, p.Shape(glyphs), 6)
}

func TestAPlanSaysWhichGlyphsItMayChange(t *testing.T) {
	src, ttf := features(t)
	f := shape.Parse(src, ttf)
	latn := shape.MakeTag("latn")
	assert.True(t, f.Plan(latn, "").Touches(run(t, src, "AV")), "the font kerns")
	assert.True(t, f.Has(latn, shape.MakeTag("smcp")))
	assert.False(t, f.Has(latn, shape.MakeTag("pcap")))
	assert.Same(t, f.Plan(latn, "smcp"), f.Plan(latn, "smcp"), "plans are made once")
}

func TestFeaturesAreReadAsHarfBuzzWritesThem(t *testing.T) {
	assert.Equal(t, []shape.Feature{
		{Tag: shape.MakeTag("smcp"), Value: 1}, {Tag: shape.MakeTag("liga"), Value: 0},
		{Tag: shape.MakeTag("kern"), Value: 1}, {Tag: shape.MakeTag("salt"), Value: 2},
	}, shape.ParseFeatures("smcp, -liga,+kern,salt=2,toolong,x=y"))
	assert.Equal(t, "cv01", shape.MakeTag("cv01").String())
	assert.Equal(t, shape.MakeTag("latn"), shape.Script("12 Hello"))
	assert.Equal(t, shape.MakeTag("cyrl"), shape.Script("Привет"))
}

func TestAFacesFeaturesGoBetweenAVariantsAndTheSettings(t *testing.T) {
	assert.Equal(t, "smcp,-liga", shape.Splice("-liga", "smcp"))
	assert.Equal(t, "onum,smcp,-liga", shape.Splice("onum|-liga", "smcp"))
	assert.Equal(t, "onum,smcp", shape.Splice("onum|", "smcp"))
	assert.Equal(t, "onum|", shape.Splice("onum|", ""), "the marker reads as a comma")
	assert.Equal(t, []shape.Feature{{Tag: shape.MakeTag("onum"), Value: 1}, {Tag: shape.MakeTag("liga"), Value: 0}},
		shape.ParseFeatures("onum|-liga"))
}

func TestAFontThatLiesShapesWithoutPanicking(t *testing.T) {
	src, ttf := features(t)
	broken := append([]byte(nil), ttf...)
	for i := 200; i < len(broken); i += 7 {
		broken[i] ^= 0x5a
	}
	f := shape.Parse(src, broken)
	for _, feats := range []string{"", "smcp,c2sc,tnum,frac,dlig"} {
		p := f.Plan(shape.MakeTag("latn"), feats)
		assert.NotPanics(t, func() { p.Shape(run(t, src, "office AVATAR 1/2 Small")) })
	}
}
