package layout

import (
	"math"
	"slices"
	"sort"

	"github.com/lantervale/ember/ui/internal/style"
)

// fitTolerance absorbs rounding where a box is checked against the room beside floats:
// percentages summing to exactly 100% must not wrap.
const fitTolerance = 0.001

var negInf = float32(math.Inf(-1)) //nolint:gochecknoglobals // infinity is never a constant

// floatSide is the side a float goes to, and the index of that side's values.
type floatSide uint8

const (
	floatLeft floatSide = iota
	floatRight
)

func sideOf(f style.Float) floatSide {
	if f == style.FloatRight {
		return floatRight
	}
	return floatLeft
}

// segment is a band of a block formatting context, the same room across it for its
// whole height: insets is how far the floats (and the block that placed them) reach in
// from each side, hasFloat whether a float is what reaches.
type segment struct {
	top, bottom float32
	insets      [2]float32
	hasFloat    [2]bool
}

// placedFloat is a float's margin box in its context: inset from the side it floats to.
type placedFloat struct {
	width, height float32
	inset, y      float32
}

// floatContext places the floats of one block formatting context, Taffy's FloatContext.
type floatContext struct {
	width    float32
	has      bool
	floats   [2][]placedFloat
	segments []segment
	// lastPlaced is the segments [start, end) the last float of each side went in.
	lastPlaced [2][2]int
	// clearBottoms is the bottom of each side's lowest float, none before its first;
	// ceiling the top of the lowest-starting float, which no later float goes above.
	clearBottoms [2]float32
	ceiling      float32
}

func newFloatContext() *floatContext {
	return &floatContext{clearBottoms: [2]float32{none, none}, ceiling: none}
}

// active reports floats reaching below minY.
func (c *floatContext) active(minY float32) bool {
	return c.has && len(c.segments) > 0 && c.segments[len(c.segments)-1].bottom > minY
}

func (c *floatContext) subdivide(i int, at float32) {
	old := &c.segments[i]
	next := *old
	next.top = at
	old.bottom = at
	c.segments = append(c.segments, segment{})
	copy(c.segments[i+2:], c.segments[i+1:])
	c.segments[i+1] = next
}

// floatFits reports whether a float of that width fits between the insets: from the
// further-in of the float edge and the containing block's on its own side, it must not
// reach a float on the other, nor leave its containing block when a float holds its side.
func floatFits(width float32, side floatSide, bfcWidth float32, floatInsets, cbInsets [2]float32) bool {
	lead, trail := int(side), 1-int(side)
	x := fmax(floatInsets[lead], cbInsets[lead])
	fitsOpposite := floatInsets[trail] == 0 || x+width <= bfcWidth-floatInsets[trail]+fitTolerance
	fitsBlock := floatInsets[lead] == 0 || x+width <= bfcWidth-cbInsets[trail]+fitTolerance
	return fitsOpposite && fitsBlock
}

// clearedThreshold is the bottom of the lowest float clearing goes below, none without one.
func (c *floatContext) clearedThreshold(clearing style.Clear) float32 {
	l, r := c.clearBottoms[0], c.clearBottoms[1]
	switch clearing {
	case style.ClearLeft:
		return l
	case style.ClearRight:
		return r
	case style.ClearBoth:
		if some(l) && some(r) {
			return fmax(l, r)
		}
		return or(l, r)
	case style.ClearNone:
	}
	return none
}

// clearedSegment is the segment the last float clearing goes below ended at, -1 without one.
func (c *floatContext) clearedSegment(clearing style.Clear) int {
	l, r := c.lastPlaced[0][1], c.lastPlaced[1][1]
	switch {
	case clearing == style.ClearLeft && l > 0:
		return l
	case clearing == style.ClearRight && r > 0:
		return r
	case clearing == style.ClearBoth && (l > 0 || r > 0):
		return max(l, r)
	}
	return -1
}

