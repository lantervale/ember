# 0117. A background keeps each longhand's list and paints its layers one by one

- Status: Accepted
- Date: 2026-11-21
- Area: rendering
- Owner: engine

## Context

Pages stack backgrounds: a gradient over a picture, several gradients, an icon over a texture. ui drew one layer, so the computed style held one value a longhand, and the cascade sets each `background-*` longhand on its own, with lists of any length.

## Decision

- The computed style keeps each background longhand as the list written (`BackgroundImages`, `BackgroundPositions`…, nil for the initial value); `BackgroundLayer(i)` builds layer i on demand, a shorter list starting again and a longer one cut, as CSS Backgrounds §2.3 says.
- The painter draws the colour in the last layer's clip box, then each layer from the last to the first through the same code that drew the one layer, each a clip of its own; a colour and the layers over it share one more clip to the border box, so layers showing in all of it join the colour's batch in `internal/draw`.
- `background-repeat` is a `Tiling` for each axis, which brings `space` and `round` along with the lists.

## Alternatives

- Layers resolved once at the end of the cascade into a slice of `BackgroundLayer`: a second copy of the same values to keep in step, and a computed style copied for a pseudo-element or a transition sharing it while one is written in place.
- Compositing the layers offscreen into one picture: a pass and a texture per element for what plain draws in order give, and nothing Chrome's output asks for.

## Consequences

A style with one layer costs as before (nil lists), and a layer costs its own draws; paint reads layers by index, never allocating. What reads the background goes through `BackgroundLayer`, not the lists.
