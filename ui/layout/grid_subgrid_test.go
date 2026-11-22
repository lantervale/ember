package layout //nolint:testpackage // tests the subgrid helpers and the kept layout

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestSubgridLinesTakeTheGridsNamesAndTheirOwn(t *testing.T) {
	st := &style.Computed{}
	st.Subgrid.Width = true
	st.Columns = []style.TrackRepeat{
		{Count: 1, Names: [][]string{{"a"}}},
		{Count: style.AutoFill, Names: [][]string{{"r"}, {"s"}}},
		{Count: 1, Names: [][]string{{"z"}}},
	}
	st.Areas = []style.GridArea{{Name: "p", Row: [2]int{1, 2}, Column: [2]int{2, 3}}}
	g := subgridLines(st, horizontal, &subgridAxis{span: 5, names: map[string][]int{"x": {0, 5}, "a": {3}}})
	assert.Equal(t, map[string][]int{
		"x": {0, 5}, "a": {0, 3}, "r": {1, 3}, "s": {2, 4}, "z": {5}, "p-start": {1}, "p-end": {2},
	}, g.named)

	parent := gridLines{explicit: 4, named: map[string][]int{"c": {1, 3}, "d": {0}, "e": {4}}}
	assert.Equal(t, map[string][]int{"c": {0, 2}, "e": {3}}, parent.within(lineSpan{1, 4}))
}

func TestItemsPastASubgridAreClamped(t *testing.T) {
	cases := map[lineSpan]lineSpan{
		{0, 1}: {0, 1}, {2, 5}: {2, 3}, {4, 6}: {2, 3}, {-3, -1}: {0, 1}, {-1, 2}: {0, 2}, {1, 3}: {1, 3},
	}
	for in, want := range cases {
		assert.Equal(t, want, clampToSubgrid(in, 3), in)
	}
}

func TestASubgridsTracksAreItsGridsWithinItsEdges(t *testing.T) {
	sizes := func(tracks []gridTrack) []float32 {
		var out []float32
		for _, tr := range tracks {
			out = append(out, tr.min.Px)
		}
		return out
	}
	sub := &subgridAxis{span: 3, gap: 10, tracks: []span{{0, 40}, {50, 80}, {90, 150}}}
	assert.Equal(t, []float32{0, 40, 10, 30, 10, 60, 0}, sizes(subgridTracks(sub, span{0, 150}, 10)))
	assert.Equal(t, []float32{0, 35, 10, 30, 10, 45, 0}, sizes(subgridTracks(sub, span{5, 135}, 10)),
		"the first and last start and end within its content box")
	assert.Equal(t, []float32{0, 30, 30, 10, 30, 50, 0}, sizes(subgridTracks(sub, span{0, 150}, 30)),
		"its own gap, centred on the grid's")
	assert.Equal(t, []float32{0, 0, 5, 30, 10, 60, 0}, sizes(subgridTracks(sub, span{45, 150}, 10)),
		"padding past its first track leaves the second where the grid has it")
}

// subgridPage is a grid of two auto columns and a fixed one, a subgrid over the first two
// holding a label in each, and a label in the third.
func subgridPage() (*dom.Node, *dom.Node, *dom.Node) {
	auto := style.Track{}
	grid := el(dom.Div, func(st *style.Computed) {
		st.Display = style.GridDisplay
		st.Columns = []style.TrackRepeat{{Count: 2, Tracks: []style.Track{auto}}, {Count: 1,
			Tracks: []style.Track{{Min: style.Px(60), Max: style.Px(60)}}, Before: []string{"last"}}}
		st.Gap = style.Size[style.Length]{Width: style.Px(6), Height: style.Px(6)}
	})
	sub := el(dom.Div, func(st *style.Computed) {
		st.Display = style.GridDisplay
		st.Subgrid.Width = true
		st.Column = [2]style.GridLine{{}, {Span: 2}}
		st.Padding.Left = style.Px(5)
	}, el(dom.Div, block, text("HH HH")), el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Column = [2]style.GridLine{{Name: "last"}, {}}
	}, text("HHH")))
	grid.Append(sub, el(dom.Div, block, text("H")))
	return el(dom.Div, block, grid), grid, sub
}

func TestAKeptSubgridFollowsItsGrid(t *testing.T) {
	changes := map[string]func(grid, sub *dom.Node){
		"nothing": func(*dom.Node, *dom.Node) {},
		"the grid's tracks": func(grid, _ *dom.Node) {
			restyle(grid, func(st *style.Computed) {
				st.Columns = []style.TrackRepeat{{Count: 1, Tracks: []style.Track{{Min: style.Px(90), Max: style.Px(90)}}},
					{Count: 2, Tracks: []style.Track{{}}}}
			})
		},
		"the grid's gap": func(grid, _ *dom.Node) {
			restyle(grid, func(st *style.Computed) { st.Gap.Width = style.Px(20) })
		},
		"the grid's names": func(grid, _ *dom.Node) {
			restyle(grid, func(st *style.Computed) { st.Columns[1].Before = nil })
		},
		"the subgrid's padding": func(_, sub *dom.Node) {
			restyle(sub, func(st *style.Computed) { st.Padding.Left = style.Px(30) })
		},
		"a label in the subgrid": func(_, sub *dom.Node) { sub.Children()[0].Children()[0].SetText("HHHHHHH") },
		"the subgrid no more": func(_, sub *dom.Node) {
			restyle(sub, func(st *style.Computed) { st.Subgrid.Width = false })
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			root, grid, sub := subgridPage()
			var r Retained
			layKept(&r, root, 400)
			change(grid, sub)
			kept := boxes(layKept(&r, root, 400))
			assert.Equal(t, boxes(lay(root, 400)), kept)
		})
	}
}
