# 0216. A clip under a rotation or a skew cuts in its own space

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A clip pushed under a rotation or a skew clipped to the screen rectangle round it, unrounded (decision 0058). What a turned box draws past its own edges, the bands of a radial or conic gradient, tiles of a repeated picture or gradient starting before its box, a background under a rounded border, then showed in the corners of that rectangle, where Chrome cuts them at the turned box.

## Decision

- The rounded clip a batch carries can be in its own space: its rect and radii as pushed, with the transform taking the screen's points back there and how many points one of its units spans. The GPU takes each fragment back through it before the rounded-rect distance, scaled to points, so the cut turns, skews and anti-aliases with what it cuts; the scissor stays the rectangle round it.
- Under a rotation or a skew, `PushClipRounded` keeps its radii this way, and `PushClip` cuts as a rounded clip with no radii: inside a square clip turned alike, as their overlap; inside any other rounded clip, that one goes on cutting.
- A turned box's gradient reaches a point past its box, and that clip cuts it, so its edges are anti-aliased as Chrome's are.

## Alternatives

- Cutting the geometry on the CPU to the turned box: exact for gradients, but every picture, tile and glyph cut too, and rounded corners as polygons.
- A mask (`PushClipMask`) of the box's shape: turned already, but a picture per box and its sampling's blur at the edge.
- A stencil buffer, as decision 0058 weighed: exact nesting, at a stencil attachment in every 2D pass.

## Consequences

Two more vectors in each batch's uniforms. Turned boxes cut their backgrounds, scrollers and overflow as Chrome does. Only the innermost rounded clip still cuts: a square clip inside a turned rounded one, or in another space, cuts as its rectangle on screen.
