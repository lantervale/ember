package ico //nolint:testpackage // tests unexported internals

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/ico/icotest"
)

//nolint:gochecknoglobals // the colours of the files these tests write
var (
	red     = color.NRGBA{255, 0, 0, 255}
	green   = color.NRGBA{0, 255, 0, 255}
	blue    = color.NRGBA{0, 0, 255, 255}
	white   = color.NRGBA{255, 255, 255, 255}
	through = color.NRGBA{}
)

// quadrants is a 6×4 picture: red, green, blue and white quarters, from
// the top-left, across.
func quadrants(x, y int) color.NRGBA {
	switch {
	case x < 3 && y < 2:
		return red
	case y < 2:
		return green
	case x < 3:
		return blue
	}
	return white
}

//nolint:gochecknoglobals // the palette of the files these tests write
var quadrantPalette = []color.NRGBA{red, green, blue, white}

func index(x, y int) uint32 {
	for i, c := range quadrantPalette {
		if c == quadrants(x, y) {
			return uint32(i)
		}
	}
	return 0
}

func bgr(x, y int) uint32 {
	c := quadrants(x, y)
	return uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16
}

func bgra(x, y int) uint32 { return bgr(x, y) | 0xff000000 }

// packed is a pixel of 16 bits: five of red, gbits of green, five of
// blue.
func packed(gbits int) func(x, y int) uint32 {
	return func(x, y int) uint32 {
		c := quadrants(x, y)
		return uint32(c.R)>>3<<(gbits+5) | uint32(c.G)>>(8-gbits)<<5 | uint32(c.B)>>3
	}
}

// quadrantBitmaps are the picture of quadrants in every way a reader
// knows.
func quadrantBitmaps() map[string]icotest.Bitmap {
	base := icotest.Bitmap{W: 6, H: 4}
	with := func(edit func(b *icotest.Bitmap)) icotest.Bitmap {
		b := base
		edit(&b)
		return b
	}
	return map[string]icotest.Bitmap{
		"4 bits":  with(func(b *icotest.Bitmap) { b.BPP, b.Palette, b.At = 4, quadrantPalette, index }),
		"8 bits":  with(func(b *icotest.Bitmap) { b.BPP, b.Palette, b.At = 8, quadrantPalette, index }),
		"16 bits": with(func(b *icotest.Bitmap) { b.BPP, b.At = 16, packed(5) }),
		"24 bits": with(func(b *icotest.Bitmap) { b.BPP, b.At = 24, bgr }),
		"32 bits": with(func(b *icotest.Bitmap) { b.BPP, b.At = 32, bgra }),
		"8 bits, a palette of four": with(func(b *icotest.Bitmap) {
			b.BPP, b.Palette, b.Used, b.At = 8, quadrantPalette, 4, index
		}),
		"24 bits from the top": with(func(b *icotest.Bitmap) { b.BPP, b.At, b.TopDown = 24, bgr, true }),
		"4 bits from the top": with(func(b *icotest.Bitmap) {
			b.BPP, b.Palette, b.At, b.TopDown = 4, quadrantPalette, index, true
		}),
		"16 bits as 565": with(func(b *icotest.Bitmap) {
			b.BPP, b.Comp, b.Masks, b.At = 16, compBitfields, []uint32{0xf800, 0x7e0, 0x1f}, packed(6)
		}),
		"16 bits as 555 by masks": with(func(b *icotest.Bitmap) {
			b.BPP, b.Comp, b.Masks, b.At = 16, compBitfields, []uint32{0x7c00, 0x3e0, 0x1f}, packed(5)
		}),
		"32 bits by masks, red low": with(func(b *icotest.Bitmap) {
			b.BPP, b.Comp, b.Masks = 32, compBitfields, []uint32{0xff, 0xff00, 0xff0000}
			b.At = func(x, y int) uint32 {
				c := quadrants(x, y)
				return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16
			}
		}),
		"a V4 header": with(func(b *icotest.Bitmap) {
			b.BPP, b.Header, b.Comp, b.At = 32, infoHeaderV4, compBitfields, bgra
			b.Masks = []uint32{0xff0000, 0xff00, 0xff, 0xff000000}
		}),
		"a V5 header, 565": with(func(b *icotest.Bitmap) {
			b.BPP, b.Header, b.Comp, b.At = 16, infoHeaderV5, compBitfields, packed(6)
			b.Masks = []uint32{0xf800, 0x7e0, 0x1f}
		}),
		"a V5 header, 24 bits": with(func(b *icotest.Bitmap) { b.BPP, b.Header, b.At = 24, infoHeaderV5, bgr }),
	}
}

