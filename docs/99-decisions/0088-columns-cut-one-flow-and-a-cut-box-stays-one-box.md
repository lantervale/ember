# 0088. Columns cut one flow, and a cut box stays one box

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

ui refused `columns`, `column-count` and `column-width` (decision 0049's subset). Pages written for browsers use them for prose, and the web explorer showed such a section as one column. Layout keeps one box per element, with one cached layout each.

## Decision

- `column-count`, `column-width`, `columns`, `column-gap`, `column-rule`, `column-fill`, `break-inside`, `break-before`, `break-after`, `orphans` and `widows` are part of the CSS subset, laid out as Chrome does and checked against it.
- A multi-column container lays its content out once, as one column of the used width, then cuts that flow into columns: between blocks, between lines, anywhere in empty room. Each node moves to its column when the tree is written.
- An element cut by columns is still one box: its rectangle is the box round its parts, as `getBoundingClientRect()`'s, and its parts are its fragments, as a span's over several lines. `Box()` and `BoxOf` answer the rectangle; paint and hit testing take the parts.
- Pictures, controls, tables, flex and grid containers, floats, scroll containers and nested multi-column containers are never cut.
- `column-span: all` is refused with a warning.

## Alternatives

- Lay each column out on its own, as browsers do, with a box per fragment: every layout would take a break position and return where it stopped, through block, inline, flex, grid and table, and one element would have several boxes everywhere a box is read (automation, `ui.Follow`, `BoxOf`, hit testing). Exact for flex containers and tables, which Chrome cuts between rows; too much for prose.
- Keep refusing columns and have the explorer rewrite them to grid: grid items don't flow from one column into the next.

## Consequences

A Document without columns pays one test per layout call and the cascade's cost of twelve more properties. Flex containers and tables in columns stay whole where Chrome cuts them, a positioned box is never cut, and a spanning heading has to stand before its columns.
