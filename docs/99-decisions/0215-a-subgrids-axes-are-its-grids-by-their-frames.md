# 0215. A subgrid's axes are its grid's by their frames

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A subgrid takes its grid's tracks ([decision 0125](0125-a-subgrids-items-size-its-grids-tracks-hoisted-to-its-lines.md)), but each lays out in its own frame: its writing mode's, mirrored under `direction: rtl`. A subgrid of another direction or writing mode than its grid's got the grid's tracks in the wrong axis or from the wrong end, and its items, sized by the grid, in the frame of the subgrid.

## Decision

- Each grid's frame says which page axis its columns and rows run along and which way; a subgrid's axis lies along the grid's axis on the same page axis, reversed when the two run opposite ways (CSS Grid 2 §9, as Chrome).
- The grid hands a subgrid its names and tracks already in the subgrid's frame: an orthogonal one gets the grid's columns as its rows, a reversed one counts them from the other end.
- Hoisting a subgrid's items, the grid maps their lines into its own, takes the subgrid's margin, border and padding from its physical style in the grid's frame, and sizes an item of another frame as though it stood in the grid's (`reframe`, as an absolutely positioned box is placed in its containing block's frame).
- Baseline-aligned items of a subgrid sharing the grid's rows join the grid's baseline groups; the shim the grid finds is the one the subgrid lays them out with.

## Alternatives

- Lay an orthogonal subgrid out as an ordinary grid: what Chrome did before subgrid; the WPT reference pages expect the shared tracks.
- Turn the subgrid's templates into the grid's frame when styling: the subgrid's own layout reads them in its frame, so both would need two copies.

## Consequences

Subgrids nest across directions and writing modes as Chrome's do. A subgrid's item sized for an orthogonal grid is reframed for each measure, which empties its layout cache; such pages are rare.