func assertQuadrants(t *testing.T, name string, img image.Image) {
	t.Helper()
	require.Equal(t, image.Rect(0, 0, 6, 4), img.Bounds(), name)
	for y := range 4 {
		for x := range 6 {
			assert.Equal(t, quadrants(x, y), color.NRGBAModel.Convert(img.At(x, y)), "%s at %d,%d", name, x, y)
		}
	}
}

func decodeBMP(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := DecodeBMP(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := Decode(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

func TestABMPOfEveryDepthDecodesToItsPixels(t *testing.T) {
	for name, b := range quadrantBitmaps() {
		assertQuadrants(t, name, decodeBMP(t, b.File()))
		cfg, err := DecodeBMPConfig(bytes.NewReader(b.File()))
		require.NoError(t, err, name)
		assert.Equal(t, 6, cfg.Width, name)
		assert.Equal(t, 4, cfg.Height, name)
	}
}

func TestAOneBitBMPTakesItsTwoColours(t *testing.T) {
	b := icotest.Bitmap{W: 9, H: 2, BPP: 1, Palette: []color.NRGBA{blue, white}, At: func(x, y int) uint32 {
		return uint32((x + y) % 2)
	}}
	img := decodeBMP(t, b.File())
	for y := range 2 {
		for x := range 9 {
			want := blue
			if (x+y)%2 == 1 {
				want = white
			}
			assert.Equal(t, want, img.At(x, y), "%d,%d", x, y)
		}
	}
}

func TestA32BitBMPHasAlphaOnlyWhereItsHeaderNamesIt(t *testing.T) {
	half := func(x, _ int) uint32 { return 0x00ff0000 | uint32(0x80*x)<<24 }
	plain := icotest.Bitmap{W: 2, H: 1, BPP: 32, At: half}
	assert.Equal(t, red, decodeBMP(t, plain.File()).At(1, 0), "no masks: no alpha, as Chrome")

	masks := []uint32{0xff0000, 0xff00, 0xff, 0xff000000}
	for name, b := range map[string]icotest.Bitmap{
		"masks after the header": {W: 2, H: 1, BPP: 32, Comp: compAlphaBitfields, Masks: masks, At: half},
		"masks of a V3 header":   {W: 2, H: 1, BPP: 32, Header: infoHeaderV3, Comp: compBitfields, Masks: masks, At: half},
		"masks of a V4 header":   {W: 2, H: 1, BPP: 32, Header: infoHeaderV4, Comp: compBitfields, Masks: masks, At: half},
		"a V5 header's alpha":    {W: 2, H: 1, BPP: 32, Header: infoHeaderV5, Masks: masks, At: half},
	} {
		img := decodeBMP(t, b.File())
		assert.Equal(t, color.NRGBA{255, 0, 0, 0}, img.At(0, 0), name)
		assert.Equal(t, color.NRGBA{255, 0, 0, 0x80}, img.At(1, 0), name)
	}

	none := icotest.Bitmap{W: 2, H: 1, BPP: 32, Header: infoHeaderV4, Comp: compBitfields, Masks: masks,
		At: func(int, int) uint32 { return 0x00ff0000 }}
	assert.Equal(t, color.NRGBA{255, 0, 0, 0}, decodeBMP(t, none.File()).At(0, 0), "all zero is all through, as Chrome")

	noAlpha := icotest.Bitmap{W: 1, H: 1, BPP: 32, Comp: compBitfields, Masks: masks[:3],
		At: func(int, int) uint32 { return 0x80ff0000 }}
	assert.Equal(t, red, decodeBMP(t, noAlpha.File()).At(0, 0), "masks without alpha give none")
}

func TestABMPsRowsMayStartInItsPalette(t *testing.T) {
	short := func(w, h int) []byte {
		b := icotest.Bitmap{W: w, H: h, BPP: 4, Palette: quadrantPalette, Used: 4, At: index}
		data := b.File()
		binary.LittleEndian.PutUint32(data[fileHeader+32:], 0)
		return data
	}
	img := decodeBMP(t, short(6, 20))
	assert.Equal(t, red, img.At(0, 0), "sixteen colours said, four before the rows: read as Chrome does")
	assert.Equal(t, white, img.At(5, 3))
	_, err := DecodeBMP(bytes.NewReader(short(6, 4)))
	assert.ErrorIs(t, err, errShort, "the file must hold the sixteen")
}

// quadrantRuns is quadrants run-length encoded: a run each side of a
// row, an end of line after each row and an end of bitmap, in bytes
// for 8 bits a pixel and in nibbles for 4.
func quadrantRuns(bpp int) []byte {
	var out []byte
	for i := range 4 {
		l, r := byte(index(0, 3-i)), byte(index(5, 3-i))
		if bpp == 4 {
			l, r = l<<4|l, r<<4|r
		}
		out = append(out, 3, l, 3, r, 0, endOfLine)
	}
	return append(out, 0, endOfBitmap)
}

func coded(comp uint32, runs []byte) icotest.Bitmap {
	bpp := 8
	if comp == compRLE4 {
		bpp = 4
	}
	return icotest.Bitmap{W: 6, H: 4, BPP: bpp, Comp: comp, Palette: quadrantPalette, Runs: runs}
}

func TestARunLengthEncodedBMPDecodesToItsPixels(t *testing.T) {
	assertQuadrants(t, "8 bits", decodeBMP(t, coded(compRLE8, quadrantRuns(8)).File()))
	assertQuadrants(t, "4 bits", decodeBMP(t, coded(compRLE4, quadrantRuns(4)).File()))
	assertQuadrants(t, "an icon", decode(t, icotest.ICO(coded(compRLE8, quadrantRuns(8)).Entry())))
}

// rows is the colours of a picture's rows, from the top, as letters:
// r, g, b and w for the quadrants' colours, k for black, . for clear.
func rows(img image.Image) []string {
	names := map[color.NRGBA]byte{red: 'r', green: 'g', blue: 'b', white: 'w', {0, 0, 0, 255}: 'k', through: '.'}
	var out []string
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		var row []byte
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c, ok := names[color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)]
			if !ok {
				c = '?'
			}
			row = append(row, c)
		}
		out = append(out, string(row))
	}
	return out
}

