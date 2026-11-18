package css

import (
	"fmt"
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

// lineName reports a grid line given as a name: an area's.
func lineName(v []Component) bool {
	return len(v) == 1 && v[0].Kind == Ident && !isWord(v[0], "auto", "span", "inherit", "initial", kwUnset)
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

// templateRows cuts the rows of a grid-template written as strings into
// its areas and each row's height, auto for a string no height follows.
func templateRows(rows []Component) ([]Component, []Component, error) {
	var areas, heights []Component
	for i := 0; i < len(rows); i++ {
		if rows[i].Kind != String {
			return nil, nil, fmt.Errorf("expected a string a row, found %q", rows[i].String())
		}
		areas = append(areas, rows[i])
		if i+1 < len(rows) && rows[i+1].Kind != String {
			i++
			heights = append(heights, rows[i])
		} else {
			heights = append(heights, word("auto"))
		}
	}
	if err := checkTemplateAreas(areas); err != nil {
		return nil, nil, err
	}
	return areas, heights, nil
}

func containsString(v []Component) bool {
	for _, c := range v {
		if c.Kind == String {
			return true
		}
	}
	return false
}
