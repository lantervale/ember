# 0170. What columns cut and a line clamp hides follow Chrome's ends

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0156](0156-columns-cut-what-they-hold-through-one-flow-of-units.md) cut tables, flex containers and positioned boxes in one flow of units, but kept grid containers, tables with captions and blocks round a nested spanner whole; [0147](0147-lines-are-clamped-as-chrome-clamps-them.md) clamped a fixed count of lines and drew some of what follows. Chrome cuts and hides these by where each box's content ends, which the one flow has to express without laying anything out again.

## Decision

- A grid container and a table with captions are walked as a flex container and a table are; a caption below comes after the rows. A table's spacing below its last row is no unit: balancing leaves it out, and its last part keeps what of it the column has room for. A row group's parts end below the spacing after their last row, not at the column's end.
- A block wrapping a spanner, that starts no formatting context and has no margin, border or padding, is laid out as what it holds: its children stand in its place among the container's parts, and it is then placed round them, its parts skipping the spanner's row.
- `line-clamp: auto` lays the block out clamped at a count of lines, the most whose height fits its `height` or `max-height`, found by bisection. A clamp that hides what follows ends below the bottom margin of the block holding its last line; past it an inline element's lines and the floats are left out of paint and hit testing, positioned boxes kept, as Chrome's.

## Alternatives

- Lay a wrapper out once per part, as Chrome's fragments: a node would have a layout per part, against the one layout per node the columns rely on.
- Find the auto clamp while laying out, line by line against the height left: every block in the clamp's flow would carry a height budget through margin collapsing, for a property Chrome ships behind a flag.

## Consequences

Grids, captioned tables and articles whose headings span from inside a wrapper break as in Chrome. A wrapper with a frame of its own, boxes side by side whose lines don't line up and table rows stay as [missing](../05-ui/10-missing.md); an auto clamp costs a few extra layouts of its box.
