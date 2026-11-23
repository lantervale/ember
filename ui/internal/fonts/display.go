package fonts

import (
	"math"
	"unicode/utf8"
)

// Display is how a face of the web shows while its file is on its way,
// as its @font-face's font-display has it.
type Display uint8

const (
	// DisplayAuto is block, as Chrome has it.
	DisplayAuto Display = iota
	DisplayBlock
	DisplaySwap
	DisplayFallback
	DisplayOptional
)

// Chrome's periods: a short one of 100 ms and a long one of 3 s.
const (
	shortPeriod = 0.1
	longPeriod  = 3
)

// periods are the ends of the block period, while the face's text is
// invisible, and of the swap period, after which a file that lands is
// not used, in seconds from when text first needed the file.
func (d Display) periods() (float32, float32) {
	inf := float32(math.Inf(1))
	switch d {
	case DisplaySwap:
		return 0, inf
	case DisplayFallback:
		return shortPeriod, longPeriod
	case DisplayOptional:
		return shortPeriod, shortPeriod
	case DisplayAuto, DisplayBlock:
	}
	return longPeriod, inf
}

// period is where a face of the web is in its font-display's periods.
type period uint8

const (
	// idle is a face whose file no text has needed, or that is settled:
	// landed in time, failed or expired.
	idle period = iota
	blocking
	swapping
)

// periodOf is where c is at the set's clock.
func (s *Set) periodOf(c *Face) period {
	if c.web == nil || c.font != nil || c.expired || c.web.state != fileWanted {
		return idle
	}
	block, swap := c.Display.periods()
	switch elapsed := s.elapsed(c.web); {
	case elapsed < block:
		return blocking
	case elapsed < swap:
		return swapping
	}
	return idle
}

// elapsed is how long file has been on its way, 0 before the set has a
// clock.
func (s *Set) elapsed(file *File) float32 {
	if !s.clocked || file.asked < 0 {
		return 0
	}
	return s.now - file.asked
}

// late reports whether c's file, landing now, comes after c's swap
// period: c stays in the next family, as its file failed.
func (s *Set) late(c *Face) bool {
	_, swap := c.Display.periods()
	return s.elapsed(c.web) >= swap
}

// Tick moves the set's clock to now, in seconds: a face whose file is
// on its way past its block period shows its text in the next family,
// and one past its swap period stays there whenever its file lands. It
// reports whether a face's period changed, which draws its text again.
func (s *Set) Tick(now float32) bool {
	s.now, s.clocked = now, true
	changed := false
	for _, c := range s.known {
		if c.web != nil && c.web.state == fileWanted && c.web.asked < 0 {
			c.web.asked = now
		}
	}
	for _, c := range s.known {
		if c.web != nil && c.web.state == fileWanted && c.font == nil && !c.expired && s.late(c) {
			c.expired, changed = true, true
		}
		if p := s.periodOf(c); p != c.period {
			c.period = p
			changed = true
		}
	}
	if changed {
		s.stale()
	}
	return changed
}

// Waiting reports whether a face's period ends while its file is on its
// way: the clock is to be ticked until it does.
func (s *Set) Waiting() bool {
	for _, c := range s.known {
		if s.periodOf(c) != idle {
			return true
		}
	}
	return false
}

// Hidden are the characters the chain draws invisible, in their room:
// those of its faces in their block period.
func (ch *Chain) Hidden(s *Set) []Range {
	var out []Range
	for _, c := range ch.entries {
		if s.periodOf(c) != blocking {
			continue
		}
		if len(c.Ranges) == 0 {
			return []Range{{Lo: 0, Hi: utf8.MaxRune}}
		}
		out = append(out, c.Ranges...)
	}
	return out
}
