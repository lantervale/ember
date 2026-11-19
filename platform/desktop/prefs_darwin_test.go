//go:build darwin

package desktop //nolint:testpackage // tests the unexported AppKit bridge to the system's preferences

import (
	"strings"
	"testing"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/platform"
)

func TestMacOSSaysItsAppearanceAndLanguagesAndWhenTheAppearanceChanges(t *testing.T) {
	var failed error
	var first, forced, back platform.Appearance
	var languages []string
	var told, quiet, again bool
	main(func() {
		if err := glfw.Init(); err != nil {
			failed = err
			return
		}
		watchPreferences()
		var asked time.Time
		told = systemChanged(&asked)
		first, languages = systemAppearance(), systemLanguages()
		quiet = !systemChanged(&asked)
		other := platform.AppearanceDark
		if first == platform.AppearanceDark {
			other = platform.AppearanceLight
		}
		overrideAppearance(other)
		again = systemChanged(&asked)
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
