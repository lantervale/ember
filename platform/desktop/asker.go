package desktop

import (
	"sync"
	"time"

	"github.com/lantervale/ember/internal/sysprefs"
)

// settle is how long after a window gains the focus the system is asked
// again: the focus a window takes back often moves between the program's
// windows, and one read answers the lot.
const settle = 250 * time.Millisecond

// asker reads the system where nothing ember listens to tells a change:
// once before the first frame, then off the frame each time one of the
// program's windows gains the focus, when a player comes back from the
// system's settings, a flurry of focus within settle reading once.
type asker struct {
	read  func() sysprefs.System
	after func(time.Duration, func())
	first sync.Once

	mu      sync.Mutex
	latest  sysprefs.System
	fresh   bool
	pending bool
}

func newAsker(read func() sysprefs.System) *asker {
	return &asker{read: read, after: func(d time.Duration, f func()) { time.AfterFunc(d, f) }}
}

// changed reports whether the system said something new since it was last
// asked, gained whether a window of the program has just gained the focus.
func (a *asker) changed(gained bool) bool {
	a.first.Do(func() { a.store(a.read()) })
	a.mu.Lock()
	defer a.mu.Unlock()
	if gained && !a.pending {
		a.pending = true
		a.after(settle, func() {
			s := a.read()
			a.mu.Lock()
			a.pending = false
			a.mu.Unlock()
			a.store(s)
		})
	}
	fresh := a.fresh
	a.fresh = false
	return fresh
}

func (a *asker) store(s sysprefs.System) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !s.Equal(a.latest) {
		a.latest, a.fresh = s, true
	}
}

func (a *asker) system() sysprefs.System {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.latest
}
