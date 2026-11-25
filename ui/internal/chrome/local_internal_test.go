//go:build chrome

package chrome

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThePinnedChromeIsAChromeForTestingVersion(t *testing.T) {
	parts := strings.Split(strings.TrimSpace(pinned), ".")
	require.Len(t, parts, 4, pinned)
	for _, p := range parts {
		assert.NotEmpty(t, p)
		assert.Empty(t, strings.Trim(p, "0123456789"), pinned)
	}
	sums, err := os.ReadFile("MD5SUMS")
	require.NoError(t, err)
	for _, platform := range []string{"mac-arm64", "mac-x64", "linux64", "win64", "win32"} {
		assert.Regexp(t, `(?m)^[0-9a-f]{32}  chrome-`+platform+`\.zip$`, string(sums))
	}
}

func TestTheChromeNamedWinsThenThePinnedOneThenTheSystems(t *testing.T) {
	lib := t.TempDir()
	assert.Equal(t, "/my/chrome", browserIn("/my/chrome", lib, "linux", "amd64"))
	assert.Equal(t, systemChrome, browserIn("", lib, "darwin", "arm64"), "nothing fetched")

	exe := pinnedIn(lib, "linux", "amd64")
	require.NoError(t, os.MkdirAll(filepath.Dir(exe), 0o750))
	require.NoError(t, os.WriteFile(exe, nil, 0o600))
	assert.Equal(t, exe, browserIn("", lib, "linux", "amd64"))
	assert.Equal(t, "/my/chrome", browserIn("/my/chrome", lib, "linux", "amd64"))
	assert.Equal(t, systemChrome, browserIn("", lib, "plan9", "amd64"), "no build for the system")
}

// The fetch script, run on a fake Chrome for Testing archive served from a
// folder, puts the executable where the harness looks for it.
func TestTaskChromeFetchesThePinnedChromeWhereTheHarnessLooks(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("no curl")
	}
	for _, c := range []struct{ goos, goarch, platform string }{
		{"darwin", "arm64", "mac-arm64"},
		{"linux", "amd64", "linux64"},
		{"windows", "amd64", "win64"},
	} {
		t.Run(c.platform, func(t *testing.T) {
			version := strings.TrimSpace(pinned)
			host, lib := t.TempDir(), t.TempDir()
			exe, err := filepath.Rel(filepath.Join(lib, version), pinnedIn(lib, c.goos, c.goarch))
			require.NoError(t, err)
			archive := filepath.Join(host, version, c.platform, "chrome-"+c.platform+".zip")
			fakeArchive(t, archive, filepath.ToSlash(exe))
			base := (&url.URL{Scheme: "file", Path: filepath.ToSlash(host)}).String()
			if runtime.GOOS == "windows" {
				base = "file:///" + filepath.ToSlash(host)
			}

			sums := filepath.Join(host, "MD5SUMS")
			writeSums(t, sums, "0123456789abcdef0123456789abcdef", c.platform)

			out, err := fetch(base, lib, sums, c.goos, c.goarch, true)
			require.NoError(t, err, out)
			assert.Contains(t, out, base+"/"+version+"/"+c.platform+"/chrome-"+c.platform+".zip")
			_, err = os.Stat(filepath.Join(lib, version))
			assert.True(t, os.IsNotExist(err), "a dry run fetches nothing")

			out, err = fetch(base, lib, sums, c.goos, c.goarch, false)
			require.Error(t, err, "an archive of another MD5 is refused")
			assert.Contains(t, out, "MD5")
			_, err = os.Stat(filepath.Join(lib, version))
			assert.True(t, os.IsNotExist(err), "and nothing is installed")

			data, err := os.ReadFile(archive)
			require.NoError(t, err)
			writeSums(t, sums, fmt.Sprintf("%x", md5.Sum(data)), c.platform)
			out, err = fetch(base, lib, sums, c.goos, c.goarch, false)
			require.NoError(t, err, out)
			assert.Equal(t, pinnedIn(lib, c.goos, c.goarch), browserIn("", lib, c.goos, c.goarch))
		})
	}
}

func fakeArchive(t *testing.T, path, exe string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	f, err := os.Create(path)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	w, err := z.Create(exe)
	require.NoError(t, err)
	_, err = w.Write([]byte("#!/bin/sh\n"))
	require.NoError(t, err)
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())
}

func writeSums(t *testing.T, path, sum, platform string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(sum+"  chrome-"+platform+".zip\n"), 0o600))
}

func fetch(base, lib, sums, goos, goarch string, dry bool) (string, error) {
	cmd := exec.Command("sh", filepath.Join("scripts", "fetch.sh"), goos, goarch)
	cmd.Env = append(os.Environ(), "EMBER_CHROME_CFT_BASE="+base, "EMBER_CHROME_LIB="+lib, "EMBER_CHROME_SUMS="+sums,
		"DRY_RUN=")
	if dry {
		cmd.Env = append(cmd.Env, "DRY_RUN=1")
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
