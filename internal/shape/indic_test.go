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

// systemFace is a face of the machine and the texts to shape in it.
type systemFace struct {
	path  string
	index int
	texts []string
}

//nolint:gochecknoglobals // the texts the system faces are compared on
var (
	devanagari = []string{
		"नमस्ते दुनिया", "हिन्दी", "कि", "क्षत्रिय", "र्क", "र्कि", "धर्म", "प्रकाश", "श्री", "क्ष्म", "द्ध",
		"त्र", "ट्र", "स्त्री", "क्\u200dष", "क्\u200cष", "रु", "रू", "हृ", "अा", "ि", "◌ि", "र्", "कं", "किं", "र्किं",
		"ज्ञान", "उद्ध्व", "पृथ्वी", "क़", "ड़", "ॐ", "द्य", "ह्म", "र्त्स्न्य", "कॆ", "१२३",
	}
	bengali = []string{
		"বাংলা ভাষা", "কি", "কো", "কৌ", "র্ক", "র্কি", "ক্ষ", "প্র", "ক্র", "ব্য", "র্য", "শ্রী", "স্ত্রী", "জ্ঞ",
		"য়", "ড়", "ঢ়", "কে", "র্কো", "ত্\u200d", "ন্দ্র", "ৎ", "গ্ধ",
	}
	gujarati = []string{"ગુજરાતી", "કિ", "ર્ક", "ક્ષ", "પ્ર", "શ્રી", "સ્ત્રી", "દ્ધ", "ર્કિ", "હ્ય"}
	telugu   = []string{"తెలుగు", "కి", "క్ష", "ప్ర", "శ్రీ", "స్త్రీ", "ర్క", "కొ", "కై", "క్క", "ర్\u200dక"}
	kannada  = []string{"ಕನ್ನಡ", "ಕಿ", "ಕ್ಷ", "ಪ್ರ", "ಶ್ರೀ", "ರ್ಕ", "ಕೊ", "ಕೈ", "ಕ್ಕ", "ರ್ಕೊ", "ಸ್ತ್ರೀ"}
	oriya    = []string{"ଓଡ଼ିଆ", "କି", "କୋ", "କୌ", "କ୍ଷ", "ପ୍ର", "ର୍କ", "କ୍ୟ", "ସ୍ତ୍ରୀ", "କେ", "ର୍କି"}
	tamil    = []string{"தமிழ்", "கொ", "கோ", "கௌ", "கெ", "க்ஷ", "ஸ்ரீ", "கை", "க்க", "து"}
	myanmar  = []string{
		"မြန်မာ", "ကြ", "ကြိ", "ကျ", "ကွ", "ကှ", "င်္ဂ", "ကေ", "ကြေ", "ကို", "ကော်", "မြေ", "သင်္ချိုင်း", "ဿ",
	}
	balinese = []string{"ᬩᬮᬶ", "ᬓᬾ", "ᬓ᭄ᬱ", "ᬓᭀ", "ᬓᬺ", "ᬓᬿ"}
	javanese = []string{"ꦗꦮ", "ꦏꦺ", "ꦏ꧀ꦱ", "ꦏꦺꦴ", "ꦏꦽ", "ꦏꦾ", "ꦏꦿ", "ꦲꦤꦕꦫꦏ"}
	sinhala  = []string{"සිංහල", "කො", "කෝ", "කෞ", "ක්\u200dෂ", "ශ්\u200dරී", "කෙ", "ක්\u200dය"}
	tibetan  = []string{"བོད་ཡིག", "སྐད", "བསྒྲུབས", "ཀྲི", "ཀླུ"}
	others   = []string{"𑆯𑆳𑆫𑆢𑆳", "𑒞𑒱𑒩𑒯𑒳𑒞𑒰", "𑐣𑐾𑐥𑐵𑐮", "ᮞᮥᮔ᮪ᮓ", "ᯅᯖᯂ᯲", "𑘦𑘻𑘚𑘲", "𑚔𑚭𑚊𑚶", "𑀩𑁆𑀭𑀸𑀳𑁆𑀫𑀻"}
)

