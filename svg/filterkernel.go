package svg

import (
	"math"
	"strings"
)

// maxOrder is the widest and highest kernel a convolution runs.
const maxOrder = 64

// convolve reads an feConvolveMatrix; ok is false for one in error,
// which draws nothing.
func convolve(props map[string]string) (Convolve, bool) {
	cv := Convolve{Order: [2]int{3, 3}}
	if v := strings.TrimSpace(props["order"]); v != "" {
		sc := &scanner{s: v}
		o := sc.list(3)
		if !sc.done() || len(o) == 0 || len(o) > 2 {
			return cv, false
		}
		if len(o) == 1 {
			o = append(o, o[0])
		}
		for i, n := range o {
			if n != math.Trunc(n) || n < 1 || n > maxOrder {
				return cv, false
			}
			cv.Order[i] = int(n)
		}
	}
	sc := &scanner{s: props["kernelMatrix"]}
	cv.Kernel = sc.list(maxOrder * maxOrder)
	if !sc.done() || len(cv.Kernel) != cv.Order[0]*cv.Order[1] {
		return cv, false
	}
	sum := 0.0
	for _, k := range cv.Kernel {
		sum += k
	}
	cv.Divisor = sum
	if v, ok := number(props["divisor"]); ok {
		cv.Divisor = v
	} else if sum == 0 {
		cv.Divisor = 1
	}
	if cv.Divisor == 0 {
		return cv, false
	}
	cv.Bias, _ = number(props["bias"])
	for i, k := range [...]string{"targetX", "targetY"} {
		cv.Target[i] = cv.Order[i] / 2
		if v, ok := number(props[k]); ok {
			if v != math.Trunc(v) || v < 0 || int(v) >= cv.Order[i] {
				return cv, false
			}
			cv.Target[i] = int(v)
		}
	}
	switch strings.TrimSpace(props["edgeMode"]) {
	case "wrap":
		cv.Edge = EdgeWrap
	case none:
		cv.Edge = EdgeNone
	}
	cv.PreserveAlpha = strings.TrimSpace(props["preserveAlpha"]) == "true"
	return cv, true
}

// channel is the channel an x or yChannelSelector names, alpha by
// default.
func channel(v string) uint8 {
	if i := strings.Index("RGB", strings.TrimSpace(v)); len(strings.TrimSpace(v)) == 1 && i >= 0 {
		return uint8(i) //nolint:gosec // one of three
	}
	return 3
}

// edgeAt is the pixel of area a kernel reads at x, y, by mode; ok is
// false where it reads clear.
func edgeAt(x, y int, area [4]int, mode EdgeMode) (int, int, bool) {
	x0, y0, x1, y1 := area[0], area[1], area[2], area[3]
	if x >= x0 && x < x1 && y >= y0 && y < y1 {
		return x, y, true
	}
	switch mode {
	case EdgeDuplicate:
		return min(max(x, x0), x1-1), min(max(y, y0), y1-1), true
	case EdgeWrap:
		w, h := x1-x0, y1-y0
		return x0 + ((x-x0)%w+w)%w, y0 + ((y-y0)%h+h)%h, true
	}
	return 0, 0, false
}

// convolve writes in weighed by p's kernel to out, over out's area, the
// kernel turned half a turn as Filter Effects asks, and reading in past
// that area by the kernel's edge mode.
func (fr *filterRun) convolve(out, in *fbuf, p *Primitive) {
	cv := &p.Convolve
	a := out.area.Sub(fr.box.Min)
	if a.Empty() {
		return
	}
	area := [4]int{a.Min.X, a.Min.Y, a.Max.X, a.Max.Y}
	ox, oy := cv.Order[0], cv.Order[1]
	div, bias := float32(1/cv.Divisor), float32(cv.Bias)
	for y := a.Min.Y; y < a.Max.Y; y++ {
		for x := a.Min.X; x < a.Max.X; x++ {
			var sum [4]float32
			for i := range oy {
				for j := range ox {
					sx, sy, ok := edgeAt(x-cv.Target[0]+j, y-cv.Target[1]+i, area, cv.Edge)
					if !ok {
						continue
					}
					k := float32(cv.Kernel[(oy-1-i)*ox+ox-1-j])
					px := in.pix[4*(sy*fr.w+sx):][:4]
					if cv.PreserveAlpha {
						if px[3] > 0 {
							for c := range 3 {
								sum[c] += px[c] / px[3] * k
							}
						}
						continue
					}
					for c := range 4 {
						sum[c] += px[c] * k
					}
				}
			}
			o := out.pix[4*(y*fr.w+x):][:4]
			alpha := min(max(sum[3]*div+bias, 0), 1)
			if cv.PreserveAlpha {
				alpha = in.pix[4*(y*fr.w+x)+3]
			}
			o[3] = alpha
			for c := range 3 {
				v := sum[c]*div + bias
				if cv.PreserveAlpha {
					v *= alpha
				}
				o[c] = min(max(v, 0), alpha)
			}
		}
	}
}

// displace writes in moved by the map's colours to out: each pixel the
// one of in at it plus Scale times the map's two channels, not
// premultiplied, less a half.
func (fr *filterRun) displace(out, in, displacement *fbuf, p *Primitive) {
	s := scales(fr.m)
	sx, sy := float32(p.Scale*s.X), float32(p.Scale*s.Y)
	h := fr.box.Dy()
	a := out.area.Sub(fr.box.Min)
	for y := a.Min.Y; y < a.Max.Y; y++ {
		for x := a.Min.X; x < a.Max.X; x++ {
			d := displacement.pix[4*(y*fr.w+x):][:4]
			var cx, cy float32
			if d[3] > 0 {
				cx, cy = d[p.Channels[0]], d[p.Channels[1]]
				if p.Channels[0] < 3 {
					cx /= d[3]
				}
				if p.Channels[1] < 3 {
					cy /= d[3]
				}
			}
			tx := x + int(math.Floor(float64(sx*(cx-0.5))+0.5))
			ty := y + int(math.Floor(float64(sy*(cy-0.5))+0.5))
			if tx < 0 || ty < 0 || tx >= fr.w || ty >= h {
				continue
			}
			copy(out.pix[4*(y*fr.w+x):][:4], in.pix[4*(ty*fr.w+tx):][:4])
		}
	}
}
