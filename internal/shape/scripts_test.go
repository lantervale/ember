package shape_test

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
)

// hbFace is a face hb-shape reads, with the texts to compare on it.
type hbFace struct {
	path  string
	index int
	texts []string
}

// loadFace reads the index-th face of path, false where the file is not.
func loadFace(t *testing.T, path string, index int) (*sfnt.Font, *shape.Font, bool) {
	t.Helper()
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}
	src, err := sfnt.Parse(file)
	if err != nil {
		c, cerr := sfnt.ParseCollection(file)
		require.NoError(t, cerr)
		src, err = c.Font(index)
		require.NoError(t, err)
	}
	return src, shape.ParseFont(src, file, index), true
}

// hbShaped is what hb-shape prints for text in the index-th face of
// path, between before and after, written as hb writes Glyphs.
func hbShaped(t *testing.T, hbShape, path string, index int, upem sfnt.Units, text, before, after string) string {
	t.Helper()
	args := []string{"--font-size=" + fmt.Sprint(upem), "--no-glyph-names", "--face-index=" + fmt.Sprint(index)}
	if before != "" {
		args = append(args, "--text-before="+before)
	}
	if after != "" {
		args = append(args, "--text-after="+after)
	}
	out, err := exec.Command(hbShape, append(args, path, text)...).Output()
	require.NoError(t, err)
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(out)), "|", "|gid"), "[", "[gid")
}

// selector reports whether r is a variation selector, which a face need
// not have.
func selector(r rune) bool {
	return r >= 0x180B && r <= 0x180F || r >= 0xFE00 && r <= 0xFE0F || r >= 0xE0100 && r <= 0xE01EF
}

// compareFaces shapes each face's texts as hb-shape does, a text whose
// characters a face lacks left out; it reports how many it compared.
func compareFaces(t *testing.T, faces []hbFace) int {
	t.Helper()
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	compared := 0
	for _, face := range faces {
		src, f, ok := loadFace(t, face.path, face.index)
		if !ok {
			continue
		}
		for _, text := range face.texts {
			glyphs := run(t, src, text)
			if slices.ContainsFunc(glyphs, func(g shape.Glyph) bool { return g.ID == 0 && !selector(g.Char) }) {
				continue
			}
			want := hbShaped(t, hbShape, face.path, face.index, src.UnitsPerEm(), text, "", "")
			p := f.Plan(shape.Script(text), "")
			assert.Equal(t, want, hb(p.Shape(glyphs), p.RightToLeft()), "%q (%U) in %s", text, []rune(text),
				face.path)
			compared++
		}
	}
	return compared
}

// TestThaiAndLaoAreShapedAsHarfBuzzShapesThem compares the Thai and Lao
// of the machine's faces with HarfBuzz's: sara am decomposed, its
// nikhahit before the tone marks above, and marks on tall consonants.
func TestThaiAndLaoAreShapedAsHarfBuzzShapesThem(t *testing.T) {
	thai := []string{
		"ภาษาไทย", "น้ำ", "ทำ", "กำ", "ป่า", "ปั่น", "ญู", "ฐุ", "ฎุ", "ที่", "ก้ำ", "คำ", "ป้ำ", "นิ้", "ฝั่ง", "ปิ่",
		"ฟ้ำ", "กุ้", "ฤๅ", "ำ", "่ำ", "ก่ำา",
	}
	lao := []string{"ພາສາລາວ", "ນ້ຳ", "ຄຳ", "ກ່ຳ", "ບໍ່", "ຂຶ້ນ"}
	const sup = "/System/Library/Fonts/Supplemental/"
	n := compareFaces(t, []hbFace{
		{"/System/Library/Fonts/ThonburiUI.ttc", 0, thai},
		{sup + "Tahoma.ttf", 0, thai},
		{"testdata/noto-thai.ttf", 0, thai},
		{"testdata/noto-lao.ttf", 0, lao},
		{"testdata/go-thai.ttf", 0, thai},
	})
	t.Logf("%d texts compared", n)
}

