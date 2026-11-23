//go:build !darwin

package desktop

import "github.com/lantervale/ember/internal/sysprefs"

// systemAsker asks Windows and a Linux desktop, which tell ember no
// change: Windows' answers are cheap calls, a Linux desktop's are tools
// ember runs, so both are read before the first frame and when a window
// gains the focus, never on a timer.
//
//nolint:gochecknoglobals // one system per process
var systemAsker = newAsker(sysprefs.Read)

// systemChanged reports whether a read of the system says something new
// since it was last asked; gained asks it again.
func systemChanged(gained bool) bool { return systemAsker.changed(gained) }

func systemPreferences() sysprefs.System { return systemAsker.system() }
