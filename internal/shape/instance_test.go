package shape_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/internal/varfont/varfonttest"
)

// variable is a variable font's layout, its file read by sfnt and by
// varfont.
type variable struct {
	src    *sfnt.Font
	vf     *varfont.Font
	layout *shape.Font
	ttf    []byte
}

func parseVariable(t *testing.T, ttf []byte) variable {
	t.Helper()
	src, err := sfnt.Parse(ttf)
	require.NoError(t, err)
	vf, err := varfont.Parse(ttf)
	require.NoError(t, err)
	return variable{src: src, vf: vf, layout: shape.Parse(src, ttf), ttf: ttf}
}

// at is the instance HarfBuzz's variations name, "wght=700,wdth=125".
func (v variable) at(t *testing.T, variations string) varfont.Coords {
	t.Helper()
	user := make([]float32, len(v.vf.Axes))
	for i, a := range v.vf.Axes {
		user[i] = a.Default
	}
	for _, s := range strings.Split(variations, ",") {
		if s == "" {
			continue
		}
		var tag string
		var value float32
		_, err := fmt.Sscanf(strings.Replace(s, "=", " ", 1), "%s %g", &tag, &value)
		require.NoError(t, err)
		if i := v.vf.Tag(tag); i >= 0 {
			user[i] = value
		}
	}
	return v.vf.Normalize(user)
}

func (v variable) shape(t *testing.T, text, variations string) string {
	t.Helper()
	p := v.layout.PlanAt(shape.Script(text), "", v.vf, v.at(t, variations))
	return hb(p.Shape(run(t, v.src, text)), p.RightToLeft())
}

const variedText = "AVTo LYx^ $z"

func TestAVariableFontIsShapedAtItsInstanceAsHarfBuzzShapesIt(t *testing.T) {
	v := parseVariable(t, varfonttest.Build(varfonttest.Options{Layout: true}))
	// What hb-shape --font-size=2048 --no-glyph-names --variations=… prints.
	for _, c := range []struct{ variations, want string }{
		{"", "[gid36=0+1286|gid57=1+1366|gid55=2+1131|gid82=3+1139|gid3=4+569|gid47=5+1039|gid60=6+1366|" +
			"gid91=7+1024|gid65=8@-812,100+0|gid3=9+569|gid7=10+1139|gid93=11+1024]"},
		{"wght=900", "[gid36=0+1246|gid57=1+1486|gid55=2+1131|gid82=3+1259|gid3=4+689|gid47=5+1039|gid60=6+1486|" +
			"gid91=7+1144|gid65=8@-792,260+0|gid3=9+689|gid54=10+1486|gid93=11+1144]"},
		{"wght=100", "[gid36=0+1318|gid57=1+1318|gid55=2+1083|gid82=3+1091|gid3=4+521|gid47=5+1031|gid60=6+1318|" +
			"gid91=7+976|gid65=8@-764,60+0|gid3=9+521|gid86=10+976|gid61=11+1203]"},
		{"wdth=150", "[gid36=0+1512|gid57=1+1912|gid55=2+1751|gid82=3+1595|gid3=4+797|gid47=5+1595|gid60=6+1912|" +
			"gid91=7+1434|gid65=8@-1042,100+0|gid3=9+797|gid7=10+1595|gid61=11+1751]"},
		{"wght=900,wdth=150", "[gid36=0+1632|gid57=1+2032|gid55=2+1871|gid82=3+1715|gid3=4+917|gid47=5+1715|" +
			"gid60=6+2032|gid91=7+1554|gid65=8@-1022,260+0|gid3=9+917|gid54=10+2032|gid93=11+1554]"},
	} {
		assert.Equal(t, c.want, v.shape(t, variedText, c.variations), "at %q", c.variations)
	}
}

