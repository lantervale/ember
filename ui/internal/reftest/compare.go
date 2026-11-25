package reftest

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

// Diff is how two pictures differ, as WPT counts it: the largest
// difference of a channel over every pixel, and how many pixels differ
// at all.
type Diff struct {
	MaxDifference int
	TotalPixels   int
}

// Equal reports whether the pictures are the same.
func (d Diff) Equal() bool { return d.MaxDifference == 0 && d.TotalPixels == 0 }

func (d Diff) String() string {
	return fmt.Sprintf("maxDifference=%d totalPixels=%d", d.MaxDifference, d.TotalPixels)
}

// Compare compares a and b pixel by pixel, over the part both cover; a
// pixel of one outside the other differs by 255.
func Compare(a, b image.Image) Diff {
	na, nb := toNRGBA(a), toNRGBA(b)
	if na.Rect.Size() == nb.Rect.Size() {
		var d Diff
		for i := 0; i < len(na.Pix); i += 4 {
			most := 0
			for c := range 4 {
				most = max(most, abs(int(na.Pix[i+c])-int(nb.Pix[i+c])))
			}
			if most > 0 {
				d.TotalPixels++
				d.MaxDifference = max(d.MaxDifference, most)
			}
		}
		return d
	}
	ra, rb := a.Bounds(), b.Bounds()
	w, h := max(ra.Dx(), rb.Dx()), max(ra.Dy(), rb.Dy())
	var d Diff
	for y := range h {
		for x := range w {
			pa, ina := at(a, x, y)
			pb, inb := at(b, x, y)
			most := 0
			if ina != inb {
				most = 255
			} else {
				for _, c := range [][2]uint8{{pa.R, pb.R}, {pa.G, pb.G}, {pa.B, pb.B}, {pa.A, pb.A}} {
					most = max(most, abs(int(c[0])-int(c[1])))
				}
			}
			if most > 0 {
				d.TotalPixels++
				d.MaxDifference = max(d.MaxDifference, most)
			}
		}
	}
	return d
}

// toNRGBA is img as non-premultiplied 8-bit colour, from its origin,
// tightly packed.
func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok && n.Rect.Min == (image.Point{}) && n.Stride == 4*n.Rect.Dx() {
		return n
	}
	if r, ok := img.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) && r.Stride == 4*r.Rect.Dx() && opaque(r) {
		return &image.NRGBA{Pix: r.Pix, Stride: r.Stride, Rect: r.Rect}
	}
	b := img.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Rect, img, b.Min, draw.Src)
	return n
}

// opaque reports whether every pixel of r is opaque, where premultiplied
// colour is the plain one.
func opaque(r *image.RGBA) bool {
	for i := 3; i < len(r.Pix); i += 4 {
		if r.Pix[i] != 0xff {
			return false
		}
	}
	return true
}

func at(img image.Image, x, y int) (color.NRGBA, bool) {
	r := img.Bounds()
	if x >= r.Dx() || y >= r.Dy() {
		return color.NRGBA{}, false
	}
	c, _ := color.NRGBAModel.Convert(img.At(r.Min.X+x, r.Min.Y+y)).(color.NRGBA)
	return c, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Passes reports whether pictures differing by d relate as rel says
// under the fuzzy rule f (nil for none), as WPT's runner decides: a match
// passes when they are the same or within f's ranges, a mismatch when
// they would not match.
func Passes(rel Rel, d Diff, f *Fuzzy) bool {
	same := d.Equal()
	if !same && f != nil {
		same = within(d.MaxDifference, f.MaxDifference) && within(d.TotalPixels, f.TotalPixels)
	}
	if rel == Mismatch {
		return !same
	}
	return same
}

func within(n int, r Range) bool { return n >= r.Min && n <= r.Max }