// segmentFrom is the first segment from index from on that ends below minY, len(segments)
// for none.
func (c *floatContext) segmentFrom(from int, minY float32) int {
	for i := min(from, len(c.segments)); i < len(c.segments); i++ {
		if c.segments[i].bottom > minY {
			return i
		}
	}
	return len(c.segments)
}

// place puts a float's margin box in the context, at or below minY, and returns its
// top-left corner.
func (c *floatContext) place(box size, minY float32, cbInsets [2]float32, side floatSide, clearing style.Clear) point {
	c.has = true
	p := c.placeInner(box, minY, cbInsets, side, clearing)
	bottom := p.y + p.height
	c.clearBottoms[side] = fmax(or(c.clearBottoms[side], bottom), bottom)
	c.ceiling = fmax(or(c.ceiling, p.y), p.y)
	c.floats[side] = append(c.floats[side], p)
	if side == floatRight {
		return point{c.width - p.inset - p.width, p.y}
	}
	return point{p.inset, p.y}
}

//nolint:funlen,gocognit // Taffy's place_floated_box_inner, kept in its shape
func (c *floatContext) placeInner(box size, minY float32, cbInsets [2]float32, side floatSide,
	clearing style.Clear) placedFloat {
	slot := int(side)
	minY = fmax(minY, fmax(or(c.ceiling, negInf), or(c.clearedThreshold(clearing), negInf)))

	floatStart := max(c.lastPlaced[0][0], c.lastPlaced[1][0])
	hwm := floatStart
	switch clearing {
	case style.ClearLeft:
		hwm = max(floatStart, c.lastPlaced[0][1]+1)
	case style.ClearRight:
		hwm = max(floatStart, c.lastPlaced[1][1]+1)
	case style.ClearBoth:
		hwm = max(c.lastPlaced[0][1], c.lastPlaced[1][1]) + 1
	case style.ClearNone:
	}

	start := c.segmentFrom(hwm, minY)
	startY := minY
	end := start
	found, foundEnd := false, false
	var inset float32
outer:
	for {
		if start >= len(c.segments) {
			inset = cbInsets[slot]
			break
		}
		first := c.segments[start]
		if !floatFits(box.w, side, c.width, first.insets, cbInsets) {
			start++
			end = max(end, start)
			continue
		}
		startY = fmax(startY, first.top)
		height := float64(first.bottom - startY)
		insets := [2]float32{fmax(0, first.insets[0]), fmax(0, first.insets[1])}
		for {
			if end >= len(c.segments) {
				found, inset = true, fmax(insets[slot], cbInsets[slot])
				break outer
			}
			last := c.segments[end]
			insets = [2]float32{fmax(insets[0], last.insets[0]), fmax(insets[1], last.insets[1])}
			if !floatFits(box.w, side, c.width, insets, cbInsets) {
				start++
				end = max(end, start)
				continue outer
			}
			if end != start {
				height += float64(last.bottom - last.top)
			}
			if height < float64(box.h) {
				end++
				continue
			}
			found, foundEnd, inset = true, true, fmax(insets[slot], cbInsets[slot])
			break outer
		}
	}

	// A float without height takes no segment; one without width still does, its edge
	// one no box that avoids floats goes past.
	if box.h == 0 {
		return placedFloat{width: box.w, y: startY, inset: inset}
	}

	if !found {
		lastEnd := negInf
		if n := len(c.segments); n > 0 {
			lastEnd = c.segments[n-1].bottom
		}
		if startY > lastEnd {
			c.segments = append(c.segments, segment{top: lastEnd, bottom: startY})
		}
		startY = fmax(lastEnd, startY)
		s := segment{top: startY, bottom: startY + box.h, insets: cbInsets}
		s.insets[slot] += box.w
		s.hasFloat[slot] = true
		c.segments = append(c.segments, s)
		c.notePlaced(side, len(c.segments)-1, len(c.segments))
		return placedFloat{width: box.w, height: box.h, y: startY, inset: cbInsets[slot]}
	}

	if startY != c.segments[start].top {
		c.subdivide(start, startY)
		start++
		if foundEnd {
			end++
		}
	}
	endY := startY + box.h
	if !foundEnd {
		if lastEnd := c.segments[len(c.segments)-1].bottom; endY > lastEnd {
			c.segments = append(c.segments, segment{top: lastEnd, bottom: endY, insets: cbInsets})
		}
		end = len(c.segments) - 1
	} else {
		for end > start && endY <= c.segments[end].top {
			end--
		}
		if c.segments[end].top < endY && endY < c.segments[end].bottom {
			c.subdivide(end, endY)
		}
	}
	for i := start; i <= end; i++ {
		c.segments[i].insets[slot] = inset + box.w
		c.segments[i].hasFloat[slot] = true
	}
	c.notePlaced(side, start, end+1)
	return placedFloat{width: box.w, height: box.h, y: startY, inset: inset}
}

