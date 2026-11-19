package sysprefs_test

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/internal/sysprefs"
)

func TestAPOSIXLocaleBecomesABCP47Tag(t *testing.T) {
	for locale, want := range map[string]string{
		"fr_FR.UTF-8":      "fr-FR",
		"en_GB":            "en-GB",
		"de":               "de",
		"EN_us.utf8":       "en-US",
		"zh_CN.GB18030":    "zh-CN",
		"de_DE@euro":       "de-DE",
		"de_DE.UTF-8@euro": "de-DE",
		"sr_RS@latin":      "sr-Latn-RS",
		"uz_UZ@cyrillic":   "uz-Cyrl-UZ",
		"sr@latin":         "sr-Latn",
		"ca_ES@valencia":   "ca-ES-valencia",
		"es_419":           "es-419",
		"ast_ES":           "ast-ES",
		"fr-CA":            "fr-CA",
		"C":                "",
		"POSIX":            "",
		"C.UTF-8":          "",
		"":                 "",
		"_FR":              "",
		"fr_FRANCE":        "fr",
		"français":         "",
		"x":                "",
	} {
		if got := sysprefs.Tag(locale); got != want {
			t.Errorf("%q: %q, want %q", locale, got, want)
		}
	}
}

func TestTheEnvironmentListsTheLanguagesInOrder(t *testing.T) {
	for _, c := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		{"nothing set", nil, nil},
		{"LANG alone", map[string]string{"LANG": "fr_FR.UTF-8"}, []string{"fr-FR"}},
		{
			"LC_MESSAGES before LANG",
			map[string]string{"LANG": "fr_FR.UTF-8", "LC_MESSAGES": "en_GB.UTF-8"},
			[]string{"en-GB"},
		},
		{
			"LC_ALL before both",
			map[string]string{"LANG": "fr_FR.UTF-8", "LC_MESSAGES": "en_GB.UTF-8", "LC_ALL": "de_DE"},
			[]string{"de-DE"},
		},
		{
			"LANGUAGE first, then the locale, each once",
			map[string]string{"LANGUAGE": "fr_CH:fr:de_CH::en_GB", "LANG": "de_CH.UTF-8"},
			[]string{"fr-CH", "fr", "de-CH", "en-GB"},
		},
		{"LANGUAGE counts for nothing in the C locale", map[string]string{"LANGUAGE": "fr", "LANG": "C"}, nil},
		{"nor without a locale", map[string]string{"LANGUAGE": "fr"}, nil},
		{"C in LC_ALL hides LANG", map[string]string{"LC_ALL": "C.UTF-8", "LANG": "fr_FR"}, nil},
	} {
		got := sysprefs.FromEnv(func(name string) string { return c.env[name] })
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
