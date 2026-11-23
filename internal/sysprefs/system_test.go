package sysprefs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/sysprefs"
	"github.com/lantervale/ember/platform"
)

func TestAForcedThemesContrastIsItsTextsFromItsBackground(t *testing.T) {
	for _, c := range []struct {
		back, text string
		want       platform.Contrast
	}{
		{"#000000", "#ffffff", platform.ContrastMore},
		{"#ffffff", "#000000", platform.ContrastMore},
		{"#000000", "#ffff00", platform.ContrastMore},
		{"#ffffff", "#767676", platform.ContrastCustom},
		{"#202020", "#5a5a5a", platform.ContrastLess},
	} {
		assert.Equal(t, c.want, sysprefs.ForcedContrast(color.Hex(c.back), color.Hex(c.text)), "%s on %s", c.text, c.back)
	}
}

func TestTheSettingsPortalSaysTheSchemeContrastAndMotion(t *testing.T) {
	var s sysprefs.System
	out := `({'org.freedesktop.appearance': {'color-scheme': <uint32 1>, 'accent-color': <(0.2, 0.5, 0.9)>, ` +
		`'contrast': <uint32 1>, 'reduced-motion': <uint32 1>}},)`
	assert.True(t, sysprefs.FromPortal(&s, out))
	assert.Equal(t, sysprefs.System{
		Appearance: platform.AppearanceDark, Contrast: platform.ContrastMore, Motion: platform.MotionReduced,
	}, s)

	s = sysprefs.System{}
	assert.True(t, sysprefs.FromPortal(&s, `({'org.freedesktop.appearance': {'color-scheme': <uint32 0>}},)`))
	assert.Equal(t, platform.AppearanceUnknown, s.Appearance, "no preference: left to the desktop's own settings")
	assert.Equal(t, platform.ContrastUnknown, s.Contrast, "an older portal says no contrast")

	s = sysprefs.System{}
	assert.True(t, sysprefs.FromPortal(&s, `({'org.freedesktop.appearance': {'color-scheme': <<uint32 2>>, `+
		`'contrast': <uint32 0>, 'reduced-motion': <uint32 0>}},)`))
	assert.Equal(t, sysprefs.System{
		Appearance: platform.AppearanceLight, Contrast: platform.ContrastNoPreference, Motion: platform.MotionFull,
	}, s)

	s = sysprefs.System{}
	assert.False(t, sysprefs.FromPortal(&s, "Error: GDBus.Error:org.freedesktop.DBus.Error.ServiceUnknown"))
	assert.Equal(t, sysprefs.System{}, s)
}

func TestGNOMEsSettingsFillWhatThePortalLeft(t *testing.T) {
	settings := map[string]string{
		"org.gnome.desktop.interface color-scheme":       "'default'",
		"org.gnome.desktop.interface gtk-theme":          "'Adwaita-dark'",
		"org.gnome.desktop.interface enable-animations":  "false",
		"org.gnome.desktop.a11y.interface high-contrast": "true",
	}
	get := func(schema, key string) (string, bool) {
		v, ok := settings[schema+" "+key]
		return v, ok
	}
	var s sysprefs.System
	sysprefs.FromGSettings(&s, get)
	assert.Equal(t, sysprefs.System{
		Appearance: platform.AppearanceDark, Motion: platform.MotionReduced, Contrast: platform.ContrastMore,
	}, s, "an old dark theme, no animations, high contrast")

	s = sysprefs.System{Appearance: platform.AppearanceLight, Motion: platform.MotionFull}
	settings["org.gnome.desktop.interface color-scheme"] = "'prefer-dark'"
	sysprefs.FromGSettings(&s, get)
	assert.Equal(t, platform.AppearanceLight, s.Appearance, "what the portal said stays")
	assert.Equal(t, platform.MotionFull, s.Motion)

	s = sysprefs.System{}
	sysprefs.FromGSettings(&s, get)
	assert.Equal(t, platform.AppearanceDark, s.Appearance)

	s = sysprefs.System{}
	sysprefs.FromGSettings(&s, func(string, string) (string, bool) { return "", false })
	assert.Equal(t, sysprefs.System{}, s, "no GNOME, nothing said")
}

