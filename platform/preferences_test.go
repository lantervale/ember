package platform //nolint:testpackage // shares window_test's package

import (
	"fmt"
	"slices"
	"testing"

	"github.com/lantervale/ember/color"
)

func TestPreferencesAreUnknownUntilTheSystemSays(t *testing.T) {
	p := NewPreferences()
	if p.Appearance() != AppearanceUnknown || len(p.Languages()) != 0 {
		t.Fatalf("with no backend: %v %v", p.Appearance(), p.Languages())
	}
	if _, changed := p.TakeAppearance(); changed {
		t.Fatal("nothing changed yet")
	}
	if _, changed := p.TakeLanguages(); changed {
		t.Fatal("nothing changed yet")
	}
	for a, word := range map[Appearance]string{
		AppearanceUnknown: "unknown", AppearanceLight: "light", AppearanceDark: "dark",
	} {
		if a.String() != word {
			t.Fatalf("an appearance prints as its word: %q, want %q", a.String(), word)
		}
	}
}

func TestAnAppearanceIsToldOncePerChange(t *testing.T) {
	p := NewPreferences()
	p.SyncAppearance(AppearanceDark)
	if p.Appearance() != AppearanceDark {
		t.Fatal("the appearance reads at once")
	}
	if a, changed := p.TakeAppearance(); !changed || a != AppearanceDark {
		t.Fatalf("the first one known is a change: %v %v", a, changed)
	}
	p.SyncAppearance(AppearanceDark)
	if _, changed := p.TakeAppearance(); changed {
		t.Fatal("the same appearance again is no change")
	}
	p.SyncAppearance(AppearanceLight)
	p.SyncAppearance(AppearanceDark)
	if _, changed := p.TakeAppearance(); changed {
		t.Fatal("there and back between two frames is no change")
	}
}

func TestLanguagesAreToldOncePerChange(t *testing.T) {
	p := NewPreferences()
	tags := []string{"fr-FR", "en-GB"}
	p.SyncLanguages(tags...)
	tags[0] = "de"
	if !slices.Equal(p.Languages(), []string{"fr-FR", "en-GB"}) {
		t.Fatalf("the languages are kept in order, apart from the caller's slice: %v", p.Languages())
	}
	if l, changed := p.TakeLanguages(); !changed || !slices.Equal(l, []string{"fr-FR", "en-GB"}) {
		t.Fatalf("the first ones known are a change: %v %v", l, changed)
	}
	p.SyncLanguages("fr-FR", "en-GB")
	if _, changed := p.TakeLanguages(); changed {
		t.Fatal("the same languages again are no change")
	}
	p.SyncLanguages("en-GB", "fr-FR")
	if l, changed := p.TakeLanguages(); !changed || l[0] != "en-GB" {
		t.Fatalf("another order is a change: %v %v", l, changed)
	}
	p.SyncLanguages()
	if l, changed := p.TakeLanguages(); !changed || len(l) != 0 || len(p.Languages()) != 0 {
		t.Fatalf("no language known any more is a change: %v %v", l, changed)
	}
}

func TestReducedMotionIsToldOncePerChange(t *testing.T) {
	p := NewPreferences()
	if p.Motion() != MotionUnknown {
		t.Fatalf("with no backend: %v", p.Motion())
	}
	if _, changed := p.TakeMotion(); changed {
		t.Fatal("nothing changed yet")
	}
	p.SyncMotion(MotionReduced)
	if m, changed := p.TakeMotion(); !changed || m != MotionReduced || p.Motion() != MotionReduced {
		t.Fatalf("the first one known is a change: %v %v", m, changed)
	}
	p.SyncMotion(MotionReduced)
	if _, changed := p.TakeMotion(); changed {
		t.Fatal("the same answer again is no change")
	}
	p.SyncMotion(MotionFull)
	if m, changed := p.TakeMotion(); !changed || m != MotionFull {
		t.Fatalf("a change is told: %v %v", m, changed)
	}
	for m, word := range map[Motion]string{MotionUnknown: "unknown", MotionFull: "full", MotionReduced: "reduced"} {
		if m.String() != word {
			t.Fatalf("a motion prints as its word: %q, want %q", m.String(), word)
		}
	}
}

func TestContrastForcedColoursAndTheHighlightAreToldOncePerChange(t *testing.T) {
	p := NewPreferences()
	if p.Contrast() != ContrastUnknown || p.ForcedColors() != ForcedColorsUnknown || p.Highlight() != (Highlight{}) {
		t.Fatalf("with no backend: %v %v %v", p.Contrast(), p.ForcedColors(), p.Highlight())
	}
	_, c := p.TakeContrast()
	_, f := p.TakeForcedColors()
	_, h := p.TakeHighlight()
	if c || f || h {
		t.Fatal("nothing changed yet")
	}
	p.SyncContrast(ContrastMore)
	p.SyncForcedColors(ForcedColorsActive)
	blue := Highlight{Background: color.Hex("#0078d7"), Text: color.White}
	p.SyncHighlight(blue)
	if c, changed := p.TakeContrast(); !changed || c != ContrastMore || p.Contrast() != ContrastMore {
		t.Fatalf("the first contrast known is a change: %v %v", c, changed)
	}
	if f, changed := p.TakeForcedColors(); !changed || f != ForcedColorsActive || p.ForcedColors() != ForcedColorsActive {
		t.Fatalf("the first forced colours known are a change: %v %v", f, changed)
	}
	if h, changed := p.TakeHighlight(); !changed || h != blue || p.Highlight() != blue {
		t.Fatalf("the first highlight known is a change: %v %v", h, changed)
	}
	p.SyncContrast(ContrastMore)
	p.SyncForcedColors(ForcedColorsActive)
	p.SyncHighlight(blue)
	_, c = p.TakeContrast()
	_, f = p.TakeForcedColors()
	_, h = p.TakeHighlight()
	if c || f || h {
		t.Fatalf("the same answers again are no change: %v %v %v", c, f, h)
	}
	p.SyncHighlight(Highlight{Background: color.Hex("#b3d7ff")})
	if _, changed := p.TakeHighlight(); !changed {
		t.Fatal("another colour is a change")
	}
	for v, word := range map[fmt.Stringer]string{
		ContrastUnknown: "unknown", ContrastNoPreference: "no-preference", ContrastMore: "more", ContrastLess: "less",
		ContrastCustom: "custom", ForcedColorsUnknown: "unknown", ForcedColorsNone: "none", ForcedColorsActive: "active",
	} {
		if v.String() != word {
			t.Fatalf("an answer prints as its word: %q, want %q", v.String(), word)
		}
	}
}
