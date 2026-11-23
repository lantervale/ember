# 0179. A collapsed column narrows a stretched table, but a flex row's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

`visibility: collapse` on a column takes its room out of the table. A table given its width by its container, stretched in a grid area or across a flex column, or sized along a flex row, could keep that width or lose the column's too; CSS Tables 3 leaves it open.

## Decision

- As Chrome: a table stretched in a grid area or across a flex column is laid out at that width, then narrowed by its collapsed columns and their spacing, and placed in its area at its narrowed width.
- A table that is an item of a flex row keeps the main size its container gave it, its rows as wide as before the collapse, whatever its own `width` or `flex`.
- A height a table collapsing rows is given (a grid area's, a flex line's, a flex column's main size) is a floor for its height before the collapse: the rows grow to it, then the collapsed ones go, so a height measured after the collapse and handed back takes nothing out twice.
- A grid item that is a table is as big as its own layout says, never the area's size forced over it.

## Alternatives

- Keep every width a container gives: simpler, and what ui did, but the grid's and the flex column's tables would stand wider than Chrome's.
- Narrow every table, flex rows included: one rule, but a flex row's items would no longer fill the sizes the flex algorithm gave them, unlike Chrome.

## Consequences

Pages laid out for Chrome keep their tables where Chrome draws them; a flex row's table keeps its collapsed column's room inside its box.
