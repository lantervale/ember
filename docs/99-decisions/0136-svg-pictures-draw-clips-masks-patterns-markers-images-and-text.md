# 0136. SVG pictures draw clips, masks, patterns, markers, images and text

- Status: Accepted
- Date: 2026-11-23
- Area: assets
- Owner: engine

## Context

ember's svg reader ([0087](0087-svg-is-read-by-embers-own-reader-imported-where-it-is-wanted.md)) left out `clip-path`, masks, patterns, markers, symbols, nested `svg`, `image` and text, so real logos (most clip) showed wrong. Each needs a way to be drawn that keeps the reader pure Go, bounded, and a model another renderer could draw.

## Decision

- The model grows, not the reader's secrets: a `Group` carries a `Clip` (shapes whose fills are unioned, cut by a further clip) and a `Mask` (items drawn apart, read by luminance or alpha, cut to a region); a `Paint` may be a `Pattern` (a tile of items in its own space); an `Item` may be an `Image` (premultiplied pixels and a transform).
- Rasterising draws a clipped or masked group in a layer, then fades it pixel by pixel by the clip's coverage and the mask's luminance; a pattern's tile is drawn once at the size it shows at and sampled, wrapping, under the shape's coverage.
- Symbols, nested `svg` elements and markers become groups clipped to their viewports, built with the transforms their attributes ask for; an SVG `image` is read as a picture of its own (four deep at most) and moved into place.
- `image` draws only what a `data:` address holds (PNG, JPEG, GIF, WebP, SVG): nothing is fetched.
- Text is set in Go's faces (the engine's, parsed by x/image's sfnt), the monospace one for a family naming one, and becomes shapes painted as any path: positions, `tspan`, `text-anchor`, kerning; no shaping.

## Alternatives

- Rasterising clips and masks as extra paths cut geometrically: exact edges, but path booleans are a large, fragile body of code, and Chrome itself masks by coverage.
- Text drawn through twod's faces: the same letters, but svg would import ember and stop being usable on its own and by servers.
- Fetching `image` files through a callback: the reader then depends on its caller's I/O; data: addresses cover icons that embed their pictures.

## Consequences

Logos with clips, masks, patterns and markers draw within a level or two of Chrome on every sample. Text reads close to Chrome but in other letters, and the reader now depends on x/text through sfnt.
