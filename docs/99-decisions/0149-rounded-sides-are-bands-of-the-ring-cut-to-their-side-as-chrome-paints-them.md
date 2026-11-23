# 0149. Rounded sides are bands of the ring cut to their side, as Chrome paints them

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Decision [0132](0132-border-sides-are-mitered-pieces-smoothed-by-their-own-geometry.md) left a rounded border square when its sides differed in width or style, solid when dashed or dotted, a trace of its first colour where two others met, outlines solid in any style, and a table's collapsed edges joined by a guess.

## Decision

- A rounded border that isn't one ring is painted side by side, as Chrome's BoxBorderPainter paints it: each side is a band of the ring between the border box and the padding box (inner corners elliptical where the widths differ), cut by a mask to the quad from its outer corners to where the line through the inner corner crosses the chord of a rounded inner corner. Sides painted alike are one run; runs go in Chrome's order (the more transparent first, then dashed, dotted and double, then the shaded styles, then solid; top, bottom, right, left).
- A band is the ring between two insets: the whole ring, an inset or outset side's shade, a groove or ridge halved about the middle, a double's two lines cut in device pixels. Dashes and dots are laid round the whole middle of the border clockwise from the top-left end of the top side, as Skia strokes a rounded rectangle, with Chrome's StyledStrokeData lengths and gap measured in device pixels, then cut to the ring and their side.
- An outline in any other style than solid or double is that border drawn round it; `auto` is Chrome's focus ring, a stroke of two thirds of its width down the box grown by its offset, round as the box or as wide as the outline, over a white one a third wide.
- A table's collapsed edges join as Chrome's ComputeEdgeJoints decides: at each corner the wider edge, then the style, then the earlier box (cells in the document's order, rows, row groups, columns, column groups, the table) wins; a winner reaches through the corner, a loser stops short, and edges are drawn in Chrome's order so that of two winners the later shows.

## Alternatives

- Strokes along the rounded outline per side in a shader: smooth without masks, but a new GPU kind, and dashes and their cuts would still need the same geometry.
- Rings per colour drawn over each other (0132's way): no masks for the first colour, but a trace where two later ones meet, and no answer for uneven widths.

## Consequences

Rounded borders, dashed and dotted ones included, outlines in every style and collapsed corners match Chrome to their antialiasing; a rounded side costs a mask per band, cached by its outline.