func TestRunsLeaveBlackWhatTheySkipAndStopWhereTheyLeaveThePicture(t *testing.T) {
	for name, c := range map[string]struct {
		comp uint32
		runs []byte
		want []string
	}{
		"written out, one padded, then an end of line": {compRLE8, []byte{
			0, 3, 0, 1, 2, 0, 0, endOfLine, 0, endOfLine, 2, 3, 0, endOfBitmap,
		}, []string{"kkkkkk", "wwkkkk", "kkkkkk", "rgbkkk"}},
		"a move over pixels and up a row": {compRLE8, []byte{
			1, 2, 0, delta, 3, 1, 2, 1, 0, endOfBitmap,
		}, []string{"kkkkkk", "kkkkkk", "kkkkgg", "bkkkkk"}},
		"a run past its row, cut there": {compRLE8, []byte{9, 3, 1, 0, 0, endOfLine, 1, 1}, []string{
			"kkkkkk", "kkkkkk", "gkkkkk", "wwwwww",
		}},
		"a move past the row, the end": {compRLE8, []byte{6, 3, 0, delta, 1, 0, 1, 1}, []string{
			"kkkkkk", "kkkkkk", "kkkkkk", "wwwwww",
		}},
		"a move to the row's end, then up": {compRLE8, []byte{0, delta, 6, 1, 0, endOfLine, 2, 2}, []string{
			"kkkkkk", "bbkkkk", "kkkkkk", "kkkkkk",
		}},
		"a move above the picture, the end": {compRLE8, []byte{1, 1, 0, delta, 0, 5, 1, 1}, []string{
			"kkkkkk", "kkkkkk", "kkkkkk", "gkkkkk",
		}},
		"pixels written past the row, the end": {compRLE8, []byte{2, 3, 0, 5, 1, 1, 1, 1, 1, 0}, []string{
			"kkkkkk", "kkkkkk", "kkkkkk", "wwkkkk",
		}},
		"pixels written past the file, the end": {compRLE8, []byte{1, 3, 0, 4, 1, 1}, []string{
			"kkkkkk", "kkkkkk", "kkkkkk", "wkkkkk",
		}},
		"rows past the top, never written": {compRLE8, []byte{
			0, endOfLine, 0, endOfLine, 0, endOfLine, 0, endOfLine, 6, 1,
		}, []string{"kkkkkk", "kkkkkk", "kkkkkk", "kkkkkk"}},
		"no end of bitmap": {compRLE8, []byte{2, 2}, []string{"kkkkkk", "kkkkkk", "kkkkkk", "bbkkkk"}},
		"nibbles, alternating": {compRLE4, []byte{5, 0x13, 0, endOfLine, 0, 3, 0x21, 0x30}, []string{
			"kkkkkk", "kkkkkk", "bgwkkk", "gwgwgk",
		}},
	} {
		assert.Equal(t, c.want, rows(decodeBMP(t, coded(c.comp, c.runs).File())), name)
	}
}

