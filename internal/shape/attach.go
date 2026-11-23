package shape

// anchor is the point of anchor table a, false for none.
func anchor(a data) (x, y int32, ok bool) {
	if a == nil || a.u16(0) < 1 || a.u16(0) > 3 {
		return 0, 0, false
	}
	return int32(int16(a.u16(2))), int32(int16(a.u16(4))), true //nolint:gosec // signed 16-bit fields
}

// cursive joins glyph i, by its entry, to the glyph before it lk does
// not pass over, by that one's exit, as subtable s of a cursive
// attachment has them: the one before ends where i starts, and the
// child, the earlier glyph of a right-to-left lookup, the later
// otherwise, is moved up or down onto its parent's anchor.
func (b *buffer) cursive(lk *lookup, s data, i int) bool {
	cov := s.sub(2)
	k := cov.coverage(b.glyphs[i].ID)
	if s.u16(0) != 1 || k < 0 || k >= int(s.u16(4)) {
		return false
	}
	entryX, entryY, ok := anchor(s.sub(6 + 4*k))
	if !ok {
		return false
	}
	p := b.prev(lk, i)
	if p < 0 {
		return false
	}
	pk := cov.coverage(b.glyphs[p].ID)
	if pk < 0 || pk >= int(s.u16(4)) {
		return false
	}
	exitX, exitY, ok := anchor(s.sub(6 + 4*pk + 2))
	if !ok {
		return false
	}
	gp, gi := &b.glyphs[p], &b.glyphs[i]
	if b.rtl {
		d := exitX + gp.XOffset
		gp.Advance -= d
		gp.XOffset -= d
		gi.Advance = entryX + gi.XOffset
	} else {
		gp.Advance = exitX + gp.XOffset
		d := entryX + gi.XOffset
		gi.Advance -= d
		gi.XOffset -= d
	}
	child, parent := p, i
	dy := entryY - exitY
	if lk.flag&rightToLeft == 0 {
		child, parent, dy = i, p, -dy
	}
	b.unchain(child, parent, maxDepth)
	c := &b.glyphs[child]
	c.attach, c.cursive, c.YOffset = int32(parent-child), true, dy //nolint:gosec // a distance in a run
	if pg := &b.glyphs[parent]; pg.attach == -c.attach {
		pg.attach, pg.YOffset = 0, 0
	}
	return true
}

// unchain turns round the cursive chain glyph i hangs from, up to
// parent, so that what hung from i hangs from it again once i hangs
// from parent.
func (b *buffer) unchain(i, parent, depth int) {
	g := &b.glyphs[i]
	if g.attach == 0 || !g.cursive || depth == 0 {
		return
	}
	chain := g.attach
	g.attach = 0
	j := i + int(chain)
	if j == parent || j < 0 || j >= len(b.glyphs) {
		return
	}
	b.unchain(j, parent, depth-1)
	o := &b.glyphs[j]
	o.YOffset = -b.glyphs[i].YOffset
	o.attach, o.cursive = -chain, true
}

// markToBase places mark i on the glyph before it that is not a mark,
// as subtable s of a mark-to-base attachment, or of a mark-to-ligature
// one on the component of the ligature the mark belongs to.
func (b *buffer) markToBase(lig bool, s data, i int) bool {
	mk := s.sub(2).coverage(b.glyphs[i].ID)
	if s.u16(0) != 1 || mk < 0 {
		return false
	}
	j := i - 1
	for j >= 0 && b.f.isMark(b.glyphs[j].ID) {
		j--
	}
	if j < 0 {
		return false
	}
	bk := s.sub(4).coverage(b.glyphs[j].ID)
	classes := int(s.u16(6))
	marks, bases := s.sub(8), s.sub(10)
	if bk < 0 || bk >= int(bases.u16(0)) {
		return false
	}
	if !lig {
		return b.attachMark(marks, mk, bases, 2+2*bk*classes, i, j)
	}
	attach := bases.sub(2 + 2*bk)
	comps := int(attach.u16(0))
	if comps == 0 {
		return false
	}
	comp := comps - 1
	m, l := b.glyphs[i], b.glyphs[j]
	if l.lig != 0 && l.lig == m.lig && compOf(m) > 0 {
		comp = min(comps, int(compOf(m))) - 1
	}
	return b.attachMark(marks, mk, attach, 2+2*comp*classes, i, j)
}

