#!/bin/sh
# Fetches the Chrome for Testing build CI checks ui against (../VERSION)
# into ui/internal/chrome/lib/<version>/, which git ignores: the ui Chrome
# tests run against it rather than the system's Chrome, so a machine and
# CI compare with the same Chrome. DRY_RUN=1 says what it would fetch and
# where, fetching nothing. The archive must have the MD5 ../MD5SUMS gives
# it, the one Google's storage states for it (its x-goog-hash).
#
#	ui/internal/chrome/scripts/fetch.sh [goos goarch]
#
# EMBER_CHROME_CFT_BASE replaces the download host (a file:// folder laid
# out as it, for the script's test), EMBER_CHROME_LIB the lib folder and
# EMBER_CHROME_SUMS the checksums.
set -eu
here=$(cd "$(dirname "$0")/.." && pwd)
version=$(tr -d '[:space:]' < "$here/VERSION")
goos=${1:-$(go env GOOS)}
goarch=${2:-$(go env GOARCH)}
case "$goos-$goarch" in
	darwin-arm64) platform=mac-arm64 ;;
	darwin-amd64) platform=mac-x64 ;;
	linux-amd64) platform=linux64 ;;
	windows-amd64) platform=win64 ;;
	windows-386) platform=win32 ;;
	*) echo "chrome: no Chrome for Testing build for $goos-$goarch" >&2; exit 1 ;;
esac
base=${EMBER_CHROME_CFT_BASE:-https://storage.googleapis.com/chrome-for-testing-public}
url="$base/$version/$platform/chrome-$platform.zip"
out="${EMBER_CHROME_LIB:-$here/lib}/$version"
if [ -n "${DRY_RUN:-}" ]; then
	echo "chrome: would fetch $url into $out/chrome-$platform"
	exit 0
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "chrome: $url"
curl -fsSL -o "$tmp/chrome.zip" "$url"
want=$(awk -v f="chrome-$platform.zip" '$2 == f { print $1 }' "${EMBER_CHROME_SUMS:-$here/MD5SUMS}")
if command -v md5sum >/dev/null 2>&1; then
	got=$(md5sum "$tmp/chrome.zip" | cut -d' ' -f1)
else
	got=$(md5 -q "$tmp/chrome.zip")
fi
if [ -z "$want" ] || [ "$got" != "$want" ]; then
	echo "chrome: chrome-$platform.zip's MD5 is $got, MD5SUMS says ${want:-nothing}" >&2
	exit 1
fi
mkdir -p "$tmp/x"
if command -v unzip >/dev/null 2>&1; then
	unzip -q -o "$tmp/chrome.zip" -d "$tmp/x"
else
	# Windows has no unzip, but its tar (bsdtar) reads zips.
	tar -xf "$tmp/chrome.zip" -C "$tmp/x"
fi
if [ ! -d "$tmp/x/chrome-$platform" ]; then
	echo "chrome: chrome-$platform not in the archive" >&2
	exit 1
fi
mkdir -p "$out"
rm -rf "$out/chrome-$platform"
mv "$tmp/x/chrome-$platform" "$out/"
echo "chrome: $out/chrome-$platform"