func TestAnICOsRunsAreCutByTheMaskThatFollowsThem(t *testing.T) {
	b := coded(compRLE8, quadrantRuns(8))
	b.Hole = func(x, y int) bool { return x == 0 }
	assert.Equal(t, []string{".rrggg", ".rrggg", ".bbwww", ".bbwww"}, rows(decode(t, icotest.ICO(b.Entry()))))

	entry := b.Entry()
	binary.LittleEndian.PutUint32(entry.Data[20:], uint32(len(entry.Data)))
	assert.Equal(t, []string{"rrrggg", "rrrggg", "bbbwww", "bbbwww"}, rows(decode(t, icotest.ICO(entry))),
		"runs whose size leaves no room for a mask")
}

func TestRunsOfTheWrongDepthOrFromTheTopAreRefused(t *testing.T) {
	b := coded(compRLE8, quadrantRuns(8))
	b.BPP = 4
	_, err := DecodeBMP(bytes.NewReader(b.File()))
	require.ErrorIs(t, err, errHeader)
	b = coded(compRLE4, quadrantRuns(4))
	b.BPP = 8
	_, err = DecodeBMP(bytes.NewReader(b.File()))
	require.ErrorIs(t, err, errHeader)
	b = coded(compRLE8, quadrantRuns(8))
	b.TopDown = true
	_, err = DecodeBMP(bytes.NewReader(b.File()))
	require.ErrorIs(t, err, errHeader)
}

func TestACursorIsReadAsAnIcon(t *testing.T) {
	bitmap := icotest.Bitmap{W: 6, H: 4, BPP: 8, Palette: quadrantPalette, At: index}
	assertQuadrants(t, "a cursor", decode(t, icotest.CUR(bitmap.Entry())))
	small := icotest.Bitmap{W: 6, H: 4, BPP: 4, Palette: []color.NRGBA{blue}, At: func(int, int) uint32 { return 0 }}
	assertQuadrants(t, "the deepest, by its bitmap: the directory says its hotspot",
		decode(t, icotest.CUR(bitmap.Entry(), small.Entry())))
	cfg, err := DecodeConfig(bytes.NewReader(icotest.CUR(bitmap.Entry())))
	require.NoError(t, err)
	assert.Equal(t, 6, cfg.Width)
}

