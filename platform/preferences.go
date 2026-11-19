package platform

import (
	"slices"

	"github.com/lantervale/ember/world"
)

// Appearance is the look the user asked their system for.
type Appearance int

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
	return "unknown"
}

// AppearanceChanged is emitted when the system's appearance changes, the
// first time it is known included.
type AppearanceChanged struct {
	Appearance Appearance
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
// a server: SyncAppearance and SyncLanguages then stand for the system.
type Preferences struct {
	*world.BasicEntity
	appearance, toldAppearance Appearance
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

// Languages is the user's languages, the one they prefer first, as BCP 47
// tags such as fr-FR or en; empty when the system has not said.
func (p *Preferences) Languages() []string { return slices.Clone(p.languages) }

// SyncAppearance records the system's appearance. Called by platform
// implementations, and by tests in their place.
func (p *Preferences) SyncAppearance(a Appearance) { p.appearance = a }

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

// TakeLanguages returns the languages and whether they changed since they
// were last taken. Called by the World's windows.
func (p *Preferences) TakeLanguages() ([]string, bool) {
	changed := !slices.Equal(p.languages, p.toldLanguages)
	p.toldLanguages = p.languages
	return p.Languages(), changed
}
