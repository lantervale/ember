package paint //nolint:testpackage // white-box: builds box trees by hand

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tablePart(p style.TablePart, bg color.Color) func(*style.Computed) {
	return func(st *style.Computed) { st.TablePart, st.Background = p, bg }
}

// layered is a table of two rows of two cells 40 by 20, 2 apart, in a row group, under
// a column over each track, the second in a column group.
func layered(column, group, rows, row color.Color) *box.Box {
	cell := func(x float32) *box.Box { return el(x, 0, 40, 20, tablePart(style.TableCell, color.Color{})) }
	r0 := el(0, 0, 82, 20, tablePart(style.TableRow, row), cell(0), cell(42))
	r1 := el(0, 22, 82, 20, tablePart(style.TableRow, color.Color{}), cell(0), cell(42))
	body := el(2, 2, 82, 42, tablePart(style.TableRowGroup, rows), r0, r1)
	first := el(2, 2, 40, 42, tablePart(style.TableColumn, column))
	second := el(0, 0, 40, 42, tablePart(style.TableColumn, column))
	cols := el(44, 2, 40, 42, tablePart(style.TableColumnGroup, group), second)
	return el(10, 10, 86, 46, func(st *style.Computed) { st.Table = true }, first, cols, body)
}

func rectsOf(r *recorder, c color.Color) []geom.Rect {
	var out []geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == c {
			out = append(out, o.Rect)
		}
	}
	return out
}

func TestATablesLayersPaintBehindItsCellsAlone(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(layered(red, green, blue, white)))
	assert.Equal(t, []geom.Rect{{X: 12, Y: 12, W: 40, H: 20}, {X: 12, Y: 34, W: 40, H: 20}, {X: 54, Y: 12, W: 40, H: 20},
		{X: 54, Y: 34, W: 40, H: 20}}, rectsOf(r, red), "each column behind its cells, never in the spacing")
	assert.Equal(t, []geom.Rect{{X: 54, Y: 12, W: 40, H: 20}, {X: 54, Y: 34, W: 40, H: 20}}, rectsOf(r, green),
		"a column group behind its columns' cells")
	assert.Len(t, rectsOf(r, blue), 4, "a row group behind the cells of its rows")
	assert.Equal(t, []geom.Rect{{X: 12, Y: 12, W: 40, H: 20}, {X: 54, Y: 12, W: 40, H: 20}}, rectsOf(r, white))
	assert.Equal(t, "column column group group column column rows rows rows rows row row",
		r.order(map[color.Color]string{red: "column", green: "group", blue: "rows", white: "row"}),
		"a column group under its columns, the columns under the row groups, those under the rows (CSS 2.1 §17.5.1)")
}

func TestATableDrawsItsCollapsedBordersOverItsCellsAndTheyDrawNone(t *testing.T) {
	bordered := func(p style.TablePart, bg color.Color) func(*style.Computed) {
		return func(st *style.Computed) {
			tablePart(p, bg)(st)
			st.Border, st.BorderStyle, st.BorderColor = style.All[float32](2), style.All(style.BorderSolid), style.All(red)
		}
	}
	cell := el(1, 0, 40, 20, bordered(style.TableCell, white))
	row := el(0, 1, 42, 20, tablePart(style.TableRow, color.Color{}), cell)
	table := el(10, 10, 42, 22, func(st *style.Computed) {
		bordered(style.NoTablePart, blue)(st)
		st.Table = true
	}, row)
	table.Collapsed = []box.Edge{
		{Rect: geom.Rect{X: 0, Y: 0, W: 42, H: 2}, Style: style.BorderSolid, Color: black, Across: true},
		{Rect: geom.Rect{X: 0, Y: 2, W: 2, H: 20}, Style: style.BorderDashed, Color: black},
		{Rect: geom.Rect{X: 0, Y: 20, W: 42, H: 4}, Style: style.BorderGroove, Color: green, Across: true},
	}
	r := newRecorder()
	Paint(r, tree(table))
	assert.Empty(t, rectsOf(r, red), "neither the table nor its cells draw their own borders")
	rects := r.of("rect")
	require.GreaterOrEqual(t, len(rects), 3)
	assert.Equal(t, blue, rects[0].Color)
	assert.Equal(t, white, rects[1].Color)
	assert.Equal(t, []geom.Rect{{X: 10, Y: 10, W: 42, H: 2}}, rectsOf(r, black)[:1],
		"the grid's edges, after the cells' backgrounds")
	assert.Greater(t, len(rectsOf(r, black)), 2, "a dashed edge in dashes")
	dark, light := shades(green)
	assert.Equal(t, []geom.Rect{{X: 10, Y: 30, W: 42, H: 2}}, rectsOf(r, dark), "a groove dark, then light")
	assert.Equal(t, []geom.Rect{{X: 10, Y: 32, W: 42, H: 2}}, rectsOf(r, light))
}

func TestACollapsedDoubleEdgeIsTwoLines(t *testing.T) {
	table := el(10, 10, 42, 22, func(st *style.Computed) { st.Table = true })
	table.Collapsed = []box.Edge{
		{Rect: geom.Rect{X: 0, Y: 0, W: 42, H: 6}, Style: style.BorderDouble, Color: green, Across: true},
		{Rect: geom.Rect{X: 0, Y: 0, W: 5, H: 22}, Style: style.BorderDouble, Color: blue},
		{Rect: geom.Rect{X: 37, Y: 0, W: 2, H: 22}, Style: style.BorderDouble, Color: red},
	}
	r := newRecorder()
	Paint(r, tree(table), Scale(1))
	assert.Equal(t, []geom.Rect{{X: 10, Y: 10, W: 42, H: 2}, {X: 10, Y: 14, W: 42, H: 2}}, rectsOf(r, green),
		"a third of its width each, across")
	assert.Equal(t, []geom.Rect{{X: 10, Y: 10, W: 2, H: 22}, {X: 13, Y: 10, W: 2, H: 22}}, rectsOf(r, blue), "down")
	assert.Equal(t, []geom.Rect{{X: 47, Y: 10, W: 2, H: 22}}, rectsOf(r, red), "solid, too thin for a gap")
}
