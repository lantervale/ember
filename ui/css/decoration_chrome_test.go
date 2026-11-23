//go:build chrome

package css_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/style"
)

//nolint:gochecknoglobals // the properties compared, a test table
var decorationProps = map[string]func(s *style.Computed) string{
	"color": func(s *style.Computed) string { return rgb(s.Color) },
	"text-decoration-line": func(s *style.Computed) string {
		var lines []string
		for i, name := range []string{"underline", "overline", "line-through"} {
			if s.TextDecoration.Line&(1<<i) != 0 {
				lines = append(lines, name)
			}
		}
		if lines == nil {
			return "none"
		}
		return strings.Join(lines, " ")
	},
	"text-decoration-style": func(s *style.Computed) string {
		return []string{"solid", "double", "dotted", "dashed"}[s.TextDecoration.Style]
	},
	"text-decoration-color": func(s *style.Computed) string { return rgb(s.TextDecoration.Color) },
	"text-decoration-thickness": func(s *style.Computed) string {
		switch t := s.TextDecoration.Thickness; {
		case t.Kind == style.Auto:
			return "auto"
		case t.IsPercent:
			return num(t.Percent) + "%"
		default:
			return px(t.Px)
		}
	},
	"text-decoration-skip-ink": func(s *style.Computed) string { return [...]string{"auto", "none", "all"}[s.SkipInk] },
}

func TestTextDecorationMatchesChrome(t *testing.T) {
	fsys := fstest.MapFS{"decoration.css": {Data: []byte(`
body { color: #2b2722; font-size: 20px; }
.under { text-decoration: underline; }
.all { text-decoration: line-through overline underline; }
.styled { text-decoration: dotted underline #c8473d 0.1em; }
.percent { text-decoration: overline double 12.5%; }
.colour { text-decoration: rgb(10 20 30 / 50%); }
.longhands { text-decoration-line: overline underline; text-decoration-style: dashed;
  text-decoration-color: currentcolor; text-decoration-thickness: 3px; color: #3a6fd8; }
.reset { text-decoration: underline dashed red 2px; text-decoration: line-through; }
.inherit { text-decoration: inherit; }
.none { text-decoration: none; }
.float { float: left; }
.red { color: #c8473d; }
.skip-all { text-decoration-skip-ink: all; }
.skip-none { text-decoration-skip-ink: NONE; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "decoration.css"))
	d.Body().Append(
		ui.P(ui.Class("under"), ui.Text("underlined "), ui.Span(ui.Text("and its span, which sets none")),
			ui.Span(ui.Class("none"), ui.Text("none")), ui.Span(ui.Class("float"), ui.Text("a float"))),
		ui.P(ui.Class("all"), ui.Text("all three")),
		ui.P(ui.Class("styled"), ui.Span(ui.Class("inherit", "red"), ui.Text("inherit keeps the used colour"))),
		ui.P(ui.Class("percent"), ui.Text("a percentage")),
		ui.P(ui.Class("colour"), ui.Text("a colour alone")),
		ui.P(ui.Class("longhands"), ui.Span(ui.Class("red"), ui.Text("currentcolor is each element's own"))),
		ui.P(ui.Class("reset"), ui.Text("the shorthand resets the rest")),
		ui.P(ui.Style("text-decoration: overline #00f"), ui.Text("a style attribute")),
		ui.P(ui.Class("under", "skip-all"), ui.Text("skips all ink "), ui.Span(ui.Text("inherited")),
			ui.Span(ui.Class("skip-none"), ui.Text("or none"))),
	)
	matchChromeProps(t, d, decorationProps, 0)
}
