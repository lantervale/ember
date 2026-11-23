# 0192. text-box-trim follows Chrome's lines into columns and floats

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

`text-box-trim` was checked against Chrome on block containers alone. The spec trims a block's first and last formatted lines, which reads as nothing for a flex, grid or table container, once per multi-column container, and says nothing of the floats standing on a trimmed line; Chrome, measured, does more in columns and moves floats in a way of its own.

## Decision

- A multi-column container's trim takes each column's first line's room above it and its last line's room below it, as Chrome's trims at each column's start and end: the columns balance on what is left, the container's flow height shared as before, and a box that starts with the raised line keeps its top at the column's.
- A float laid out on a block's trimmed first line rises with it by the trim, while the room it takes from the lines stays where the line was laid out: the lines after the first rise by the trim too, so the second can be narrowed by the float, as in Chrome.
- A flex, grid or table container's trim reaches none of its items and cells, nor does a block's reach the text between its blocks: anonymous blocks take no `text-box-trim` of their own. A table cell trims its own lines as any block container.

## Alternatives

- The spec's reading for columns, trimming the container's first and last lines alone: Chrome's columns show a raised first line in every column, so pages would differ by a few pixels a column.
- Moving the floats' room with them: simpler, but lines would break where Chrome's don't.

## Consequences

Columns, floats and the boxes round them lay out as Chrome lays them; what wasn't measured (a column's last line before a block's bottom padding, the lines of boxes side by side cut by columns) keeps its whole room, recorded as missing.
