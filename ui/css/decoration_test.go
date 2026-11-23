package css //nolint:testpackage // tests the unexported expansion and reads computed styles off dom

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDecorationParses(t *testing.T) {
	for _, src := range []string{
		"text-decoration: none",
		"text-decoration: underline",
		"text-decoration: underline overline line-through",
		"text-decoration: dotted underline",
		"text-decoration: underline #f00 2px",
		"text-decoration: line-through double currentcolor from-font",
		"text-decoration: red",
		"text-decoration: inherit",
		"text-decoration-line: overline underline",
		"text-decoration-style: dashed",
		"text-decoration-color: rgb(1 2 3 / 50%)",
		"text-decoration-thickness: auto",
		"text-decoration-thickness: 0.1em",
		"text-decoration-thickness: 10%",
		"text-underline-offset: auto",
		"text-underline-offset: -0.1em",
		"text-underline-offset: calc(2px + 10%)",
		"text-underline-position: auto",
		"text-underline-position: under",
		"text-underline-position: from-font",
		"text-underline-position: right under",
		"text-underline-position: from-font left",
	} {
		_, warns := Parse("t.css", "p { "+src+" }")
		assert.Empty(t, warns, src)
	}
}

func TestTextDecorationRefusesWhatItCannotDraw(t *testing.T) {
	for src, why := range map[string]string{
		"text-decoration: underline wavy":          "wavy",
		"text-decoration-style: wavy":              "wavy",
		"text-decoration-line: blink":              "blink",
		"text-decoration: underline underline":     "underline",
		"text-decoration: none underline":          "none",
		"text-decoration-line: underline none":     "none",
		"text-decoration: underline solid dotted":  "dotted",
		"text-decoration: underline 1px 2px":       "2px",
		"text-decoration-thickness: -1px":          "negative",
		"text-decoration-thickness: 2":             "unit",
		"text-decoration-line: ":                   "",
		"text-underline-offset: 2":                 `found "2"`,
		"text-underline-offset: from-font":         "from-font",
		"text-underline-position: under from-font": "second of from-font and under",
		"text-underline-position: left right":      "second of left and right",
		"text-underline-position: auto under":      "auto",
		"text-underline-position: over":            "over",
	} {
		_, warns := Parse("t.css", "p { "+src+" }")
		require.Len(t, warns, 1, src)
		assert.Contains(t, warns[0].Error(), why, src)
	}
}

func TestTextDecorationExpands(t *testing.T) {
	initials := map[string]string{
		"text-decoration-line": "none", "text-decoration-style": "solid",
		"text-decoration-color": "currentcolor", "text-decoration-thickness": "auto",
	}
	for src, set := range map[string]map[string]string{
		"none":      {},
		"underline": {"text-decoration-line": "underline"},
		"line-through underline": {
			"text-decoration-line": "line-through underline",
		},
		"dotted red underline 3px": {
			"text-decoration-line": "underline", "text-decoration-style": "dotted",
			"text-decoration-color": "red", "text-decoration-thickness": "3px",
		},
		"overline from-font double": {
			"text-decoration-line": "overline", "text-decoration-style": "double",
			"text-decoration-thickness": "from-font",
		},
		"#00f": {"text-decoration-color": "#00f"},
	} {
		got := expanded(t, "text-decoration", src)
		for name, initial := range initials {
			want := initial
			if v, ok := set[name]; ok {
				want = v
			}
			assert.Equal(t, want, got[name], "%s: %s", src, name)
		}
	}
}

// decorated computes a page of one div holding kids under src.
func decorated(t *testing.T, src string, kids ...*dom.Node) *dom.Node {
	t.Helper()
	root := page(with(el(dom.Div, "box"), kids...))
	Compute(root, []*Sheet{sheet(t, src)})
	return root
}

