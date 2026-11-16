# 0075. A wall seen from above is lit from every side

- Status: Accepted
- Date: 2026-11-16
- Area: rendering
- Owner: engine

## Context

A flat tile that blocks light takes its light just past the left, right and bottom of what it blocks with, never from above: a wall's face looks down the screen, and a light behind it must not light it. A wall drawn from above, such as the rim of rock round a top-down cave, has floor on any side. A rim with its floor to the north stayed dark however near the light (Lantervale's Act 0 crypt, lantervale/lantervale#117).

## Decision

- Light's tile property `lit` says how a tile that blocks light is lit: `front`, the default, as a wall's face; `around`, from the brightest light just past any of its four sides, as a tall tile is at its base.
- `light.TileLitAround(ids…)` sets it on a tileset made in code; the tileset tool shows it with light's other properties.
- The draw list lights a drawing round a box as `LitFacing` or `LitAround` (`SetLightBox`), so a lit tile costs the same either way.

## Alternatives

- Light every blocking tile from every side: a face would be lit by a light behind its wall, through the wall.
- Guess from the tile's picture or its neighbours which way its floor lies: wrong for tiles placed by hand, and a cost on every map change.
- Light such tiles per pixel: they stand in the shadow of what they block with, so they would go dark at their far side.

## Consequences

A tileset says once, per tile, whether it is a face or a top; tiles that say nothing draw as before.
