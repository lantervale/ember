# 0151. Stroked text is a font of its own, its glyphs' outlines stroked on the CPU

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

`-webkit-text-stroke` draws each glyph's outline, half outside and half inside, over its fill, and with `-webkit-text-fill-color: transparent` alone, for outlined text. ui drew text only as glyph coverage from an atlas, and the 2D renderer fills no path.

## Decision

- A stroked text draws from a font of its own (`draw.Style.Stroke`, `twod.TextStroke(width)` for games): each glyph's outline is flattened and stroked as Skia strokes it, a quadrilateral for each straight piece and a mitred wedge (bevelled past a limit of 4) on the outside of each corner, all turned the same way so the rasterizer adds them into one shape, cached in the font's atlas as any glyph. Its advances are the plain font's, so layout never knows.
- Its coverage is exact, never darkened: Chrome strokes a path, not a glyph mask.
- ui draws the fill, then the stroke over it; the selected part takes the `::selection`'s stroke width and colour where it sets them, and its `color` for the fill, its `-webkit-text-fill-color` read and left unused as in Chrome.
- `thin`, `medium` and `thick` are drawn 1, 3 and 5 points wide, as Chrome draws them, though its getComputedStyle writes 48ths of the font size; the width flips halfway through a transition, the colours move.

## Alternatives

- Drawing the text again round itself (the made-up bold's ring): no inside half, no way to leave the fill out.
- Filling paths on the GPU (stencil or a path shader): exact at any scale, but a new kind of draw for text alone, where an atlas entry per glyph and width is drawn as every glyph is.
- `svg`'s stroker: pure, but it would pull the SVG package into every Document.

## Consequences

Outlined and stroked text match Chrome's to its anti-aliasing; each width and size is an atlas of its own, and a glyph's curves stroked from straight pieces can mitre a little further than Chrome's at a sharp corner between short curves.
