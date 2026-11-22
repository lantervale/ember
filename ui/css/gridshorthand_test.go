package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestGridShorthand(t *testing.T) {
	px := func(v float32) style.Track { return style.Track{Min: style.Px(v), Max: style.Px(v)} }
	one := func(ts ...style.Track) []style.TrackRepeat {
		var out []style.TrackRepeat
		for _, tr := range ts {
			out = append(out, style.TrackRepeat{Count: 1, Tracks: []style.Track{tr}})
		}
		return out
	}
	auto := []style.Track{{}}
	const before = "grid-auto-rows: 7px; grid-auto-columns: 8px; grid-auto-flow: column; gap: 5px; " +
		"grid-template-areas: 'q'; "

	s := gridOf(t, before+"grid: 10px 20px / 30px")
	assert.Equal(t, one(px(10), px(20)), s.Rows)
	assert.Equal(t, one(px(30)), s.Columns)
	assert.Nil(t, s.Areas)
	assert.Equal(t, auto, s.AutoRows, "the implicit tracks go back to auto")
	assert.Equal(t, auto, s.AutoColumns)
	assert.Equal(t, style.FlowRow, s.Flow)
	assert.Equal(t, style.Px(5), s.Gap.Width, "the gaps stay")

	s = gridOf(t, before+"grid: 'a b' 40px 'c d' / 1fr 2fr")
	assert.Len(t, s.Areas, 4)
	assert.Len(t, s.Rows, 2)
	assert.Len(t, s.Columns, 2)

	s = gridOf(t, before+"grid: 10px / auto-flow 30px 40px")
	assert.Equal(t, one(px(10)), s.Rows)
	assert.Nil(t, s.Columns)
	assert.Equal(t, []style.Track{px(30), px(40)}, s.AutoColumns)
	assert.Equal(t, auto, s.AutoRows)
	assert.Equal(t, style.FlowColumn, s.Flow)

	s = gridOf(t, before+"grid: 10px / dense auto-flow")
	assert.Equal(t, style.FlowColumnDense, s.Flow)
	assert.Equal(t, auto, s.AutoColumns)

	s = gridOf(t, before+"grid: auto-flow dense 50px / 60px 70px")
	assert.Equal(t, style.FlowRowDense, s.Flow)
	assert.Equal(t, []style.Track{px(50)}, s.AutoRows)
	assert.Equal(t, auto, s.AutoColumns)
	assert.Equal(t, one(px(60), px(70)), s.Columns)
	assert.Nil(t, s.Rows)

	s = gridOf(t, before+"grid: auto-flow / [a] repeat(2, 1fr) [b]")
	assert.Equal(t, style.FlowRow, s.Flow)
	assert.Len(t, s.Columns, 1)
	assert.Equal(t, []string{"a"}, s.Columns[0].Before)

	s = gridOf(t, before+"grid: none")
	assert.Nil(t, s.Rows)
	assert.Nil(t, s.Columns)
	assert.Equal(t, auto, s.AutoColumns)
	assert.Equal(t, style.FlowRow, s.Flow)

	s = gridOf(t, "grid: subgrid [a] / auto-flow 20px")
	assert.True(t, s.Subgrid.Height)
	assert.False(t, s.Subgrid.Width)
	assert.Equal(t, []style.TrackRepeat{{Count: 1, Names: [][]string{{"a"}}}}, s.Rows)

	for _, src := range []string{
		"10px", "auto-flow", "auto-flow / auto-flow", "dense 10px / 20px", "auto-flow auto-flow / 1fr",
		"10px / auto-flow dense dense", "10px / auto-flow 1fr / 2fr", "auto-flow 10px", "auto-flow 'a' / 10px",
		"10px / 20px auto-flow",
	} {
		_, warns := Parse("t.css", "div { grid: "+src+" }")
		assert.Len(t, warns, 1, src)
	}
}

func TestSubgridTemplates(t *testing.T) {
	s := gridOf(t, "grid-template-columns: subgrid")
	assert.True(t, s.Subgrid.Width)
	assert.Nil(t, s.Columns)

	s = gridOf(t, "grid-template-rows: subgrid [a] [] [b c] repeat(2, [d] [e]) repeat(auto-fill, [f])")
	assert.True(t, s.Subgrid.Height)
	assert.Equal(t, []style.TrackRepeat{
		{Count: 1, Names: [][]string{{"a"}}},
		{Count: 1, Names: [][]string{nil}},
		{Count: 1, Names: [][]string{{"b", "c"}}},
		{Count: 2, Names: [][]string{{"d"}, {"e"}}},
		{Count: style.AutoFill, Names: [][]string{{"f"}}},
	}, s.Rows)

	s = gridOf(t, "grid-template-columns: subgrid [a]; grid-template-columns: 10px")
	assert.False(t, s.Subgrid.Width, "a later template is no subgrid")

	s = gridOf(t, "grid-template: subgrid [top] / subgrid [l] [r]")
	assert.True(t, s.Subgrid.Width)
	assert.True(t, s.Subgrid.Height)
	assert.Len(t, s.Columns, 2)

	for _, src := range []string{
		"subgrid 10px", "subgrid [a] 1fr", "subgrid repeat(2, 10px)", "subgrid repeat(0, [a])",
		"subgrid repeat(auto-fit, [a])", "subgrid repeat(auto-fill, [a]) repeat(auto-fill, [b])", "subgrid [span]",
		"subgrid subgrid", "10px subgrid", "subgrid repeat(2)",
	} {
		_, warns := Parse("t.css", "div { grid-template-columns: "+src+" }")
		assert.Len(t, warns, 1, src)
	}
}
