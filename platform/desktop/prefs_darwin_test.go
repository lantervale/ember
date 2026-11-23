//go:build darwin

package desktop //nolint:testpackage // tests the unexported AppKit bridge to the system's preferences

import (
	"strings"
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/platform"
)

func TestMacOSSaysItsPreferencesAndWhenTheAppearanceChanges(t *testing.T) {
	var failed error
	var first, forced, back platform.Appearance
	var motion platform.Motion
	var contrast platform.Contrast
	var highlight platform.Highlight
	var languages []string
	var told, quiet, again bool
	main(func() {
		if err := glfw.Init(); err != nil {
			failed = err
			return
		}
		watchPreferences()
		told = systemChanged()
		first, motion, languages = systemAppearance(), systemMotion(), systemLanguages()
		contrast, highlight = systemContrast(), systemHighlight()
		quiet = !systemChanged()
		other := platform.AppearanceDark
		if first == platform.AppearanceDark {
			other = platform.AppearanceLight
		}
		overrideAppearance(other)
		again = systemChanged()
		forced = systemAppearance()
		overrideAppearance(platform.AppearanceUnknown)
		back = systemAppearance()
	})
	if failed != nil {
		t.Skipf("no AppKit here: %v", failed)
	}
	if !told || !quiet {
		t.Fatalf("the preferences are to read once at the start, then only when told: %v %v", told, quiet)
	}
	if first == platform.AppearanceUnknown {
		t.Fatal("macOS knows whether it is light or dark")
	}
	if motion == platform.MotionUnknown {
		t.Fatal("macOS knows whether it reduces motion")
	}
	if contrast != platform.ContrastNoPreference && contrast != platform.ContrastMore {
		t.Fatalf("macOS says whether it increases contrast: %v", contrast)
	}
	if highlight.Background.A != 1 || highlight.Inactive.A != 1 || highlight.Text.A != 0 {
		t.Fatalf("macOS says the colours it selects in, the text keeping its own: %+v", highlight)
	}
	if len(languages) == 0 {
		t.Fatal("macOS has at least one preferred language")
	}
	for _, tag := range languages {
		if tag == "" || strings.ContainsAny(tag, "_\n ") {
			t.Fatalf("%q is no BCP 47 tag", tag)
		}
	}
	if !again || forced == first || forced == platform.AppearanceUnknown {
		t.Fatalf("the app's appearance changing is observed and read: told %v, %v after %v", again, forced, first)
	}
	if back != first {
		t.Fatalf("the system's appearance is back once the app follows it again: %v, was %v", back, first)
	}
}
