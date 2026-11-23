package shape

// vowelConstraints puts a dotted circle before the last character of
// each sequence of vowels that would draw as another vowel, default
// ignorable characters between them passed over, as HarfBuzz does for
// the Indic scripts and the USE's.
func (b *buffer) vowelConstraints() {
	var out []Glyph
	g := b.glyphs
	next := func(i int) int {
		i++
		for i < len(g) && defaultIgnorable(g[i].Char) {
			i++
		}
		return i
	}
	copied := 0
	for i := 0; i < len(g); i = next(i) {
		last := -1
		for _, seq := range vowelConstraints {
			if seq[0] != g[i].Char {
				continue
			}
			at := i
			for _, r := range seq[1:] {
				if at = next(at); at >= len(g) || g[at].Char != r {
					at = -1
					break
				}
			}
			if at >= 0 && (last < 0 || at < last) {
				last = at
			}
		}
		if last < 0 {
			continue
		}
		out = append(out, g[copied:last]...)
		dc := g[last]
		dc.Char, dc.ID, dc.cc = dottedCircle, b.f.nominal(dottedCircle), 0
		out = append(out, dc)
		copied, i = last, last
	}
	if out == nil {
		return
	}
	out = append(out, g[copied:]...)
	b.glyphs = out
}
