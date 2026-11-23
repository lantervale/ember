//go:build chrome

package css_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

// TestEnvMatchesChrome: a desktop page has no title bar area of its own
// (Chrome without the window controls overlay), so env(titlebar-area-*)
// takes its fallback or is invalid at computed-value time; the safe area
// and keyboard insets are 0px; a name nothing defines, or one in other
// case, takes its fallback.
func TestEnvMatchesChrome(t *testing.T) {
	vals := []string{"env(titlebar-area-width)", "env(titlebar-area-width, 50px)", "env(titlebar-area-x, 7px)",
		"env(titlebar-area-height, 8px)", "env(nonsense, 12px)", "env(nonsense)", "env(safe-area-inset-top, 9px)",
		"env(safe-area-inset-right)", "calc(env(safe-area-inset-left) + 5px)", "env(keyboard-inset-width, 6px)",
		"env(titlebar-area-width,)", "calc(env(titlebar-area-y, 3px) * 2)", "env(keyboard-inset-height, 2px) 4px",
		"env(KEYBOARD-INSET-TOP, 1px)"}
	src := "div { margin-left: 2px; padding-left: 3px }\n"
	d := ui.NewDocument()
	for i, v := range vals {
		src += fmt.Sprintf("#d%d { margin-left: %s; padding-left: %s }\n", i, v, v)
		d.Body().Append(ui.Div(ui.ID(fmt.Sprint("d", i))))
	}
	d.AddSheet(src)
	matchChromeProps(t, d, map[string]func(*style.Computed) string{
		"margin-left":  func(s *style.Computed) string { return length(s.Margin.Left) },
		"padding-left": func(s *style.Computed) string { return length(s.Padding.Left) },
	}, 2)
}
