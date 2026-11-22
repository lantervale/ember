package desktop //nolint:testpackage // tests the driver's unexported sync of the system's preferences

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestAScriptedRunAsksTheSystemNothing(t *testing.T) {
	w := world.New()
	prefs := platform.NewPreferences()
	w.AddEntities(prefs)
	d := &driver{world: w, scripted: true}
	w.AddView(&d.preferences)
	d.syncPreferences()
	if prefs.Appearance() != platform.AppearanceUnknown || prefs.Motion() != platform.MotionUnknown ||
		len(prefs.Languages()) != 0 {
		t.Fatalf("a capture plays the same on any machine: %v %v", prefs.Appearance(), prefs.Languages())
	}
	prefs.SyncAppearance(platform.AppearanceDark)
	prefs.SyncMotion(platform.MotionReduced)
	prefs.SyncLanguages("fr-FR")
	d.syncPreferences()
	if prefs.Appearance() != platform.AppearanceDark || prefs.Motion() != platform.MotionReduced ||
		!slices.Equal(prefs.Languages(), []string{"fr-FR"}) {
		t.Fatal("what the program simulates stays")
	}
}

func TestTheDriverGivesEveryPreferencesWhatItLastRead(t *testing.T) {
	w := world.New()
	early := platform.NewPreferences()
	w.AddEntities(early)
	d := &driver{
		world: w, appearance: platform.AppearanceLight, motion: platform.MotionReduced, languages: []string{"en-GB", "fr"},
	}
	w.AddView(&d.preferences)
	d.tellPreferences()
	late := platform.NewPreferences()
	w.AddEntities(late)
	d.tellPreferences()
	for _, p := range []*platform.Preferences{early, late} {
		if p.Appearance() != platform.AppearanceLight || p.Motion() != platform.MotionReduced ||
			!slices.Equal(p.Languages(), []string{"en-GB", "fr"}) {
			t.Fatalf("preferences added at any time hold the system's: %v %v", p.Appearance(), p.Languages())
		}
	}
}
