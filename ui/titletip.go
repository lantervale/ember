package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

// The title attribute shows as Chrome's tooltip does (decision 0142): once
// the pointer rests half a second over the element, or one inside it whose
// nearest title is its own, a box of class ember-title-tip shows its text,
// lines kept, 10 points right of the pointer and 15 below it, above the
// pointer where it would cross the window's foot and held inside its
// right edge. It hides when the pointer leaves for another title, a button
// or a key is pressed, or after 10 seconds, and shows again only over
// another title. The box is the html's last child while it shows, fixed
// and out of the pointer's way; the user-agent sheet gives it Chrome's
// look and kit's sheet kit's tooltip's.

const (
	titleDelay   = 0.5
	titleTimeout = 10
	titleOffsetX = 10
	titleOffsetY = 15
	// titleMargin is how far a tip held inside the window keeps from its
	// edges.
	titleMargin = 4
)

// titleTip is the title tooltip's state: the element whose title the
// pointer is over and since when, the one a press or a key dismissed, the
// box when it shows and where the pointer was then.
type titleTip struct {
	over      *Node
	since     float32
	dismissed *Node
	box       *Node
	shownAt   float32
	pointer   geom.Vec2
	placed    bool
}

// titled is the element whose title shows over the chain, innermost
// first, and its text: the nearest title, none when it is empty.
func titled(c []*Node) (*Node, string) {
	for _, n := range c {
		if v, ok := n.Attr("title"); ok {
			if v == "" {
				return nil, ""
			}
			return n, v
		}
	}
	return nil, ""
}

// titleTips shows, keeps or hides the title tooltip for this frame.
func (d *Document) titleTips(in *input.State) {
	tt := &d.title
	over, text := titled(d.hovered)
	if over != tt.over {
		tt.over, tt.since, tt.dismissed = over, d.clock, nil
		d.hideTitle()
	}
	f := inputfeed.Of(in)
	if over != nil && (in.Pressed(input.MouseLeft) || in.Pressed(input.MouseRight) || in.Pressed(input.MouseMiddle) ||
		f.Changed() && anyKeyPressed(f)) {
		tt.dismissed = over
		d.hideTitle()
	}
	switch {
	case over == nil || tt.dismissed == over:
	case tt.box != nil && d.clock-tt.shownAt >= titleTimeout:
		tt.dismissed = over
		d.hideTitle()
	case tt.box != nil:
		tt.box.SetText(text)
	case d.clock-tt.since >= titleDelay:
		d.showTitle(text)
	}
}

func anyKeyPressed(f inputfeed.Feed) bool {
	for k := input.Key(0); k <= input.KeyMenu; k++ {
		if f.RawPressed(int(k)) {
			return true
		}
	}
	return false
}

// showTitle shows text in the title box below the pointer, placed once it
// is laid out (placeTitle).
func (d *Document) showTitle(text string) {
	tt := &d.title
	b := Div(Class("ember-title-tip"), Attr("aria-hidden", "true"))
	b.SetText(text)
	tt.box, tt.shownAt, tt.pointer, tt.placed = b, d.clock, d.pointerAt, false
	setStyleVar(b, "left", px(d.pointerAt.X+titleOffsetX))
	setStyleVar(b, "top", px(d.pointerAt.Y+titleOffsetY))
	d.html.Append(b)
}

func (d *Document) hideTitle() {
	if b := d.title.box; b != nil {
		d.title.box = nil
		if b.Parent() != nil {
			b.Parent().Remove(b)
		}
	}
}

// TitleTip is the title tooltip showing, nil for none: its text is the
// title it shows.
func (d *Document) TitleTip() *Node { return d.title.box }

// placeTitle moves the title box shown since the last layout inside the
// viewport, above the pointer when it crosses the foot, and reports
// whether it moved.
func (d *Document) placeTitle() bool {
	tt := &d.title
	if tt.box == nil || tt.placed {
		return false
	}
	tt.placed = true
	b := d.tree.Of[tt.box]
	if b == nil {
		return false
	}
	r := b.Rect()
	x, y := r.X, r.Y
	if x+r.W > d.laidAt.X-titleMargin {
		x = max(d.laidAt.X-titleMargin-r.W, titleMargin)
	}
	if y+r.H > d.laidAt.Y-titleMargin {
		y = max(tt.pointer.Y-r.H, titleMargin)
	}
	if x == r.X && y == r.Y {
		return false
	}
	setStyleVar(tt.box, "left", px(x))
	setStyleVar(tt.box, "top", px(y))
	return true
}
