package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func listItem(ls style.ListStyle) func(*style.Computed) {
	return func(st *style.Computed) { st.Display, st.ListItem, st.ListStyle = style.Block, true, ls }
}

func markerOfBox(b *box.Box) *box.Box {
	for _, c := range b.Children {
		if c.Kind == box.Marker {
			return c
		}
	}
	return nil
}

func TestAListItemsMarkerStandsOutsideItBesideItsFirstLine(t *testing.T) {
	list := func(st *style.Computed) { st.Display = style.Block; st.Padding.Left = style.Px(40) }
	a := el(dom.Li, listItem(style.ListDisc), text("HH HH"))
	b := el(dom.Li, listItem(style.ListDecimal), text("HH"))
	c := el(dom.Li, listItem(style.ListDecimal), el(dom.P, block, text("HH")))
	none := el(dom.Li, listItem(style.ListNone), text("HH"))
	ol := el(dom.Ol, list, a, b, c, none)
	dom.Data(b).Ordinal, dom.Data(c).Ordinal = 8, 9
	tr := lay(el(dom.Div, width(100), ol), 200)

	assert.Equal(t, geom.Rect{X: 40, W: 60, H: 10}, tr.Of[a].Rect(), "the marker takes no room in the item")
	disc := markerOfBox(tr.Of[a])
	require.NotNil(t, disc)
	assert.Less(t, disc.Pos.X+disc.Size.X, float32(0), "a bullet stands before the item's content")
	assert.Greater(t, disc.Pos.X, float32(-40), "within the list's padding")

	num := markerOfBox(tr.Of[b])
	require.NotNil(t, num)
	require.Len(t, num.Lines, 1)
	assert.Equal(t, "8. ", num.Lines[0].Text, "the number css counted for the item")
	assert.Equal(t, float32(0), num.Pos.X+num.Size.X, "a number ends where the content starts")
	assert.Equal(t, float32(-30), num.Pos.X, "three glyphs wide")

	inner := markerOfBox(tr.Of[c])
	require.NotNil(t, inner, "the first line may be a block's inside the item")
	assert.Equal(t, "9. ", inner.Lines[0].Text)
	assert.Nil(t, markerOfBox(tr.Of[none]), "list-style-type: none draws no marker")
}

func TestABrBreaksTheLine(t *testing.T) {
	br := el(dom.Br, nil)
	p := el(dom.P, block, text("HH"), br, text("HHH HH"))
	tr := lay(el(dom.Div, width(100), p), 200)
	assert.Equal(t, []fragmentAt{{"HH", 0, 0}, {"HHH HH", 0, 10}}, fragments(tr.Of[p]))
	assert.Equal(t, float32(20), tr.Of[p].Size.Y)
	assert.Equal(t, geom.Rect{X: 20, W: 0, H: 10}, tr.Of[br].Rect(), "the br is at the end of its line, no wider than nothing")

	two := el(dom.P, block, text("HH"), el(dom.Br, nil), el(dom.Br, nil), text("HH"))
	tr = lay(el(dom.Div, width(100), two), 200)
	assert.Equal(t, float32(30), tr.Of[two].Size.Y, "two in a row leave an empty line")
}

func TestAFieldsetsLegendStandsOnItsTopBorder(t *testing.T) {
	legend := el(dom.Legend, func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Width = style.Length{Kind: style.FitContent}
	}, text("HHH"))
	p := el(dom.P, func(st *style.Computed) { st.Display = style.Block; st.Margin.Top = style.Px(6) }, text("HH"))
	set := el(dom.Fieldset, func(st *style.Computed) {
		st.Display, st.FlowRoot = style.Block, true
		st.Border = style.Sides[float32]{Top: 2, Right: 2, Bottom: 2, Left: 2}
		st.Padding = style.Sides[style.Length]{Top: style.Px(4), Right: style.Px(8), Bottom: style.Px(5), Left: style.Px(8)}
	}, legend, p)
	tr := lay(el(dom.Div, width(100), set), 200)

	assert.Equal(t, geom.Rect{X: 10, Y: 0, W: 30, H: 10}, tr.Of[legend].Rect(), "from the fieldset's top, as wide as its text")
	assert.Equal(t, geom.Rect{X: 10, Y: 20, W: 80, H: 10}, tr.Of[p].Rect(), "below the legend, the padding and its own margin")
	assert.Equal(t, float32(37), tr.Of[set].Size.Y)
	assert.Equal(t, float32(4), tr.Of[set].Wrap.Top, "the border is drawn through the legend's middle")

	bare := el(dom.Fieldset, func(st *style.Computed) {
		st.Display, st.FlowRoot = style.Block, true
		st.Border = style.Sides[float32]{Top: 2, Right: 2, Bottom: 2, Left: 2}
	}, el(dom.P, block, text("HH")))
	tr = lay(el(dom.Div, width(100), bare), 200)
	assert.Equal(t, float32(14), tr.Of[bare].Size.Y, "without a legend, a plain bordered block")
	assert.Equal(t, float32(0), tr.Of[bare].Wrap.Top)
}

func TestSubAndSuperShiftTheirTextFromTheBaseline(t *testing.T) {
	baselines := func(b *box.Box) map[string]float32 {
		out := map[string]float32{}
		var walk func(b *box.Box, y float32)
		walk = func(b *box.Box, y float32) {
			for _, f := range b.Lines {
				out[f.Text] = y + f.Baseline
			}
			for _, c := range b.Children {
				walk(c, y+c.Pos.Y)
			}
		}
		walk(b, 0)
		return out
	}
	plain := el(dom.P, block, text("HH"))
	tr := lay(el(dom.Div, width(200), plain), 200)
	base := baselines(tr.Of[plain])["HH"]

	sub := el(dom.Sub, func(st *style.Computed) { st.VerticalAlign = style.AlignSub }, text("HHH"))
	sup := el(dom.Sup, func(st *style.Computed) { st.VerticalAlign = style.AlignSuper }, text("HHHH"))
	p := el(dom.P, block, text("HH"), sub, sup)
	tr = lay(el(dom.Div, width(200), p), 200)
	got := baselines(tr.Of[p])
	// a fifth of the parent's 10px font and one more below, a third and one more above
	assert.InDelta(t, 3, got["HHH"]-got["HH"], 0.001)
	assert.InDelta(t, -(10.0/3 + 1), got["HHHH"]-got["HH"], 0.001)
	assert.InDelta(t, 10.0/3+1, got["HH"]-base, 0.001, "the line grows above for the sup")
	assert.InDelta(t, 10+3+10.0/3+1, tr.Of[p].Size.Y, 0.001, "and below for the sub")
	assert.InDelta(t, tr.Of[sup].Pos.Y+3+10.0/3+1, tr.Of[sub].Pos.Y, 0.001, "their boxes move with their text")
}
