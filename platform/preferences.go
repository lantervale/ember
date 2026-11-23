package platform

import (
	"slices"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/world"
)

// Appearance is the look the user asked their system for.
type Appearance int

// unknown is the word of an answer the system has not given.
const unknown = "unknown"

const (
	// AppearanceUnknown is the appearance of a system that has not said,
	// or cannot: before the first frame, in a test, a hidden run or a
	// server, and where the backend has no dependable way to ask.
	AppearanceUnknown Appearance = iota
	AppearanceLight
	AppearanceDark
)

func (a Appearance) String() string {
	switch a {
	case AppearanceLight:
		return "light"
	case AppearanceDark:
		return "dark"
	case AppearanceUnknown:
	}
	return unknown
}

// AppearanceChanged is emitted when the system's appearance changes, the
// first time it is known included.
type AppearanceChanged struct {
	Appearance Appearance
}

// Motion is whether the user asked their system for less motion on
// screen.
type Motion int

const (
	// MotionUnknown is the answer of a system that has not said, or
	// cannot, as AppearanceUnknown.
	MotionUnknown Motion = iota
	// MotionFull is a system that animates as it likes.
	MotionFull
	// MotionReduced is a system asked to reduce motion.
	MotionReduced
)

func (m Motion) String() string {
	switch m {
	case MotionFull:
		return "full"
	case MotionReduced:
		return "reduced"
	case MotionUnknown:
	}
	return unknown
}

// MotionChanged is emitted when the system's reduced motion setting
// changes, the first time it is known included.
type MotionChanged struct {
	Motion Motion
}

// Contrast is the contrast the user asked their system for, as CSS's
// prefers-contrast words it.
type Contrast int

const (
	// ContrastUnknown is the answer of a system that has not said, or
	// cannot, as AppearanceUnknown.
	ContrastUnknown Contrast = iota
	ContrastNoPreference
	ContrastMore
	ContrastLess
	// ContrastCustom is a system whose forced colours stand neither far
	// apart nor close.
	ContrastCustom
)

func (c Contrast) String() string {
	switch c {
	case ContrastNoPreference:
		return "no-preference"
	case ContrastMore:
		return "more"
	case ContrastLess:
		return "less"
	case ContrastCustom:
		return "custom"
	case ContrastUnknown:
	}
	return unknown
}

// ContrastChanged is emitted when the system's contrast changes, the
// first time it is known included.
type ContrastChanged struct {
	Contrast Contrast
}

// ForcedColors is whether the system forces a palette of its own on
// programs, as Windows' contrast themes do.
type ForcedColors int

const (
	// ForcedColorsUnknown is the answer of a system that has not said,
	// or cannot, as AppearanceUnknown.
	ForcedColorsUnknown ForcedColors = iota
	ForcedColorsNone
	ForcedColorsActive
)

func (f ForcedColors) String() string {
	switch f {
	case ForcedColorsNone:
		return "none"
	case ForcedColorsActive:
		return "active"
	case ForcedColorsUnknown:
	}
	return unknown
}

// ForcedColorsChanged is emitted when the system starts or stops forcing
// its colours, the first time it is known included.
type ForcedColorsChanged struct {
	ForcedColors ForcedColors
}

// Highlight is the colours the system selects text in: Background behind
// it while its window is focused, Inactive while it isn't, and Text, the
// text's own over them. A colour of no alpha is one the system does not
// say.
type Highlight struct {
	Background, Inactive, Text color.Color
}

// HighlightChanged is emitted when the system's selection colours change,
// the first time they are known included.
type HighlightChanged struct {
	Highlight Highlight
}

// LanguagesChanged is emitted when the user's languages change, the first
// time they are known included.
type LanguagesChanged struct {
	Languages []string
}

// Preferences is what the user chose on the system the program runs on,
// and an entity: once it is in the World the platform keeps it up to date,
// and each change is emitted as the next frame begins.
//
//	prefs := platform.NewPreferences()
//	w.AddEntity(prefs)
//
// Nothing is known until the program runs, nor in a test, a hidden run or
// a server: the Sync methods then stand for the system.
type Preferences struct {
	*world.BasicEntity
	appearance, toldAppearance Appearance
	motion, toldMotion         Motion
	contrast, toldContrast     Contrast
	forced, toldForced         ForcedColors
	highlight, toldHighlight   Highlight
	languages, toldLanguages   []string
}

