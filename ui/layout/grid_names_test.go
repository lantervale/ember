package layout //nolint:testpackage // resolves lines with the package's own types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func TestGridLinesResolveNamesAsTheSpecCountsThem(t *testing.T) {
	px := style.Track{Min: style.Px(40), Max: style.Px(40)}
	st := &style.Computed{}
	st.Columns = []style.TrackRepeat{
		{Count: 1, Tracks: []style.Track{px}, Before: []string{"a"}},
		{Count: 3, Tracks: []style.Track{px}, Names: [][]string{{"col"}, {"gap"}}},
		{Count: 1, Tracks: []style.Track{px}, Before: []string{"b"}, After: []string{"end"}},
	}
	st.Areas = []style.GridArea{{Name: "head", Row: [2]int{1, 2}, Column: [2]int{2, 4}}}
	g := newGridLines(st, horizontal, 0, 5)
	line := func(v int) ozPlacement { return ozPlacement{placeLine, v} }
	span := func(v int) ozPlacement { return ozPlacement{placeSpan, v} }
	name := func(n string, l int) style.GridLine { return style.GridLine{Name: n, Line: l} }
	spanOf := func(n string, s int) style.GridLine { return style.GridLine{Name: n, Span: s} }
	cases := map[string]struct {
		lines [2]style.GridLine
		want  ozLines
	}{
		"col 2":               {[2]style.GridLine{name("col", 2), {}}, ozLines{line(2), ozPlacement{}}},
		"col -1 / span gap 2": {[2]style.GridLine{name("col", -1), spanOf("gap", 2)}, ozLines{line(3), line(6)}},
		"span col 2 / b":      {[2]style.GridLine{spanOf("col", 2), name("b", 0)}, ozLines{line(2), line(4)}},
		"2 / span col":        {[2]style.GridLine{{Line: 2}, spanOf("col", 1)}, ozLines{line(1), line(2)}},
		"a / gap 2":           {[2]style.GridLine{name("a", 0), name("gap", 2)}, ozLines{line(0), line(3)}},
		"gap / end":           {[2]style.GridLine{name("gap", 0), name("end", 0)}, ozLines{line(2), line(5)}},
		"head":                {[2]style.GridLine{name("head", 0), name("head", 0)}, ozLines{line(1), line(3)}},
		"head-end / head-start": {[2]style.GridLine{name("head-end", 0), name("head-start", 0)},
			ozLines{line(3), line(1)}},
		"nope":              {[2]style.GridLine{name("nope", 0), {}}, ozLines{line(6), ozPlacement{}}},
		"nope -2 / nope 3":  {[2]style.GridLine{name("nope", -2), name("nope", 3)}, ozLines{line(-2), line(8)}},
		"span nope / 2":     {[2]style.GridLine{spanOf("nope", 1), {Line: 2}}, ozLines{line(-1), line(1)}},
		"1 / span nope 2":   {[2]style.GridLine{{Line: 1}, spanOf("nope", 2)}, ozLines{line(0), line(7)}},
		"span col / auto":   {[2]style.GridLine{spanOf("col", 3), {}}, ozLines{span(1), ozPlacement{}}},
		"span 2 / span col": {[2]style.GridLine{{Span: 2}, spanOf("col", 1)}, ozLines{span(2), ozPlacement{}}},
		"col 5":             {[2]style.GridLine{name("col", 5), {}}, ozLines{line(7), ozPlacement{}}},
		"-5 col":            {[2]style.GridLine{name("col", -5), {}}, ozLines{line(-2), ozPlacement{}}},
		"-1 / span 2":       {[2]style.GridLine{{Line: -1}, {Span: 2}}, ozLines{line(5), span(2)}},
	}
	for src, c := range cases {
		assert.Equal(t, c.want, g.placement(c.lines), src)
	}
}