func (c *floatContext) notePlaced(side floatSide, start, end int) {
	c.lastPlaced[side][0] = max(c.lastPlaced[side][0], start)
	c.lastPlaced[side][1] = max(c.lastPlaced[side][1], end)
}

// bfcSlot is room beside the floats for a box that must not overlap them: one that
// establishes its own formatting context, a table, a replaced element. x and y are its
// border box's corner, borderWidth the room for the border box, stretchWidth what an auto
// width fills; segment is the band it starts in, -1 below every float.
type bfcSlot struct {
	segment                   int
	x, y                      float32
	borderWidth, stretchWidth float32
}

// findBFCSlot is the first band at or below minY, after the band after, for such a box
// with those left and right margins; the caller checks its width and asks again past a
// band too narrow.
func (c *floatContext) findBFCSlot(minY float32, cbInsets, margins [2]float32, clearing style.Clear,
	after int) bfcSlot {
	marginInsets := [2]float32{cbInsets[0] + margins[0], cbInsets[1] + margins[1]}
	free := c.width - marginInsets[0] - marginInsets[1]
	open := bfcSlot{segment: -1, x: marginInsets[0], y: minY, borderWidth: free, stretchWidth: free}
	if !c.active(minY) {
		return open
	}
	minY = fmax(minY, or(c.clearedThreshold(clearing), negInf))
	i := c.segmentFrom(max(after+1, c.clearedSegment(clearing)+1), minY)
	if i >= len(c.segments) {
		open.y = fmax(c.segments[len(c.segments)-1].bottom, minY)
		return open
	}
	s := c.segments[i]
	var fit, stretch [2]float32
	fit[0] = marginInsets[0]
	if s.hasFloat[0] {
		fit[0] = fmax(s.insets[0], marginInsets[0])
	}
	stretch[0] = fit[0]
	if s.hasFloat[1] {
		fit[1] = fmax(s.insets[1], cbInsets[1])
		stretch[1] = fmax(s.insets[1], marginInsets[1])
	} else {
		fit[1] = fmin(marginInsets[1], cbInsets[1])
		stretch[1] = marginInsets[1]
	}
	return bfcSlot{
		segment:      i,
		x:            fit[0],
		y:            fmax(s.top, minY),
		borderWidth:  c.width - fit[0] - fit[1],
		stretchWidth: c.width - stretch[0] - stretch[1],
	}
}

// lineSlot is the room a line box of that height has at y between the floats and the
// block's own edges: where it starts and how wide it is, and, when a float narrows it,
// where the room next changes below y (none otherwise).
func (c *floatContext) lineSlot(y, height float32, cbInsets [2]float32) (float32, float32, float32) {
	insets := cbInsets
	next := none
	if c.active(y) {
		bottom := y + fmax(height, 0)
		first := none
		for _, s := range c.segments {
			if s.bottom <= y {
				continue
			}
			if s.top > y && s.top >= bottom {
				break
			}
			if !some(first) {
				first = s.bottom
			}
			for side := range 2 {
				if s.hasFloat[side] && s.insets[side] > insets[side] {
					insets[side] = s.insets[side]
					next = first
				}
			}
		}
	}
	return insets[0], c.width - insets[0] - insets[1], next
}