func TestTextDecorationComputes(t *testing.T) {
	sp := el(dom.Span, "s")
	decorated(t, `
#box { color: #f00; font-size: 20px; text-decoration: underline overline dashed; text-decoration-thickness: 0.1em; }
#s { text-decoration: line-through #00f 25%; }
`, sp)
	box := styleOf(sp.Parent()).TextDecoration
	assert.Equal(t, style.Underline|style.Overline, box.Line)
	assert.Equal(t, style.DecorationDashed, box.Style)
	assert.Equal(t, hex("#f00"), box.Color, "currentcolor is the element's colour")
	assert.Equal(t, style.Px(2), box.Thickness)

	s := styleOf(sp).TextDecoration
	assert.Equal(t, style.LineThrough, s.Line)
	assert.Equal(t, style.DecorationSolid, s.Style, "the shorthand resets what it leaves out")
	assert.Equal(t, hex("#00f"), s.Color)
	assert.Equal(t, style.Length{Kind: style.Fixed, Percent: 25, IsPercent: true}, s.Thickness)
}

func TestTextDecorationIsNotInheritedButInheritCopiesIt(t *testing.T) {
	plain, asked := el(dom.Span, "plain"), el(dom.Span, "asked")
	decorated(t, `
#box { text-decoration: underline dotted #0f0 3px; }
#asked { text-decoration: inherit; }
`, plain, asked)
	assert.Equal(t, style.TextDecoration{Color: styleOf(plain).Color}, styleOf(plain).TextDecoration)
	assert.Equal(t, styleOf(asked.Parent()).TextDecoration, styleOf(asked).TextDecoration)
}

func TestTextDecorationInitialIsNoLineInTheTextsColour(t *testing.T) {
	sp := el(dom.Span, "s")
	decorated(t, `#s { color: #00f; }`, sp)
	assert.Equal(t, style.TextDecoration{Color: hex("#00f")}, styleOf(sp).TextDecoration)
	assert.Empty(t, styleOf(sp).Decorations)
}

func TestADecorationIsDrawnOnItsDescendantsText(t *testing.T) {
	em := el(dom.Em, "em")
	sp := with(el(dom.Span, "s"), em)
	txt := dom.NewText("words")
	em.Append(txt)
	decorated(t, `
#box { color: #f00; font-size: 20px; text-decoration: underline; }
#s { color: #00f; font-size: 30px; }
#em { text-decoration: line-through 2px; }
`, sp)
	box := styleOf(sp.Parent())
	want := style.Decoration{TextDecoration: box.TextDecoration, Font: box.Font}
	assert.Equal(t, []style.Decoration{want}, box.Decorations)
	assert.Equal(t, []style.Decoration{want}, styleOf(sp).Decorations,
		"in the decorating box's colour and font, not the span's")

	own := style.Decoration{TextDecoration: styleOf(em).TextDecoration, Font: styleOf(em).Font}
	assert.Equal(t, hex("#00f"), own.Color)
	assert.Equal(t, []style.Decoration{want, own}, styleOf(em).Decorations, "the outer box's first, its own over it")
	assert.Same(t, styleOf(em), styleOf(txt), "a text node draws its parent's")
}

func TestABlockTakesADecorationInItsOwnFont(t *testing.T) {
	p := with(el(dom.P, "p"), el(dom.Span, "s"))
	decorated(t, `
#box { font-size: 20px; text-decoration: underline; }
#p { font-size: 30px; }
#s { font-size: 10px; }
`, p)
	box := styleOf(p.Parent())
	want := []style.Decoration{{TextDecoration: box.TextDecoration, Font: styleOf(p).Font}}
	assert.Equal(t, want, styleOf(p).Decorations, "its lines start anew: their root inline box decorates")
	assert.Equal(t, want, styleOf(p.Children()[0]).Decorations)
}

