# 0076. A tile object casts as its tile

- Status: Accepted
- Date: 2026-11-17
- Area: rendering
- Owner: engine

## Context

A tile placed as a Tiled tile object became a plain sprite: it cast nothing from its tile's shapes and was lit per pixel, while the same tile in a map casts from its shapes, at its height, in its group, and is lit just past what it blocks with. Lantervale's level rocks are tile objects, so the game cast them from a box it made by hand, and a rock beside a wall went black in the wall's shadow (lantervale/lantervale#117).

## Decision

- `Tiled.Sprite` remembers the tile it stands for and the cell under its foot, in its own space, scaled as its width.
- The light package casts that tile's shapes from the sprite, as it does on that cell of a map: its shapes or its whole solid cell, flipped, at its `height`, in its `occluder_group`. A sprite whose tile blocks no light is never looked at.
- twod lights it as that tile in a map: a tall tile at its base, round what it blocks with; another that blocks light by its face, or from every side when its `lit` is `around` ([0075](0075-a-wall-seen-from-above-is-lit-from-every-side.md)).
- `light.NoShadows()` on the sprite stops it casting; `light.Occluder` or `Silhouette` on it cast as they say instead.

## Alternatives

- A light option the game puts on each object's sprite: every game would have to remember it, and the default would stay wrong.
- Draw tile objects through a tile map: an object is placed and sized freely, off the grid.

## Consequences

A tile casts and is lit the same wherever it is placed. A flipped tall tile in a map is now lit round what it blocks with flipped, as it casts.
