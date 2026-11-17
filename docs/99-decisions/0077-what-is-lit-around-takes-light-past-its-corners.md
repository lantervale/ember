# 0077. What is lit around takes light past its corners

- Status: Accepted
- Date: 2026-11-17
- Area: rendering
- Owner: engine

## Context

What is lit around, a tall tile at its base or a tile with `lit: around` ([0075](0075-a-wall-seen-from-above-is-lit-from-every-side.md)), took the brightest light just past the middle of each of its four sides. A cave's rim tile meeting the floor only at a corner, the inside of a step, never saw that floor and stayed dark (lantervale/lantervale#117).

## Decision

- What is lit around takes the brightest of eight points: past the middle of each side, as before, and past each corner of its box.
- A wall's face, lit from its sides and below, is unchanged.

## Alternatives

- A shadow shape drawn small in the tile's floor corner: the rest of the cell then lets light through the rim.
- Light rim tiles per pixel: they stand in the shadow of what they block with.

## Consequences

Four more reads of the light picture for each pixel lit around. A tall tile can now catch light past a corner, so it may be lit a little more often at the end of another's shadow.
