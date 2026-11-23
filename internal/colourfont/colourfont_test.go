package colourfont_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/colourfont"
	"github.com/lantervale/ember/internal/colourfont/colourtest"
)

func parse(t *testing.T, ttf []byte) *colourfont.Font {
	t.Helper()
	f, err := sfnt.Parse(ttf)
	require.NoError(t, err)
	cf := colourfont.Parse(ttf, f)
	require.NotNil(t, cf)
	return cf
}

var ink = color.NRGBA{0x33, 0x33, 0x33, 0xff} //nolint:gochecknoglobals // the fixtures' text colour

// at is the colour drawn at x, y em from the glyph's origin, y up, at
// 100 pixels an em.
func at(t *testing.T, cf *colourfont.Font, r rune, req colourfont.Request, x, y float64) color.NRGBA {
	t.Helper()
	req.PPEM = 100
	img, origin, ok := cf.Draw(colourtest.GlyphID(r), req)
	require.True(t, ok, "%c is drawn in colour", r)
	p := image.Pt(int(x*100)-origin.X, int(-y*100)-origin.Y)
	c, isNRGBA := color.NRGBAModel.Convert(img.At(p.X, p.Y)).(color.NRGBA)
	require.True(t, isNRGBA)
	return c
}

func near(t *testing.T, want, got color.NRGBA, msg string) {
	t.Helper()
	within(t, want, got, 3, msg)
}

func within(t *testing.T, want, got color.NRGBA, tol int, msg string) {
	t.Helper()
	d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	assert.True(t, d(want.R, got.R) <= tol && d(want.G, got.G) <= tol && d(want.B, got.B) <= tol &&
		d(want.A, got.A) <= tol, "%s: want %v, got %v", msg, want, got)
}

// The fixtures' shapes, in ems: the square from 0.05 to 0.55 both ways,
// the disc round (0.30, 0.325) 0.22 wide.
const (
	squareCorner = 0.08
	discX, discY = 0.302, 0.325
)

func TestCOLRv0LayersAreDrawnInTheirPalette(t *testing.T) {
	cf := parse(t, colourtest.COLRv0())
	assert.Equal(t, colourfont.Layers, cf.Kind(colourtest.GlyphID('A')))
	assert.Equal(t, colourfont.None, cf.Kind(colourtest.GlyphID('x')))
	assert.Equal(t, 3, cf.Palettes())
	near(t, colourtest.Default[2], at(t, cf, 'A', colourfont.Request{}, squareCorner, squareCorner), "the square")
	near(t, colourtest.Default[0], at(t, cf, 'A', colourfont.Request{}, discX, discY), "the disc over it")
	near(t, colourtest.Light[0], at(t, cf, 'A', colourfont.Request{Palette: 1}, discX, discY), "another palette")
	blue := color.NRGBA{0, 0, 0xff, 0xff}
	over := colourfont.Request{Palette: 2, Overrides: []colourfont.Override{{Index: 0, Color: blue}}}
	near(t, blue, at(t, cf, 'A', over, discX, discY), "an entry overridden")
	near(t, colourtest.Dark[2], at(t, cf, 'A', over, squareCorner, squareCorner), "the rest of its palette")
	near(t, ink, at(t, cf, 'B', colourfont.Request{Foreground: ink}, discX-0.21, discY), "the text's colour")
	assert.Equal(t, 1, cf.PaletteFor(false))
	assert.Equal(t, 2, cf.PaletteFor(true))
}

func TestCOLRv1PaintsGradientsTransformsAndComposites(t *testing.T) {
	cf := parse(t, colourtest.COLRv1())
	req := colourfont.Request{Foreground: ink}
	within(t, colourtest.Default[0], at(t, cf, 'A', req, 0.055, 0.3), 12, "a linear gradient's first stop")
	within(t, colourtest.Default[2], at(t, cf, 'A', req, 0.545, 0.3), 12, "and its last")
	mid := at(t, cf, 'A', req, 0.3, 0.3)
	near(t, colourtest.Default[3], mid, "and its middle")
	near(t, at(t, cf, 'A', req, 0.3, 0.1), mid, "its lines run across it")
	assert.NotEqual(t, at(t, cf, 'B', req, discX, discY+0.05), at(t, cf, 'B', req, discX, discY+0.15),
		"a radial gradient changes outwards")
	// The sweep goes from 180° to 360°, counter-clockwise: its first
	// stop lies left of its centre, padded above, and its last right,
	// just below.
	near(t, colourtest.Default[0], at(t, cf, 'C', req, 0.1, 0.27), "a sweep's start")
	within(t, colourtest.Default[2], at(t, cf, 'C', req, 0.5, 0.245), 12, "and its end")
	multiplied := at(t, cf, 'D', req, discX, discY)
	near(t, color.NRGBA{0x2a * 0xf0 / 255, 0x5a * 0xb0 / 255, 0xd0 * 0x20 / 255, 0xff}, multiplied, "multiplied")
	near(t, colourtest.Default[1], at(t, cf, 'E', req, 0.3, 0.6), "a square turned reaches higher")
	assert.Zero(t, at(t, cf, 'E', req, squareCorner, squareCorner).A, "and leaves its corner")
	near(t, colourtest.Default[2], at(t, cf, 'F', req, -0.1, 0.5), "skewed left as it rises")
	assert.Zero(t, at(t, cf, 'F', req, 0.5, 0.5).A, "away from its right")
	g := at(t, cf, 'G', req, discX+0.2, discY+0.15)
	assert.Greater(t, int(g.R), 0xe0, "a disc moved, at 0.7 over the triangle")
	near(t, ink, at(t, cf, 'H', req, discX-0.21, discY), "a ring in the text's colour")
	near(t, color.NRGBA{0x33, 0x33, 0x33, 0x80}, at(t, cf, 'J', req, 0.3, 0.3), "at half")
	near(t, colourtest.Default[0], at(t, cf, 'N', req, discX, discY), "src-in keeps the disc")
	assert.Zero(t, at(t, cf, 'N', req, squareCorner, squareCorner).A, "and nothing else")
	for r := 'A'; r <= 'N'; r++ {
		assert.Equal(t, colourfont.Layers, cf.Kind(colourtest.GlyphID(r)), "%c", r)
	}
}

