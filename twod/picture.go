package twod

import (
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/render"
)

// Picture is what a Canvas drew between BeginPicture and EndPicture, kept
// apart to be shown as one image through effects (DrawPicture): a group
// blurred, recoloured or shadowed as a whole. It holds for the frame it
// was drawn in.
type Picture struct {
	pass  *draw.Pass
	outer *draw.List
	rect  geom.Rect
	shown int
}

// Rect is where the picture lies, in the points of the surface it was
// drawn on: what was drawn in it, the margin around, on whole pixels.
func (p *Picture) Rect() geom.Rect { return p.rect }

// kept is a pass a Canvas keeps from frame to frame for its pictures, so
// the GPU keeps its texture, with the list it draws when it is a
// picture's own and the picture.
type kept struct {
	pass *draw.Pass
	list *draw.List
	pic  Picture
}

// keep is the next pass kept for this frame's pictures.
func (u *Canvas) keep() *kept {
	if u.keptUsed == len(u.kept) {
		u.kept = append(u.kept, &kept{pass: draw.NewPass(), list: &draw.List{}})
	}
	k := u.kept[u.keptUsed]
	u.keptUsed++
	return k
}

// surface is the size, in points and pixels, of what the canvas draws on
// now: the window, or the pass of a camera's Resolution or render texture.
func (u *Canvas) surface() (geom.Vec2, geom.Vec2) {
	if u.pass != nil {
		return u.pass.Screen, geom.V2(float32(u.pass.Width), float32(u.pass.Height))
	}
	s := u.Screen()
	screen, fb := geom.V2(s.W, s.H), u.framebuffer()
	if fb.X <= 0 || fb.Y <= 0 {
		fb = screen
	}
	return screen, fb
}

// BeginPicture draws what is drawn next into a picture of its own, over a
// transparent surface as large as the canvas's, until EndPicture. The
// transform and modulate carry into it; the clips do not, as the picture
// is shown under those in place when it is drawn.
func (u *Canvas) BeginPicture() {
	k := u.keep()
	screen, fb := u.surface()
	p := k.pass
	p.List = k.list
	p.Width, p.Height, p.Screen = max(int(fb.X+0.5), 1), max(int(fb.Y+0.5), 1), screen
	p.Effects = p.Effects[:0]
	p.Target.Filter = Linear
	p.Name = "picture"
	p.List.Reset(u.list.Time())
	p.List.SnapVertices = u.list.SnapVertices
	p.List.SetTransform(u.list.Transform())
	p.List.SetModulate(u.list.Modulate())
	p.List.RestoreLight(u.list.Light())
	k.pic = Picture{pass: p, outer: u.list}
	u.passes = append(u.passes, p)
	u.open = append(u.open, &k.pic)
	u.list = p.List
}

// EndPicture ends the picture the last BeginPicture began, cut to what was
// drawn in it and margin points around, where effects may reach, and
// draws where the canvas drew before it.
func (u *Canvas) EndPicture(margin float32) *Picture {
	if len(u.open) == 0 {
		return nil
	}
	pic := u.open[len(u.open)-1]
	u.open = u.open[:len(u.open)-1]
	u.list = pic.outer
	if r, ok := cropRect(pic.pass, margin); ok {
		crop(pic.pass, r)
		pic.rect = r
	}
	return pic
}

// DrawPicture shows pic, through effects in order, over what is drawn,
// in the clip in place. Drawn again, it shows again over itself, through
// other effects: a shadow first, then the picture over it.
func (u *Canvas) DrawPicture(pic *Picture, effects ...*render.Effect) {
	u.showPicture(pic, draw.Alpha, effects)
}

// SumPicture shows pic as DrawPicture does, its colours and alpha added
// to what is drawn: two pictures cut apart join without a seam.
func (u *Canvas) SumPicture(pic *Picture, effects ...*render.Effect) {
	u.showPicture(pic, draw.Plus, effects)
}

// PushClipPicture cuts what is drawn next to pic's alpha where it lies,
// nothing showing outside it, until PopClip; Inverted cuts it where pic
// is opaque, all showing outside it. Only the innermost picture or mask
// cuts, as PushClipMask says. pic must be drawn in the frame.
func (u *Canvas) PushClipPicture(pic *Picture, opts ...MaskOption) {
	var m maskSpec
	for _, o := range opts {
		o.mask(&m)
	}
	xf := u.list.Transform()
	u.list.SetTransform(geom.Identity2D())
	defer u.list.SetTransform(xf)
	if pic == nil {
		pic = &Picture{}
	}
	var mask *render.Texture
	if pic.pass != nil {
		mask = pic.pass.Target
	}
	if m.inverted {
		u.list.PushClipMaskInverted(pic.rect, mask)
		return
	}
	u.list.PushClipMask(pic.rect, mask)
}

func (u *Canvas) showPicture(pic *Picture, blend draw.Blend, effects []*render.Effect) {
	if pic == nil || pic.rect.W <= 0 || pic.rect.H <= 0 {
		return
	}
	p := pic.pass
	if pic.shown > 0 {
		k := u.keep()
		k.pass.List, k.pass.Width, k.pass.Height, k.pass.Screen = p.List, p.Width, p.Height, p.Screen
		k.pass.Target.Filter, k.pass.Name = Linear, p.Name
		i := slices.Index(u.passes, p)
		u.passes = slices.Insert(u.passes, i+1, k.pass)
		p = k.pass
	}
	pic.shown++
	p.Effects = append(p.Effects[:0], effects...)
	xf, mod := u.list.Transform(), u.list.Modulate()
	u.list.Picture(pic.rect, p, blend, nil)
	u.list.SetTransform(xf)
	u.list.SetModulate(mod)
}

// Effect is an effect running shader that the canvas keeps from frame to
// frame: the n-th asked for with a shader in a frame is the n-th of the
// frame before, so its shader stays compiled. Set its uniforms each time.
func (u *Canvas) Effect(shader string) *render.Effect {
	if u.effects == nil {
		u.effects = map[string]*effects{}
	}
	e := u.effects[shader]
	if e == nil {
		e = &effects{}
		u.effects[shader] = e
	}
	if e.used == len(e.all) {
		e.all = append(e.all, render.NewEffect(shader))
	}
	e.used++
	return e.all[e.used-1]
}

// effects are the effects a canvas keeps for one shader, used of them
// this frame.
type effects struct {
	all  []*render.Effect
	used int
}

// resetPictures forgets the pictures and effects of the frame before.
func (u *Canvas) resetPictures() {
	u.keptUsed = 0
	u.open = u.open[:0]
	for _, e := range u.effects {
		e.used = 0
	}
}
