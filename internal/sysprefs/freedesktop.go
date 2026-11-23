package sysprefs

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
)

// portalValue is one uint32 of org.freedesktop.appearance in what gdbus
// prints of the settings portal's answer.
var portalValue = regexp.MustCompile(`'(color-scheme|contrast|reduced-motion)': <+uint32 (\d+)>`)

// FromPortal reads into s what the settings portal (xdg-desktop-portal)
// says of org.freedesktop.appearance, its ReadAll answer as gdbus prints
// it: color-scheme 1 dark and 2 light, contrast 1 higher, reduced-motion
// 1 reduced. It reports whether the portal said anything: a portal that
// has no preference for a scheme then means light.
func FromPortal(s *System, out string) bool {
	answered := false
	for _, m := range portalValue.FindAllStringSubmatch(out, -1) {
		answered = true
		switch v := m[2]; m[1] {
		case "color-scheme":
			s.Appearance = map[string]platform.Appearance{"1": platform.AppearanceDark, "2": platform.AppearanceLight}[v]
		case "contrast":
			s.Contrast = platform.ContrastNoPreference
			if v == "1" {
				s.Contrast = platform.ContrastMore
			}
		case "reduced-motion":
			s.Motion = platform.MotionFull
			if v == "1" {
				s.Motion = platform.MotionReduced
			}
		}
	}
	return answered
}

// FromGSettings reads into s what GNOME's settings say where s does not
// know yet, get asking one key of a schema as gsettings prints it: the
// color-scheme, else a theme named dark; whether it animates; its high
// contrast.
func FromGSettings(s *System, get func(schema, key string) (string, bool)) {
	const desktop = "org.gnome.desktop.interface"
	if s.Appearance == platform.AppearanceUnknown {
		switch v, _ := get(desktop, "color-scheme"); v {
		case "'prefer-dark'":
			s.Appearance = platform.AppearanceDark
		case "'prefer-light'":
			s.Appearance = platform.AppearanceLight
		default:
			if theme, ok := get(desktop, "gtk-theme"); ok && darkTheme(strings.Trim(theme, "'")) {
				s.Appearance = platform.AppearanceDark
			}
		}
	}
	if s.Motion == platform.MotionUnknown {
		if v, ok := get(desktop, "enable-animations"); ok {
			s.Motion = motionOf(v != "false")
		}
	}
	if s.Contrast == platform.ContrastUnknown {
		if v, ok := get("org.gnome.desktop.a11y.interface", "high-contrast"); ok {
			s.Contrast = platform.ContrastNoPreference
			if v == "true" {
				s.Contrast = platform.ContrastMore
			}
		}
	}
}

// FromKDE reads into s what KDE's kdeglobals say: the colours it selects
// in, and where s does not know yet whether its windows are dark and
// whether it animates (an AnimationDurationFactor of 0 is none).
func FromKDE(s *System, kdeglobals string) {
	if back, ok := rgbOf(iniValue(kdeglobals, "Colors:Selection", "BackgroundNormal")); ok {
		s.Highlight.Background = back
		if text, said := rgbOf(iniValue(kdeglobals, "Colors:Selection", "ForegroundNormal")); said {
			s.Highlight.Text = text
		}
	}
	if s.Appearance == platform.AppearanceUnknown {
		if window, ok := rgbOf(iniValue(kdeglobals, "Colors:Window", "BackgroundNormal")); ok {
			s.Appearance = platform.AppearanceLight
			if luminance(window) < 0.18 {
				s.Appearance = platform.AppearanceDark
			}
		}
	}
	if s.Motion == platform.MotionUnknown {
		if v, ok := iniValue(kdeglobals, "KDE", "AnimationDurationFactor"); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				s.Motion = motionOf(f > 0)
			}
		}
	}
}

// FromGTK reads into s what a GTK settings.ini says where s does not know
// yet: a dark theme preferred or named, and whether it animates.
func FromGTK(s *System, settings string) {
	if s.Appearance == platform.AppearanceUnknown {
		prefer, _ := iniValue(settings, "Settings", "gtk-application-prefer-dark-theme")
		theme, _ := iniValue(settings, "Settings", "gtk-theme-name")
		if truthy(prefer) || darkTheme(theme) {
			s.Appearance = platform.AppearanceDark
		}
	}
	if s.Motion == platform.MotionUnknown {
		if v, ok := iniValue(settings, "Settings", "gtk-enable-animations"); ok {
			s.Motion = motionOf(truthy(v))
		}
	}
}

func motionOf(animates bool) platform.Motion {
	if animates {
		return platform.MotionFull
	}
	return platform.MotionReduced
}

func darkTheme(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, "-dark") || strings.HasSuffix(name, ":dark")
}

func truthy(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// iniValue is the value of key in section of an ini file, as KDE and GTK
// write theirs.
func iniValue(ini, section, key string) (string, bool) {
	in := false
	lines := bufio.NewScanner(strings.NewReader(ini))
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if strings.HasPrefix(line, "[") {
			in = line == "["+section+"]"
			continue
		}
		if k, v, ok := strings.Cut(line, "="); in && ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

// rgbOf reads KDE's r,g,b colour.
func rgbOf(v string, ok bool) (color.Color, bool) {
	parts := strings.Split(v, ",")
	if !ok || len(parts) < 3 {
		return color.Color{}, false
	}
	var ch [3]uint8
	for i := range ch {
		n, err := strconv.ParseUint(strings.TrimSpace(parts[i]), 10, 8)
		if err != nil {
			return color.Color{}, false
		}
		ch[i] = uint8(n)
	}
	return color.RGB8(ch[0], ch[1], ch[2]), true
}
