//go:build !darwin

package desktop

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lantervale/ember/internal/sysprefs"
)

// repoll is how often the system is asked again where nothing ember
// listens to tells a change: Windows keeps its appearance in the registry,
// and a Linux desktop answers through tools ember runs.
const repoll = 2 * time.Second

// asker reads the system off the frame, as a Linux desktop's tools take
// milliseconds to answer: once at the start before the first frame, then
// every repoll for as long as the program runs.
//
//nolint:gochecknoglobals // one system per process
var asker struct {
	start  sync.Once
	latest atomic.Pointer[sysprefs.System]
	fresh  atomic.Bool
}

// systemChanged reports whether a read of the system says something new
// since it was last asked.
func systemChanged() bool {
	asker.start.Do(func() {
		first := sysprefs.Read()
		asker.latest.Store(&first)
		asker.fresh.Store(true)
		go func() {
			for range time.Tick(repoll) {
				if s := sysprefs.Read(); !s.Equal(*asker.latest.Load()) {
					asker.latest.Store(&s)
					asker.fresh.Store(true)
				}
			}
		}()
	})
	return asker.fresh.Swap(false)
}

func systemPreferences() sysprefs.System { return *asker.latest.Load() }