// TestHangulIsComposedAsHarfBuzzComposesIt compares Hangul syllables
// and jamo with HarfBuzz's: jamo composed into a syllable the face has,
// syllables it lacks decomposed, old jamo shaped by ljmo, vjmo and
// tjmo, and tone marks put before their syllable.
func TestHangulIsComposedAsHarfBuzzComposesIt(t *testing.T) {
	texts := []string{
		"한국어", "한글", "한", "가", "ᄀᆞ", "ᄀᆞᆫ", "각",
		"가ᇿ", "가〮", "〮", "가〯", "ꥠᅡ", "ᄀힰ", "각ᆨ",
		"ᅟᅡ", "각〮",
	}
	const sup = "/System/Library/Fonts/Supplemental/"
	n := compareFaces(t, []hbFace{
		{"/System/Library/Fonts/AppleSDGothicNeo.ttc", 0, texts},
		{sup + "AppleMyungjo.ttf", 0, texts},
		{"testdata/noto-hangul.ttf", 0, texts},
	})
	t.Logf("%d texts compared", n)
}

// TestHebrewIsShapedAsHarfBuzzShapesIt compares pointed Hebrew with
// HarfBuzz's: points reordered as fonts expect them, and a face with no
// mark positioning drawing a letter and its point as their
// presentation form.
func TestHebrewIsShapedAsHarfBuzzShapesIt(t *testing.T) {
	texts := []string{
		"שָׁלוֹם", "בְּרֵאשִׁית", "שׁ", "שּׁ", "שׂ", "בּ", "יִ", "אַ", "אָ", "וֹ", "ײַ", "כֿ", "פּ", "בְַֽ",
		"בִָ֑", "תּוֹרָה", "שִׂמְחָה",
	}
	const sup = "/System/Library/Fonts/Supplemental/"
	n := compareFaces(t, []hbFace{
		{"/System/Library/Fonts/SFHebrew.ttf", 0, texts},
		{sup + "Tahoma.ttf", 0, texts},
		{sup + "Arial Unicode.ttf", 0, texts},
		{"testdata/noto-hebrew.ttf", 0, texts},
		{"testdata/go-hebrew.ttf", 0, texts},
	})
	t.Logf("%d texts compared", n)
}

// TestJoiningScriptsOfTheUSEAreShapedAsHarfBuzzShapesThem compares the
// scripts HarfBuzz 14 gives the Universal Shaping Engine that join as
// Arabic does, and Arabic's modifier marks, with HarfBuzz's.
func TestJoiningScriptsOfTheUSEAreShapedAsHarfBuzzShapesThem(t *testing.T) {
	const sup = "/System/Library/Fonts/Supplemental/"
	n := compareFaces(t, []hbFace{
		{sup + "NotoSansMongolian-Regular.ttf", 0, []string{
			"ᠮᠣᠩᠭᠣᠯ", "ᠠ᠋ᠠ", "ᠭ᠌ᠠ", "ᠨ᠋", "ᠠᠮᠠ᠋ᠷ", "ᠪᠢᠴᠢᠭ", "ᠠ᠏ᠠ",
		}},
		{sup + "NotoSansNKo-Regular.ttf", 0, []string{"ߒߞߏ ߞߊ߲ߜߍ", "ߓߊ߫", "ߞߎߡߊ߲"}},
		{sup + "NotoSansAdlam-Regular.ttf", 0, []string{"𞤀𞤣𞤤𞤢𞤥", "𞤆𞤵𞤤𞤢𞤪", "𞤢𞥄𞤢"}},
		{sup + "NotoSansPhagsPa-Regular.ttf", 0, []string{"ꡀꡁꡂ", "ꡏꡡꡃ"}},
		{sup + "NotoSansMandaic-Regular.ttf", 0, []string{"ࡀࡁࡂ", "ࡌࡀࡍࡃࡀࡉࡀ"}},
		{sup + "NotoSansManichaean-Regular.ttf", 0, []string{"𐫀𐫁𐫂", "𐫃𐫄𐫅𐫆"}},
		{sup + "NotoSansPsalterPahlavi-Regular.ttf", 0, []string{"𐮀𐮁𐮂", "𐮃𐮄𐮅"}},
		{sup + "NotoSansHanifiRohingya-Regular.ttf", 0, []string{"𐴀𐴁𐴂", "𐴌𐴟𐴇𐴥𐴝𐴚𐴒𐴙𐴝"}},
		{sup + "NotoSansSyriac-Regular.ttf", 0, []string{"ܫܠܡܐ ܥܠܘܟܘܢ", "ܐܬܐ"}},
		{"/System/Library/Fonts/SFArabic.ttf", 0, []string{
			"بَٔ", "بِٕ", "أُ", "بَّٔ",
			"ؤَ", "ئٰ", "بِۧ",
		}},
	})
	t.Logf("%d texts compared", n)
}

