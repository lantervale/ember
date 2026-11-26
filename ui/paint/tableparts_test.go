package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

func part(p style.TablePart, edit func(s *style.Computed)) func(s *style.Computed) {
	return func(s *style.Computed) {
		s.TablePart = p
		edit(s)
	}
}

func table(s *style.Computed) { s.Table = true }

func TestAColumnAndAColumnGroupDrawNoOutline(t *testing.T) {
	outline := func(s *style.Computed) {
		s.OutlineWidth, s.OutlineStyle, s.OutlineColor = 10, style.BorderSolid, color.Black
	}
	for _, p := range []style.TablePart{style.TableColumn, style.TableColumnGroup} {
		r := newRecorder()
		Paint(r, tree(el(0, 0, 200, 100, table, el(0, 0, 100, 100, part(p, outline)))))
		assert.Empty(t, r.of("border"), "part %d", p)
	}
	r := newRecorder()
	Paint(r, tree(el(0, 0, 200, 100, nil, el(0, 0, 100, 100, part(style.TableColumnGroup, outline)))))
	assert.NotEmpty(t, r.of("border"), "outside a table, a box of the display is laid out otherwise and outlined")
}

func TestARowGroupsOverflowClipsNothing(t *testing.T) {
	hidden := func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}
	for _, p := range []style.TablePart{style.TableRowGroup, style.TableHeaderGroup, style.TableFooterGroup,
		style.TableRow} {
		cell := el(0, 0, 150, 20, part(style.TableCell, func(s *style.Computed) { s.Background = color.Black }))
		r := newRecorder()
		Paint(r, tree(el(0, 0, 400, 100, table, el(0, 0, 100, 20, part(p, hidden), cell))))
		for _, o := range r.of("rect") {
			assert.NotEqual(t, geom.Rect{W: 100, H: 20}, o.Clip, "part %d: %v", p, o)
		}
	}
}