func TestKDEsGlobalsSayItsSelectionColoursDarknessAndAnimations(t *testing.T) {
	const breezeDark = `[ColorEffects:Inactive]
ChangeSelectionColor=true

[Colors:Selection]
BackgroundAlternate=30,87,116
BackgroundNormal=61,174,233
ForegroundNormal=252,252,252

[Colors:Window]
BackgroundNormal=32,35,38

[KDE]
AnimationDurationFactor=0
`
	var s sysprefs.System
	sysprefs.FromKDE(&s, breezeDark)
	assert.Equal(t, sysprefs.System{
		Appearance: platform.AppearanceDark, Motion: platform.MotionReduced,
		Highlight: platform.Highlight{Background: color.RGB8(61, 174, 233), Text: color.RGB8(252, 252, 252)},
	}, s)

	s = sysprefs.System{}
	sysprefs.FromKDE(&s, "[Colors:Window]\nBackgroundNormal=239,240,241\n[KDE]\nAnimationDurationFactor=0.5\n")
	assert.Equal(t, sysprefs.System{Appearance: platform.AppearanceLight, Motion: platform.MotionFull}, s)

	s = sysprefs.System{}
	sysprefs.FromKDE(&s, "[Colors:Selection]\nBackgroundNormal=not,a,colour\n")
	assert.Equal(t, sysprefs.System{}, s)
}

func TestGTKsSettingsSayADarkThemeAndAnimations(t *testing.T) {
	var s sysprefs.System
	sysprefs.FromGTK(&s, "[Settings]\ngtk-application-prefer-dark-theme=1\ngtk-enable-animations=false\n")
	assert.Equal(t, sysprefs.System{Appearance: platform.AppearanceDark, Motion: platform.MotionReduced}, s)

	s = sysprefs.System{}
	sysprefs.FromGTK(&s, "[Settings]\ngtk-theme-name=Yaru-dark\ngtk-enable-animations=1\n")
	assert.Equal(t, sysprefs.System{Appearance: platform.AppearanceDark, Motion: platform.MotionFull}, s)

	s = sysprefs.System{}
	sysprefs.FromGTK(&s, "[Other]\ngtk-application-prefer-dark-theme=1\n")
	assert.Equal(t, sysprefs.System{}, s, "outside its section")
}

func TestABrowsersComputedColourIsRead(t *testing.T) {
	for in, want := range map[string]color.Color{
		"rgb(0, 120, 215)":         color.Hex("#0078d7"),
		"rgba(128, 188, 254, 0.6)": color.Hex("#80bcfe").Alpha(0.6),
		"rgb(128 188 254 / 0.6)":   color.Hex("#80bcfe").Alpha(0.6),
		" rgb(255,255,255) ":       color.White,
	} {
		got, ok := sysprefs.ComputedColor(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "Highlight", "hsl(0, 0%, 0%)", "rgb(1, 2)", "rgb(1, 2, 300)", "rgba(1, 2, 3, 2)",
		"rgb(1, 2, 3"} {
		_, ok := sysprefs.ComputedColor(bad)
		assert.False(t, ok, bad)
	}
}

func TestSystemsAreEqualWhenTheySayTheSame(t *testing.T) {
	a := sysprefs.System{Appearance: platform.AppearanceDark, Languages: []string{"fr-FR", "en"}}
	b := a
	b.Languages = []string{"fr-FR", "en"}
	assert.True(t, a.Equal(b))
	b.Languages = []string{"en", "fr-FR"}
	assert.False(t, a.Equal(b))
	b = a
	b.Highlight.Text = color.White
	assert.False(t, a.Equal(b))
}
