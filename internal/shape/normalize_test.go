package shape_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
)

// TestMarksAFaceDoesNotPlaceSitByTheirClassesAsHarfBuzzPutsThem
// compares, where hb-shape is, Go Regular, which has no GPOS, shaping
// marks on letters (placed by their combining classes, under, over and
// beside), characters it lacks drawn from those it decomposes to, and
// marks composed with their base into a character it has.
func TestMarksAFaceDoesNotPlaceSitByTheirClassesAsHarfBuzzPutsThem(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	path := t.TempDir() + "/go.ttf"
	require.NoError(t, os.WriteFile(path, goregular.TTF, 0o600))
	src, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	f := shape.Parse(src, goregular.TTF)
	for _, text := range []string{
		"q\u0301", "x\u0323", "M\u0300\u0301", "a\u0308\u0304", "g\u0327", "e\u0301", "o\u0328\u0301",
		"ḉ", "ǖ", "ṏ", "ạ", "ǻ", "Ω", "x\u0301\u0323", "i\u0302\u0330", "A\u030A\u0301", "fi\u0303",
	} {
		glyphs := run(t, src, text)
		out, herr := exec.Command(hbShape, "--font-size="+fmt.Sprint(src.UnitsPerEm()), "--no-glyph-names", path,
			text).Output()
		require.NoError(t, herr)
		want := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(out)), "|", "|gid"), "[", "[gid")
		p := f.Plan(shape.Script(text), "")
		assert.Equal(t, want, hb(p.Shape(glyphs)), "%q (%U)", text, []rune(text))
	}
}
