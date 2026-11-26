package layout //nolint:testpackage // tests the bidi pieces of inline layout

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func rtl(edit func(*style.Computed)) func(*style.Computed) {
	return func(st *style.Computed) {
		if edit != nil {
			edit(st)
		}
		st.Dir = style.RTL
	}
}

func inlineEl(children ...*dom.Node) *dom.Node { return el(dom.Span, nil, children...) }

func TestABreakEndsItsLineOnTheLineEndSide(t *testing.T) {
	br := el(dom.Br, nil)
	p := el(dom.P, rtl(width(100)), inlineEl(text("HHH")), br, inlineEl(text("HH")))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(70), tr.Of[br].Pos.X, "left of the first line's text, at its end")
}

func TestARightToLeftItemsMarkerStandsOnItsRight(t *testing.T) {
	a := el(dom.Li, rtl(listItem(style.ListDisc)), text("HH"))
	b := el(dom.Li, rtl(listItem(style.ListDisc)), text("HH"))
	dom.Data(b).MarkerText, dom.Data(b).HasMarkerText = "12. ", true
	tr := lay(el(dom.Div, width(100), el(dom.Ol, rtl(func(st *style.Computed) {
		st.Display = style.Block
		st.Padding.Right = style.Px(40)
	}), a, b)), 200)
	disc := markerOfBox(tr.Of[a])
	assert.Greater(t, disc.Pos.X, float32(60), "past the content's right")
	assert.Less(t, disc.Pos.X+disc.Size.X, float32(100), "within the list's padding")
	num := markerOfBox(tr.Of[b])
	assert.Equal(t, float32(60), num.Pos.X, "a number starts where the content ends")
	assert.Equal(t, " .12", num.Lines[0].Text, "the number in a right-to-left line")
}

func TestVisualText(t *testing.T) {
	assert.Equal(t, "abc", visualText("abc", 0))
	assert.Equal(t, "גבא", visualText("אבג", 0))
	assert.Equal(t, "12 בא", visualText("אב 12", 1))
	assert.Equal(t, "(א)", visualText("(א)", 1), "brackets mirrored")
}

func TestABlockOverridingBidiShowsItsTextInItsDirection(t *testing.T) {
	override := func(dir style.Direction) func(*style.Computed) {
		return func(st *style.Computed) {
			st.Dir = dir
			st.Bidi = style.BidiOverride
			st.Display = style.Block
		}
	}
	shown := func(b *box.Box) []string {
		var out []string
		for _, c := range b.Children {
			for _, f := range c.Lines {
				out = append(out, f.Shown())
			}
		}
		return out
	}
	rtlP := el(dom.P, override(style.RTL), text("txeT relliF"), el(dom.Br, nil), text("dc"))
	ltrP := el(dom.P, override(style.LTR), text("ab אבג"))
	tr := lay(el(dom.Div, block, rtlP, ltrP), 400)
	assert.Equal(t, []string{"Filler Text", "cd"}, shown(tr.Of[rtlP]), "each line, past the break too")
	assert.Equal(t, []string{"ab אבג"}, shown(tr.Of[ltrP]), "Hebrew left to right, as written")
}
