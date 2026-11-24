# 0165. A filter element runs on the GPU over an element's picture, and a drop cap is a trimmed float

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Pages name SVG filters from CSS (`filter: url(#glow)`) and make drop caps with `initial-letter`; ui refused both. The svg package reads filter elements and runs them on the CPU over pictures it rasterises (decision [0150](0150-svg-filters-run-on-the-cpu-over-the-pixels-they-show-at.md)), but an element's picture is a GPU pass (decision [0118](0118-a-filtered-element-is-a-picture-drawn-through-wgsl-effects.md)), and ui never imports svg (decision [0087](0087-svg-is-read-by-embers-own-reader-imported-where-it-is-wanted.md)).

## Decision

- `url(#id)` in `filter` and `backdrop-filter` names a `filter` element in an `svg` element of the Document. The svg package reads it for the element's border box (`Picture.Filter`: units of the box, user space from its corner); ui/svg hands ui its own model of it (`ui/internal/fx`), so ui still imports no svg.
- paint runs it primitive by primitive on `twod.Canvas` pictures: each result a picture cut to its subregion, blurs, offsets, colour matrices, transfers and morphology as WGSL effects, composites through picture clips and sums, blends through `DrawPictureOver`, linearRGB as an effect into linear light and one back; the last result is cut to the filter's region. A picture cut by is first shown plain, as a clip reads the pass of its first showing. CSS functions before or after a `url()` filter the picture between. A list naming no filter element is left out whole; one in another file is refused.
- `initial-letter` on a `::first-letter` computes, in the cascade, as Chrome lays it out: the letter's font sized so its capitals span the cap height of the first line to the baseline of the size-th, floated at the line's start, as wide as its ink (fonts' glyph bounds, a negative text-indent for its bearing) and as tall as its capitals (a line height that puts the baseline at the bottom), its baseline on the sink line's by its margin; a letter raised above the first line pushes the lines down by an empty block before it. No sink, or `drop`, is the size rounded up, as Chrome's lines show. Since Chrome 158 it is the size rounded down, the letter rising above the first line by the fraction left, which pushes the lines down as a raised letter does; ui follows 158, and its Chrome comparison of fractional sizes runs from 158 on.

## Alternatives

- Read the element's pixels back and run svg's CPU filter: a GPU stall each frame and a texture upload back; the pure packages would also draw differently from the GPU path.
- A shader per filter graph, every input bound at once: needs textures in effects beyond the one picture, a GPU change.
- `initial-letter` in layout, as its own kind of box: exact for every writing mode, but a second float path where the float one already shortens the lines as Chrome does.

## Consequences

`filter: url(#id)` costs a pass per primitive and more for the pictures cut by; the steps between pictures are 8 bits a channel. A drop cap is laid out as any float, so selection, hit testing and painting need nothing new; vertical text and `initial-letter` on an element are not read.
