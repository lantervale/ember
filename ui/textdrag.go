package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// A press inside the selection drags its text, as Chrome (decision 0222):
// the press leaves the selection as it is, and once the pointer is
// textDragStart from it along either axis it is a drag of the text,
// which a ghost of the text follows (the box of class ember-drag-image);
// let go over a text field that takes text, the text goes in where the
// pointer is, moved out of the field it came from or copied from the
// page or with the platform's copy key held, and is selected there. Let
// go anywhere else, or back on the selection, it does nothing. A press
// let go where it fell clears the selection, a field's caret then there;
// one that moved less than the drag needs leaves it. On macOS the press
// drags only once held textDragDelay: a move before then that stays short
// of textDragStart selects from the press, as a press outside the
// selection would, and one past it has a caret follow the pointer.
const (
	textDragStart = 3
	textDragDelay = 0.15
)

// textDrag is a press inside the selection, the page's or field's, and
// the drag of its text it becomes.
type textDrag struct {
	// armed is the press held; on its drag; moved the pointer gone from
	// where it fell; caret the caret following the pointer instead.
	armed, on, moved, caret bool
	from                    geom.Vec2
	at                      float32
	// field is the text field whose selection the press fell in, nil for
	// the page's, and place where the press fell in it or in the page.
	field   *Node
	fieldAt int
	place   Position
	text    string
	// grab is the press from the ghost's corner.
	grab  geom.Vec2
	ghost *Node
	// over is the text field the pointer is over that takes the drop, and
	// overAt the place in it.
	over   *Node
	overAt int
}

// pressTextDrag arms a drag of the selected text when a left press at
// falls inside the selection: the focused field's, which takes, or the
// page's. It reports whether it did, the press then leaving the
// selection as it is.
func (d *Document) pressTextDrag(in *input.State, takes *Node, at geom.Vec2) bool {
	g := &d.textDrag
	*g = textDrag{}
	if inputfeed.Of(in).RawDown(int(input.KeyShift)) {
		return false
	}
	t := textOf(d.machine(takes))
	if t != nil && !d.pressFieldDrag(takes, t, at) || t == nil && !d.pressPageDrag(at) {
		return false
	}
	g.armed, g.from, g.at = true, at, d.clock
	return true
}

// pressFieldDrag arms the drag of field n's selection if the press at
// falls in it, n focused: not a password's, nor the second press of a
// run.
func (d *Document) pressFieldDrag(n *Node, t *control.Text, at geom.Vec2) bool {
	g := &d.textDrag
	c := &d.textClicks
	if d.focused() != n || t.Password() || c.on == n && d.clicksOn(c.at, c.where, at) {
		return false
	}
	i, ok := d.positionAt(n, t, at)
	from, to := t.Editor.Selection()
	if !ok || from == to || i < from || i > to {
		return false
	}
	g.field, g.fieldAt, g.text = n, i, t.Editor.Selected()
	g.grab = at.Sub(d.fieldCorner(n, t, from, at))
	*c = textClicks{on: n, at: d.clock, where: at, count: 1}
	return true
}

// pressPageDrag arms the drag of the page's selection if the press at
// falls in it, where a press would begin a selection: not the second
// press of a run.
func (d *Document) pressPageDrag(at geom.Vec2) bool {
	g := &d.textDrag
	s := &d.sel
	if s.anchor == s.focus || !d.startsSelection() || s.clicks > 0 && d.clicksOn(s.clickAt, s.clickWhere, at) {
		return false
	}
	p, ok := d.placeUnder(at)
	start, end := d.ordered(s.anchor, s.focus)
	if !ok || before(p, start) || before(end, p) {
		return false
	}
	g.place, g.text = p, d.Selection().Text
	if r := d.SelectionRects(); len(r) > 0 {
		g.grab = at.Sub(geom.V2(r[0].X, r[0].Y))
	}
	s.clicks, s.clickAt, s.clickWhere = 1, d.clock, at
	return true
}

// clicksOn reports whether a press at goes on a run of clicks whose last
// press was at the clock's when, at where.
func (d *Document) clicksOn(when float32, where, at geom.Vec2) bool {
	return d.clock-when <= doubleClick && at.Sub(where).Len() <= clickSlop
}

