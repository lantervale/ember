//go:build !windows && (!unix || darwin)

package sysprefs

import "os"

// Read is the user's languages as the environment has them: macOS says
// the rest through AppKit, in platform/desktop.
func Read() System { return System{Languages: FromEnv(os.Getenv)} }
