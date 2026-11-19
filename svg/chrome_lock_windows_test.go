//go:build chrome && windows

package svg_test

import "testing"

func lockChrome(*testing.T) func() { return func() {} }
