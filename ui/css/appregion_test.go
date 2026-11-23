package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAppRegionDragHoldsWhatIsInItUntilNoDragTakesItBack(t *testing.T) {
	inner, back, deep := el(dom.Span, "inner"), el(dom.Div, "back"), el(dom.Span, "deep")
	back.Append(deep)
	bar := el(dom.Div, "bar")
	bar.Append(inner, back)
	plain := el(dom.Div, "plain")
	root := page(bar, plain)
	Compute(root, []*Sheet{sheet(t, `#bar { app-region: drag } #back { app-region: no-drag }`)})
	if styleOf(bar).AppRegion != style.AppRegionDrag || styleOf(inner).AppRegion != style.AppRegionDrag {
		t.Error("drag, and what is in it with it")
	}
	if styleOf(back).AppRegion != style.AppRegionNoDrag || styleOf(deep).AppRegion != style.AppRegionNoDrag {
		t.Error("no-drag takes its box out, and what is in it")
	}
	if styleOf(plain).AppRegion != style.AppRegionNone {
		t.Error("none is the initial value")
	}
}

func TestAppRegionIsReadPlainAndAsWebkits(t *testing.T) {
	_, warns := Parse("t.css", "p { app-region: drag; app-region: no-drag; app-region: none }")
	if len(warns) != 0 {
		t.Errorf("app-region: %s", warnings(warns))
	}
	if _, warns = Parse("t.css", "p { app-region: move }"); len(warns) != 1 {
		t.Errorf("app-region: move gave %d warnings, want 1", len(warns))
	}
	s, warns := Parse("t.css", "p { -webkit-app-region: drag }", Browser())
	if len(warns) != 0 || len(s.Rules) != 1 || s.Rules[0].Decls[0].Name != "app-region" {
		t.Errorf("-webkit-app-region: %s", warnings(warns))
	}
}
