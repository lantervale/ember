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
		case t.Kind == style.None:
			return "from-font"
		case t.IsPercent:
			return num(t.Percent) + "%"
		default:
			return px(t.Px)
		}
	},
	"text-decoration-skip-ink": func(s *style.Computed) string { return [...]string{"auto", "none", "all"}[s.SkipInk] },
	"text-underline-offset":    func(s *style.Computed) string { return lengthCSS(s.UnderlineOffset) },
	"text-underline-position": func(s *style.Computed) string {
		var words []string
		for _, w := range []struct {
			bit  style.UnderlinePosition
			name string
		}{{style.UnderFromFont, "from-font"}, {style.UnderUnder, "under"}, {style.UnderLeft, "left"},
			{style.UnderRight, "right"}} {
			if s.UnderlinePosition&w.bit != 0 {
				words = append(words, w.name)
			}
		}
		if words == nil {
			return "auto"
		}
		return strings.Join(words, " ")
	},
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

func TestUnderlinePlacementMatchesChrome(t *testing.T) {
	fsys := fstest.MapFS{"underline.css": {Data: []byte(`
body { font-size: 20px; text-decoration: underline; }
.offset { text-underline-offset: 3px; }
.em { text-underline-offset: 0.25em; font-size: 16px; }
.percent { text-underline-offset: 15%; }
.negative { text-underline-offset: -2px; }
.calc { text-underline-offset: calc(1em + 10%); }
.auto { text-underline-offset: AUTO; }
.under { text-underline-position: under; }
.font { text-underline-position: from-font; }
.left { text-underline-position: left; }
.under-right { text-underline-position: right under; }
.font-left { text-underline-position: from-font left; }
.thick { text-decoration-thickness: from-font; }
.big { font-size: 40px; }
`)}}
	d := ui.NewDocument(ui.Stylesheet(fsys, "underline.css"))
	d.Body().Append(
		ui.P(ui.Class("offset"), ui.Text("3px "), ui.Span(ui.Class("big"), ui.Text("inherited")),
			ui.Span(ui.Class("auto"), ui.Text("auto"))),
		ui.P(ui.Class("em"), ui.Text("em "), ui.Span(ui.Class("big"), ui.Text("computed where set"))),
		ui.P(ui.Class("percent"), ui.Text("15% "), ui.Span(ui.Class("big"), ui.Text("inherited as a percentage"))),
		ui.P(ui.Class("negative"), ui.Text("negative")),
		ui.P(ui.Class("calc"), ui.Text("calc")),
		ui.P(ui.Class("under"), ui.Text("under "), ui.Span(ui.Class("left"), ui.Text("left"))),
		ui.P(ui.Class("font"), ui.Text("from-font "), ui.Span(ui.Text("inherited"))),
		ui.P(ui.Class("under-right"), ui.Text("right under")),
		ui.P(ui.Class("font-left"), ui.Text("from-font left")),
		ui.P(ui.Class("thick"), ui.Text("from-font thickness")),
	)
	matchChromeProps(t, d, decorationProps, 0)
}