func TestABMPSayingWhatNoReaderKnowsIsRefused(t *testing.T) {
	good := icotest.Bitmap{W: 6, H: 4, BPP: 24, At: bgr}.File()
	put16 := func(at int, v uint16) []byte {
		data := bytes.Clone(good)
		binary.LittleEndian.PutUint16(data[at:], v)
		return data
	}
	put32 := func(at int, v uint32) []byte {
		data := bytes.Clone(good)
		binary.LittleEndian.PutUint32(data[at:], v)
		return data
	}
	cases := map[string]struct {
		data []byte
		want error
	}{
		"no BM":                             {put16(0, 0x4142), errNoBMP},
		"a header of 12 bytes":              {put32(14, 12), errHeader},
		"a header of 64 bytes":              {put32(14, 64), errHeader},
		"2 bits a pixel":                    {put16(28, 2), errDepth},
		"0 bits a pixel":                    {put16(28, 0), errDepth},
		"JPEG inside":                       {put32(30, 4), errCompression},
		"masks at 24 bits":                  {put32(30, compBitfields), errHeader},
		"no width":                          {put32(18, 0), errNoSize},
		"a width below zero":                {put32(18, 0xffffffff), errNoSize},
		"no height":                         {put32(22, 0), errNoSize},
		"the lowest height":                 {put32(22, 0x80000000), errHeader},
		"a width over the cap":              {put32(18, MaxSide+1), errTooLarge},
		"the widest width":                  {put32(18, 0x7fffffff), errTooLarge},
		"a height over the cap":             {put32(22, MaxSide+1), errTooLarge},
		"rows past the file":                {put32(10, uint32(len(good)+1)), errOffset},
		"rows at the far end":               {put32(10, 0xffffffff), errOffset},
		"rows in the file header":           {put32(10, 4), errOffset},
		"rows in the icotest.Bitmap header": {put32(10, fileHeader+8), errOffset},
		"rows starting a byte too late":     {put32(10, fileHeader+infoHeader+1), errShort},
		"a height the file lacks":           {put32(22, 5), errShort},
		"a height of thousands":             {put32(22, 4000), errShort},
	}
	for name, c := range cases {
		_, err := DecodeBMP(bytes.NewReader(c.data))
		assert.ErrorIs(t, err, c.want, name)
		_, err = DecodeBMPConfig(bytes.NewReader(c.data))
		assert.ErrorIs(t, err, c.want, name)
	}

	wide := icotest.Bitmap{W: 6000, H: 1, BPP: 1, Palette: []color.NRGBA{red, blue}, At: index}.File()
	binary.LittleEndian.PutUint32(wide[22:], 6000)
	_, err := DecodeBMP(bytes.NewReader(wide))
	require.ErrorIs(t, err, errTooLarge, "each side under the cap, the pixels over it")

	many := icotest.Bitmap{W: 6, H: 4, BPP: 4, Palette: quadrantPalette, Used: 17, At: index}.File()
	_, err = DecodeBMP(bytes.NewReader(many))
	require.ErrorIs(t, err, errPalette)

	masks := icotest.Bitmap{
		W: 1, H: 1, BPP: 16, Comp: compBitfields, Masks: []uint32{0xf800, 0x7e0, 0x1f}, At: packed(6),
	}.File()
	_, err = DecodeBMP(bytes.NewReader(masks[:fileHeader+infoHeader+8]))
	assert.ErrorIs(t, err, errOffset, "masks cut short, and the rows with them")
}

func TestEveryTruncatedBMPIsAnError(t *testing.T) {
	for name, b := range quadrantBitmaps() {
		data := b.File()
		for n := range len(data) {
			_, err := DecodeBMP(bytes.NewReader(data[:n]))
			require.Error(t, err, "%s cut at %d of %d", name, n, len(data))
		}
	}
}

func TestMasksOfAnyShapeGiveAColour(t *testing.T) {
	for _, masks := range [][]uint32{
		{0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff}, {0, 0, 0, 0}, {0x80000001, 0x5555, 0xaaaa, 1},
		{0xffff0000, 0xff00, 0xf0, 0xf},
	} {
		b := icotest.Bitmap{W: 3, H: 1, BPP: 32, Header: infoHeaderV3, Comp: compBitfields, Masks: masks,
			At: func(x, _ int) uint32 { return uint32(x) * 0x7fffffff }}
		_, err := DecodeBMP(bytes.NewReader(b.File()))
		require.NoError(t, err)
	}
}

func TestAnICOOfEveryDepthDecodesToItsPixels(t *testing.T) {
	for name, b := range quadrantBitmaps() {
		if b.TopDown {
			continue
		}
		assertQuadrants(t, name, decode(t, icotest.ICO(b.Entry())))
		cfg, err := DecodeConfig(bytes.NewReader(icotest.ICO(b.Entry())))
		require.NoError(t, err, name)
		assert.Equal(t, 6, cfg.Width, name)
		assert.Equal(t, 4, cfg.Height, name)
	}
}