func TestAClipBoxCutsWhatItsGlyphPaints(t *testing.T) {
	cf := parse(t, colourtest.COLRv1())
	img, origin, ok := cf.Draw(colourtest.GlyphID('H'), colourfont.Request{PPEM: 100, Foreground: ink})
	require.True(t, ok)
	b := img.Bounds().Add(origin)
	assert.GreaterOrEqual(t, b.Min.X, 5)
	assert.LessOrEqual(t, b.Max.X, 56)
	assert.GreaterOrEqual(t, b.Min.Y, -51)
}

func TestBitmapGlyphsAreScaledFromTheirStrike(t *testing.T) {
	for name, ttf := range map[string][]byte{"sbix": colourtest.Sbix(true), "CBDT": colourtest.CBDT(false)} {
		t.Run(name, func(t *testing.T) {
			cf := parse(t, ttf)
			assert.Equal(t, colourfont.Bitmap, cf.Kind(colourtest.GlyphID('A')))
			below := 0.0
			if name == "CBDT" {
				below = 13.0 / 64
			}
			near(t, colourtest.Default[0], at(t, cf, 'A', colourfont.Request{}, 0.5, 0.5-below), "the disc")
			near(t, colourtest.Default[2], at(t, cf, 'A', colourfont.Request{}, 0.16, 0.16-below), "the square")
			assert.Zero(t, at(t, cf, 'A', colourfont.Request{}, 0.05, 0.05-below).A, "the margin")
		})
	}
}

func TestAnSbixGlyphShowsWithinItsOutlinesBounds(t *testing.T) {
	cf := parse(t, colourtest.Sbix(true))
	img, origin, ok := cf.Draw(colourtest.GlyphID('B'), colourfont.Request{PPEM: 100})
	require.True(t, ok)
	f, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	var buf sfnt.Buffer
	segs, err := f.LoadGlyph(&buf, sfnt.GlyphIndex(colourtest.GlyphID('B')), 100*64, nil)
	require.NoError(t, err)
	right := segs.Bounds().Max.X.Ceil()
	for y := range img.Bounds().Dy() {
		for x := right - origin.X + 1; x < img.Bounds().Dx(); x++ {
			assert.Zero(t, img.RGBAAt(x, y).A, "past B's right at %d, %d", x, y)
		}
	}
}

func TestSVGGlyphsAreDrawnByTheSVGPackage(t *testing.T) {
	for _, zipped := range []bool{false, true} {
		cf := parse(t, colourtest.SVG(zipped))
		assert.Equal(t, colourfont.SVG, cf.Kind(colourtest.GlyphID('A')))
		near(t, colourtest.Default[0], at(t, cf, 'A', colourfont.Request{}, discX, 0.25), "the circle")
		near(t, colourtest.Default[2], at(t, cf, 'A', colourfont.Request{}, squareCorner, squareCorner), "the square")
		red := color.NRGBA{0xaa, 0, 0, 0xff}
		near(t, red, at(t, cf, 'C', colourfont.Request{Foreground: red}, discX-0.22, discY), "currentColor")
		_, _, ok := cf.Draw(colourtest.GlyphID('A'), colourfont.Request{PPEM: 100, NoSVG: true})
		assert.False(t, ok, "left to the outline")
	}
}

func TestAFontWithoutColourGlyphsIsNone(t *testing.T) {
	f, err := sfnt.Parse(goregular.TTF)
	require.NoError(t, err)
	assert.Nil(t, colourfont.Parse(goregular.TTF, f))
	var none *colourfont.Font
	assert.Equal(t, colourfont.None, none.Kind(3))
}

func TestACutColourTableDrawsWithoutPanicking(t *testing.T) {
	for _, ttf := range [][]byte{colourtest.COLRv0(), colourtest.COLRv1(), colourtest.Sbix(true),
		colourtest.CBDT(true), colourtest.SVG(true)} {
		f, err := sfnt.Parse(ttf)
		require.NoError(t, err)
		tables := colourtest.Tables(ttf)
		for _, tag := range []string{"COLR", "CPAL", "sbix", "CBLC", "CBDT", "SVG "} {
			whole, ok := tables[tag]
			if !ok {
				continue
			}
			for cut := 0; cut < len(whole); cut += max(1, len(whole)/40) {
				tables[tag] = whole[:cut]
				cut := colourtest.Assemble(tables)
				cf := colourfont.Parse(cut, f)
				for r := 'A'; r <= 'N'; r++ {
					cf.Draw(colourtest.GlyphID(r), colourfont.Request{PPEM: 40})
				}
			}
			tables[tag] = whole
		}
	}
}
