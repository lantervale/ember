package css //nolint:testpackage // reads computed styles off dom with the package's test helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestTrackListsNameTheirLines(t *testing.T) {
	px := func(v float32) style.Track { return style.Track{Min: style.Px(v), Max: style.Px(v)} }
	s := gridOf(t, "grid-template-columns: [full-start] 10px [content-start] 20px [content-end b] 30px [full-end]")
	assert.Equal(t, []style.TrackRepeat{
		{Count: 1, Tracks: []style.Track{px(10)}, Before: []string{"full-start"}},
		{Count: 1, Tracks: []style.Track{px(20)}, Before: []string{"content-start"}},
		{Count: 1, Tracks: []style.Track{px(30)}, Before: []string{"content-end", "b"}, After: []string{"full-end"}},
	}, s.Columns)

	s = gridOf(t, "grid-template-rows: [a] repeat(2, [col] 40px 50px [gap]) [Z] 60px")
	assert.Equal(t, []style.TrackRepeat{
		{Count: 2, Tracks: []style.Track{px(40), px(50)}, Names: [][]string{{"col"}, nil, {"gap"}}, Before: []string{"a"}},
		{Count: 1, Tracks: []style.Track{px(60)}, Before: []string{"Z"}},
	}, s.Rows)

	s = gridOf(t, "grid-template-columns: repeat(auto-fill, [c] 80px) [last] 10px")
	assert.Equal(t, []style.TrackRepeat{
		{Count: style.AutoFill, Tracks: []style.Track{px(80)}, Names: [][]string{{"c"}, nil}},
		{Count: 1, Tracks: []style.Track{px(10)}, Before: []string{"last"}},
	}, s.Columns)

	s = gridOf(t, "grid-template-columns: 10px repeat(2, 20px)")
	assert.Nil(t, s.Columns[1].Names, "a repeat naming nothing")

	for _, src := range []string{
		"[a] [b] 10px", "[span] 10px", "[auto] 10px", "[1] 10px", "[a, b] 10px", "[a]", "[inherit] 10px",
		"repeat(2, [a] [b] 10px)", "repeat(2, [a])", "10px [a] [b]", "[a] [b]",
	} {
		_, warns := Parse("t.css", "div { grid-template-columns: "+src+" }")
		assert.Len(t, warns, 1, src)
	}
}

func TestGridLinesByName(t *testing.T) {
	cases := map[string]style.GridLine{
		"a":        {Name: "a"},
		"a 2":      {Name: "a", Line: 2},
		"-1 a":     {Name: "a", Line: -1},
		"span a":   {Name: "a", Span: 1},
		"span a 3": {Name: "a", Span: 3},
		"2 span b": {Name: "b", Span: 2},
		"span 2":   {Span: 2},
		"-3":       {Line: -3},
	}
	for src, want := range cases {
		s := gridOf(t, "grid-column-start: "+src)
		assert.Equal(t, want, s.Column[0], src)
	}
	for _, src := range []string{
		"a b", "span", "0 a", "span -1 a", "span 0", "auto a", "a 2 3", "span a b", "span span", "1 2",
	} {
		_, warns := Parse("t.css", "div { grid-row-end: "+src+" }")
		assert.Len(t, warns, 1, src)
	}
	s := gridOf(t, "grid-column: a 2 / span b")
	assert.Equal(t, [2]style.GridLine{{Name: "a", Line: 2}, {Name: "b", Span: 1}}, s.Column)
	s = gridOf(t, "grid-row: a 2")
	assert.Equal(t, [2]style.GridLine{{Name: "a", Line: 2}, {}}, s.Row)
	s = gridOf(t, "grid-column: full")
	assert.Equal(t, [2]style.GridLine{{Name: "full"}, {Name: "full"}}, s.Column, "a name alone is both ends")
	s = gridOf(t, "grid-area: c 2 / d")
	assert.Equal(t, [2]style.GridLine{{Name: "c", Line: 2}, {}}, s.Row)
	assert.Equal(t, [2]style.GridLine{{Name: "d"}, {Name: "d"}}, s.Column)
}

func TestGridTemplateNamesItsLines(t *testing.T) {
	px := func(v float32) style.Track { return style.Track{Min: style.Px(v), Max: style.Px(v)} }
	auto := style.Track{Min: style.Length{Kind: style.Auto}, Max: style.Length{Kind: style.Auto}}
	s := gridOf(t, "grid-template: [top] 'a a' 40px [mid] [x] 'b c' [bottom end] / [l] 100px [r]")
	assert.Equal(t, []style.TrackRepeat{
		{Count: 1, Tracks: []style.Track{px(40)}, Before: []string{"top"}},
		{Count: 1, Tracks: []style.Track{auto}, Before: []string{"mid", "x"}, After: []string{"bottom", "end"}},
	}, s.Rows)
	assert.Equal(t, []style.TrackRepeat{
		{Count: 1, Tracks: []style.Track{px(100)}, Before: []string{"l"}, After: []string{"r"}},
	}, s.Columns)
	assert.Len(t, s.Areas, 3)

	s = gridOf(t, "grid-template: [r1] 30px [r2] / [c1] 100px [c2]")
	assert.Equal(t, []string{"r2"}, s.Rows[0].After)
	assert.Equal(t, []string{"c1"}, s.Columns[0].Before)

	for _, src := range []string{"[a] [b] 'x'", "'x' [a] [b] [c] 'y'", "'x' 10px 20px", "[a] 10px 'x'", "'x' [a] 10px"} {
		_, warns := Parse("t.css", "div { grid-template: "+src+" }")
		assert.Len(t, warns, 1, src)
	}
}
