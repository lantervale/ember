//go:build !windows

package sysprefs

import (
	"os"

	"github.com/lantervale/ember/platform"
)

// Languages is the user's languages as the environment has them.
func Languages() []string { return FromEnv(os.Getenv) }

// Appearance is unknown: a Linux desktop says whether it is dark through
// its settings portal, over D-Bus, which ember does not speak.
func Appearance() platform.Appearance { return platform.AppearanceUnknown }
