package twod

import "github.com/lantervale/ember/internal/draw"

// Gap is a range of what a Canvas drew, from one Mark to a later one,
// that a recording leaves out: something drawn anew each frame in its
// place, between what was drawn before and after it.
type Gap struct{ From, To Mark }

// KeepAround keeps what u drew since m in r as Keep does, but for gaps,
// in drawing order, and reports whether ReplayAround can draw it again.
func (u *Canvas) KeepAround(m Mark, gaps []Gap, r *Recording) bool {
	r.gapped = true
	if !u.list.SameState(m.m) {
		r.parts = r.parts[:0]
		return false
	}
	n := len(gaps) + 1
	if cap(r.parts) < n {
		r.parts = append(r.parts[:cap(r.parts)], make([]draw.Recording, n-cap(r.parts))...)
	}
	r.parts = r.parts[:n]
	from := m.m
	for i := range n {
		to := u.list.Mark()
		if i < len(gaps) {
			to = gaps[i].From.m
		}
		if !u.list.RecordRange(from, to, &r.parts[i]) {
			r.parts = r.parts[:0]
			return false
		}
		if i < len(gaps) {
			from = gaps[i].To.m
		}
	}
	return true
}

// ReplayAround draws r, kept by KeepAround, again, calling fill to draw
// gap number i in its place, and reports whether it did: not when r
// holds nothing replayable, u's transform, modulate, blend or clip are
// not what they were when r was recorded, or a fill left them otherwise
// than it found them, which takes back what the replay drew.
func (u *Canvas) ReplayAround(r *Recording, fill func(gap int)) bool {
	return r.gapped && u.list.ReplayAround(r.parts, fill)
}
