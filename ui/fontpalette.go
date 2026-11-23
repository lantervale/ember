package ui

import (
	"image/color"
	"math"

	"github.com/lantervale/ember/internal/colourfont"
	"github.com/lantervale/ember/internal/draw"
)

// colourLook is how a face of family draws its colour glyphs in the
// font-palette named palette: the palette an @font-palette-values of
// the family gives that name, the last one of the sheets, or light or
// dark; its SVG glyphs drawn by their outlines, as Chrome draws them.
func (d *Document) colourLook(family, palette string) draw.Look {
	look := draw.Look{NoSVG: true}
	switch palette {
	case "":
	case "light":
		look.Palette = &draw.Palette{Light: true}
	case "dark":
		look.Palette = &draw.Palette{Dark: true}
	default:
		for _, s := range d.authored {
			if s.disabled {
				continue
			}
			for _, v := range s.sheet().FontPaletteValues() {
				if v.Name != palette || !v.For(family) {
					continue
				}
				p := &draw.Palette{Index: v.Base, Light: v.Light, Dark: v.Dark}
				for _, o := range v.Overrides {
					c := color.NRGBA{R: unit(o.Color.R), G: unit(o.Color.G), B: unit(o.Color.B), A: unit(o.Color.A)}
					p.Overrides = append(p.Overrides, colourfont.Override{Index: o.Index, Color: c})
				}
				look.Palette = p
			}
		}
	}
	return look
}

func unit(v float64) uint8 { return uint8(math.Round(min(max(v, 0), 1) * 255)) }
