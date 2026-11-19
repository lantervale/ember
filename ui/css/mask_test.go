package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestAMaskIsReadAsABackgroundImageIs(t *testing.T) {
	s := gridOf(t, "mask-image: url(https://x/i.svg); mask-size: 12px 50%; mask-repeat: no-repeat; "+
		"mask-position: right 2px top 50%")
	if s.MaskImage.Source != "https://x/i.svg" {
		t.Errorf("mask-image %q", s.MaskImage.Source)
	}
	if s.MaskSize.Width != style.Px(12) || s.MaskSize.Height.Percent != 50 {
		t.Errorf("mask-size %v, want 12px 50%%", s.MaskSize)
	}
	if !s.MaskNoRepeat {
		t.Error("mask-repeat: no-repeat is not read")
	}
	if p := s.MaskPosition; p.X != style.Px(2) || !p.FromRight || p.Y.Percent != 50 || p.FromBottom {
		t.Errorf("mask-position %v, want 2px from the right, 50%%", p)
	}

	s = gridOf(t, "mask-image: url(a.png); mask-repeat: repeat-y; mask-image: none")
	if s.MaskImage.Source != "" || s.MaskNoRepeat || s.MaskRepeat != style.RepeatY {
		t.Errorf("none, repeat-y: %q %v %v", s.MaskImage.Source, s.MaskNoRepeat, s.MaskRepeat)
	}
	if s.BackgroundImage.Source != "" || s.BackgroundRepeat != style.RepeatBoth {
		t.Error("a mask is no background")
	}
}

func TestWhatAMaskDoesNotDrawIsRefused(t *testing.T) {
	for _, decl := range []string{
		"mask-image: linear-gradient(#000, transparent)",
		"mask-image: url(a.png), url(b.png)",
		"mask: url(a.png) center / 12px no-repeat",
		"mask-repeat: space",
		"mask-position: right 2px center",
	} {
		_, warns := Parse("t.css", "div { "+decl+" }")
		if len(warns) != 1 {
			t.Errorf("%s: warnings %v, want one", decl, warns)
		}
	}
	_, warns := Parse("t.css", "div { mask-image: linear-gradient(#000, #fff) }")
	if len(warns) != 1 || !strings.Contains(warns[0].Error(), "none or one url()") {
		t.Errorf("a gradient mask: %v", warns)
	}
}
