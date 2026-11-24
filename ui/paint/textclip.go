package paint

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// clipsText reports whether a layer of st's background, or its colour,
// shows only through text (background-clip: text).
func clipsText(st *style.Computed) bool {
	for _, c := range st.BackgroundClips {
		if c == style.UnderText {
			return true
		}
	}
	return false
}

// textMask is a picture of the glyphs and decorations of e's text and of
// all it holds, opaque where they are, as Chrome clips a background to
// text (decision 0144); nil on a surface that draws no pictures or in a
// scan, where what is clipped to text is left out.
func (p *painter) textMask(e entry) *twod.Picture {
	on, ok := p.s.(Pictures)
	if !ok || p.probing {
		return nil
	}
	held := p.clip
	p.use(nil)
	on.BeginPicture()
	p.pictures++
	p.glyphsOnly = true
	p.glyphs(e.b, e.at)
	p.glyphsOnly = false
	p.pictures--
	mask := on.EndPicture(0)
	p.use(held)
	return mask
}

// glyphs draws the text runs under b, whose border box is at at.
func (p *painter) glyphs(b *box.Box, at geom.Vec2) {
	origin := at.Sub(b.Scroll)
	for _, c := range b.Children {
		ca := origin.Add(c.Pos)
		switch c.Kind {
		case box.TextRun:
			p.text(entry{b: c, at: ca}, 1)
		case box.Element, box.AnonymousBlock:
			if st := shown(c); st == nil || st.Display != style.DisplayNone {
				p.glyphs(c, ca)
			}
		case box.Marker, box.Viewport, box.ControlPart:
		}
	}
}

// throughText draws what paint does inside mask, the element's text, when
// clip is text; else as it is.
func (p *painter) throughText(mask *twod.Picture, clip style.BackgroundBox, paint func()) {
	if clip != style.UnderText {
		paint()
		return
	}
	on, ok := p.s.(Pictures)
	if !ok || mask == nil {
		return
	}
	on.PushClipPicture(mask)
	paint()
	p.s.PopClip()
}
