//go:build chrome

package chrome

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/shard"
)

// pool gives t a pool of n slots of its own: os.TempDir reads TMP and TEMP
// on Windows, TMPDIR elsewhere, so the slots of other packages' Chrome
// runs stay apart.
func pool(t *testing.T, n int) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv(slotsEnv, strconv.Itoa(n))
}

func TestSlotsAreHalfTheCPUsUnlessSet(t *testing.T) {
	t.Setenv(slotsEnv, "")
	assert.Equal(t, max(1, runtime.NumCPU()/2), slots())
	t.Setenv(slotsEnv, "0")
	assert.Equal(t, max(1, runtime.NumCPU()/2), slots())
	t.Setenv(slotsEnv, "six")
	assert.Equal(t, max(1, runtime.NumCPU()/2), slots())
	t.Setenv(slotsEnv, "6")
	assert.Equal(t, 6, slots())
}

func TestTheFirstSlotIsTheLockOlderHarnessesTake(t *testing.T) {
	pool(t, 1)
	assert.Equal(t, "ember-chrome.lock", filepath.Base(slotPath(0)))
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "ember-chrome.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, lockFile(f, false))
	_, err = takeSlot(false)
	require.ErrorIs(t, err, errNoSlot, "an older harness's Chrome holds the one slot")
	require.NoError(t, unlockFile(f))
	unlock, err := takeSlot(false)
	require.NoError(t, err)
	unlock()
}

func TestThePoolHoldsAsManyChromesAsItHasSlots(t *testing.T) {
	pool(t, 3)
	var unlocks []func()
	for range 3 {
		unlock, err := takeSlot(false)
		require.NoError(t, err)
		unlocks = append(unlocks, unlock)
	}
	assert.EqualValues(t, 3, slotsHeld.Load())
	_, err := takeSlot(false)
	require.ErrorIs(t, err, errNoSlot)

	got := make(chan func(), 1)
	go func() {
		u, lockErr := lockChrome()
		assert.NoError(t, lockErr)
		got <- u
	}()
	time.Sleep(200 * time.Millisecond)
	select {
	case <-got:
		t.Fatal("a fourth run took a slot while all three were held")
	default:
	}
	unlocks[1]()
	select {
	case u := <-got:
		u()
	case <-time.After(10 * time.Second):
		t.Fatal("a slot let go was never taken")
	}
	unlocks[0]()
	unlocks[2]()
	assert.EqualValues(t, 0, slotsHeld.Load())
}

func TestARunWaitingForASlotSaysWhoHoldsThem(t *testing.T) {
	pool(t, 2)
	a, err := lockChrome()
	require.NoError(t, err)
	b, err := lockChrome()
	require.NoError(t, err)
	var held []string
	for i := range 2 {
		note, readErr := os.ReadFile(slotPath(i))
		require.NoError(t, readErr)
		assert.Contains(t, string(note), fmt.Sprintf("pid %d ", os.Getpid()))
		held = append(held, string(note))
	}

	var mu sync.Mutex
	var notes []string
	old, oldAfter := waitNote, waitAfter
	waitNote = func(s string) { mu.Lock(); notes = append(notes, s); mu.Unlock() }
	waitAfter = 10 * time.Millisecond
	defer func() { waitNote, waitAfter = old, oldAfter }()

	got := make(chan func(), 1)
	go func() {
		u, lockErr := lockChrome()
		assert.NoError(t, lockErr)
		got <- u
	}()
	time.Sleep(200 * time.Millisecond)
	a()
	(<-got)()
	b()

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, notes, "the waiting run said so")
	assert.Contains(t, notes[0], "waiting for one of the 2 Chrome slots")
	for _, h := range held {
		assert.Contains(t, notes[0], h)
	}
	for i := range 2 {
		after, _ := os.ReadFile(slotPath(i))
		assert.Empty(t, string(after), "a slot let go names no holder")
	}
}