// Preferred is any entity that is the system's preferences.
type Preferred interface {
	world.BasicFace
	GetPreferences() *Preferences
}

func NewPreferences() *Preferences { return &Preferences{BasicEntity: world.NewBasic()} }

func (p *Preferences) GetPreferences() *Preferences { return p }

// Appearance is whether the system shows itself light or dark,
// AppearanceUnknown when it has not said.
func (p *Preferences) Appearance() Appearance { return p.appearance }

// Motion is whether the system asks programs to reduce motion,
// MotionUnknown when it has not said.
func (p *Preferences) Motion() Motion { return p.motion }

// Contrast is the contrast the system asks programs for, ContrastUnknown
// when it has not said.
func (p *Preferences) Contrast() Contrast { return p.contrast }

// ForcedColors is whether the system forces its own colours,
// ForcedColorsUnknown when it has not said.
func (p *Preferences) ForcedColors() ForcedColors { return p.forced }

// Highlight is the colours the system selects text in, those it does not
// say clear.
func (p *Preferences) Highlight() Highlight { return p.highlight }

// SyncContrast records the contrast the system asks for. Called by
// platform implementations, and by tests in their place.
func (p *Preferences) SyncContrast(c Contrast) { p.contrast = c }

// SyncForcedColors records whether the system forces its colours. Called
// by platform implementations, and by tests in their place.
func (p *Preferences) SyncForcedColors(f ForcedColors) { p.forced = f }

// SyncHighlight records the system's selection colours. Called by
// platform implementations, and by tests in their place.
func (p *Preferences) SyncHighlight(h Highlight) { p.highlight = h }

// TakeContrast returns the contrast and whether it changed since it was
// last taken. Called by the World's windows.
func (p *Preferences) TakeContrast() (Contrast, bool) {
	changed := p.contrast != p.toldContrast
	p.toldContrast = p.contrast
	return p.contrast, changed
}

// TakeForcedColors returns whether colours are forced and whether that
// changed since it was last taken. Called by the World's windows.
func (p *Preferences) TakeForcedColors() (ForcedColors, bool) {
	changed := p.forced != p.toldForced
	p.toldForced = p.forced
	return p.forced, changed
}

// TakeHighlight returns the selection colours and whether they changed
// since they were last taken. Called by the World's windows.
func (p *Preferences) TakeHighlight() (Highlight, bool) {
	changed := p.highlight != p.toldHighlight
	p.toldHighlight = p.highlight
	return p.highlight, changed
}

// Languages is the user's languages, the one they prefer first, as BCP 47
// tags such as fr-FR or en; empty when the system has not said.
func (p *Preferences) Languages() []string { return slices.Clone(p.languages) }

// SyncAppearance records the system's appearance. Called by platform
// implementations, and by tests in their place.
func (p *Preferences) SyncAppearance(a Appearance) { p.appearance = a }

// SyncMotion records whether the system asks to reduce motion. Called
// by platform implementations, and by tests in their place.
func (p *Preferences) SyncMotion(m Motion) { p.motion = m }

// SyncLanguages records the user's languages, the preferred one first.
// Called by platform implementations, and by tests in their place.
func (p *Preferences) SyncLanguages(tags ...string) {
	if !slices.Equal(p.languages, tags) {
		p.languages = slices.Clone(tags)
	}
}

// TakeAppearance returns the appearance and whether it changed since it
// was last taken. Called by the World's windows.
func (p *Preferences) TakeAppearance() (Appearance, bool) {
	changed := p.appearance != p.toldAppearance
	p.toldAppearance = p.appearance
	return p.appearance, changed
}

// TakeMotion returns the motion answer and whether it changed since it
// was last taken. Called by the World's windows.
func (p *Preferences) TakeMotion() (Motion, bool) {
	changed := p.motion != p.toldMotion
	p.toldMotion = p.motion
	return p.motion, changed
}

// TakeLanguages returns the languages and whether they changed since they
// were last taken. Called by the World's windows.
func (p *Preferences) TakeLanguages() ([]string, bool) {
	changed := !slices.Equal(p.languages, p.toldLanguages)
	p.toldLanguages = p.languages
	return p.Languages(), changed
}
