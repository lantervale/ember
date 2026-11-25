//go:build chrome

package chrome

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// pinned is the Chrome for Testing version CI checks ui against, which
// task chrome fetches into lib.
//
//go:embed VERSION
var pinned string

// systemChrome is the Chrome installed on macOS.
const systemChrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// browser is the Chrome the tests run: EMBER_CHROME, else the pinned
// Chrome for Testing task chrome fetched beside this package, else the
// system's.
func browser() string {
	_, file, _, _ := runtime.Caller(0)
	return browserIn(os.Getenv("EMBER_CHROME"), filepath.Join(filepath.Dir(file), "lib"), runtime.GOOS, runtime.GOARCH)
}

func browserIn(named, lib, goos, goarch string) string {
	if named != "" {
		return named
	}
	if p := pinnedIn(lib, goos, goarch); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return systemChrome
}

// pinnedIn is where fetch.sh puts the pinned Chrome's executable under
// lib for goos and goarch, empty where Chrome for Testing has no build.
func pinnedIn(lib, goos, goarch string) string {
	var platform, exe string
	switch goos + "-" + goarch {
	case "darwin-arm64", "darwin-amd64":
		platform = map[string]string{"arm64": "mac-arm64", "amd64": "mac-x64"}[goarch]
		exe = "Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
	case "linux-amd64":
		platform, exe = "linux64", "chrome"
	case "windows-amd64", "windows-386":
		platform = map[string]string{"amd64": "win64", "386": "win32"}[goarch]
		exe = "chrome.exe"
	default:
		return ""
	}
	return filepath.Join(lib, strings.TrimSpace(pinned), "chrome-"+platform, filepath.FromSlash(exe))
}
