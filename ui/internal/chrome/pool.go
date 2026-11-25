//go:build chrome

package chrome

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lantervale/ember/ui/internal/shard"
)

// Chrome runs, across every test process on the machine, take one of a
// pool of slots, each a file locked while a Chrome runs: as many Chromes
// run at once as there are slots, each with its own profile and its own
// pipe or page, so none sees another's.

// slotsEnv sets how many Chromes may run at once on the machine.
const slotsEnv = "EMBER_CHROME_SLOTS"

// slots is EMBER_CHROME_SLOTS, else half the CPUs, at least one.
func slots() int {
	if n, err := strconv.Atoi(os.Getenv(slotsEnv)); err == nil && n > 0 {
		return n
	}
	return max(1, runtime.NumCPU()/2)
}

// slotPath is the file slot i locks. Slot 0's is the one lock harnesses
// before the pool took, so those still running wait for it and the pool
// for them.
func slotPath(i int) string {
	if i == 0 {
		return filepath.Join(os.TempDir(), "ember-chrome.lock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("ember-chrome-%d.lock", i))
}

// waitNote tells that a run waits for a slot; waitAfter is how long it
// waits before saying so, and again between notes; pollEvery how often it
// tries the slots meanwhile.
//
//nolint:gochecknoglobals // the pool tests swap them to watch the notes quickly
var (
	waitNote  = func(s string) { fmt.Fprintln(os.Stderr, s) }
	waitAfter = 5 * time.Second
	pollEvery = 50 * time.Millisecond
)

// slotsHeld counts the slots this process holds: a run asking for one
// more while all are taken could be waiting on itself.
//
//nolint:gochecknoglobals // the process's own count of the slots it holds
var slotsHeld atomic.Int32

// errNoSlot is every slot taken.
var errNoSlot = errors.New("chrome: every Chrome slot is taken")

// lockChrome takes a free slot, waiting for one, and writes who holds it
// into its file: full suites in several sessions take every slot for an
// hour, and a run waiting behind them says whom it waits for instead of
// seeming to hang.
func lockChrome() (func(), error) { return takeSlot(true) }

// takeSlot takes a free slot, waiting for one when wait is set, else
// failing with errNoSlot at once.
func takeSlot(wait bool) (func(), error) {
	n := slots()
	f, err := trySlots(n)
	if f == nil && err == nil && wait {
		start := time.Now()
		next := start.Add(waitAfter)
		for f == nil && err == nil {
			time.Sleep(pollEvery)
			if f, err = trySlots(n); f == nil && err == nil && time.Now().After(next) {
				waitNote(fmt.Sprintf("chrome: waiting for one of the %d Chrome slots in %s for %s, held by %s",
					n, os.TempDir(), time.Since(start).Round(time.Second), holders(n)))
				next = time.Now().Add(waitAfter)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errNoSlot
	}
	// The holder's note only tells those waiting whom for: a failed write
	// leaves the slot held all the same.
	_ = f.Truncate(0)
	note := fmt.Sprintf("pid %d (%s) since %s", os.Getpid(), filepath.Base(os.Args[0]), time.Now().Format(time.TimeOnly))
	_, _ = f.WriteAt([]byte(note), 0)
	slotsHeld.Add(1)
	return func() {
		slotsHeld.Add(-1)
		_ = f.Truncate(0)
		// Closing the file releases the lock too, should unlocking fail.
		_ = unlockFile(f)
		f.Close()
	}, nil
}

// trySlots takes the first free one of n slots without waiting, starting
// at one of this process's own so processes spread over them; nil when
// every one is taken.
func trySlots(n int) (*os.File, error) {
	first := os.Getpid() % n
	for k := range n {
		f, err := os.OpenFile(slotPath((first+k)%n), os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return nil, err
		}
		if lockFile(f, false) == nil {
			return f, nil
		}
		f.Close()
	}
	return nil, nil //nolint:nilnil // no slot free is no error: the caller waits or says so
}

// holders is who holds each of n slots, as their files say.
func holders(n int) string {
	var held []string
	for i := range n {
		if b, err := os.ReadFile(slotPath(i)); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			held = append(held, strings.TrimSpace(string(b)))
		}
	}
	return strings.Join(held, "; ")
}

// Shard skips t, a subtest of a test split across shards (shard.SplitEnv),
// unless it falls in the shard this process runs (shard.Env); without a
// shard it runs. It fails a test split that SplitEnv doesn't name, whose
// other subtests would run nowhere.
func Shard(t testing.TB) {
	t.Helper()
	i, n, ok := shard.Parse(os.Getenv(shard.Env))
	if !ok {
		return
	}
	why, err := shard.Skip(t.Name(), i, n, shard.Split(os.Getenv(shard.SplitEnv)))
	if err != nil {
		t.Fatal(err)
	}
	if why != "" {
		t.Skip(why)
	}
}
