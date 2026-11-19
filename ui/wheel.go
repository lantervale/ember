package ui

import "github.com/lantervale/ember/geom"

// WheelScroll has the wheel, a trackpad's two fingers too, scroll the
// Document f times what it says: 2 twice as far, 0.5 half, a negative f
// the other way, 0 not at all. 1, the default, is the wheel as it comes.
// It is what the person at the window asked of their wheel, so the
// factor of the Document in the window holds for those it shows in its
// iframes, whatever their own. Only scrolling is scaled: the window's
// input, and a number field the wheel steps, read the wheel as it is
// (decision 0110).
func WheelScroll(f float32) DocumentOption { return func(d *Document) { d.SetWheelScroll(f) } }

// WheelScroll is how many times what the wheel says the Document scrolls
// by, 1 unless set: its own, also while a Document around it has another.
func (d *Document) WheelScroll() float32 { return d.wheel }

// SetWheelScroll changes the factor from the next turn of the wheel.
func (d *Document) SetWheelScroll(f float32) { d.wheel = f }

// wheeled is the wheel's turn w as the Document scrolls by it: by the
// factor of the Document in the window.
func (d *Document) wheeled(w geom.Vec2) geom.Vec2 {
	root := d
	for root.framed != nil {
		root = root.framed.host
	}
	return w.Scale(root.wheel)
}
