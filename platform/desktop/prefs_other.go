//go:build !darwin

package desktop

import (
	"time"

	"github.com/lantervale/ember/internal/sysprefs"
	"github.com/lantervale/ember/platform"
)

// repoll is how often the system is asked again where nothing ember
// listens to tells a change: Windows keeps its appearance in the registry.
const repoll = 2 * time.Second

// systemChanged reports whether the preferences are to read again: at the
// start, then every repoll since asked.
func systemChanged(asked *time.Time) bool {
	if now := time.Now(); now.Sub(*asked) >= repoll {
		*asked = now
		return true
	}
	return false
}

func systemAppearance() platform.Appearance { return sysprefs.Appearance() }

func systemLanguages() []string { return sysprefs.Languages() }
