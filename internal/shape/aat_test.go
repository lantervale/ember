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

//nolint:gochecknoglobals // the texts the stripped faces are compared on
var (
	khmer    = []string{"ភាសាខ្មែរ", "កើ", "កៀ", "ក្រ", "ក្រុ", "ស្ត្រី", "ក្ស", "កេ", "ក្ក", "ប៉ុ", "ក៏", "ព្រះ"}
	gurmukhi = []string{"ਪੰਜਾਬੀ", "ਕਿ", "ਕ੍ਰ", "ਕ੍ਵ", "ਸ੍ਹ", "ਕੌ", "ਖ਼", "ਕੀਂ", "ਕੁ"}
)

// TestTheMachinesAATFacesStrippedShapeAsHarfBuzzShapesThem compares,
// where hb-shape and fontTools are, the shaping of Khmer, Gurmukhi,
// Sinhala and other faces of the machine whose AAT tables HarfBuzz
// would apply in place of GSUB, those tables taken out.
func TestTheMachinesAATFacesStrippedShapeAsHarfBuzzShapesThem(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	python := "/opt/homebrew/opt/fonttools/libexec/bin/python"
	if _, err = os.Stat(python); err != nil {
		t.Skip("no fontTools")
	}
	const sup = "/System/Library/Fonts/Supplemental/"
	const strip = "import sys\nfrom fontTools.ttLib import TTFont\n" +
		"f = TTFont(sys.argv[1], fontNumber=int(sys.argv[3]))\n" +
		"for t in ('morx', 'kerx', 'mort', 'kern'):\n    if t in f: del f[t]\nf.save(sys.argv[2])\n"
	compared := 0
	for _, face := range []systemFace{
		{sup + "Khmer Sangam MN.ttf", 0, khmer},
		{sup + "Khmer MN.ttc", 0, khmer},
		{sup + "Gurmukhi MN.ttc", 0, gurmukhi},
		{sup + "Sinhala MN.ttc", 0, sinhala},
		{sup + "Myanmar MN.ttc", 0, myanmar},
		{sup + "Bangla MN.ttc", 0, bengali},
		{sup + "Devanagari Sangam MN.ttc", 0, devanagari},
		{sup + "Telugu MN.ttc", 0, telugu},
		{sup + "Kannada MN.ttc", 0, kannada},
		{sup + "Oriya MN.ttc", 0, oriya},
	} {
		if _, err = os.Stat(face.path); err != nil {
			continue
		}
		path := t.TempDir() + "/face.ttf"
		out, perr := exec.Command(python, "-I", "-c", strip, face.path, path, fmt.Sprint(face.index)).CombinedOutput()
		require.NoError(t, perr, string(out))
		file, rerr := os.ReadFile(path)
		require.NoError(t, rerr)
		src, serr := sfnt.Parse(file)
		require.NoError(t, serr)
		f := shape.Parse(src, file)
		for _, text := range face.texts {
			glyphs := run(t, src, text)
			if slices.ContainsFunc(glyphs, func(g shape.Glyph) bool {
				return g.ID == 0 && g.Char != 0x200C && g.Char != 0x200D
			}) {
				continue
			}
			hbOut, herr := exec.Command(hbShape, "--font-size="+fmt.Sprint(src.UnitsPerEm()), "--no-glyph-names", path,
				text).Output()
			require.NoError(t, herr)
			want := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(hbOut)), "|", "|gid"), "[", "[gid")
			p := f.Plan(shape.Script(text), "")
			assert.Equal(t, want, hb(p.Shape(glyphs), p.RightToLeft()), "%q (%U) in %s", text, []rune(text),
				face.path)
			compared++
		}
	}
	t.Logf("%d texts compared", compared)
}
