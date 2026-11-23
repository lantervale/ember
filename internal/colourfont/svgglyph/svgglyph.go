// Package svgglyph draws a colour font's SVG glyphs through ember's svg
// package. ui/svg registers Read with colourfont, so only a program that
// imports ui/svg links the svg reader (decision 0087).
package svgglyph

import (
	"image"
	"image/color"

	"github.com/lantervale/ember/svg"
)

// Read draws doc into a picture width by height pixels, its currentColor fg.
func Read(doc []byte, width, height int, fg color.NRGBA) (*image.RGBA, error) {
	pic, err := svg.NewPicture(doc, svg.CurrentColor(fg))
	if err != nil {
		return nil, err
	}
	return pic.Rasterise(width, height)
}
