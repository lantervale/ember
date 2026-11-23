package svg

import "math"

// maxMorphology is the widest radius, in pixels, a morphology is run
// with.
const maxMorphology = 512

// blur blurs pix, the run's box, by sx and sy pixels of standard
// deviation, as Filter Effects asks: a Gaussian kernel under 2 pixels,
// three box blurs from there.
func (fr *filterRun) blur(pix []float32, sx, sy float64) {
	w, h := fr.w, fr.box.Dy()
	line, tmp := make([]float32, 4*max(w, h)), make([]float32, 4*max(w, h))
	if sx > 0 {
		for y := range h {
			row := pix[4*y*w:][:4*w]
			copy(line, row)
			blurLine(line[:4*w], tmp[:4*w], sx)
			copy(row, line)
		}
	}
	if sy > 0 {
		for x := range w {
			for y := range h {
				copy(line[4*y:4*y+4], pix[4*(y*w+x):])
			}
			blurLine(line[:4*h], tmp[:4*h], sy)
			for y := range h {
				copy(pix[4*(y*w+x):][:4], line[4*y:4*y+4])
			}
		}
	}
}

func blurLine(line, tmp []float32, sigma float64) {
	if sigma < 2 {
		gaussian(line, tmp, sigma)
		return
	}
	d := int(math.Floor(sigma*3*math.Sqrt(2*math.Pi)/4 + 0.5))
	if d%2 == 1 {
		for range 3 {
			box(line, tmp, d/2, d/2)
		}
		return
	}
	box(line, tmp, d/2, d/2-1)
	box(line, tmp, d/2-1, d/2)
	box(line, tmp, d/2, d/2)
}

// box averages each pixel of line with l pixels before it and r after,
// those past the ends clear.
func box(line, tmp []float32, l, r int) {
	n := len(line) / 4
	var sum [4]float32
	for j := 0; j <= min(r, n-1); j++ {
		for c := range 4 {
			sum[c] += line[4*j+c]
		}
	}
	scale := 1 / float32(l+r+1)
	for i := range n {
		for c := range 4 {
			tmp[4*i+c] = sum[c] * scale
		}
		if in := i + 1 + r; in < n {
			for c := range 4 {
				sum[c] += line[4*in+c]
			}
		}
		if out := i - l; out >= 0 {
			for c := range 4 {
				sum[c] -= line[4*out+c]
			}
		}
	}
	copy(line, tmp)
}

func gaussian(line, tmp []float32, sigma float64) {
	n := len(line) / 4
	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float32, 2*radius+1)
	total := 0.0
	for i := range kernel {
		x := float64(i - radius)
		v := math.Exp(-x * x / (2 * sigma * sigma))
		kernel[i] = float32(v)
		total += v
	}
	for i := range kernel {
		kernel[i] /= float32(total)
	}
	for i := range n {
		var sum [4]float32
		for k, wt := range kernel {
			j := i + k - radius
			if j < 0 || j >= n {
				continue
			}
			for c := range 4 {
				sum[c] += line[4*j+c] * wt
			}
		}
		copy(tmp[4*i:4*i+4], sum[:])
	}
	copy(line, tmp)
}

// morphology thins (or, dilate, thickens) pix by rx and ry pixels: each
// channel the least (most) of those within the radii, clear past the
// box.
func (fr *filterRun) morphology(pix []float32, rx, ry int, dilate bool) {
	w, h := fr.w, fr.box.Dy()
	rx, ry = min(rx, maxMorphology), min(ry, maxMorphology)
	line, tmp := make([]float32, 4*max(w, h)), make([]float32, 4*max(w, h))
	pass := func(n, r int) {
		for i := range n {
			var best [4]float32
			if !dilate {
				best = [4]float32{1, 1, 1, 1}
			}
			for j := i - r; j <= i+r; j++ {
				for c := range 4 {
					v := float32(0)
					if j >= 0 && j < n {
						v = line[4*j+c]
					}
					if dilate {
						best[c] = max(best[c], v)
					} else {
						best[c] = min(best[c], v)
					}
				}
			}
			copy(tmp[4*i:4*i+4], best[:])
		}
		copy(line[:4*n], tmp[:4*n])
	}
	if rx > 0 {
		for y := range h {
			row := pix[4*y*w:][:4*w]
			copy(line, row)
			pass(w, rx)
			copy(row, line)
		}
	}
	if ry > 0 {
		for x := range w {
			for y := range h {
				copy(line[4*y:4*y+4], pix[4*(y*w+x):])
			}
			pass(h, ry)
			for y := range h {
				copy(pix[4*(y*w+x):][:4], line[4*y:4*y+4])
			}
		}
	}
}
