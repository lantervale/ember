package css //nolint:testpackage // uses the unexported test helpers (el, page, warnings)

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestIndividualTransformsCompute(t *testing.T) {
	d := el(dom.Div, "d")
	d.SetStyle("font-size: 10px; translate: 50% 2em; rotate: 0 0 -3 0.25turn; scale: 150%")
	if warns := Compute(page(d), nil); len(warns) > 0 {
		t.Fatal(warnings(warns))
	}
	s := styleOf(d)
	if want := (style.Translation{X: style.Percent(50), Y: style.Px(20), Set: true}); s.Translate.X.Percent != 50 ||
		s.Translate.Y != want.Y || !s.Translate.Set {
		t.Errorf("translate: %+v, want %+v", s.Translate, want)
	}
	if s.Rotate != (style.Rotation{Deg: -90, Set: true}) {
		t.Errorf("rotate: %+v", s.Rotate)
	}
	if s.Scale != (style.Scaling{X: 1.5, Y: 1.5, Set: true}) {
		t.Errorf("scale: %+v", s.Scale)
	}
	if !s.Transformed() || len(s.Transform) > 0 {
		t.Errorf("transformed %v, transform %v", s.Transformed(), s.Transform)
	}
}

// TestThreeDIndividualTransformsAreRefused has a value moving the box
// out of the page refused, as a 3D transform function is, the one
// before it standing; one of no z left whole is kept.
func TestThreeDIndividualTransformsAreRefused(t *testing.T) {
	for _, tc := range []struct{ decl, refused string }{
		{"translate: 1px 2px 3px", ""}, {"translate: 1px 2px 1em", ""}, {"rotate: x 10deg", ""},
		{"rotate: 0 1 0 10deg", ""}, {"rotate: 1 1 1 10deg", ""}, {"rotate: 0 0 0 10deg", ""},
		{"scale: 1 1 2", ""}, {"scale: 2 2 50%", ""},
		{"rotate: 0", "angle"}, {"rotate: 10deg 10deg", "axis"}, {"rotate: w 10deg", "axis"},
		{"translate: 1px 2px 3%", "percentage"}, {"scale: 1px", "number"}, {"translate: 1px 2px 0 0", "three"},
	} {
		d := el(dom.Div, "d")
		d.SetStyle("translate: 4px; rotate: 4deg; scale: 4; " + tc.decl)
		got := warnings(Compute(page(d), nil))
		want := tc.refused
		if want == "" {
			want = "3D"
		}
		if !strings.Contains(got, want) {
			t.Errorf("%s: warnings %q, want one saying %q", tc.decl, got, want)
		}
		s := styleOf(d)
		if s.Translate.X != style.Px(4) || s.Rotate.Deg != 4 || s.Scale.X != 4 {
			t.Errorf("%s: %+v %+v %+v, want the values before it", tc.decl, s.Translate, s.Rotate, s.Scale)
		}
	}
	d := el(dom.Div, "d")
	d.SetStyle("translate: 1px 2px 0; rotate: 10deg 0 0 1; scale: 2 3 100%")
	if warns := Compute(page(d), nil); len(warns) > 0 {
		t.Fatal(warnings(warns))
	}
	if s := styleOf(d); s.Translate.Y != style.Px(2) || s.Rotate.Deg != 10 || s.Scale.Y != 3 {
		t.Errorf("%+v %+v %+v", s.Translate, s.Rotate, s.Scale)
	}
}
