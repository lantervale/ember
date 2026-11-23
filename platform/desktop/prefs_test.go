package desktop //nolint:testpackage // tests the driver's unexported sync of the system's preferences

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/sysprefs"
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
		prefs.Contrast() != platform.ContrastUnknown || prefs.Highlight() != (platform.Highlight{}) ||
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
	blue := platform.Highlight{Background: color.Hex("#0078d7"), Text: color.White}
	d := &driver{world: w, system: sysprefs.System{
		Appearance: platform.AppearanceLight, Motion: platform.MotionReduced, Contrast: platform.ContrastMore,
		ForcedColors: platform.ForcedColorsActive, Highlight: blue, Languages: []string{"en-GB", "fr"},
	}}
	w.AddView(&d.preferences)
	d.tellPreferences()
	late := platform.NewPreferences()
	w.AddEntities(late)
	d.tellPreferences()
	for _, p := range []*platform.Preferences{early, late} {
		if p.Appearance() != platform.AppearanceLight || p.Motion() != platform.MotionReduced ||
			p.Contrast() != platform.ContrastMore || p.ForcedColors() != platform.ForcedColorsActive ||
			p.Highlight() != blue || !slices.Equal(p.Languages(), []string{"en-GB", "fr"}) {
			t.Fatalf("preferences added at any time hold the system's: %v %v", p.Appearance(), p.Languages())
		}
	}
}
