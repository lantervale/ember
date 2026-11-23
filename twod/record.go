package twod

import (
	"runtime/debug"

	"github.com/lantervale/ember/internal/draw"
)

// Mark is a point in what a Canvas draws, to record from.
type Mark struct{ m draw.Mark }

// Recording is what a Canvas drew from a Mark, kept to draw again: a UI
// host's last frame, replayed when nothing in it changed. Games draw their
// entities each frame and never need one.
type Recording struct {
	r draw.Recording
	// parts are the ranges KeepAround kept round its gaps, gapped set.
	parts  []draw.Recording
	gapped bool
}

// Mark is where u's drawing is now.
func (u *Canvas) Mark() Mark { return Mark{u.list.Mark()} }

// Try runs do and gives what it panicked with and the stack there, nil
// when it did not. After a panic the canvas is as it was before do, its
// clips, transform, scope and cursor included, and what do had drawn by
// then stays: for a host that draws something it does not trust to draw
// to the end, and says what became of it.
func (u *Canvas) Try(do func()) (any, []byte) {
	before, list := *u, u.list
	var (
		m        draw.Mark
		clips    int
		panicked any
		stack    []byte
	)
	if list != nil {
		m, clips = list.Mark(), list.Clips()
	}
	func() {
		defer func() {
			if panicked = recover(); panicked == nil {
				return
			}
			stack = debug.Stack()
			*u = before
			if list != nil {
				list.Resume(m, clips)
			}
		}()
		do()
	}()
	return panicked, stack
}

// Keep keeps what u drew since m in r, reusing its room, and reports
// whether Replay can draw it again: not when it was lit, masked, cut by
// a picture or a pass's picture, or when u's transform, modulate, blend or
// clip moved since m.
func (u *Canvas) Keep(m Mark, r *Recording) bool {
	r.gapped = false
	return u.list.Record(m.m, &r.r)
}

// Replay draws r again, at the same place, and reports whether it did:
// not when r holds nothing replayable or u's transform, modulate, blend or
// clip are not what they were when r was recorded.
func (u *Canvas) Replay(r *Recording) bool { return !r.gapped && u.list.Replay(&r.r) }
