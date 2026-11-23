package draw

// SpaceAdvance is the advance of the space character r in a face that
// lacks it, as HarfBuzz falls back to its own space and Chrome draws it
// in that face rather than in another: a fraction of em, the face's size,
// for the em and en spaces and those cut from them, a digit's advance for
// the figure space, a full stop's (else a comma's) for the punctuation
// space, half a space for the narrow no-break space. advance is the
// face's advance of a character, false where it lacks it; the result is
// false for a character that is no such space, or whose measure the face
// lacks too.
func SpaceAdvance(r rune, em float32, advance func(rune) (float32, bool)) (float32, bool) {
	switch r {
	case ' ', ' ':
		return em, true
	case ' ', ' ':
		return em / 2, true
	case ' ':
		return em / 3, true
	case ' ':
		return em / 4, true
	case ' ':
		return em / 5, true
	case ' ':
		return em / 6, true
	case ' ':
		return em / 16, true
	case ' ':
		return em * 4 / 18, true
	case ' ':
		for d := '0'; d <= '9'; d++ {
			if a, ok := advance(d); ok {
				return a, true
			}
		}
	case ' ':
		if a, ok := advance('.'); ok {
			return a, true
		}
		return advance(',')
	case ' ':
		if a, ok := advance(' '); ok {
			return a / 2, true
		}
	}
	return 0, false
}
