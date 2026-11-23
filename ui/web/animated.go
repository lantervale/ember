package web

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"time"

	"golang.org/x/image/webp"

	"github.com/lantervale/ember/ui/internal/animated"
)

// decodeGIF reads an animated GIF as Chrome plays it: its frames drawn
// on a clear canvas, each disposed of as it says before the next (left,
// cleared, whatever background the file names, or put back to what was
// under it), played once without a loop count, for ever with one of 0,
// and n+1 times with one of n. A GIF of one frame is nil.
func decodeGIF(data []byte) (*animated.Picture, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil || len(g.Image) < 2 {
		return nil, err
	}
	plays := 1
	if g.LoopCount >= 0 {
		plays = g.LoopCount + 1
		if g.LoopCount == 0 {
			plays = 0
		}
	}
	canvas := animated.NewCanvas(g.Config.Width, g.Config.Height, plays)
	for i, frame := range g.Image {
		var disposal byte
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		var saved []byte
		if disposal == gif.DisposalPrevious {
			saved = canvas.Save()
		}
		at := frame.Bounds().Intersect(canvas.Rect)
		draw.Draw(canvas.NRGBA, at, frame, at.Min, draw.Over)
		if !canvas.Show(time.Duration(g.Delay[i]) * 10 * time.Millisecond) {
			break
		}
		switch disposal {
		case gif.DisposalBackground:
			canvas.Clear(at)
		case gif.DisposalPrevious:
			canvas.Restore(saved)
		}
	}
	return canvas.Picture(), nil
}

func decodeWebP(data []byte) (*animated.Picture, error) {
	return animated.DecodeWebP(data, func(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) })
}
