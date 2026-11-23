package desktop //nolint:testpackage // tests the unexported reader of the system's preferences

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/internal/sysprefs"
	"github.com/lantervale/ember/platform"
)

// scriptedAsker is an asker over a system the test sets, whose debounced
// reads wait until the test fires them.
func scriptedAsker() (*asker, *sysprefs.System, *int, *[]func()) {
	sys, reads := &sysprefs.System{Appearance: platform.AppearanceLight}, new(int)
	var timers []func()
	a := newAsker(func() sysprefs.System {
		*reads++
		return *sys
	})
	a.after = func(d time.Duration, f func()) {
		if d != settle {
			panic("not the settle")
		}
		timers = append(timers, f)
	}
	return a, sys, reads, &timers
}

func TestTheSystemIsReadBeforeTheFirstFrameThenOnlyOnFocus(t *testing.T) {
	a, sys, reads, timers := scriptedAsker()
	require.True(t, a.changed(false), "read before the first frame")
	assert.Equal(t, platform.AppearanceLight, a.system().Appearance)
	assert.Equal(t, 1, *reads)

	sys.Appearance = platform.AppearanceDark
	for range 10 {
		assert.False(t, a.changed(false), "no window gained the focus: nothing asked")
	}
	assert.Equal(t, 1, *reads)
	assert.Empty(t, *timers)

	assert.False(t, a.changed(true), "the read waits for the focus to settle")
	require.Len(t, *timers, 1)
	(*timers)[0]()
	assert.Equal(t, 2, *reads)
	assert.True(t, a.changed(false), "what it read is told the next frame")
	assert.Equal(t, platform.AppearanceDark, a.system().Appearance)
	assert.False(t, a.changed(false), "and once")
}

func TestAFlurryOfFocusReadsTheSystemOnce(t *testing.T) {
	a, sys, reads, timers := scriptedAsker()
	a.changed(false)
	sys.Motion = platform.MotionReduced
	for range 5 {
		a.changed(true)
	}
	require.Len(t, *timers, 1, "one read for the lot")
	(*timers)[0]()
	assert.Equal(t, 2, *reads)
	assert.True(t, a.changed(false))

	a.changed(true)
	require.Len(t, *timers, 2, "a focus once the read is done asks again")
	(*timers)[1]()
	assert.Equal(t, 3, *reads)
	assert.False(t, a.changed(false), "the same answers are no change")
}

func TestTheDriverAsksAgainWhenAWindowGainsTheFocus(t *testing.T) {
	one, two := platform.NewWindow(), platform.NewWindow()
	d := &driver{natives: map[*platform.Window]*native{one: {window: one}, two: {window: two}}}
	var asked []bool
	frame := func(first, second bool) {
		one.SetFocused(first)
		two.SetFocused(second)
		asked = append(asked, d.focusGained())
	}
	frame(false, false)
	frame(true, false)
	frame(true, false)
	frame(false, true)
	frame(false, false)
	frame(false, true)
	assert.Equal(t, []bool{false, true, false, false, false, true}, asked,
		"the program coming back to the front, not the focus moving between its windows")
}
