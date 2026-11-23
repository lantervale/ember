# 0156. Columns cut what they hold through one flow of units

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A multi-column container lays its content out once, as one column, then cuts that flow into columns by its units: lines, monolithic boxes, the edges of blocks. Tables, flex containers and absolutely positioned boxes were monolithic, where Chrome cuts a table between its rows, a flex container between its lines and inside its items, and a positioned box whose containing block is in the columns.

## Decision

- The walk that makes the units goes into a table and its row groups (a row stays whole), into a flex container's items, and into the absolutely positioned boxes a cut box is the containing block of; placing the boxes goes down the same way, each box shifted by the column its top shows in.
- Units that overlap in the flow (a flex row's items, a positioned box beside the lines, a float) are one unit: a column ends only where none of them has a line across it.
- `break-inside: avoid` joins a box's units into one but where a block in it forces a break, as Chrome gives way to a forced break.

## Alternatives

- Lay each column out again with what it holds, as Chrome's fragmentation does: every box would need to resume mid-way, a rewrite of every algorithm for a rare page.
- Cut each of the boxes side by side where its own content allows, as Chrome does: the columns' starts are one place in the flow for all of them, so each would need starts of its own.

## Consequences

Tables, flex rows and positioned boxes split across columns as in Chrome where their lines line up; boxes side by side whose lines don't are cut only between them all, a gap kept in [what is missing](../05-ui/10-missing.md).
