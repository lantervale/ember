# 0105. A table is what display says, with the boxes it misses

- Status: Accepted
- Date: 2026-11-19
- Area: rendering
- Owner: engine

## Context

ui laid tables out by their elements and read `display: table` elsewhere as a block that shrinks (decision 0094), refusing rows, cells and `border-collapse`. Pages of the web make tables of `div`s for equal-height columns and centred cells, restyle `td`s as blocks on narrow screens, and collapse the borders of nearly every data table; the CSS 2.1 tests of WPT are written in both.

## Decision

- An element is a table, a row group, a row, a cell, a column, a column group or a caption by its `display`; ui's own sheet gives the table elements theirs, so a sheet can take it away (`td { display: block }`) or give it to anything.
- The layout makes the anonymous boxes of CSS 2.1 §17.2.1 as Chrome does: a row round loose cells, a table round loose rows, cells and groups, a cell round what stands in a row; white space alone between parts is dropped.
- `col` and `colgroup` are elements (`ui.Col`, `ui.Colgroup`), and columns size their tracks and paint behind their cells; the backgrounds of a table are the layers of §17.5.1.
- `border-collapse: collapse` resolves each edge of the grid by §17.6.2.1; a cell and the table take half of their edges as borders, and the table draws the edges over its cells.
- One table layout serves all of it: the simple path stays for tables of plain cells, and columns, spans, baselines and collapsed borders go through the general one.

## Alternatives

- Keep tables by element and map `display: table-cell` on to flex: equal heights come out, but not the widths a table gives its columns, nor anonymous boxes, and every difference from Chrome would be ours.
- A second layout for tables of other elements: two table algorithms to keep equal.
- Draw collapsed borders from each cell (its half of each edge): two halves of one dashed or wide edge never line up, and a hidden border would need its neighbour's half suppressed anyway.

## Consequences

Tables of any elements, real tables restyled and collapsed borders lay out as in Chrome, checked case by case against it. A table with columns, spans, baseline cells or collapsed borders takes the general layout, which measures each cell again where the simple one keeps rows; what the borders do where edges of different widths meet is ui's own, a few pixels from Chrome's, and is in [What is missing](../05-ui/10-missing.md).
