# 0132. Border sides are mitered pieces smoothed by their own geometry

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

A border whose sides differ in colour was drawn as hard-edged trapezoids, its diagonals stair-stepped where Chrome blends the two colours, and drawn square when the box was rounded. `border-style: double` was refused, and a double needs the same mitres with a gap cut out of each side.

## Decision

- A square border of more than one colour, an inset or outset one, and a double beside solid sides are each side's trapezoid cut to its lines (a double's two, as Chromium cuts them in whole device pixels: the outer `(w+1)/3`, the inner from `(2w+1)/3`, solid under 3 pixels) and drawn as triangles.
- Where two sides of other colours meet, each side's last half device pixel before the diagonal is a strip whose corners carry the colour, fading to the colour halfway between the two in encoded sRGB on the diagonal itself, as Chrome blends it; a clear side draws its half too, fading out, so a tooltip's arrow is smooth on both edges. The pieces tile the border, so nothing is drawn twice.
- A rounded border as wide all round is a ring per run of sides of one colour, the first opaque run whole and the others over it through a mask of their sides, cut on the 45-degree lines from the corners; the mask's edge is the blend.
- A double whose every side is double, or rounded, is two borders: its outer lines, and its inner ones inside the gap, each drawn as any border. Outlines and a table's collapsed edges cut their double the same way.

## Alternatives

- Analytic coverage in a shader for every side: smooth everywhere, but a new GPU kind for one look, and the 2D batcher would split at each border.
- A mask per side for square borders too: a texture per box and colour, where triangles with coloured corners cost nothing and match Chrome to the pixel.
- Two rectangles per double side: overlapping corners where sides differ in colour, and the gap would cross the mitre.

## Consequences

Square borders of other colours match Chrome's pixels on their diagonals; a rounded one keeps its corners. Where three or four colours meet round a rounded box, two masked rings meet over the first and let a trace of it through their shared edge. A rounded border of uneven widths is still drawn square.
