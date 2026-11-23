//go:build unix && !darwin

package sysprefs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lantervale/ember/platform"
)

// asking is how long one of the desktop's tools may take to answer.
const asking = time.Second

// Read is what a Linux desktop says, read-only: the settings portal over
// the session bus through gdbus, GNOME's settings through gsettings where
// it said nothing, KDE's kdeglobals and GTK's settings.ini after them,
// and the languages from the environment. ember speaks no D-Bus itself,
// so each read runs those tools: call it off the frame.
func Read() System {
	s := System{Languages: FromEnv(os.Getenv)}
	answered := false
	if out, ok := run("gdbus", "call", "--session", "--timeout", "1",
		"--dest", "org.freedesktop.portal.Desktop", "--object-path", "/org/freedesktop/portal/desktop",
		"--method", "org.freedesktop.portal.Settings.ReadAll", "['org.freedesktop.appearance']"); ok {
		answered = FromPortal(&s, out)
	}
	if _, err := exec.LookPath("gsettings"); err == nil {
		FromGSettings(&s, func(schema, key string) (string, bool) { return run("gsettings", "get", schema, key) })
	}
	if dir, err := os.UserConfigDir(); err == nil {
		if ini, ok := read(filepath.Join(dir, "kdeglobals")); ok {
			FromKDE(&s, ini)
		}
		for _, gtk := range []string{"gtk-4.0", "gtk-3.0"} {
			if ini, ok := read(filepath.Join(dir, gtk, "settings.ini")); ok {
				FromGTK(&s, ini)
			}
		}
	}
	if answered && s.Appearance == platform.AppearanceUnknown {
		s.Appearance = platform.AppearanceLight
	}
	return s
}

func read(path string) (string, bool) {
	b, err := os.ReadFile(path)
	return string(b), err == nil
}

// run is what a tool prints, false where it isn't there or fails.
func run(name string, args ...string) (string, bool) {
	if _, err := exec.LookPath(name); err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), asking)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