// fieldCorner is the top-left of the line of field n where at falls, at
// position from along it: where the selected text's ghost begins.
func (d *Document) fieldCorner(n *Node, t *control.Text, from int, at geom.Vec2) geom.Vec2 {
	b := d.tree.Of[n]
	if b == nil || t.Layout == nil {
		return at
	}
	c := b.ContentRect()
	x := n.Box().X + c.X - b.Scroll.X + t.Layout.X(from)
	y := n.Box().Y + c.Y
	if a, ok := t.Layout.(*area); ok {
		row := a.Row(a.Line(from))
		y += row.Y - b.Scroll.Y
		x += row.X
	}
	return geom.V2(x, y)
}

// textDragPointer moves an armed press inside the selection: it holds
// the pointer until the press becomes a drag of the text or is let go,
// the release then a caret where it fell. It reports whether it took the
// pointer.
func (d *Document) textDragPointer(in *input.State, at geom.Vec2) bool {
	g := &d.textDrag
	if !g.armed || in.Pressed(input.MouseLeft) {
		return false
	}
	if !g.on {
		if !in.Down(input.MouseLeft) {
			g.armed = false
			if !g.moved {
				d.caretAtPress()
			}
			return false
		}
		if g.caret {
			d.caretAt(at)
			return true
		}
		move := at.Sub(g.from)
		g.moved = g.moved || move != geom.Vec2{}
		past := abs(move.X) >= textDragStart || abs(move.Y) >= textDragStart
		early := d.platform == textedit.Mac && d.clock-g.at < textDragDelay
		switch {
		case early && past:
			g.caret = true
			d.caretAt(at)
			return true
		case early && g.moved:
			g.armed = false
			d.selectFromPress()
			return false
		case !past:
			return true
		}
		g.on = true
		d.pressed = nil
		d.showGhost()
	}
	if g.ghost != nil {
		setStyleVar(g.ghost, "left", px(at.X-g.grab.X))
		setStyleVar(g.ghost, "top", px(at.Y-g.grab.Y))
	}
	g.over, g.overAt = nil, 0
	for _, n := range d.hovered {
		if t := textOf(d.machine(n)); t != nil {
			if i, ok := d.positionAt(n, t, at); ok && !t.Editor.ReadOnly && !disabledField(n) {
				g.over, g.overAt = n, i
			}
			break
		}
	}
	if in.Pressed(input.KeyEscape) {
		d.endTextDrag()
		return true
	}
	if !in.Down(input.MouseLeft) {
		d.dropText(d.copyKey(in))
		d.endTextDrag()
	}
	return true
}

// copyKey reports the key held that has a drop copy the text of a field
// rather than move it: Option on macOS, Control elsewhere.
func (d *Document) copyKey(in *input.State) bool {
	f := inputfeed.Of(in)
	if d.platform == textedit.Mac {
		return f.RawDown(int(input.KeyAlt))
	}
	return f.RawDown(int(input.KeyControl))
}

// caretAtPress is the release of the armed press without a drag: a
// field's caret where it fell, the page's selection cleared.
func (d *Document) caretAtPress() {
	g := &d.textDrag
	if g.field == nil {
		d.ClearSelection()
		return
	}
	if t := textOf(d.machine(g.field)); t != nil && d.focused() == g.field {
		t.Editor.Place(g.fieldAt, false)
	}
}

// caretAt puts the caret of the field pressed, or of the page, at the
// place nearest at.
func (d *Document) caretAt(at geom.Vec2) {
	g := &d.textDrag
	if g.field == nil {
		if p, ok := d.positionNear(at, nil); ok {
			d.point(p, p)
		}
		return
	}
	if t := textOf(d.machine(g.field)); t != nil {
		if i, ok := d.positionAt(g.field, t, at); ok {
			t.Editor.Place(i, false)
		}
	}
}

// selectFromPress has the armed press select from where it fell, as a
// press outside the selection does: the pointer's next moves extend it.
func (d *Document) selectFromPress() {
	g := &d.textDrag
	if g.field == nil {
		s := &d.sel
		d.clearOtherSelections()
		s.held, s.unit = true, byCharacter
		d.point(g.place, g.place)
		return
	}
	if t := textOf(d.machine(g.field)); t != nil {
		t.Editor.Place(g.fieldAt, false)
	}
}