// markToMark places mark i on the mark before it the lookup does not
// pass over, as subtable s of a mark-to-mark attachment, when both sit
// on one base or one component of a ligature.
func (b *buffer) markToMark(lk *lookup, s data, i int) bool {
	mk := s.sub(2).coverage(b.glyphs[i].ID)
	if s.u16(0) != 1 || mk < 0 {
		return false
	}
	flag := lk.flag &^ (ignoreBase | ignoreLigatures | ignoreMarks)
	j := i - 1
	for j >= 0 && b.f.skips(flag, lk.markSet, b.glyphs[j].ID) {
		j--
	}
	if j < 0 || !b.f.isMark(b.glyphs[j].ID) {
		return false
	}
	m1, m2 := b.glyphs[i], b.glyphs[j]
	c1, c2 := compOf(m1), compOf(m2)
	switch {
	case m1.lig == m2.lig && (m1.lig == 0 || c1 == c2):
	case m1.lig != m2.lig && ((m1.lig > 0 && c1 == 0) || (m2.lig > 0 && c2 == 0)):
	default:
		return false
	}
	k := s.sub(4).coverage(m2.ID)
	marks2 := s.sub(10)
	if k < 0 || k >= int(marks2.u16(0)) {
		return false
	}
	return b.attachMark(s.sub(8), mk, marks2, 2+2*k*int(s.u16(6)), i, j)
}

// attachMark places mark i on glyph j: the mark's anchor, record mk of
// mark array marks, on j's anchor of the mark's class among those at
// row in table anchors; false when j has none for the class.
func (b *buffer) attachMark(marks data, mk int, anchors data, row int, i, j int) bool {
	if mk >= int(marks.u16(0)) {
		return false
	}
	class := int(marks.u16(2 + 4*mk))
	mx, my, ok := anchor(marks.sub(2 + 4*mk + 2))
	if !ok {
		return false
	}
	bx, by, ok := anchor(anchors.sub(row + 2*class))
	if !ok {
		return false
	}
	g := &b.glyphs[i]
	g.XOffset, g.YOffset = bx-mx, by-my
	g.attach, g.cursive = int32(j-i), false //nolint:gosec // a distance in a run
	return true
}

// zeroMarks takes away the advances of the marks, as HarfBuzz does once
// GPOS has placed them; adjust moves each back by its advance, for a
// face whose marks no GPOS places.
func (b *buffer) zeroMarks(adjust bool) {
	for i := range b.glyphs {
		g := &b.glyphs[i]
		if !b.f.isMark(g.ID) {
			continue
		}
		if adjust {
			g.XOffset -= g.Advance
		}
		g.Advance = 0
	}
}

// propagate adds to glyph i's offset that of the glyph it is attached
// to, that one's own first: a cursive child moves up or down with its
// parent, a mark with its base and by the advances between them.
func (b *buffer) propagate(i, depth int) {
	g := &b.glyphs[i]
	if g.attach == 0 || depth == 0 {
		return
	}
	j := i + int(g.attach)
	g.attach = 0
	if j < 0 || j >= len(b.glyphs) {
		return
	}
	b.propagate(j, depth-1)
	p := b.glyphs[j]
	if g.cursive {
		g.YOffset += p.YOffset
		return
	}
	g.XOffset += p.XOffset
	g.YOffset += p.YOffset
	if j >= i {
		return
	}
	if b.rtl {
		for k := j + 1; k <= i; k++ {
			g.XOffset += b.glyphs[k].Advance
		}
		return
	}
	for k := j; k < i; k++ {
		g.XOffset -= b.glyphs[k].Advance
	}
}
