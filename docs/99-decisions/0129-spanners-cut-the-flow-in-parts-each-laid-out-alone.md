# 0129. Spanners cut the flow in parts, each laid out alone

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: engine

## Context

[0088](0088-columns-cut-one-flow-and-a-cut-box-stays-one-box.md) lays a multi-column container's content out once, as one column, and cuts that flow into columns. `column-span: all` was refused: a spanner is as wide as the container, so it can't be laid out in a column-wide flow, and the content after it starts a new row of columns balanced on its own.

## Decision

- The container's children are cut at its spanners into parts: the runs between them, and each run of adjacent spanners. Each part is laid out alone, as its own formatting context, one after the other in the flow: a run at the column width, cut into a row of balanced columns; spanners at the container's content width, shown as one column at x 0. Nothing collapses across parts but the margins of adjacent spanners, as Chrome's.
- Every part still gives each node one layout in the one flow: the column set lists where each column, or spanner, starts in the flow and where it shows, so placing boxes, cutting them and shifting lines work as before.
- Only children of the container span; a spanner deeper down (Chrome's, under blocks that start no formatting context) is laid out in its column.
- Rules are drawn per row (`box.Columns.Rows`), never across a spanner.

## Alternatives

- Lay the whole flow out once with spanners widened in the block layout: their margins would collapse with the columns' content and floats would cross them, where Chrome does neither.
- Lay out each column row as a block of its own in the layout tree: an anonymous node per row would show in the box tree and break the one-box-per-element rule 0088 keeps.

## Consequences

A container with spanners is laid out once per part, each part a flow of its own; a plain container is one part, laid out as before. Nested spanners stay a gap in [missing](../05-ui/10-missing.md).