// Runs at once, as many as the slots, each run Chrome with a profile of
// its own: the fake Chrome logs when it starts and ends and which profile
// it was given.
func TestRunsShareThePoolEachWithItsOwnProfile(t *testing.T) {
	const n, runs = 2, 6
	pool(t, n)
	fakeChrome(t, false)
	log := filepath.Join(t.TempDir(), "runs.log")
	t.Setenv("EMBER_FAKE_CHROME_LOG", log)
	var wg sync.WaitGroup
	errs := make(chan error, runs)
	for range runs {
		wg.Go(func() {
			unlock, err := lockChrome()
			if err == nil {
				_, err = runRaw([]byte("<html></html>"), geom.V2(100, 100), "--dump-dom", css.Media{})
				unlock()
			}
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	f, err := os.Open(log)
	require.NoError(t, err)
	defer f.Close()
	at, most := 0, 0
	var profiles []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		what, profile, _ := strings.Cut(sc.Text(), " ")
		switch what {
		case "start":
			at++
			most = max(most, at)
			assert.NotContains(t, profiles, profile, "a profile used twice")
			profiles = append(profiles, profile)
		case "end":
			at--
		}
	}
	require.NoError(t, sc.Err())
	assert.Len(t, profiles, runs)
	assert.Equal(t, n, most, "as many Chromes at once as slots, no more")
}

// logRun is the fake Chrome's log of its run: its start, a moment of work
// so runs overlap, its end, each with its profile.
func logRun(log string, args []string) {
	i := slices.IndexFunc(args, func(a string) bool { return strings.HasPrefix(a, "--user-data-dir=") })
	if i < 0 {
		return
	}
	profile := strings.TrimPrefix(args[i], "--user-data-dir=")
	write := func(what string) {
		f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return
		}
		defer f.Close()
		_, _ = fmt.Fprintf(f, "%s %s\n", what, profile)
	}
	lock, err := os.OpenFile(log+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return
	}
	defer lock.Close()
	_ = lockFile(lock, true)
	write("start")
	_ = unlockFile(lock)
	time.Sleep(300 * time.Millisecond)
	_ = lockFile(lock, true)
	write("end")
	_ = unlockFile(lock)
}

func TestTheVersionUnderAHeldSlotIsReadInAFreeOne(t *testing.T) {
	forget(t)
	pool(t, 2)
	fakeChrome(t, false)
	unlock, err := lockChrome()
	require.NoError(t, err)
	defer unlock()
	v, err := versionWithin(t, 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, 158, v, "the fake Chrome's")
}

func TestTheVersionUnderEverySlotHeldFailsAtOnce(t *testing.T) {
	forget(t)
	pool(t, 2)
	fakeChrome(t, false)
	a, err := lockChrome()
	require.NoError(t, err)
	defer a()
	b, err := lockChrome()
	require.NoError(t, err)
	defer b()
	_, err = versionWithin(t, 5*time.Second)
	require.Error(t, err, "a version read while this process holds every slot")
}

// versionWithin is Version, failing t should it take longer than limit.
func versionWithin(t *testing.T, limit time.Duration) (int, error) {
	t.Helper()
	type answer struct {
		v   int
		err error
	}
	asked := make(chan answer, 1)
	go func() {
		v, err := Version()
		asked <- answer{v, err}
	}()
	select {
	case a := <-asked:
		return a.v, a.err
	case <-time.After(limit):
		t.Fatal("the version waited for a slot its own process holds")
		return 0, errors.New("unreachable")
	}
}

// Two sessions of the real Chrome at once, on the same origin, each in a
// slot and a profile of its own: what one page stores the other never
// sees, and each drives its own page.
func TestTwoSessionsAtOnceKeepApart(t *testing.T) {
	pool(t, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "<!DOCTYPE html><title>%s</title><p>x</p>", r.URL.Query().Get("name"))
	}))
	defer srv.Close()
	a := Open(t, srv.URL+"/?name=a", geom.V2(200, 100), 1)
	b := Open(t, srv.URL+"/?name=b", geom.V2(300, 100), 1)
	assert.EqualValues(t, 2, slotsHeld.Load())

	a.Eval(`localStorage.setItem("who", "a"), document.cookie = "who=a"`, nil)
	var stored, cookie, title string
	var width float64
	b.Eval(`String(localStorage.getItem("who"))`, &stored)
	b.Eval(`document.cookie`, &cookie)
	b.Eval(`document.title`, &title)
	b.Eval(`innerWidth`, &width)
	assert.Equal(t, "null", stored, "b's profile holds none of a's storage")
	assert.Empty(t, cookie, "nor its cookies")
	assert.Equal(t, "b", title)
	assert.InDelta(t, 300, width, 0.5)

	a.Eval(`localStorage.getItem("who")`, &stored)
	a.Eval(`document.title`, &title)
	assert.Equal(t, "a", stored)
	assert.Equal(t, "a", title)
}

func TestASplitTestRunsItsShardsSubtestsAlone(t *testing.T) {
	ran := map[string]bool{}
	run := func(env string) {
		t.Setenv(shard.Env, env)
		t.Setenv(shard.SplitEnv, t.Name())
		for k := range 20 {
			t.Run(fmt.Sprint("page-", k), func(t *testing.T) {
				Shard(t)
				ran[env+" "+t.Name()] = true
			})
		}
	}
	run("")
	run("0/2")
	run("1/2")
	alone, first, second := 0, 0, 0
	for name := range ran {
		switch {
		case strings.HasPrefix(name, " "):
			alone++
		case strings.HasPrefix(name, "0/2 "):
			first++
		default:
			second++
		}
	}
	assert.Equal(t, 20, alone, "without a shard every subtest runs")
	assert.Equal(t, 20, first+second, "with two, each runs in one")
	assert.Positive(t, first)
	assert.Positive(t, second)
}
