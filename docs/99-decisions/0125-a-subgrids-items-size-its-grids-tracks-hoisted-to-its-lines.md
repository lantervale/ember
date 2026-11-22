# 0125. A subgrid's items size its grid's tracks, hoisted to its lines

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

`subgrid` makes an item's tracks its grid's, so the subgrid's items must size the grid's tracks, but ui's grid is a port of Taffy, which has no subgrids and sizes each grid on its own.

## Decision

- The grid sizing an axis places each subgrid's items (and theirs, for nested subgrids) and sizes its tracks with them in the subgrid's place, as items of its own at the grid's lines, each with more margin: the margin, border and padding of the subgrids whose edge it stands on, and half what a subgrid's gap differs from the grid's beside a gutter. Each subgrid adds an empty item on each edge track holding its own edges, as Chrome does.
- The grid then hands each subgrid the tracks it spans (and the names of their lines); the subgrid lays out with them as fixed tracks, its first and last cut to its content box, and is stretched over them whatever its size and self-alignment.
- An item hoisted for one axis on tracks the subgrid has of its own in the other is measured at the width a layout of the subgrid gives it there.

## Alternatives

- One sizing over the grid and every subgrid as a single track problem, as the spec words it: a rewrite of the Taffy port.
- Sizing a subgrid as an ordinary item, then forcing its tracks: its items would never widen the grid's tracks, the point of a subgrid.

## Consequences

The Taffy port stays whole and subgrids reuse its sizing, at the cost of placing a subgrid's items twice and a probe layout of row subgrids; what the hoisted items lack (baselines shared across a subgrid) is recorded in docs/05-ui/10-missing.md.