func TestAnICOsMaskCutsWhatHasNoAlpha(t *testing.T) {
	corner := func(x, y int) bool { return x == 0 && y == 0 || x == 5 && y == 3 }
	for name, b := range quadrantBitmaps() {
		if b.TopDown || b.BPP == 32 {
			continue
		}
		b.Hole = corner
		img := decode(t, icotest.ICO(b.Entry()))
		for y := range 4 {
			for x := range 6 {
				want := quadrants(x, y)
				if corner(x, y) {
					want = through
				}
				assert.Equal(t, want, img.At(x, y), "%s at %d,%d", name, x, y)
			}
		}
	}

	one := icotest.Bitmap{W: 33, H: 2, BPP: 1, Palette: []color.NRGBA{blue, white},
		At:   func(x, _ int) uint32 { return uint32(x % 2) },
		Hole: func(x, y int) bool { return x == 32 && y == 1 }}
	img := decode(t, icotest.ICO(one.Entry()))
	assert.Equal(t, blue, img.At(32, 0))
	assert.Equal(t, through, img.At(32, 1))
	assert.Equal(t, white, img.At(31, 1))
}

func TestA32BitICOUsesItsAlphaAndElseItsMask(t *testing.T) {
	hole := func(x, y int) bool { return x == 0 && y == 0 }
	alpha := icotest.Bitmap{W: 6, H: 4, BPP: 32, Hole: hole, At: func(x, y int) uint32 {
		if x == 5 && y == 3 {
			return bgr(x, y) | 0x40000000
		}
		return bgra(x, y)
	}}
	img := decode(t, icotest.ICO(alpha.Entry()))
	assert.Equal(t, red, img.At(0, 0), "the mask says nothing where there is alpha")
	assert.Equal(t, color.NRGBA{255, 255, 255, 0x40}, img.At(5, 3))

	zero := icotest.Bitmap{W: 6, H: 4, BPP: 32, Hole: hole, At: bgr}
	img = decode(t, icotest.ICO(zero.Entry()))
	assert.Equal(t, through, img.At(0, 0), "alpha all zero: the mask cuts")
	assert.Equal(t, white, img.At(5, 3))

	e := zero.Entry()
	e.Data = e.Data[:len(e.Data)-stride(6, 1)*4]
	img = decode(t, icotest.ICO(e))
	assert.Equal(t, red, img.At(0, 0), "no mask and no alpha: opaque")

	e = icotest.Bitmap{W: 6, H: 4, BPP: 24, At: bgr}.Entry()
	e.Data = e.Data[:len(e.Data)-1]
	_, err := Decode(bytes.NewReader(icotest.ICO(e)))
	assert.ErrorIs(t, err, errShort, "24 bits need their mask whole")
}

func TestAnICOMayHoldAPNG(t *testing.T) {
	data := icotest.ICO(icotest.Entry{W: 7, H: 5, BPP: 32, Data: icotest.PNG(7, 5, green)})
	img := decode(t, data)
	assert.Equal(t, image.Rect(0, 0, 7, 5), img.Bounds())
	assert.Equal(t, green, color.NRGBAModel.Convert(img.At(6, 4)))
	cfg, err := DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, 7, cfg.Width)
	assert.Equal(t, 5, cfg.Height)
}

func TestAnICOShowsItsLargestPictureThenItsDeepest(t *testing.T) {
	flat := func(w, bpp int, c color.NRGBA) icotest.Entry {
		b := icotest.Bitmap{W: w, H: w, BPP: bpp, Palette: []color.NRGBA{c}, At: func(int, int) uint32 { return 0 }}
		if bpp > 8 {
			b.Palette, b.At = nil, func(int, int) uint32 { return uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16 | 0xff000000 }
		}
		return b.Entry()
	}
	img := decode(t, icotest.ICO(flat(16, 32, red), flat(48, 8, green), flat(32, 32, blue)))
	assert.Equal(t, 48, img.Bounds().Dx(), "the largest, whatever its depth")
	assert.Equal(t, green, img.At(0, 0))

	img = decode(t, icotest.ICO(flat(16, 4, red), flat(16, 32, blue), flat(16, 8, green)))
	assert.Equal(t, blue, img.At(0, 0), "of one size, the deepest")

	unsaid := flat(16, 32, blue)
	unsaid.BPP = 0
	img = decode(t, icotest.ICO(flat(16, 8, red), unsaid))
	assert.Equal(t, blue, img.At(0, 0), "a depth the directory leaves out is read from the icotest.Bitmap")

	img = decode(t, icotest.ICO(flat(16, 24, red), icotest.Entry{W: 16, H: 16, Data: icotest.PNG(16, 16, green)}))
	assert.Equal(t, green, color.NRGBAModel.Convert(img.At(0, 0)), "a PNG counts as 32 bits")

	img = decode(t, icotest.ICO(flat(64, 8, red), icotest.Entry{W: 0, H: 0, BPP: 32, Data: icotest.PNG(256, 256, green)}))
	assert.Equal(t, 256, img.Bounds().Dx(), "a side of 0 is 256")
}

