# 0078. A tile lit around is lit round its bodies too

- Status: Accepted
- Date: 2026-11-17
- Area: rendering
- Owner: engine

## Context

A cave's rim drawn from above is one tile thick, with floor on both sides at its steps. A light beside a step put the floor just past the step's top edge behind the rim, a thin wedge of shadow that read as a fault (lantervale/lantervale#117). Casting the rim only from a strip on its far side stops the wedge, but a tile that blocks light was lit round what it blocks with alone, so such a rim took its light inside its own cell, in its neighbours' shadow, and went dark.

## Decision

- A tile lit around ([0075](0075-a-wall-seen-from-above-is-lit-from-every-side.md)) is lit round its `body` shapes as well as what it blocks light with.
- A tile lit by its face, and every tile that does not say `around`, is lit round what it blocks light with, as before.

## Alternatives

- Light every tile round its bodies: the walls and props of a game already drawn with bodies and shadow shapes would change how they are lit.
- A separate shape class for what a tile is lit round: one more thing to draw, where the body already says what the wall is.
- Leave the rim casting from its whole cell: the wedges stay.

## Consequences

A wall seen from above is drawn as what it is (its body) and what of it casts (its shadow shapes), and lit as the whole wall. A tile lit around with a body much larger than what it casts is lit further out.
