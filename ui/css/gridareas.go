package css

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/style"
)

// areaRows reads grid-template-areas' strings as rows of names, "" for a
// cell no area takes.
func areaRows(v []Component) ([][]string, error) {
	var rows [][]string
	for _, c := range v {
		if c.Kind != String {
			return nil, fmt.Errorf("expected none or a string a row, found %q", c.String())
		}
		row := strings.Fields(c.Text)
		for i, name := range row {
			if strings.Trim(name, ".") == "" {
				row[i] = ""
			}
		}
		if len(row) == 0 || len(rows) > 0 && len(row) != len(rows[0]) {
			return nil, fmt.Errorf("every row of grid-template-areas names as many cells")
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("expected none or a string a row")
	}
	return rows, nil
}

// gridAreas are the named areas of rows, each a rectangle of cells.
func gridAreas(rows [][]string) ([]style.GridArea, error) {
	var out []style.GridArea
	at := map[string]int{}
	cells := map[string]int{}
	for y, row := range rows {
		for x, name := range row {
			if name == "" {
				continue
			}
			i, ok := at[name]
			if !ok {
				i = len(out)
				at[name] = i
				out = append(out, style.GridArea{Name: name, Row: [2]int{y + 1, y + 2}, Column: [2]int{x + 1, x + 2}})
			}
			a := &out[i]
			a.Row[1], a.Column[0], a.Column[1] = max(a.Row[1], y+2), min(a.Column[0], x+1), max(a.Column[1], x+2)
			cells[name]++
		}
	}
	for _, a := range out {
		if (a.Row[1]-a.Row[0])*(a.Column[1]-a.Column[0]) != cells[a.Name] {
			return nil, fmt.Errorf("the area %q of grid-template-areas is not a rectangle", a.Name)
		}
	}
	return out, nil
}

func checkTemplateAreas(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	rows, err := areaRows(v)
	if err != nil {
		return err
	}
	_, err = gridAreas(rows)
	return err
}

func templateAreas(_ *ctx, v []Component) []style.GridArea {
	if isWord(v[0], kwNone) {
		return nil
	}
	rows, _ := areaRows(v)
	areas, _ := gridAreas(rows)
	return areas
}

// lineName reports a grid line given as a name alone.
func lineName(v []Component) bool {
	return len(v) == 1 && gridIdent(v[0])
}

// expandGridArea reads grid-area's row-start / column-start / row-end /
// column-end: a line left out is the name of the one it pairs with, auto
// when that is no name.
func expandGridArea(long []string, v []Component, out map[string][]Component) error {
	parts := splitOn(v, isSlash)
	if len(parts) > 4 {
		return fmt.Errorf("expected up to four lines")
	}
	for i, p := range parts {
		if err := checkGridLine(p); err != nil {
			return err
		}
		out[long[i]] = p
	}
	for i := len(parts); i < 4; i++ {
		from := out[long[max(i-2, 0)]]
		if !lineName(from) {
			from = []Component{word("auto")}
		}
		out[long[i]] = from
	}
	return nil
}

// expandGridTemplate reads grid-template: none, rows / columns, or a
// string a row with its height after it, then / columns.
func expandGridTemplate(v []Component, out map[string][]Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	parts := splitOn(v, isSlash)
	if len(parts) > 2 || len(parts[0]) == 0 {
		return fmt.Errorf("expected none, rows / columns, or a string a row")
	}
	rows := parts[0]
	if containsString(rows) {
		areas, heights, err := templateRows(rows)
		if err != nil {
			return err
		}
		out["grid-template-areas"] = areas
		rows = heights
	} else if len(parts) != 2 {
		return fmt.Errorf("expected rows / columns")
	}
	if err := checkTracks(rows); err != nil {
		return err
	}
	out["grid-template-rows"] = rows
	if len(parts) == 2 {
		if err := checkTracks(parts[1]); err != nil {
			return err
		}
		out["grid-template-columns"] = parts[1]
	}
	return nil
}

// expandGrid reads grid into its longhands (template rows, columns and
// areas, auto rows and columns, auto flow): a grid-template, or rows /
// auto-flow and the implicit columns' sizes, or auto-flow and the implicit
// rows' sizes / columns, dense beside auto-flow either way.
func expandGrid(long []string, v []Component, out map[string][]Component) error {
	parts := splitOn(v, isSlash)
	if len(parts) != 2 {
		return expandGridTemplate(v, out)
	}
	for i := range 2 {
		if flow, sizes, ok := autoFlow(parts[i]); ok {
			return expandAutoFlow(long, i == 1, flow, sizes, parts[1-i], out)
		}
	}
	return expandGridTemplate(v, out)
}

// expandAutoFlow sets grid's longhands for auto-flow on the side of the
// columns or of the rows: the other's template, the implicit sizes, the flow.
func expandAutoFlow(long []string, columns bool, flow, sizes, template []Component,
	out map[string][]Component) error {
	if err := checkTracks(template); err != nil {
		return err
	}
	templateAt, autoAt := 1, 3
	if columns {
		templateAt, autoAt = 0, 4
		flow[0] = word("column")
	}
	if len(sizes) > 0 {
		if err := repeated(trackSize, 64)(sizes); err != nil {
			return err
		}
		out[long[autoAt]] = sizes
	}
	out[long[templateAt]] = template
	out[long[5]] = flow
	return nil
}

// autoFlow reads auto-flow and dense, in either order, at the start of v:
// grid-auto-flow's value, and the sizes after them.
func autoFlow(v []Component) ([]Component, []Component, bool) {
	flow, dense, i := false, false, 0
scan:
	for ; i < len(v); i++ {
		switch {
		case isWord(v[i], "auto-flow") && !flow:
			flow = true
		case isWord(v[i], "dense") && !dense:
			dense = true
		default:
			break scan
		}
	}
	if !flow {
		return nil, nil, false
	}
	value := []Component{word("row")}
	if dense {
		value = append(value, word("dense"))
	}
	return value, v[i:], true
}

// templateRows cuts the rows of a grid-template written as strings into
// its areas and its row tracks: each row's height, auto for a string no
// height follows, between the names of their lines, those after a row and
// before the next on one line.
func templateRows(rows []Component) ([]Component, []Component, error) {
	var areas, tracks []Component
	var after *Component
	for i := 0; i < len(rows); {
		names := after
		after = nil
		if rows[i].Kind == LBracket {
			names = joinNames(names, rows[i])
			i++
		}
		if i == len(rows) || rows[i].Kind != String {
			return nil, nil, fmt.Errorf("expected a string a row, its height and the names of its lines")
		}
		areas = append(areas, rows[i])
		i++
		height := word("auto")
		if i < len(rows) && rows[i].Kind != String && rows[i].Kind != LBracket {
			height = rows[i]
			i++
		}
		if i < len(rows) && rows[i].Kind == LBracket {
			after = &rows[i]
			i++
		}
		if names != nil {
			tracks = append(tracks, *names)
		}
		tracks = append(tracks, height)
	}
	if after != nil {
		tracks = append(tracks, *after)
	}
	if err := checkTemplateAreas(areas); err != nil {
		return nil, nil, err
	}
	return areas, tracks, nil
}

// joinNames is the bracket of the names of a and b, b alone when a is nil.
func joinNames(a *Component, b Component) *Component {
	if a == nil {
		return &b
	}
	joined := *a
	joined.Args = append(slices.Clone(a.Args), b.Args...)
	return &joined
}

func containsString(v []Component) bool {
	for _, c := range v {
		if c.Kind == String {
			return true
		}
	}
	return false
}
