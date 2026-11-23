package style_test

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/style"
)

// The shades of accents Chrome drew on a hovered and a pressed range
// (TestAccentShadesAreChromes measures them live).
func TestAnAccentIsShadedAsChromes(t *testing.T) {
	for _, c := range []struct{ accent, hovered, pressed string }{
		{"#ee0044", "#b60034", "#ff2765"},
		{"#33aa77", "#4cc994", "#267f59"},
		{"#808080", "#646464", "#9c9c9c"},
		{"#8b8b8b", "#a7a7a7", "#6f6f6f"},
		{"#000000", "#000000", "#1c1c1c"},
		{"#ffffff", "#ffffff", "#e3e3e3"},
	} {
		a := color.Hex(c.accent)
		for _, s := range []struct {
			shade style.Shade
			want  string
		}{{style.ShadeHovered, c.hovered}, {style.ShadePressed, c.pressed}} {
			got, want := style.AccentShade(a, s.shade, style.LightRange(a)).SRGB(), color.Hex(s.want).SRGB()
			for i := range got {
				if d := (got[i] - want[i]) * 255; d > 1.01 || d < -1.01 {
					t.Errorf("%s shade %d: %v, want %s", c.accent, s.shade, color.FromSRGB(got).Hex(), s.want)
					break
				}
			}
		}
	}
	if a := color.Hex("#ee004480"); style.AccentShade(a, style.ShadeRest, false) != a ||
		style.AccentShade(a, style.ShadeHovered, false).A != a.A {
		t.Error("the shades keep the accent's alpha")
	}
}