func systemFaces() []systemFace {
	const sup = "/System/Library/Fonts/Supplemental/"
	return []systemFace{
		{"/System/Library/Fonts/Kohinoor.ttc", 0, devanagari},
		{sup + "ITFDevanagari.ttc", 0, devanagari},
		{"/System/Library/Fonts/KohinoorBangla.ttc", 0, bengali},
		{"/System/Library/Fonts/KohinoorGujarati.ttc", 3, gujarati},
		{"/System/Library/Fonts/KohinoorTelugu.ttc", 0, telugu},
		{"/System/Library/Fonts/NotoSansKannada.ttc", 3, kannada},
		{"/System/Library/Fonts/NotoSansOriya.ttc", 0, oriya},
		{sup + "Tamil Sangam MN.ttc", 0, tamil},
		{"/System/Library/Fonts/NotoSansMyanmar.ttc", 6, myanmar},
		{sup + "NotoSerifBalinese-Regular.ttf", 0, balinese},
		{sup + "NotoSansJavanese-Regular.otf", 0, javanese},
		{sup + "NotoSansSharada-Regular.ttf", 0, others},
		{sup + "NotoSansTirhuta-Regular.ttf", 0, others},
		{sup + "NotoSansNewa-Regular.ttf", 0, others},
		{sup + "NotoSansSundanese-Regular.ttf", 0, others},
		{sup + "NotoSansBatak-Regular.ttf", 0, others},
		{sup + "NotoSansModi-Regular.ttf", 0, others},
		{sup + "NotoSansTakri-Regular.ttf", 0, others},
		{sup + "NotoSansBrahmi-Regular.ttf", 0, others},
		{"testdata/noto-devanagari.ttf", 0, devanagari},
		{"/usr/share/fonts/truetype/noto/NotoSansDevanagari-Regular.ttf", 0, devanagari},
		{"/usr/share/fonts/truetype/noto/NotoSansSinhala-Regular.ttf", 0, sinhala},
		{"/usr/share/fonts/truetype/noto/NotoSansTibetan-Regular.ttf", 0, tibetan},
	}
}

// TestTheMachinesSyllabicFacesShapeAsHarfBuzzShapesThem compares, where
// hb-shape and the faces are, the shaping of words of the Indic scripts,
// Myanmar and scripts of the Universal Shaping Engine in faces of the
// machine that have no AAT tables, with HarfBuzz's.
func TestTheMachinesSyllabicFacesShapeAsHarfBuzzShapesThem(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	compared := map[string]int{}
	for _, face := range systemFaces() {
		file, rerr := os.ReadFile(face.path)
		if rerr != nil {
			continue
		}
		src, perr := sfnt.Parse(file)
		if perr != nil {
			c, cerr := sfnt.ParseCollection(file)
			require.NoError(t, cerr)
			src, perr = c.Font(face.index)
			require.NoError(t, perr)
		}
		f := shape.ParseFont(src, file, face.index)
		for _, text := range face.texts {
			glyphs := run(t, src, text)
			if slices.ContainsFunc(glyphs, func(g shape.Glyph) bool {
				return g.ID == 0 && g.Char != 0x200C && g.Char != 0x200D
			}) {
				continue
			}
			out, herr := exec.Command(hbShape, "--font-size="+fmt.Sprint(src.UnitsPerEm()), "--no-glyph-names",
				"--face-index="+fmt.Sprint(face.index), face.path, text).Output()
			require.NoError(t, herr)
			want := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(out)), "|", "|gid"), "[", "[gid")
			p := f.Plan(shape.Script(text), "")
			assert.Equal(t, want, hb(p.Shape(glyphs), p.RightToLeft()), "%q (%U) in %s", text, []rune(text),
				face.path)
			compared[face.path]++
		}
	}
	t.Logf("texts compared by face: %v", compared)
}

// TestDevanagariIsReorderedAndFormedAsHarfBuzzDoes shapes Devanagari in
// a subset of Noto Sans Devanagari (testdata/noto-devanagari.ttf) as
// hb-shape prints it: pre-base matras, reph, half and below forms,
// conjuncts, an explicit half form by ZWJ and a ZWNJ breaking one, a
// nukta form composed, and dotted circles where a vowel sign has no
// base or would draw as another vowel.
func TestDevanagariIsReorderedAndFormedAsHarfBuzzDoes(t *testing.T) {
	src, ttf := testFont(t, "noto-devanagari.ttf")
	f := shape.Parse(src, ttf)
	for _, c := range []struct{ text, want string }{
		{"कि", "[gid604=0+259|gid23=0+762]"},
		{"र्कि", "[gid628=0+259|gid23=0+762|gid649=0+0]"},
		{"क्ष", "[gid176=0+712]"},
		{"प्रकाश", "[gid271=0+568|gid23=3+762|gid64=3+259|gid56=5+680]"},
		{"श्री", "[gid281=0+707|gid66=0+259]"},
		{"स्त्री", "[gid212=0+389|gid266=2+552|gid66=2+259]"},
		{"क्\u200dष", "[gid180=0+603|gid1=0+0|gid57=3+578]"},
		{"क्\u200cष", "[gid23=0+762|gid79=0@-221,0+0|gid1=2+0|gid57=3+578]"},
		{"किं", "[gid616=0+259|gid23=0+762|gid649=0+0]"},
		{"क़", "[gid90=0+762]"},
		{"अा", "[gid7=0+764|gid132=0+510|gid64=0+259]"},
		{"हिन्दी", "[gid604=0+259|gid59=0+531|gid199=2+309|gid40=4+520|gid650=4+259]"},
		{"ि", "[gid65=0+259|gid132=0+510]"},
	} {
		p := f.Plan(shape.Script(c.text), "")
		assert.Equal(t, c.want, hb(p.Shape(run(t, src, c.text))), "%q", c.text)
	}
}