// floatWidths sums what a block's floats add to its intrinsic width, Taffy's
// FloatIntrinsicWidthCalculator: side by side under max-content or a width to fit in,
// the widest alone under min-content. run is the floats since the last block in the flow.
type floatWidths struct {
	avail        space
	sides        [2]float32
	run          [2]float32
	contribution float32
	widest       float32
}

func (f *floatWidths) add(width float32, side floatSide, clearing style.Clear) {
	if f.avail.kind == minContent {
		f.contribution = fmax(f.contribution, width)
	} else {
		if clearing == style.ClearLeft || clearing == style.ClearBoth {
			f.sides[0], f.run[0] = 0, 0
		}
		if clearing == style.ClearRight || clearing == style.ClearBoth {
			f.sides[1], f.run[1] = 0, 0
		}
		f.sides[side] += width
		f.run[side] += width
		f.contribution = fmax(f.contribution, f.sides[0]+f.sides[1])
	}
	f.widest = fmax(f.widest, width)
}

// leading is how wide the floats since the last block in the flow stand side by side.
func (f *floatWidths) leading() float32 { return f.run[0] + f.run[1] }

func (f *floatWidths) result() float32 {
	if f.avail.definite() {
		return fmax(fmin(f.contribution, f.avail.v), f.widest)
	}
	return f.contribution
}

// blockFormattingContext is what the blocks of one formatting context share: its floats.
type blockFormattingContext struct {
	floats *floatContext
}

// blockContext is one block's place in its formatting context, Taffy's BlockContext: nil
// stands for the root of a new one.
type blockContext struct {
	bfc *blockFormattingContext
	// y is the block's border top below the context's, insets its border box's sides in
	// from the context's, contentInsets its content box's.
	y             float32
	insets        [2]float32
	contentInsets [2]float32
	// floatHeight is how far down the floats under the block reach, from its border top.
	floatHeight float32
	root        bool
	// adjoining marks, by side, a float placed while the margins round it still collapse,
	// so its place may move with them: an element that clears it gets clearance whatever
	// its own place. topAdjoining is adjoining when the block's first content settled.
	adjoining       [2]bool
	topAdjoining    [2]bool
	hasTopAdjoining bool
}

func newRootContext() *blockContext {
	return &blockContext{bfc: &blockFormattingContext{floats: newFloatContext()}, floatHeight: negInf, root: true}
}

func (c *blockContext) sub(y float32, insets [2]float32) *blockContext {
	in := [2]float32{c.insets[0] + insets[0], c.insets[1] + insets[1]}
	return &blockContext{bfc: c.bfc, y: c.y + y, insets: in, contentInsets: in, floatHeight: negInf,
		adjoining: c.adjoining}
}

func (c *blockContext) setContentInset(l, r float32) {
	c.contentInsets = [2]float32{c.insets[0] + l, c.insets[1] + r}
}

func (c *blockContext) hasFloats() bool { return c.bfc.floats.has }

func (c *blockContext) active(minY float32) bool { return c.bfc.floats.active(minY + c.y) }

func (c *blockContext) place(box size, minY float32, side floatSide, clearing style.Clear, adjoins bool) point {
	if adjoins {
		c.adjoining[side] = true
	}
	p := c.bfc.floats.place(box, minY+c.y, c.contentInsets, side, clearing)
	p.y -= c.y
	p.x -= c.insets[0]
	c.floatHeight = fmax(c.floatHeight, p.y+box.h)
	return p
}

func (c *blockContext) findBFCSlot(minY float32, margins [2]float32, clearing style.Clear, after int) bfcSlot {
	s := c.bfc.floats.findBFCSlot(minY+c.y, c.contentInsets, margins, clearing, after)
	s.y -= c.y
	s.x -= c.insets[0]
	return s
}

