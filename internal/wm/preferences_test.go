package wm //nolint:testpackage // shares windows_test's package

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestTheWorldTellsEachChangeOfThePreferencesOnce(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	prefs := platform.NewPreferences()
	w.AddEntities(prefs)
	var looks world.Reader[platform.AppearanceChanged]
	var tongues world.Reader[platform.LanguagesChanged]
	w.RunFrames(1)
	if looks.Count(w) != 0 || tongues.Count(w) != 0 {
		t.Fatal("with no backend nothing is known, and nothing changes")
	}

	prefs.SyncAppearance(platform.AppearanceDark)
	prefs.SyncLanguages("fr-FR", "en-GB")
	w.RunFrames(1)
	if got := slices.Collect(looks.Read(w)); len(got) != 1 || got[0].Appearance != platform.AppearanceDark {
		t.Fatalf("the appearance now known is told once: %v", got)
	}
	got := slices.Collect(tongues.Read(w))
	if len(got) != 1 || !slices.Equal(got[0].Languages, []string{"fr-FR", "en-GB"}) {
		t.Fatalf("the languages now known are told once: %v", got)
	}

	prefs.SyncAppearance(platform.AppearanceDark)
	prefs.SyncLanguages("fr-FR", "en-GB")
	w.RunFrames(2)
	if looks.Count(w) != 0 || tongues.Count(w) != 0 {
		t.Fatal("the same values again tell nothing")
	}

	prefs.SyncAppearance(platform.AppearanceLight)
	w.RunFrames(1)
	if got := slices.Collect(looks.Read(w)); len(got) != 1 || got[0].Appearance != platform.AppearanceLight {
		t.Fatalf("a change is told once: %v", got)
	}
	if tongues.Count(w) != 0 {
		t.Fatal("the languages did not change")
	}

	w.RemoveEntity(prefs)
	w.RunFrames(1)
	prefs.SyncAppearance(platform.AppearanceDark)
	w.RunFrames(1)
	if looks.Count(w) != 0 {
		t.Fatal("preferences out of the World tell it nothing")
	}
}

func TestTheWorldTellsEachChangeOfReducedMotionOnce(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	prefs := platform.NewPreferences()
	w.AddEntities(prefs)
	var motions world.Reader[platform.MotionChanged]
	w.RunFrames(1)
	if motions.Count(w) != 0 {
		t.Fatal("with no backend nothing is known, and nothing changes")
	}
	prefs.SyncMotion(platform.MotionReduced)
	w.RunFrames(1)
	if got := slices.Collect(motions.Read(w)); len(got) != 1 || got[0].Motion != platform.MotionReduced {
		t.Fatalf("the answer now known is told once: %v", got)
	}
	w.RunFrames(2)
	if motions.Count(w) != 0 {
		t.Fatal("the same answer tells nothing")
	}
	prefs.SyncMotion(platform.MotionFull)
	w.RunFrames(1)
	if got := slices.Collect(motions.Read(w)); len(got) != 1 || got[0].Motion != platform.MotionFull {
		t.Fatalf("a change is told once: %v", got)
	}
}
