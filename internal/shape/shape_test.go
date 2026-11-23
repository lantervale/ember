package shape_test

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
)

func features(t *testing.T) (*sfnt.Font, []byte) { return testFont(t, "go-features.ttf") }

func testFont(t *testing.T, name string) (*sfnt.Font, []byte) {
	t.Helper()
	ttf, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	f, err := sfnt.Parse(ttf)
	require.NoError(t, err)
	return f, ttf
}

// run is text's glyphs as the font's cmap has them, each of its rune's
// cluster, a mark of its base's.
func run(t *testing.T, f *sfnt.Font, text string) []shape.Glyph {
	t.Helper()
	var buf sfnt.Buffer
	var out []shape.Glyph
	for i, r := range []rune(text) {
		g, err := f.GlyphIndex(&buf, r)
		require.NoError(t, err)
		cluster := int32(i)
		if len(out) > 0 && unicode.Is(unicode.Mn, r) {
			cluster = out[len(out)-1].Cluster
		}
		out = append(out, shape.Glyph{ID: uint16(g), Cluster: cluster, Char: r})
	}
	return out
}

// hb writes glyphs as hb-shape does, a right-to-left run from its last.
func hb(glyphs []shape.Glyph, rtl ...bool) string {
	if len(rtl) > 0 && rtl[0] {
		glyphs = slices.Clone(glyphs)
		slices.Reverse(glyphs)
	}
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
	for _, c := range []struct{ font, text string }{
		{"go-features.ttf", "office AVATAR 1/2 Small"},
		{"go-arabic.ttf", "بَّسسس لاِ هههه e\u0301\u0301 23"},
	} {
		src, ttf := testFont(t, c.font)
		for step := 3; step <= 11; step += 4 {
			broken := append([]byte(nil), ttf...)
			for i := 200; i < len(broken); i += step {
				broken[i] ^= 0x5a
			}
			f := shape.Parse(src, broken)
			for _, feats := range []string{"", "smcp,c2sc,tnum,frac,dlig,sups"} {
				p := f.Plan(shape.Script(c.text), feats)
				assert.NotPanics(t, func() { p.Shape(run(t, src, c.text)) })
			}
		}
	}
}

func TestArabicIsJoinedAndItsMarksPlacedAsHarfBuzzDoes(t *testing.T) {
	src, ttf := testFont(t, "go-arabic.ttf")
	f := shape.Parse(src, ttf)
	// What hb-shape --font-size=2048 --features=… testdata/go-arabic.ttf prints.
	for _, c := range []struct{ features, text, want string }{
		{"", "بتب", "[gid99=2+1399|gid102=1+1099|gid97=0+1399]"},
		{"", "بسم", "[gid115=2+1966|gid106=1+1544|gid97=0+1399]"},
		{"", "دار", "[gid132=2+682|gid128=1+817|gid130=0+1139]"},
		{"", "اب", "[gid96=1+1139|gid128=0+817]"},
		{"", "ه ه", "[gid120=2+1139|gid1=1+569|gid120=0+1139]"},
		{"", "ـبـ", "[gid136=2+520|gid98=1+1659|gid136=0+520]"},
		{"", "مممم", "[gid115=3+1966|gid114=2+2226|gid114=1+2226|gid113=0+1966]"},
		{"", "لا", "[gid138=0+1445]"},
		{"", "يلا", "[gid139=1+1705|gid125=0+1284]"},
		{"", "ـلا", "[gid139=1+1705|gid136=0+520]"},
		{"", "بم", "[gid115=1+1966|gid137=0+1539]"},
		{"", "ببم", "[gid115=2+1966|gid98=1+1659|gid97=0+1399]"},
		{"", "بمب", "[gid99=2+1399|gid114=1+2226|gid137=0+1539]"},
		{"", "سسس", "[gid107=2+1284|gid106=1@0,-200+1544|gid105=0@0,-260+1284]"},
		{"", "هههه", "[gid123=3+1399|gid122=2@0,-80+1659|gid122=1@0,-120+1659|gid121=0@0,-120+1399]"},
		{"", "سهس", "[gid107=2+1284|gid122=1@0,-80+1659|gid105=0@0,-140+1284]"},
		{"", "بَ", "[gid140=0@449,1200+0|gid96=0+1139]"},
		{"", "بِ", "[gid141=0@449,-300+0|gid96=0+1139]"},
		{"", "بَّ", "[gid140=0@489,1640+0|gid142=0@449,1200+0|gid96=0+1139]"},
		{"", "بَّ", "[gid140=0@489,1640+0|gid142=0@449,1200+0|gid96=0+1139]"},
		{"", "سِّ", "[gid141=0@392,-300+0|gid142=0@392,1200+0|gid104=0+1024]"},
		{"", "لَا", "[gid140=0@180,1300+0|gid138=0+1445]"},
		{"", "لاِ", "[gid141=0@780,-300+0|gid138=0+1445]"},
		{"", "بلَا", "[gid140=1@180,1300+0|gid139=1+1705|gid97=0+1399]"},
		{"", "لِاَ", "[gid140=0@780,1300+0|gid141=0@180,-300+0|gid138=0+1445]"},
		{"", "بلاَ", "[gid140=1@780,1300+0|gid139=1+1705|gid97=0+1399]"},
		{"-curs", "سسس", "[gid107=2+1284|gid106=1+1544|gid105=0+1284]"},
		{"-init,-medi", "بتب", "[gid99=2+1399|gid100=1+579|gid96=0+1139]"},
		{"-mark", "بَ", "[gid140=0+0|gid96=0+1139]"},
		{"-mkmk", "بَّ", "[gid140=0@449,1200+0|gid142=0@449,1200+0|gid96=0+1139]"},
		{"-rlig", "بلا", "[gid129=2+1077|gid110=1+1068|gid97=0+1399]"},
		{"-calt", "بم", "[gid115=1+1966|gid97=0+1399]"},
		{"isol", "بتب", "[gid99=2+1399|gid102=1+1099|gid97=0+1399]"},
		{"subs", "H2O", "[gid41=0+1479|gid149=1+683|gid48=2+1593]"},
		{"sups", "x2 3", "[gid89=0+1024|gid148=1+683|gid1=2+569|gid150=3+683]"},
		{"", "a e\u0301", "[gid66=0+1139|gid1=1+569|gid70=2+1139|gid143=2@-690,1200+0]"},
		{"", "e\u0301\u0301", "[gid70=0+1139|gid143=0@-690,1200+0|gid143=0@-690,1200+0]"},
	} {
		p := f.Plan(shape.Script(c.text), c.features)
		assert.Equal(t, c.want, hb(p.Shape(run(t, src, c.text)), p.RightToLeft()), "%q in %q", c.text, c.features)
	}
}

