package ui

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/css"
)

// Preferences has the Document's sheets answer prefers-color-scheme and
// prefers-reduced-motion as the system does, through the preferences the
// program added to the World (decision 0121): a change restyles where a
// rule asks, as the next frame begins. Without it, or while the system
// has not said, the Document answers light and no preference, the same on
// any machine. ColorScheme and ReducedMotion win over it.
func Preferences(prefs *platform.Preferences) DocumentOption {
	return func(d *Document) { d.prefs = prefs }
}

// Preferences is the system's preferences the Document follows, nil for
// none.
func (d *Document) Preferences() *platform.Preferences { return d.prefs }

// SetPreferences changes the preferences the Document follows from the
// next layout; nil follows none.
func (d *Document) SetPreferences(prefs *platform.Preferences) { d.prefs = prefs }

// ColorScheme has the Document answer prefers-color-scheme with a, the
// program's own over the system's, for a game with a theme switch of its
// own and for tests. AppearanceUnknown leaves the answer to the
// Preferences.
func ColorScheme(a platform.Appearance) DocumentOption { return func(d *Document) { d.scheme = a } }

// ColorScheme is the scheme the Document answers prefers-color-scheme
// with: the program's, else the Preferences', else its host's in an
// iframe, else light. An element whose color-scheme names both light and
// dark is drawn in it (decision 0126).
func (d *Document) ColorScheme() platform.Appearance {
	switch {
	case d.scheme != platform.AppearanceUnknown:
		return d.scheme
	case d.prefs != nil && d.prefs.Appearance() != platform.AppearanceUnknown:
		return d.prefs.Appearance()
	case d.framed != nil:
		return d.framed.host.ColorScheme()
	}
	return platform.AppearanceLight
}

// SetColorScheme changes the program's answer from the next layout,
// which styles the tree again where a rule asks; AppearanceUnknown
// follows the Preferences again.
func (d *Document) SetColorScheme(a platform.Appearance) { d.scheme = a }

// ReducedMotion has the Document's sheets answer
// prefers-reduced-motion: reduce, where a page that asks stops or
// shortens its animations (decision 0114), whatever the Preferences say.
func ReducedMotion() DocumentOption { return func(d *Document) { d.motion = platform.MotionReduced } }

// ReducedMotion reports whether the Document answers
// prefers-reduced-motion: reduce: as the program said, else as the
// Preferences say, else as its host in an iframe, else not.
func (d *Document) ReducedMotion() bool {
	switch {
	case d.motion != platform.MotionUnknown:
		return d.motion == platform.MotionReduced
	case d.prefs != nil && d.prefs.Motion() != platform.MotionUnknown:
		return d.prefs.Motion() == platform.MotionReduced
	case d.framed != nil:
		return d.framed.host.ReducedMotion()
	}
	return false
}

// SetReducedMotion changes the answer from the next layout, which styles
// the tree again where a rule asks, and from then on wins over the
// Preferences.
func (d *Document) SetReducedMotion(on bool) {
	d.motion = platform.MotionFull
	if on {
		d.motion = platform.MotionReduced
	}
}

// medium is what the Document answers the media features that aren't
// its viewport's with. The pointer is its window's, the outermost
// Document's, and so is whether that window is inactive.
func (d *Document) medium() css.Media {
	top := d
	for top.framed != nil {
		top = top.framed.host
	}
	return css.Media{Dark: d.ColorScheme() == platform.AppearanceDark, ReducedMotion: d.ReducedMotion(), Touch: top.touch,
		Inactive: top.inactive}
}

// notePointer records whether the pointer last used is a finger on the
// screen, a coarse one that doesn't hover, until the mouse is used: a
// trackpad's fingers move the mouse's pointer, and keys and pads point
// with nothing.
func (d *Document) notePointer(in *input.State) {
	for _, t := range in.Touches() {
		if t.Source == input.Touchscreen && t.Phase == input.TouchBegan {
			d.touch = true
		}
	}
	if in.LastDevice() == input.MouseDevice {
		d.touch = false
	}
}
