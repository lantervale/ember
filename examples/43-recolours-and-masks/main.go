// Recolours and masks: variants of a picture made once, at load. Above, the
// crate in its own colours, through a palette made in code, and through
// one loaded from a picture of swatches; in the middle, an orb whose glow
// is its core's colour half clear, as it is, through swaps at any alpha,
// and with one more swap at the glow's alpha only; below, a wall tile as
// it is, cut by a disc mask, and by the same mask inverted.
//
//	go run ./examples/43-recolours-and-masks
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed crate.png stone.png orb.png tiles.png disc.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · recolours and masks"),
		platform.Size(640, 510),
		platform.FixedSize(),
	)
	sprites, err := variants()
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range sprites {
		win.Add(s)
	}

	ember.New().
		AddEntities(win).
		MustRun()
}

// variants makes the nine pictures and stands them in three rows.
func variants() ([]*twod.Sprite, error) {
	// wall is the brick tile in tiles.png, and disc the whole of disc.png.
	wall := geom.Rect{X: 16, W: 16, H: 16}
	disc := geom.Rect{W: 16, H: 16}

	crate, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	moss := twod.NewPalette(
		twod.Swap(color.Hex("#d99a5b"), color.Hex("#a8d47e")),
		twod.Swap(color.Hex("#b5763a"), color.Hex("#6fa64c")),
		twod.Swap(color.Hex("#8a5226"), color.Hex("#447a33")),
		twod.Swap(color.Hex("#5a341c"), color.Hex("#264721")),
	)
	mossy, err := twod.Recolour(crate, moss)
	if err != nil {
		return nil, err
	}
	stone, err := twod.LoadPalette(files, "stone.png")
	if err != nil {
		return nil, err
	}
	stony, err := twod.LoadTexture(files, "crate.png", twod.Pixelated(), twod.Recoloured(stone))
	if err != nil {
		return nil, err
	}

	orb, err := twod.LoadTexture(files, "orb.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	// glow is the alpha orb.png draws its glow at, in its core's colour.
	const glow = 128
	core := color.Hex("#5fcde4")
	swaps := []twod.ColorSwap{
		twod.Swap(color.Hex("#cbf1f7"), color.Hex("#ffd9a0")),
		twod.Swap(core, color.Hex("#f2803a")),
		twod.Swap(color.Hex("#306082"), color.Hex("#8a3b1c")),
	}
	ember, err := twod.Recolour(orb, twod.NewPalette(swaps...))
	if err != nil {
		return nil, err
	}
	haloed, err := twod.Recolour(orb, twod.NewPalette(append(swaps, twod.SwapAt(core, glow, color.Hex("#ffe08a")))...))
	if err != nil {
		return nil, err
	}

	tiles, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	mask, err := twod.LoadTexture(files, "disc.png")
	if err != nil {
		return nil, err
	}
	cut, err := twod.Masked(tiles, wall, mask, disc)
	if err != nil {
		return nil, err
	}
	holed, err := twod.Masked(tiles, wall, mask, disc, twod.Inverted())
	if err != nil {
		return nil, err
	}

	return []*twod.Sprite{
		twod.NewSprite(crate, twod.At(160, 110), twod.Scale(6, 6)),
		twod.NewSprite(mossy, twod.At(320, 110), twod.Scale(6, 6)),
		twod.NewSprite(stony, twod.At(480, 110), twod.Scale(6, 6)),
		twod.NewSprite(orb, twod.At(160, 255), twod.Scale(6, 6)),
		twod.NewSprite(ember, twod.At(320, 255), twod.Scale(6, 6)),
		twod.NewSprite(haloed, twod.At(480, 255), twod.Scale(6, 6)),
		twod.NewSprite(tiles, twod.Region(wall), twod.At(160, 400), twod.Scale(6, 6)),
		twod.NewSprite(cut, twod.At(320, 400), twod.Scale(6, 6)),
		twod.NewSprite(holed, twod.At(480, 400), twod.Scale(6, 6)),
	}, nil
}
