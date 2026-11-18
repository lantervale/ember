# 0094. Display table off a table is a block that shrinks and holds its floats

- Status: Accepted
- Date: 2026-11-18
- Area: rendering
- Owner: engine

## Context

ui lays tables out by their elements (`table`, `tr`, `td`) and refused `display: table` anywhere else. Pages of the web say it on elements that hold no row and no cell: the empty `::after { content: ""; display: table; clear: both }` that ends a container of floats, and `display: table; margin: 0 auto` round a block of unknown width. Refused, the first left every such container without height, so a WordPress theme's header, menu and slider lay over each other.

## Decision

- `display: table` on an element that is no `table` is a block that makes its own formatting context and, with no width, is as wide as its content: the one anonymous cell CSS wraps round children that are no rows or cells. `inline-table` is the same as an `inline-block`.
- A `table` element is laid out as a table whatever its `display` says of the two.
- `table-row`, `table-cell`, `table-row-group` and the other parts stay refused with their warning on every element.
- It is read in every sheet, not only under `ui.BrowserSheets()`: it is CSS as Chrome lays it out, checked against it.

## Alternatives

- Anonymous table boxes for any element (rows, cells, columns from `display`): what CSS says in full, and a second way into the table layout that works by element today; nothing asked for it yet but equal-height columns, which flex and grid give.
- Read it as `flow-root`: holds the floats, but full width, so a centred `display: table` block would not shrink.
- Read it only under `ui.BrowserSheets()`: a screen's sheet would keep a warning for something ui now lays out as Chrome does.

## Consequences

A clearfix and a shrinking block work as in a browser. An element with `display: table` whose children say `table-cell` still stacks them as blocks, with their warnings: that layout is in [What is missing](../05-ui/10-missing.md).
