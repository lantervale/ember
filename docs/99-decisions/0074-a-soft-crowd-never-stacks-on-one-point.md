# 0074. A soft crowd never stacks on one point

- Status: Superseded by [0079](0079-a-soft-crowd-presses-body-against-body.md)
- Date: 2026-11-16
- Area: core
- Owner: engine

## Context

[0028](0028-a-soft-crowd-spreads-by-density.md) spread soft bodies by the density of grid cells, which let bodies overlap anywhere inside a cell under capacity. In Lantervale's crypt, 150 monsters pressing on the hero stood up to 25 on the very same point. Two causes: over a level-wide span the grid doubled its cells to 64 points, and inside any cell bodies slide to the same spot (the middle of a hard body's side, the edge of a full cell). Merlin: "once created they should not stack".

## Decision

- 0028 stands except as said here: soft bodies pass through each other, sweep exactly against solids and hard bodies, which they never push, and spread by density, each push capped, sliding, and in a fixed order.
- The crowd's cells are twice the largest soft body's side over any span up to 2^18 cells (about 16,000 points a side for 16-point bodies). A step clears only the cells written since the last, so it costs what its bodies touch, not the span. Only past that limit do cells grow.
- Once a step, two soft bodies whose centres are within a fifth of the smaller side are pushed to half a side apart. Each body is weighed against at most 8 others of its cell and those of the cells across a near edge. Pushes are worked out from the step's snapshot and added to the density push: one capped, sliding nudge per body.

## Alternatives

- Pairwise pushes alone: rejected by 0028, as a pile held only by them compresses without bound. Here density still bounds the pile, and pairs only keep bodies off one point.
- Keep cells at body scale but leave bodies free inside a cell: free, but 42% of a settled 20,000 pile stays within 2 points of another.
- Keeping bodies apart as an opt-in, so big hordes keep 0028's cost: an option every game would need to know about, for a pile nobody wants.

## Consequences

No two soft bodies stand on one point. Per fixed step: 300 bodies 0.068 ms (0.026 before), 20,000 spread out 3.6 ms (2.5), 20,000 settled in one pile 6.7 ms (1.8): there every step pushes about 40% of the pile apart against the chase. Later: that cost is the per-body pair pass (`separate`) and the nudges it adds in `spread`, the place to cut it.
