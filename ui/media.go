package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/paint"
)

// Preferences has the Document's sheets answer prefers-color-scheme,
// prefers-reduced-motion, prefers-contrast and forced-colors as the
// system does, through the preferences the program added to the World
// (decisions 0121, 0155): a change restyles where a rule asks, as the
// next frame begins. Without it, or while the system has not said, the
// Document answers light, no preference and none, the same on any
// machine. ColorScheme, ReducedMotion, PrefersContrast and ForcedColors
// win over it. A selection no ::selection rule colours is drawn in the
// system's selection colours where it says them.
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

// Contrast is the contrast a Document answers prefers-contrast with.
type Contrast uint8

const (
	ContrastNoPreference Contrast = iota
	ContrastMore
	ContrastLess
	ContrastCustom
)

// platform is c as the system's preferences word it.
func (c Contrast) platform() platform.Contrast {
	switch c {
	case ContrastMore:
		return platform.ContrastMore
	case ContrastLess:
		return platform.ContrastLess
	case ContrastCustom:
		return platform.ContrastCustom
	case ContrastNoPreference:
	}
	return platform.ContrastNoPreference
}

// contrastOf is the system's known contrast c as a Document answers it.
func contrastOf(c platform.Contrast) Contrast {
	switch c {
	case platform.ContrastMore:
		return ContrastMore
	case platform.ContrastLess:
		return ContrastLess
	case platform.ContrastCustom:
		return ContrastCustom
	case platform.ContrastUnknown, platform.ContrastNoPreference:
	}
	return ContrastNoPreference
}

func (c Contrast) String() string {
	switch c {
	case ContrastMore:
		return "more"
	case ContrastLess:
		return "less"
	case ContrastCustom:
		return "custom"
	case ContrastNoPreference:
	}
	return "no-preference"
}

// PrefersContrast has the Document's sheets answer prefers-contrast with
// c, where a page that asks draws its borders and text stronger or
// softer, whatever the Preferences say.
func PrefersContrast(c Contrast) DocumentOption {
	return func(d *Document) { d.contrast = c.platform() }
}

// Contrast is what the Document answers prefers-contrast with: as the
// program said, else as the Preferences say, else as its host in an
// iframe, else no-preference.
func (d *Document) Contrast() Contrast {
	switch {
	case d.contrast != platform.ContrastUnknown:
		return contrastOf(d.contrast)
	case d.prefs != nil && d.prefs.Contrast() != platform.ContrastUnknown:
		return contrastOf(d.prefs.Contrast())
	case d.framed != nil:
		return d.framed.host.Contrast()
	}
	return ContrastNoPreference
}

// SetContrast changes the answer from the next layout, which styles the
// tree again where a rule asks, and from then on wins over the
// Preferences.
func (d *Document) SetContrast(c Contrast) { d.contrast = c.platform() }

// ForcedColors has the Document's sheets answer forced-colors: active,
// as on a system whose high-contrast theme forces its own colours,
// whatever the Preferences say. It answers the query only: ui forces no
// colour itself.
func ForcedColors() DocumentOption {
	return func(d *Document) { d.forced = platform.ForcedColorsActive }
}

// ForcedColors reports whether the Document answers forced-colors:
// active: as the program said, else as the Preferences say, else as its
// host in an iframe, else not.
func (d *Document) ForcedColors() bool {
	switch {
	case d.forced != platform.ForcedColorsUnknown:
		return d.forced == platform.ForcedColorsActive
	case d.prefs != nil && d.prefs.ForcedColors() != platform.ForcedColorsUnknown:
		return d.prefs.ForcedColors() == platform.ForcedColorsActive
	case d.framed != nil:
		return d.framed.host.ForcedColors()
	}
	return false
}

// SetForcedColors changes the answer from the next layout, and from then
// on wins over the Preferences.
func (d *Document) SetForcedColors(on bool) {
	d.forced = platform.ForcedColorsNone
	if on {
		d.forced = platform.ForcedColorsActive
	}
}

// highlight is the system's selection colours a selection with no
// ::selection rule is drawn in: the Preferences', else its host's in an
// iframe, none known else.
func (d *Document) systemHighlight() paint.Highlight {
	switch {
	case d.prefs != nil && d.prefs.Highlight() != (platform.Highlight{}):
		h := d.prefs.Highlight()
		return paint.Highlight{Background: h.Background, Inactive: h.Inactive, Text: h.Text}
	case d.framed != nil:
		return d.framed.host.systemHighlight()
	}
	return paint.Highlight{}
}

// TouchOnly has the Document answer as on a device with a touchscreen and
// no mouse: pointer and any-pointer coarse, hover and any-hover none.
// Without it a mouse is always there to hover, beside the finger a
// Document notes on its screen (hover: none while it points).
func TouchOnly() DocumentOption { return func(d *Document) { d.touchOnly = true } }

// TouchOnly reports whether the Document answers as on a device with no
// mouse: as the program said, or its host in an iframe.
func (d *Document) TouchOnly() bool {
	return d.touchOnly || d.framed != nil && d.framed.host.TouchOnly()
}

// SetTouchOnly changes the answer from the next layout.
func (d *Document) SetTouchOnly(on bool) { d.touchOnly = on }

// initialFont is the font media queries measure their font units in:
// font's initial value.
//
//nolint:gochecknoglobals // a constant font
var initialFont = style.Font{Family: []string{"sans-serif"}, Size: 16, Weight: 400}

// medium is what the Document answers the media features that aren't
// its viewport's with. The pointer is its window's, the outermost
// Document's, and so is whether that window is inactive.
func (d *Document) medium() css.Media {
	top := d
	for top.framed != nil {
		top = top.framed.host
	}
	m := css.Media{Dark: d.ColorScheme() == platform.AppearanceDark, ReducedMotion: d.ReducedMotion(),
		ForcedColors: d.ForcedColors(), Touch: top.touch, TouchOnly: d.TouchOnly(), Font: d.fontUnits(initialFont),
		Inactive: top.inactive}
	if c := d.Contrast(); c != ContrastNoPreference {
		m.Contrast = c.String()
	}
	return m
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

// titleBarArea is what env(titlebar-area-*) reads in a viewport of size
// points: the part of its window's title bar the system leaves the
// Document, between its buttons, in the Document's points; none for a
// Document in an iframe, or in a window whose system keeps its title bar
// above the content (decision 0142).
func (d *Document) titleBarArea(size geom.Vec2) css.TitleBarArea {
	win := d.owner.win
	if d.framed != nil || win == nil {
		return css.TitleBarArea{}
	}
	a := win.TitleBarArea()
	if a.Height <= 0 {
		return css.TitleBarArea{}
	}
	z := d.Zoom()
	return css.TitleBarArea{X: a.Left / z, Width: max(size.X-(a.Left+a.Right)/z, 0), Height: a.Height / z}
}
