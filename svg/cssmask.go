package svg

import "strings"

// maskLayer is a mask of CSS: its image, how it is read, repeated and
// placed, from the mask shorthand and its longhands.
type maskLayer struct {
	image     string
	luminance bool
	// repeat is along x and y.
	repeat       [2]bool
	origin, clip string
}

// maskLayerOf reads an element's mask properties; odd is set for what
// they ask that is not drawn: more layers than one, a size or a place.
func maskLayerOf(props map[string]string) (layer maskLayer, odd bool) {
	layer = maskLayer{repeat: [2]bool{true, true}, origin: "border-box", clip: "border-box"}
	if v := strings.TrimSpace(props["mask"]); v != "" {
		layers := splitCommas(v)
		odd = len(layers) > 1
		boxes := 0
		repeats := 0
		for _, w := range shapeArgs(layers[0]) {
			lw := strings.ToLower(w)
			switch {
			case strings.Contains(w, "(") || lw == none:
				layer.image = w
			case lw == "alpha" || lw == "luminance" || lw == "match-source":
				layer.luminance = lw == "luminance"
			case setRepeat(&layer, lw, repeats):
				repeats++
			case lw == "no-clip":
				layer.clip = lw
			case isBox(lw):
				if boxes == 0 {
					layer.origin, layer.clip = lw, lw
				} else {
					layer.clip = lw
				}
				boxes++
			case lw == "add" || lw == "subtract" || lw == "intersect" || lw == "exclude":
			default:
				odd = true
			}
		}
	}
	if v := strings.TrimSpace(props["mask-image"]); v != "" {
		layers := splitCommas(v)
		odd = odd || len(layers) > 1
		layer.image = layers[0]
	}
	if v := strings.ToLower(strings.TrimSpace(props["mask-mode"])); v != "" {
		layer.luminance = v == "luminance"
	}
	if v := strings.TrimSpace(props["mask-repeat"]); v != "" {
		layer.repeat = [2]bool{true, true}
		for i, w := range strings.Fields(strings.ToLower(v)) {
			setRepeat(&layer, w, i)
		}
	}
	if v := strings.ToLower(strings.TrimSpace(props["mask-origin"])); isBox(v) {
		layer.origin = v
	}
	if v := strings.ToLower(strings.TrimSpace(props["mask-clip"])); isBox(v) || v == "no-clip" {
		layer.clip = v
	}
	for _, k := range [...]string{"mask-size", "mask-position"} {
		if v := strings.TrimSpace(props[k]); v != "" && v != valAuto && v != "0% 0%" && v != "0 0" {
			odd = true
		}
	}
	return layer, odd
}

// isImage reports a CSS image function, which a mask draws or is noted
// for; any other value is no mask, as Chrome reads it.
func isImage(v string) bool {
	open := strings.IndexByte(v, '(')
	if open <= 0 {
		return false
	}
	switch name := strings.ToLower(strings.TrimSpace(v[:open])); {
	case strings.HasSuffix(name, "gradient"), name == "image", name == "image-set", name == "cross-fade",
		name == "element", name == "paint", strings.HasPrefix(name, "-webkit-"):
		return true
	}
	return false
}

func isBox(w string) bool {
	switch w {
	case "border-box", "padding-box", "content-box", "margin-box", "fill-box", "stroke-box", "view-box":
		return true
	}
	return false
}

// setRepeat reads the nth word of a mask's repeat into layer; false for
// a word that is none.
func setRepeat(layer *maskLayer, w string, nth int) bool {
	switch w {
	case "repeat-x":
		layer.repeat = [2]bool{true, false}
	case "repeat-y":
		layer.repeat = [2]bool{false, true}
	case "repeat", "space", "round", "no-repeat":
		on := w != "no-repeat"
		if nth == 0 {
			layer.repeat = [2]bool{on, on}
		} else {
			layer.repeat[1] = on
		}
	default:
		return false
	}
	return true
}

// cssMask is the mask of a CSS gradient over an element whose user space
// m takes to the picture's, drawing items within bbox there: the
// gradient sized to the origin box, as Chrome places it on an SVG
// element, the border box its fill box, repeated, and cut to the clip
// box, the border box its stroke box. Nil for an image this does not
// draw.
func (b *builder) cssMask(layer maskLayer, st *style, m Matrix, items []Item, bbox Box) *Mask {
	originName := layer.origin
	if originName == "border-box" {
		originName = "fill-box"
	}
	origin, _ := b.referenceBox(originName, items, bbox)
	g, ok := cssGradient(layer.image, origin, st)
	if !ok {
		return nil
	}
	tile := &Shape{
		Path: rectPath(origin.X, origin.Y, origin.W, origin.H, 0, 0), Transform: Identity(),
		Fill: Paint{Kind: GradientPaint, Gradient: g, Opacity: 1},
	}
	mk := &Mask{Alpha: !layer.luminance}
	stroke, _ := b.referenceBox("stroke-box", items, bbox)
	area := union(stroke, Box{0, 0, b.vw, b.vh})
	if layer.clip != "no-clip" {
		area, _ = b.referenceBox(layer.clip, items, bbox)
		mk.Region = rectClip(area, m)
	}
	if !layer.repeat[0] && !layer.repeat[1] {
		tile.Transform = m
		mk.Items = []Item{{Shape: tile}}
		return mk
	}
	band := area
	if !layer.repeat[0] {
		band.X, band.W = origin.X, origin.W
	}
	if !layer.repeat[1] {
		band.Y, band.H = origin.Y, origin.H
	}
	pattern := &Pattern{Items: []Item{{Shape: tile}}, Tile: origin, Transform: Identity()}
	mk.Items = []Item{{Shape: &Shape{
		Path: rectPath(band.X, band.Y, band.W, band.H, 0, 0), Transform: m,
		Fill: Paint{Kind: PatternPaint, Pattern: pattern, Opacity: 1},
	}}}
	return mk
}
