package paint

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
)

// verticalText paints the fragments of a run of vertical lines: each one as the
// horizontal line it is in its frame, turned a quarter clockwise onto the page, its
// glyphs' tops to the right, or anticlockwise for sideways-lr, their tops to the left.
func (p *painter) verticalText(e entry, alpha float32) {
	saved := p.s.Transform()
	defer p.s.SetTransform(saved)
	up := shown(e.b).Mode.Upward()
	for i := range e.b.Lines {
		f := e.b.Lines[i]
		r := f.Rect
		line, turn := f, geom.Transform2D{X: geom.V2(0, 1), Y: geom.V2(-1, 0),
			Origin: geom.V2(e.at.X+r.X+r.W, e.at.Y+r.Y)}
		line.Rect = geom.Rect{W: r.H, H: r.W}
		line.Baseline = r.X + r.W - f.Baseline
		if up {
			turn = geom.Transform2D{X: geom.V2(0, -1), Y: geom.V2(1, 0), Origin: geom.V2(e.at.X+r.X, e.at.Y+r.Y+r.H)}
			line.Baseline = f.Baseline - r.X
		}
		one := *e.b
		one.Lines = []box.Fragment{line}
		p.s.SetTransform(saved.Mul(turn))
		p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
	}
}
