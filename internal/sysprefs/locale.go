// Package sysprefs asks the system a program runs on for its user's
// preferences, where pure Go can: the languages and the appearance that
// platform.Preferences holds. It never changes them.
package sysprefs

import (
	"slices"
	"strings"
)

// Tag is the BCP 47 tag of a POSIX locale such as fr_FR.UTF-8@euro: its
// language and territory, the script or variant a modifier names, the
// codeset dropped. Empty for the C locale and for what is no locale.
func Tag(locale string) string {
	locale, modifier, _ := strings.Cut(locale, "@")
	locale, _, _ = strings.Cut(locale, ".")
	language, territory, _ := strings.Cut(strings.ReplaceAll(locale, "-", "_"), "_")
	if n := len(language); n < 2 || n > 3 || !all(language, letter) {
		return ""
	}
	tag := strings.ToLower(language)
	variant := ""
	switch strings.ToLower(modifier) {
	case "latin":
		tag += "-Latn"
	case "cyrillic":
		tag += "-Cyrl"
	case "devanagari":
		tag += "-Deva"
	case "valencia":
		variant = "-valencia"
	}
	if len(territory) == 2 && all(territory, letter) || len(territory) == 3 && all(territory, digit) {
		tag += "-" + strings.ToUpper(territory)
	}
	return tag + variant
}

func letter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func digit(c byte) bool { return c >= '0' && c <= '9' }

func all(s string, is func(byte) bool) bool {
	for i := range len(s) {
		if !is(s[i]) {
			return false
		}
	}
	return true
}

// FromEnv is the user's languages as a POSIX environment gives them, the
// preferred one first: the locale of messages, from LC_ALL, else
// LC_MESSAGES, else LANG, behind the list in LANGUAGE, which as gettext
// has it counts only outside the C locale.
func FromEnv(getenv func(string) string) []string {
	locale := ""
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale = getenv(name); locale != "" {
			break
		}
	}
	base := Tag(locale)
	if base == "" {
		return nil
	}
	var tags []string
	for _, l := range append(strings.Split(getenv("LANGUAGE"), ":"), locale) {
		if tag := Tag(l); tag != "" && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}
