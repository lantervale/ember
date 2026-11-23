# 0185. Boxes side by side are lanes, and a framed block round a spanner is laid out per part

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0156](0156-columns-cut-what-they-hold-through-one-flow-of-units.md) made boxes side by side one unit, cut only where none of them has a line across a column's end, and kept a table's rows whole; [0170](0170-what-columns-cut-and-a-line-clamp-hides-follow-chromes-ends.md) spanned a spanner deeper down only through blocks with no margin, border or padding. Chrome cuts each of the boxes side by side where its own content allows, cuts a row's cells, and slices a framed block round a spanner, its top edges above it and its bottom ones below.

## Decision

- The items of a flex or grid container and the cells of a row whose units overlap are one unit of lanes, each lane the units of one box. Where they don't fit, each lane is filled into the columns on its own, from the column the unit starts in; the flow after them starts in their last column as far down as the lowest lane goes there, their container's bottom edges then a column of their own when the column has no room for them. A box laid out to the others' bottom ends with the lowest of them in that column, as Chrome stretches a row's parts.
- Each lane keeps its own column starts: its box and what it holds are placed by a copy of the columns with its starts, the one layout of every node kept.
- A block round a spanner with a margin, a border or padding is laid out once per part of the flow, holding the children of that part, sliced: no bottom edges above a spanner, no top ones below. It is then placed round its parts, from the top of the first to the bottom of the last, what it holds moved there.

## Alternatives

- Lay each column out again with what it holds, as Chrome's fragmentation does: every algorithm would have to resume mid-way.
- Keep the one start per column for everything and cut boxes side by side only where their lines line up: tables with uneven cells and flex rows of unlike lines would keep spilling whole into one column.

## Consequences

Flex and grid rows, table rows and framed blocks round a spanner break as in Chrome. A positioned box beside the lines of its containing block is still one unit with them, and a lane's own forced breaks are its own: the boxes beside it don't follow.
