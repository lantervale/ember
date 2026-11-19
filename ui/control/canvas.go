package control

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// Canvas is a canvas element: the game's Draw, which paint calls in the
// element's content box. Version moves on each Invalidate; Kept is paint's
// last drawing of it, replayed while it still holds. Live, it is drawn
// each frame and nothing of it is kept, as what an iframe shows is: the
// Document in it replays itself.
type Canvas struct {
	Draw    func(c *twod.Canvas)
	Version uint64
	Kept    Drawing
	Live    bool
}

func (c *Canvas) Invalidate() { c.Version++ }

// Drawing is a canvas's drawing as recorded: Draw's at Version, in the
// content box At, under Transform at Scale.
type Drawing struct {
	Recording twod.Recording
	Version   uint64
	At        geom.Rect
	Transform geom.Transform2D
	Scale     float32
	Ok        bool
}