// TestTheMachinesRightToLeftFacesShapeAsHarfBuzzShapesThem compares,
// where hb-shape and the faces are, the shaping of sentences in Arabic,
// Syriac, Hebrew, N'Ko, Adlam and Thaana faces of the machine with
// HarfBuzz's, each in the faces that have all its characters.
func TestTheMachinesRightToLeftFacesShapeAsHarfBuzzShapesThem(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	texts := []string{
		"بِسْمِ ٱللَّهِ ٱلرَّحْمَٰنِ ٱلرَّحِيمِ",
		"السلام عليكم ورحمة الله وبركاته",
		"لا إله إلا الله، كتب الطالب الدرس.",
		"ـبـ ممم لالا",
		"ܫܠܡܐ ܥܠܘܟܘܢ",
		"בְּרֵאשִׁית בָּרָא אֱלֹהִים",
		"ߒߞߏ ߞߊ߲ߜߍ",
		"𞤀𞤣𞤤𞤢𞤥 𞤆𞤵𞤤𞤢𞤪",
		"ދިވެހި",
		"Été, naïve café",
	}
	for _, path := range []string{
		"/System/Library/Fonts/SFArabic.ttf",
		"/System/Library/Fonts/Supplemental/NotoSansSyriac-Regular.ttf",
		"/System/Library/Fonts/SFHebrew.ttf",
		"/System/Library/Fonts/Supplemental/NotoSansNKo-Regular.ttf",
		"/System/Library/Fonts/Supplemental/NotoSansAdlam-Regular.ttf",
		"/System/Library/Fonts/Supplemental/NotoSansThaana-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoSansArabic-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoNaskhArabic-Regular.ttf",
	} {
		ttf, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		src, err := sfnt.Parse(ttf)
		require.NoError(t, err)
		f := shape.Parse(src, ttf)
		for _, text := range texts {
			glyphs := run(t, src, text)
			if slices.ContainsFunc(glyphs, func(g shape.Glyph) bool { return g.ID == 0 }) {
				continue
			}
			out, err := exec.Command(hbShape, "--font-size="+fmt.Sprint(src.UnitsPerEm()), "--no-glyph-names",
				path, text).Output()
			require.NoError(t, err)
			want := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(out)), "|", "|gid"), "[", "[gid")
			p := f.Plan(shape.Script(text), "")
			assert.Equal(t, want, hb(p.Shape(glyphs), p.RightToLeft()), "%q in %s", text, path)
		}
	}
}