// TestVariationSequencesTakeTheirGlyphs compares characters followed by
// variation selectors with HarfBuzz's: a sequence the face's cmap maps
// is drawn with its glyph, one it does not as its character and the
// selector as nothing.
func TestVariationSequencesTakeTheirGlyphs(t *testing.T) {
	texts := []string{
		"葛\U000E0100", "葛\U000E0101", "辻\U000E0100", "辻\U000E0101", "葛︀", "a︀b", "邉\U000E0108",
		"葛\U000E0101\U000E0100", "葛",
	}
	n := compareFaces(t, []hbFace{
		{"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc", 0, texts},
		{"testdata/go-thai.ttf", 0, []string{"a\uFE00b\uFE01a\uFE01", "a\uFE00", "b\uFE01c", "a\uFE0Fb"}},
	})
	t.Logf("%d texts compared", n)
}

// TestLongSyllablesAreShapedAsHarfBuzzShapesThem compares syllables and
// mark sequences past the lengths HarfBuzz handles apart: more than 32
// marks, which it leaves unsorted, and syllables of more than 127
// characters, whose clusters it merges whole.
func TestLongSyllablesAreShapedAsHarfBuzzShapesThem(t *testing.T) {
	long := []string{
		"क" + strings.Repeat("्क", 70), "र्" + strings.Repeat("क्", 66) + "ष",
		"क" + strings.Repeat("॒॑", 20), "क" + strings.Repeat("॒॑", 17) + "ि",
		"क" + strings.Repeat("़", 40) + "ि", strings.Repeat("क्", 64) + "कि",
		strings.Repeat("\u0930\u094D\u0915\u093F", 40),
	}
	// HarfBuzz's handling of long syllables changes between releases;
	// shape ports 14.2.1's.
	if out, err := exec.Command("hb-shape", "--version").Output(); err == nil &&
		!strings.HasPrefix(string(out), "hb-shape (HarfBuzz) 14.2.1\n") {
		t.Skipf("hb-shape is not HarfBuzz 14.2.1: %s", strings.SplitN(string(out), "\n", 2)[0])
	}
	n := compareFaces(t, []hbFace{{"testdata/noto-devanagari.ttf", 0, long}})
	t.Logf("%d texts compared", n)
}

// TestTheTextRoundARunJoinsItsLetters compares runs shaped between the
// text before and after them with hb-shape given the same context: a
// letter that joins takes the form the letters round the run give it.
func TestTheTextRoundARunJoinsItsLetters(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	cases := []struct{ path, before, text, after string }{
		{"/System/Library/Fonts/SFArabic.ttf", "ب", "ب", "ب"},
		{"/System/Library/Fonts/SFArabic.ttf", "ب", "بب", ""},
		{"/System/Library/Fonts/SFArabic.ttf", "", "بب", "ب"},
		{"/System/Library/Fonts/SFArabic.ttf", "ا", "ب", "ا"},
		{"/System/Library/Fonts/SFArabic.ttf", "بَ", "ب", "َب"},
		{"/System/Library/Fonts/SFArabic.ttf", "ب ", "ب", " ب"},
		{"/System/Library/Fonts/SFArabic.ttf", "abc", "َ", ""},
		{"/System/Library/Fonts/Supplemental/NotoSansMongolian-Regular.ttf", "ᠮ", "ᠣ", "ᠩ"},
		{"/System/Library/Fonts/Supplemental/NotoSansNKo-Regular.ttf", "ߞ", "ߊ", "ߜ"},
		{"/System/Library/Fonts/Supplemental/NotoSansSyriac-Regular.ttf", "ܫ", "ܠ", "ܡ"},
		{"testdata/noto-devanagari.ttf", "क", "ि", ""},
	}
	for _, c := range cases {
		src, f, ok := loadFace(t, c.path, 0)
		if !ok {
			continue
		}
		want := hbShaped(t, hbShape, c.path, 0, src.UnitsPerEm(), c.text, c.before, c.after)
		p := f.Plan(shape.Script(c.text+c.before), "")
		assert.Equal(t, want, hb(p.ShapeIn(run(t, src, c.text), c.before, c.after), p.RightToLeft()),
			"%q|%q|%q in %s", c.before, c.text, c.after, c.path)
	}
}
