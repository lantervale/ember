package draw //nolint:testpackage // white-box: reaches a glyph's coverage in the atlas

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// row is the coverage across g's middle row, as runs: [covered, clear,
// covered…] widths in atlas pixels, a pixel covered past half.
func row(f *Font, g glyph) []int {
	y := (g.rect.Min.Y + g.rect.Max.Y) / 2
	var runs []int
	on := false
	for x := g.rect.Min.X; x < g.rect.Max.X; x++ {
		c := f.Atlas.AlphaAt(x, y).A > 127
		if len(runs) == 0 && !c {
			continue
		}
		if len(runs) == 0 || c != on {
			runs, on = append(runs, 0), c
		}
		runs[len(runs)-1]++
	}
	if !on && len(runs) > 0 {
		runs = runs[:len(runs)-1]
	}
	return runs
}

func TestAStrokedGlyphIsItsOutlineAndKeepsItsAdvance(t *testing.T) {
	face, err := ParseFace("go", goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	plain, stroked := face.ExactSize(40), Style{Face: face, Size: 40, Exact: true, Stroke: 2}.Font()
	p, s := plain.glyph('l'), stroked.glyph('l')
	if p.advance != s.advance {
		t.Errorf("advance %v, stroked %v", p.advance, s.advance)
	}
	filled, outline := row(plain, p), row(stroked, s)
	if len(filled) != 1 || len(outline) != 3 {
		t.Fatalf("the stem's middle row: filled %v, stroked %v; want one run, then two round a hole", filled, outline)
	}
	// 2 points of stroke are 4 atlas pixels, half on either side of each edge
	if w := outline[0] + outline[1] + outline[2] - filled[0]; w < 3 || w > 5 {
		t.Errorf("the stroked stem is %d atlas pixels wider than the filled one, want 4", w)
	}
	if outline[0] < 3 || outline[0] > 5 || outline[2] < 3 || outline[2] > 5 {
		t.Errorf("the stroke's sides are %d and %d atlas pixels wide, want 4", outline[0], outline[2])
	}
	if (Style{Face: face, Size: 40, Exact: true, Stroke: 2}).Font() != stroked {
		t.Error("a second stroked font of the same size and width")
	}
}

func TestAStrokesSharpCornersAreMitred(t *testing.T) {
	face, err := ParseFace("go", goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	plain, stroked := face.ExactSize(40), Style{Face: face, Size: 40, Exact: true, Stroke: 4}.Font()
	top := func(f *Font, g glyph) float32 {
		for y := g.rect.Min.Y; y < g.rect.Max.Y; y++ {
			for x := g.rect.Min.X; x < g.rect.Max.X; x++ {
				if f.Atlas.AlphaAt(x, y).A > 127 {
					return g.offset.Y + float32(y-g.rect.Min.Y)/oversample
				}
			}
		}
		return 0
	}
	if up := top(plain, plain.glyph('M')) - top(stroked, stroked.glyph('M')); up < 3 {
		t.Errorf("the stroked M's sharp top is %v above the filled one's, want a mitre past the stroke's half, 2", up)
	}
	if up := top(plain, plain.glyph('o')) - top(stroked, stroked.glyph('o')); up < 1.5 || up > 2.5 {
		t.Errorf("the stroked o's top is %v above the filled one's, want the stroke's half, 2", up)
	}
}
