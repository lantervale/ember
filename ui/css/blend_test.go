package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestBlendModesAndIsolationAreRead(t *testing.T) {
	s := gridOf(t, "mix-blend-mode: color-dodge; isolation: isolate; background-blend-mode: multiply, luminosity")
	if s.BlendMode != style.BlendColorDodge || !s.Isolate {
		t.Errorf("mix-blend-mode %v, isolation %v", s.BlendMode, s.Isolate)
	}
	if !slices.Equal(s.BackgroundBlends, []style.BlendMode{style.BlendMultiply, style.BlendLuminosity}) {
		t.Errorf("background-blend-mode %v", s.BackgroundBlends)
	}
	if l := s.BackgroundLayer(3); l.Blend != style.BlendLuminosity {
		t.Errorf("layer 3 repeats the list: %v", l.Blend)
	}
	if !s.BackgroundBlended() {
		t.Error("a layer blends")
	}
	if d := gridOf(t, ""); d.BlendMode != style.BlendNormal || d.Isolate || d.BackgroundBlended() {
		t.Errorf("initial: %+v %v %v", d.BlendMode, d.Isolate, d.BackgroundBlends)
	}
}

func TestBackdropFiltersAreReadAsFilters(t *testing.T) {
	s := gridOf(t, "color: #00f; backdrop-filter: blur(4px) saturate(180%) drop-shadow(1px 2px)")
	want := []style.Filter{
		{Kind: style.Blur, Amount: 4},
		{Kind: style.Saturate, Amount: 1.8},
		{Kind: style.DropShadow, Shadow: style.Shadow{X: 1, Y: 2, Color: hex("#00f"), CurrentColor: true}},
	}
	if len(s.BackdropFilters) != len(want) {
		t.Fatalf("backdrop filters %+v", s.BackdropFilters)
	}
	for i := range want {
		if got := s.BackdropFilters[i]; got.Kind != want[i].Kind || !nearf(got.Amount, want[i].Amount) ||
			got.Shadow != want[i].Shadow {
			t.Errorf("filter %d: %+v, want %+v", i, got, want[i])
		}
	}
	if len(s.Filters) != 0 {
		t.Errorf("filter untouched: %+v", s.Filters)
	}
}