// hugePNG is a PNG file's start saying it is w by h.
func hugePNG(w, h uint32) []byte {
	ihdr := []byte("IHDR")
	ihdr = binary.BigEndian.AppendUint32(ihdr, w)
	ihdr = binary.BigEndian.AppendUint32(ihdr, h)
	ihdr = append(ihdr, 8, 6, 0, 0, 0)
	out := binary.BigEndian.AppendUint32([]byte(pngSignature), 13)
	out = append(out, ihdr...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(ihdr))
}

func pngICO(data []byte) []byte { return icotest.ICO(icotest.Entry{W: 16, H: 16, Data: data}) }

func TestAnICOSayingWhatItDoesNotHoldIsRefused(t *testing.T) {
	good := icotest.ICO(icotest.Bitmap{W: 6, H: 4, BPP: 24, At: bgr}.Entry())
	const dir, pic = dirHeader, dirHeader + dirEntry
	put16 := func(at int, v uint16) []byte {
		data := bytes.Clone(good)
		binary.LittleEndian.PutUint16(data[at:], v)
		return data
	}
	put32 := func(at int, v uint32) []byte {
		data := bytes.Clone(good)
		binary.LittleEndian.PutUint32(data[at:], v)
		return data
	}
	cases := map[string]struct {
		data []byte
		want error
	}{
		"a first word that isn't zero":   {put16(0, 1), errNoICO},
		"a third kind":                   {put16(2, 3), errNoICO},
		"no entry":                       {put16(4, 0), errEmpty},
		"more entries than the file has": {put16(4, 2), errEntry},
		"65535 entries":                  {put16(4, 0xffff), errShort},
		"an entry past the file":         {put32(dir+12, uint32(len(good))+1), errEntry},
		"an entry at the far end":        {put32(dir+12, 0xffffffff), errEntry},
		"an entry in the directory":      {put32(dir+12, 4), errEntry},
		"an entry longer than the file":  {put32(dir+8, uint32(len(good))), errEntry},
		"an entry of the longest length": {put32(dir+8, 0xffffffff), errEntry},
		"an entry of no length":          {put32(dir+8, 0), errShort},
		"an entry of a header only":      {put32(dir+8, infoHeader), errShort},
		"a height that isn't doubled":    {put32(pic+8, 5), errHeader},
		"rows from the top":              {put32(pic+8, 0xfffffff8), errHeader},
		"a width over the cap":           {put32(pic+4, MaxSide+1), errTooLarge},
		"a height over the cap":          {put32(pic+8, 2*MaxSide+2), errTooLarge},
		"a width the entry lacks":        {put32(pic+4, 7), errShort},
		"no width":                       {put32(pic+4, 0), errNoSize},
		"a header of 12 bytes":           {put32(pic, 12), errHeader},
		"run-length encoded at 24 bits":  {put32(pic+16, compRLE8), errHeader},
		"a PNG over the cap":             {pngICO(hugePNG(MaxSide+1, 1)), errTooLarge},
		"a PNG of too many pixels":       {pngICO(hugePNG(8192, 8192)), errTooLarge},
	}
	for name, c := range cases {
		_, err := Decode(bytes.NewReader(c.data))
		assert.ErrorIs(t, err, c.want, name)
		_, err = DecodeConfig(bytes.NewReader(c.data))
		assert.ErrorIs(t, err, c.want, name)
	}

	_, err := Decode(bytes.NewReader(icotest.ICO(icotest.Entry{W: 16, H: 16, Data: hugePNG(0x7fffffff, 0x7fffffff)})))
	require.Error(t, err, "a PNG of the largest size")
	_, err = Decode(bytes.NewReader(icotest.ICO(icotest.Entry{W: 16, H: 16, Data: icotest.PNG(16, 16, red)[:40]})))
	require.Error(t, err, "a PNG cut short")

	small := icotest.Bitmap{W: 2, H: 2, BPP: 24, At: bgr}
	second := icotest.ICO(icotest.Bitmap{W: 6, H: 4, BPP: 24, At: bgr}.Entry(), small.Entry())
	binary.LittleEndian.PutUint32(second[dirHeader+dirEntry+12:], uint32(len(second)))
	binary.LittleEndian.PutUint32(second[dirHeader+dirEntry+8:], 1)
	_, err = Decode(bytes.NewReader(second))
	assert.ErrorIs(t, err, errEntry, "an entry that isn't the one shown is checked too")
}