func TestAPlanIsMadeOncePerInstance(t *testing.T) {
	v := parseVariable(t, varfonttest.Build(varfonttest.Options{Layout: true}))
	latn := shape.MakeTag("latn")
	bold := v.at(t, "wght=900")
	assert.Same(t, v.layout.PlanAt(latn, "", v.vf, bold), v.layout.PlanAt(latn, "", v.vf, v.at(t, "wght=900")))
	assert.NotSame(t, v.layout.PlanAt(latn, "", v.vf, bold), v.layout.Plan(latn, ""))
	assert.Same(t, v.layout.Plan(latn, ""), v.layout.PlanAt(latn, "", v.vf, nil), "nil is the default instance")
	assert.Same(t, v.layout.Plan(latn, ""), v.layout.PlanAt(latn, "", nil, bold), "no font has no instance")
}

func TestAVariableFontThatLiesShapesWithoutPanicking(t *testing.T) {
	ttf := varfonttest.Build(varfonttest.Options{Layout: true})
	v := parseVariable(t, ttf)
	for step := 3; step <= 11; step += 4 {
		broken := append([]byte(nil), ttf...)
		for i := 200; i < len(broken); i += step {
			broken[i] ^= 0x5a
		}
		f := shape.Parse(v.src, broken)
		for _, at := range []string{"", "wght=900", "wght=100,wdth=150"} {
			p := f.PlanAt(shape.Script(variedText), "", v.vf, v.at(t, at))
			assert.NotPanics(t, func() { p.Shape(run(t, v.src, variedText)) })
		}
	}
}

// TestVariableFontsShapeAtEachInstanceAsHarfBuzzShapesThem compares,
// where hb-shape is, the shaping of the test font over a grid of its
// instances, and of the machine's variable faces whose layout varies
// (GDEF's store, GSUB's or GPOS's feature variations) at some of theirs.
func TestVariableFontsShapeAtEachInstanceAsHarfBuzzShapesThem(t *testing.T) {
	hbShape, err := exec.LookPath("hb-shape")
	if err != nil {
		t.Skip("no hb-shape")
	}
	dir := t.TempDir()
	test := dir + "/layout.ttf"
	require.NoError(t, os.WriteFile(test, varfonttest.Build(varfonttest.Options{Layout: true}), 0o600))
	var grid []string
	for w := 100; w <= 900; w += 50 {
		for d := 50; d <= 150; d += 25 {
			grid = append(grid, fmt.Sprintf("wght=%d,wdth=%d", w, d))
		}
	}
	faces := []struct {
		path       string
		texts      []string
		variations []string
	}{
		{test, []string{variedText, "LYAV Tox^x^ $$zz"}, grid},
		{"/System/Library/Fonts/SFNS.ttf", []string{"AVATAR Toyota, Wave $5", "office ffi 1/2 ‘quote’"},
			[]string{"wght=100", "wght=700", "wght=1000", "opsz=12", "opsz=28,wght=600", "wdth=30,wght=900"}},
		{"/System/Library/Fonts/SFArabic.ttf", []string{"بِسْمِ ٱللَّهِ ٱلرَّحْمَٰنِ", "السلام عليكم"},
			[]string{"wght=100", "wght=700", "wght=1000"}},
		{"/System/Library/Fonts/SFHebrew.ttf", []string{"בְּרֵאשִׁית בָּרָא"}, []string{"wght=200", "wght=900"}},
	}
	compared := 0
	for _, face := range faces {
		ttf, rerr := os.ReadFile(face.path)
		if rerr != nil {
			continue
		}
		v := parseVariable(t, ttf)
		for _, text := range face.texts {
			for _, variations := range face.variations {
				out, herr := exec.Command(hbShape, "--font-size="+fmt.Sprint(v.src.UnitsPerEm()), "--no-glyph-names",
					"--variations="+variations, face.path, text).Output()
				require.NoError(t, herr)
				want := strings.ReplaceAll(strings.TrimSpace(string(out)), "[", "[gid")
				want = strings.ReplaceAll(want, "|", "|gid")
				assert.Equal(t, want, v.shape(t, text, variations), "%s: %q at %s", face.path, text, variations)
				compared++
			}
		}
	}
	t.Logf("compared %d runs", compared)
}