// slotDown is slot narrowed to the room the floats leave all the way down a box h tall
// from its top: the bands below its own that start above the box's bottom narrow it too.
func (c *blockContext) slotDown(slot bfcSlot, h float32, margins [2]float32, clearing style.Clear) bfcSlot {
	if slot.segment < 0 || !some(h) || h <= 0 {
		return slot
	}
	segments := c.bfc.floats.segments
	for i := slot.segment + 1; i < len(segments) && segments[i].top-c.y < slot.y+h; i++ {
		below := c.findBFCSlot(segments[i].top-c.y, margins, clearing, i-1)
		if below.segment < 0 {
			continue
		}
		left := fmax(slot.x, below.x)
		right := fmin(slot.x+slot.borderWidth, below.x+below.borderWidth)
		slot.stretchWidth = fmin(slot.stretchWidth, below.stretchWidth)
		slot.x, slot.borderWidth = left, right-left
	}
	return slot
}

// lineSlot is the float context's, from the block's border box.
func (c *blockContext) lineSlot(y, height float32) (float32, float32, float32) {
	x, w, next := c.bfc.floats.lineSlot(y+c.y, height, c.contentInsets)
	return x - c.insets[0], w, next - c.y
}

func (c *blockContext) clearedThreshold(clearing style.Clear) float32 {
	return c.bfc.floats.clearedThreshold(clearing) - c.y
}

func (c *blockContext) hasAdjoining(clearing style.Clear) bool {
	switch clearing {
	case style.ClearLeft:
		return c.adjoining[0]
	case style.ClearRight:
		return c.adjoining[1]
	case style.ClearBoth:
		return c.adjoining[0] || c.adjoining[1]
	case style.ClearNone:
	}
	return false
}

// commitStrut notes content settled in the block: the floats before it no longer move
// with the margins after it.
func (c *blockContext) commitStrut() {
	if !c.hasTopAdjoining {
		c.topAdjoining, c.hasTopAdjoining = c.adjoining, true
	}
	c.adjoining = [2]bool{}
}

func (c *blockContext) topAdjoiningFloats() [2]bool {
	if c.hasTopAdjoining {
		return c.topAdjoining
	}
	return c.adjoining
}

// clone is a copy of the context that placing floats in leaves c as it was.
func (c *floatContext) clone() *floatContext {
	d := *c
	d.segments = slices.Clone(c.segments)
	d.floats = [2][]placedFloat{slices.Clone(c.floats[0]), slices.Clone(c.floats[1])}
	return &d
}

// lineFloat is a float after content in a block's lines: placed as its lines are broken,
// on the line its place in the text falls on where it fits beside what stands before it
// there, below that line otherwise.
type lineFloat struct {
	node *node
	// run is the run the float stands before in its content's runs.
	run int
	// pending is set while its block's layout places it among the lines, with its margin
	// box and the context the lines start from; at is then its margin box's corner from
	// the lines' block, and first marks one on their first line.
	pending  bool
	box      size
	side     floatSide
	clearing style.Clear
	placed   bool
	at       point
	first    bool
}

// floatsFrom is what the floats of the context were before the lines of a block whose
// pending floats they place: each breaking of its lines starts from it again.
type floatsFrom struct {
	floats      *floatContext
	floatHeight float32
}

// pendingFloats reports floats placed among c's lines.
func (c *inlineContent) pendingFloats() bool {
	return c != nil && len(c.lineFloats) > 0 && c.lineFloats[0].pending
}

// pieceOf is the first piece of run r or after it, len(pieces) past the last.
func (c *inlineContent) pieceOf(r int) int {
	return sort.Search(len(c.pieces), func(i int) bool { return c.pieces[i].run >= r })
}

// widthBefore is how far line l reaches before piece k, the spaces it would end in left
// out: the room what stands before a float there takes.
func (c *inlineContent) widthBefore(l line, k int) float32 {
	at := l.pen(c.unit, 0)
	held := -1
	for j := l.from; j < min(k, l.to); j++ {
		p := c.pieces[j]
		if p.kind == gap || p.kind == tab {
			if held < 0 {
				held = j
			}
			continue
		}
		for ; held >= 0 && held < j; held++ {
			at.step(c.pieces[held], 0)
		}
		held = -1
		at.step(p, 0)
	}
	return at.end()
}