func TestEveryTruncatedICOIsAnError(t *testing.T) {
	files := map[string][]byte{"a PNG": icotest.ICO(icotest.Entry{W: 7, H: 5, BPP: 32, Data: icotest.PNG(7, 5, green)})}
	for name, b := range quadrantBitmaps() {
		if !b.TopDown {
			files[name] = icotest.ICO(b.Entry())
		}
	}
	for name, data := range files {
		for n := range len(data) {
			_, err := Decode(bytes.NewReader(data[:n]))
			require.Error(t, err, "%s cut at %d of %d", name, n, len(data))
		}
	}
}

// endless is a reader that never ends.
type endless struct{}

func (endless) Read(p []byte) (int, error) { return len(p), nil }

func TestAFileOverTheLongestReadIsRefused(t *testing.T) {
	if testing.Short() {
		t.Skip("reads over a hundred megabytes")
	}
	_, err := Decode(endless{})
	require.ErrorIs(t, err, errFileSize)
	_, err = DecodeBMP(io.MultiReader(bytes.NewReader([]byte("BM")), endless{}))
	assert.ErrorIs(t, err, errFileSize)
}

// seeds are the valid files of these tests, and a few broken ones.
func seeds(f *testing.F, file func(b icotest.Bitmap) []byte) {
	for _, b := range quadrantBitmaps() {
		f.Add(file(b))
	}
	f.Add(file(coded(compRLE8, quadrantRuns(8))))
	f.Add(file(coded(compRLE4, quadrantRuns(4))))
	f.Add(file(coded(compRLE8, []byte{0, 3, 0, 1, 2, 0, 0, delta, 3, 1, 2, 1, 0, endOfBitmap})))
	f.Add(file(icotest.Bitmap{W: 9, H: 2, BPP: 1, Palette: []color.NRGBA{blue, white}, At: func(x, y int) uint32 {
		return uint32((x + y) % 2)
	}, Hole: func(x, y int) bool { return x == y }}))
}

// read decodes data, which must not panic, and checks a picture that
// comes out is within the caps and is the size its config says.
func read(t *testing.T, data []byte, decode func(io.Reader) (image.Image, error),
	config func(io.Reader) (image.Config, error)) {
	t.Helper()
	cfg, cfgErr := config(bytes.NewReader(data))
	img, err := decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	require.NoError(t, cfgErr)
	size := img.Bounds().Size()
	require.Equal(t, image.Pt(cfg.Width, cfg.Height), size)
	require.Positive(t, size.X)
	require.Positive(t, size.Y)
	require.LessOrEqual(t, size.X, MaxSide)
	require.LessOrEqual(t, size.Y, MaxSide)
	require.LessOrEqual(t, size.X*size.Y, MaxPixels)
}

func FuzzDecode(f *testing.F) {
	seeds(f, func(b icotest.Bitmap) []byte {
		b.TopDown = false
		return icotest.ICO(b.Entry())
	})
	f.Add(icotest.ICO(icotest.Entry{W: 7, H: 5, BPP: 32, Data: icotest.PNG(7, 5, green)}))
	f.Add(icotest.ICO(icotest.Bitmap{W: 6, H: 4, BPP: 24, At: bgr}.Entry(),
		icotest.Entry{W: 16, H: 16, Data: icotest.PNG(16, 16, red)},
		icotest.Bitmap{W: 6, H: 4, BPP: 4, Palette: quadrantPalette, At: index}.Entry()))
	f.Add(icotest.ICO(icotest.Entry{W: 16, H: 16, Data: hugePNG(MaxSide+1, 1)}))
	f.Fuzz(func(t *testing.T, data []byte) { read(t, data, Decode, DecodeConfig) })
}

func FuzzDecodeBMP(f *testing.F) {
	seeds(f, icotest.Bitmap.File)
	f.Fuzz(func(t *testing.T, data []byte) { read(t, data, DecodeBMP, DecodeBMPConfig) })
}