func TestFloatsAbsolutesAndAtomicInlinesStopDecorations(t *testing.T) {
	for name, decl := range map[string]string{
		"float":        "float: left",
		"absolute":     "position: absolute",
		"fixed":        "position: fixed",
		"inline-block": "display: inline-block",
		"inline-flex":  "display: inline-flex",
		"inline-grid":  "display: inline-grid",
	} {
		stop, inner := el(dom.Span, "stop"), el(dom.Span, "in")
		stop.Append(inner)
		decorated(t, "#box { text-decoration: underline; } #stop { "+decl+"; }", stop)
		assert.Empty(t, styleOf(stop).Decorations, name)
		assert.Empty(t, styleOf(inner).Decorations, "%s: nor what it holds", name)
	}
}

func TestWhatStopsDecorationsStillDrawsItsOwn(t *testing.T) {
	stop := el(dom.Span, "stop")
	decorated(t, "#box { text-decoration: underline; } #stop { float: left; text-decoration: overline; }", stop)
	require.Len(t, styleOf(stop).Decorations, 1)
	assert.Equal(t, style.Overline, styleOf(stop).Decorations[0].Line)
}

func TestRelativeAndInFlowBoxesKeepDecorations(t *testing.T) {
	for _, decl := range []string{"position: relative", "display: block", "display: flex", "position: sticky"} {
		kept := el(dom.Span, "kept")
		decorated(t, "#box { text-decoration: underline; } #kept { "+decl+"; }", kept)
		assert.Len(t, styleOf(kept).Decorations, 1, decl)
	}
}

func TestTextDecorationNoneDoesNotUndoAnAncestors(t *testing.T) {
	sp := el(dom.Span, "s")
	decorated(t, "#box { text-decoration: underline; } #s { text-decoration: none; }", sp)
	assert.Len(t, styleOf(sp).Decorations, 1)
}

func TestADecorationThatChangesRestylesDescendantsWithoutLayout(t *testing.T) {
	sp := el(dom.Span, "s")
	root := decorated(t, ".on { text-decoration: underline; }", sp)
	box := sp.Parent()
	clean := func(n *dom.Node) bool {
		dom.Data(n).Dirty = 0
		return true
	}
	root.Walk(clean)
	box.AddClass("on")
	Compute(root, []*Sheet{sheet(t, ".on { text-decoration: underline; }")})
	assert.Len(t, styleOf(sp).Decorations, 1, "the span is restyled though nothing it inherits changed")
	assert.Zero(t, dom.Data(box).Dirty&dom.DirtyLayout, "a decoration moves nothing")
	assert.Zero(t, dom.Data(sp).Dirty&dom.DirtyLayout)
}

func TestAnUnderlinesPlaceIsTheDecoratingBoxs(t *testing.T) {
	sp := el(dom.Span, "s")
	p := with(el(dom.P, "p"), sp)
	p.SetAttr("lang", "ja-JP")
	decorated(t, `
#box { font-size: 20px; text-decoration: underline; text-underline-offset: 10%; text-underline-position: under; }
#p { writing-mode: vertical-rl; text-decoration: overline; }
#s { text-underline-offset: 3px; text-underline-position: from-font right; }
`, p)
	box, ps := styleOf(p.Parent()), styleOf(p)
	assert.Equal(t, style.Length{Kind: style.Fixed, Percent: 10, IsPercent: true}, ps.UnderlineOffset,
		"inherited as a percentage")
	assert.Equal(t, style.UnderUnder, ps.UnderlinePosition)
	outer := style.Decoration{TextDecoration: box.TextDecoration, Font: ps.Font, Offset: box.UnderlineOffset,
		Position: style.UnderUnder}
	own := style.Decoration{TextDecoration: ps.TextDecoration, Font: ps.Font, Offset: ps.UnderlineOffset,
		Position: style.UnderUnder, RightInVertical: true}
	assert.Equal(t, []style.Decoration{outer, own}, styleOf(sp).Decorations,
		"each box's offset and position, whatever the span sets; Japanese vertical lines underlined on the right")
	assert.Equal(t, style.UnderFromFont|style.UnderRight, styleOf(sp).UnderlinePosition)
}
