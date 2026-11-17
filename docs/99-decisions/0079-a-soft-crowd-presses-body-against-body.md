# 0079. A soft crowd presses body against body

- Status: Accepted
- Date: 2026-11-17
- Area: core
- Owner: engine

## Context

Under [0074](0074-a-soft-crowd-never-stacks-on-one-point.md) a soft body could not move into a crowd cell that was full, and a move along an axis into one was dropped. A crowd coming from one side therefore stopped on the cells' edges. In Lantervale's crypt, 150 monsters chasing the hero stood in vertical files, three in four of them within 2 points of a 32-point line. Two in three touching pairs overlapped by more than a third of a body, and the monsters queued on the side they came from instead of going round. Merlin: monsters "aligned vertically", "they should not stack".

## Decision

- Soft bodies touch pair by pair, read as the ellipses their boxes hold. Ellipses keep a crowd from settling into rows.
- Once a step, the crowd notes for each body the nearest 8 bodies within a quarter of a body of touching it, with the room left to each. A soft body's `Move` keeps out of them: the part of its step that would go past that room is dropped, and the rest is kept. So it slides along them and round them, never stopping on a grid line.
- Once a step, bodies that overlap are pushed apart, each half the way out. Each body is weighed against at most 8 others of its cell and of each cell next to it that it reaches.
- Density stays as the bound on a pile. Capacity rises from 1 to 1.25, a little over what touching bodies fill (1.15). Only a cell over capacity spills, down its slope and only into cells with room.
- Otherwise 0074 and [0028](0028-a-soft-crowd-spreads-by-density.md) stand: soft bodies stop exactly at solids and at hard bodies, never push them, and the same steps give the same positions.

## Alternatives

- Pairwise contact below a number of bodies and density above it: a crowd would change its look as it grows past the threshold.
- Keep the density rule and soften it: files come from stopping a move at a cell edge, whatever the edge's limit.
- A full scan of neighbours on every move: the same look, at twice this cost for 20,000 bodies.

## Consequences

The crowd presses round its target: about one body in nine within 2 points of a cell line (chance is one in eight), no touching pair overlapping by more than a third of a body, and a third of a crowd coming from one side going round to the far side. A 20,000-body pile stays within 1.5 times capacity.

What a fixed step costs, before and after (CPU ms, the medians of three runs interleaved on a loaded Mac):

| Soft bodies | Before | Now |
|---|---|---|
| 300 spread over a level | 0.067 | 0.10 |
| 300 piled on their goal | 0.068 | 0.11 |
| 2,000 piled | 0.53 | 0.99 |
| 20,000 spread | 3.6 | 5.9 |
| 20,000 piled | 6.6 | 12.7 |

The cost is the pair pass (`separate`) and the contact each move reads.
