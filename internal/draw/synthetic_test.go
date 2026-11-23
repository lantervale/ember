package draw //nolint:testpackage // white-box: reaches a glyph's coverage in the atlas

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/internal/varfont/varfonttest"
)

func coverage(f *Font, r rune) (int, image.Rectangle) {
	g := f.glyph(r)
	sum := 0
	for y := g.rect.Min.Y; y < g.rect.Max.Y; y++ {
		for x := g.rect.Min.X; x < g.rect.Max.X; x++ {
			sum += int(f.Atlas.AlphaAt(x, y).A)
		}
	}
	return sum, g.rect
}

func TestAMadeUpBoldCoversMoreAndKeepsItsAdvance(t *testing.T) {
	face, err := ParseFace("go", goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	plain, bold := face.ExactSize(24), face.Styled(Look{Bold: true}).ExactSize(24)
	pc, pr := coverage(plain, 'l')
	bc, br := coverage(bold, 'l')
	if bc <= pc*11/10 {
		t.Errorf("a bold l covers %d, the regular %d", bc, pc)
	}
	if br.Dx() < pr.Dx() {
		t.Errorf("a bold l is %d pixels wide, the regular %d", br.Dx(), pr.Dx())
	}
	if plain.glyph('l').advance != bold.glyph('l').advance {
		t.Errorf("advance %v, made bold %v", plain.glyph('l').advance, bold.glyph('l').advance)
	}
}

func TestAMadeUpItalicLeansRight(t *testing.T) {
	face, err := ParseFace("go", goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	upright, slanted := face.ExactSize(40), face.Styled(Look{Oblique: true}).ExactSize(40)
	u, s := upright.glyph('I'), slanted.glyph('I')
	if lean := s.size.X - u.size.X; lean < 0.2*u.size.Y || lean > 0.3*u.size.Y {
		t.Errorf("an I %v tall leans %v, want a quarter of its height", u.size.Y, lean)
	}
	if u.advance != s.advance {
		t.Errorf("advance %v, slanted %v", u.advance, s.advance)
	}
}

func TestAScaledFaceDrawsAndAdvancesLarger(t *testing.T) {
	face, err := ParseFace("go", goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	plain, scaled := face.Size(20), face.Styled(Look{Scale: 1.5}).Size(20)
	if a, b := plain.glyph('M').advance, scaled.glyph('M').advance; b < a*1.49 || b > a*1.51 {
		t.Errorf("advance %v, scaled by 1.5 %v", a, b)
	}
	if a, b := plain.glyph('M').size.Y, scaled.glyph('M').size.Y; b < a*1.3 {
		t.Errorf("an M %v tall, scaled by 1.5 %v", a, b)
	}
}

func TestAnUnkernedFaceKernsNoPair(t *testing.T) {
	ttf, err := os.ReadFile(filepath.Join("testdata", "go-kerned.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	face, err := ParseFace("kerned", ttf)
	if err != nil {
		t.Fatal(err)
	}
	kerned, unkerned := face.Size(40), face.Styled(Look{NoKern: true}).Size(40)
	if k := kerned.kern(kerned.glyph('A'), kerned.glyph('V')); k == 0 {
		t.Fatal("the face kerns no AV")
	}
	if k := unkerned.kern(unkerned.glyph('A'), unkerned.glyph('V')); k != 0 {
		t.Errorf("unkerned AV: %v", k)
	}
}

func TestAVariableFontsInstanceDrawsItsOwnOutlinesAndAdvances(t *testing.T) {
	ttf := varfonttest.Weighty()
	face, err := ParseFace("weighty", ttf)
	if err != nil {
		t.Fatal(err)
	}
	vf, err := varfont.Parse(ttf)
	if err != nil {
		t.Fatal(err)
	}
	at := func(user ...float32) *Font {
		return face.Styled(Look{Variation: &Variation{Font: vf, Coords: vf.Normalize(user)}}).ExactSize(24)
	}
	plain, bold, wide, still := face.ExactSize(24), at(900, 100), at(400, 150), at(400, 100)
	pc, pr := coverage(plain, 'l')
	bc, _ := coverage(bold, 'l')
	sc, sr := coverage(still, 'l')
	if bc <= pc*13/10 {
		t.Errorf("an l at 900 covers %d, at 400 %d", bc, pc)
	}
	if sc != pc || sr.Size() != pr.Size() {
		t.Errorf("the default instance covers %d in %v, the file as it is %d in %v", sc, sr.Size(), pc, pr.Size())
	}
	const units = 24.0 / 2048
	got, want := bold.glyph('l').advance-plain.glyph('l').advance, float32(2*varfonttest.Bolder*units)
	if !near(got, want) {
		t.Errorf("an l at 900 is %v wider, want %v", got, want)
	}
	if w, p := wide.Measure("AVOH").X, plain.Measure("AVOH").X; w < p*1.35 {
		t.Errorf("AVOH at a width of 150 is %v, at 100 %v", w, p)
	}
	if _, wr := coverage(wide, 'O'); wr.Dx() < pr.Dx()*2 {
		t.Errorf("an O at a width of 150 is %d pixels wide, an l %d", wr.Dx(), pr.Dx())
	}
}

func near(a, b float32) bool { return a-b < 1e-3 && b-a < 1e-3 }