// showGhost shows the dragged text in its source's font, as the box of
// class ember-drag-image the pointer carries.
func (d *Document) showGhost() {
	g := &d.textDrag
	src := g.field
	if src == nil {
		src = g.place.Node.Parent()
	}
	b := Div(Class("ember-drag-image"), Attr("aria-hidden", "true"))
	b.SetText(g.text)
	if st := dom.Data(src).Style; st != nil {
		setStyleVar(b, "font", fontShorthand(st.Font))
		setStyleVar(b, "color", rgba(st.Color))
	}
	setStyleVar(b, "left", px(g.from.X-g.grab.X))
	setStyleVar(b, "top", px(g.from.Y-g.grab.Y))
	g.ghost = b
	d.html.Append(b)
}

// endTextDrag ends the drag, its ghost gone.
func (d *Document) endTextDrag() {
	g := &d.textDrag
	if b := g.ghost; b != nil && b.Parent() != nil {
		b.Parent().Remove(b)
	}
	*g = textDrag{}
}

// TextDragImage is the ghost of the text a drag carries, nil while no
// text is dragged.
func (d *Document) TextDragImage() *Node { return d.textDrag.ghost }

// dropText puts the dragged text in the field the pointer is over, at
// the place under it, selected there and the field focused: moved out
// of the field it came from unless copying, copied from the page. Dropped
// back on its own selection it changes nothing.
func (d *Document) dropText(copying bool) {
	g := &d.textDrag
	to := g.over
	t := textOf(d.machine(to))
	if t == nil || g.text == "" {
		return
	}
	at, ok := d.takeOut(g.overAt, copying)
	if !ok {
		return
	}
	n := len(t.Editor.Runes())
	t.Editor.Replace(at, at, g.text)
	t.Editor.Select(at, at+len(t.Editor.Runes())-n)
	d.ClearSelection()
	d.moveFocus(func() bool { return d.focus.Set(to, false) })
	d.fire(to, control.Input)
}

// takeOut deletes the dragged text from the field it came from, unless
// copying or it came from the page, and says where at, the place of the
// drop, stands once it went; false for a drop back on the selection.
func (d *Document) takeOut(at int, copying bool) (int, bool) {
	g := &d.textDrag
	src := textOf(d.machine(g.field))
	if src == nil {
		return at, true
	}
	from, end := src.Editor.Selection()
	if g.over == g.field && at >= from && at <= end {
		return at, false
	}
	if copying {
		return at, true
	}
	if d.platform == textedit.Mac {
		from, end = smartDelete(src.Editor.Runes(), from, end)
	}
	if g.over == g.field && at > end {
		at -= end - from
	}
	src.Editor.Replace(from, end, "")
	d.fire(g.field, control.Input)
	return at, true
}

// smartDelete is the run from from to end that moving it out deletes,
// as Chrome on macOS: with the space before it, else the one after it,
// unless the run begins or
// ends with a space itself, so that no two spaces are left together.
func smartDelete(r []rune, from, end int) (int, int) {
	space := func(i int) bool { return i >= 0 && i < len(r) && (r[i] == ' ' || r[i] == '\t') }
	switch {
	case space(from) || space(end-1):
	case space(from - 1):
		from--
	case space(end):
		end++
	}
	return from, end
}

// fontShorthand is f as the font shorthand writes it.
func fontShorthand(f style.Font) string {
	var b strings.Builder
	if f.Italic {
		b.WriteString("italic ")
	}
	fmt.Fprintf(&b, "%g %s", f.Weight, px(f.Size))
	for i, family := range f.Family {
		sep := ", "
		if i == 0 {
			sep = " "
		}
		if !genericFamily(family) {
			family = strconv.Quote(family)
		}
		b.WriteString(sep + family)
	}
	return b.String()
}

// rgba is c as CSS writes it.
func rgba(c color.Color) string {
	v := c.SRGB()
	return fmt.Sprintf("rgb(%g %g %g / %g)", v[0]*255, v[1]*255, v[2]*255, c.A)
}

func genericFamily(name string) bool {
	switch name {
	case "serif", "sans-serif", "monospace", "cursive", "fantasy", "system-ui", "math", "emoji", "fangsong",
		"ui-serif", "ui-sans-serif", "ui-monospace", "ui-rounded":
		return true
	}
	return false
}
