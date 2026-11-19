// Package cull says whether paint and hit testing skip the boxes out of
// view or off the pointer (decision 0101), for the tests that hold the
// skipping to the whole walk: EMBER_UI_CULL=off walks every box, and
// EMBER_UI_CULL=check has every paint and hit test done both ways and a
// difference between them reported.
package cull

import (
	"os"
	"sync/atomic"
)

// Mode is how paint walks the boxes.
type Mode int32

const (
	// On skips the boxes whose ink is out of view or misses the pointer.
	On Mode = iota
	// Off walks every box.
	Off
	// Check walks both ways and reports where they differ.
	Check
)

//nolint:gochecknoglobals // one switch for the process, set from the environment or by a test
var (
	mode   atomic.Int32
	differ atomic.Pointer[func(what string)]
)

//nolint:gochecknoinits // the environment's switch, read once
func init() {
	switch os.Getenv("EMBER_UI_CULL") {
	case "off":
		mode.Store(int32(Off))
	case "check":
		mode.Store(int32(Check))
	}
}

// Now is the mode paint walks in.
func Now() Mode { return Mode(mode.Load()) }

// Set has paint walk in mode m and, when differ is not nil, tells it of
// each difference Check finds, which panics otherwise. It returns what
// puts the last mode and report back.
func Set(m Mode, report func(what string)) func() {
	was, told := mode.Swap(int32(m)), differ.Load()
	if report != nil {
		differ.Store(&report)
	} else {
		differ.Store(nil)
	}
	return func() {
		mode.Store(was)
		differ.Store(told)
	}
}

// Differ reports a difference Check found.
func Differ(what string) {
	if report := differ.Load(); report != nil {
		(*report)(what)
		return
	}
	panic("ui: skipping the boxes out of view changed " + what)
}
