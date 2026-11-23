# 0150. SVG filters run on the CPU over the pixels they show at

- Status: Superseded by [0161](0161-svg-filters-run-in-the-elements-own-space-with-every-primitive-chrome-draws.md) for a filter under a rotation or a skew, which runs in the element's own space; the rest stands
- Date: 2026-11-23
- Area: assets
- Owner: engine

## Context

Logos and icons use `filter` for drop shadows and glows, `mix-blend-mode` for overlapping colours, `vector-effect` for hairlines that stay thin, and `clip-path` with a basic shape; decision [0136](0136-svg-pictures-draw-clips-masks-patterns-markers-images-and-text.md) left them out, so such files drew without their shadows or blends. The svg reader is pure Go and its model is meant for other renderers too, so a filter needs a form in the model and a way to run that stays on the CPU.

## Decision

- The model grows: a `Group` carries `Filters` (each a region, the transform of the filtered element's user space, and `Primitive`s reading `SourceGraphic`, `SourceAlpha` or earlier results by index) and a `Blend` mode; a `Shape` may be `NonScalingStroke`. Units, subregions, `in`/`result` names and `color-interpolation-filters` are resolved when the file is read.
- A filtered group is drawn in a layer over its filter region, wider by what its primitives can spread (blur, offsets, radii), then run primitive by primitive on premultiplied float pixels, in linear light unless the primitive asks for sRGB. Lengths go to pixels by the transform's scale along each axis. Regions and subregions are whole pixels, rounded out, as Chrome cuts them; the filtered element's picture is drawn past the region, as Chrome reads it.
- Blurs are a Gaussian kernel below two pixels of deviation and three box blurs from there, as Filter Effects describes them. The primitives drawn are `feGaussianBlur`, `feOffset`, `feFlood`, `feColorMatrix`, `feComponentTransfer`, `feComposite`, `feMerge`, `feBlend`, `feDropShadow`, `feMorphology` and `feTile`; any other makes a clear result and is named in `Unsupported`.
- CSS filter functions on an element are primitives of a filter without a region, in sRGB; `drop-shadow()`'s blur is its deviation, as Chrome draws it in SVG.
- `mix-blend-mode` composites the group's layer with CSS compositing's formulas over what is under it within the nearest layer; `isolation: isolate` makes a layer.
- `clip-path` of a basic shape or a box keyword is a clip of one shape in the element's fill, stroke or view box.
- Text gains `letter-spacing`, `word-spacing`, `text-decoration` (at the face's post and OS/2 positions, painted per character as Chrome paints them) and `textPath` (each glyph's middle on the path, turned with it, hidden past its ends). A UTF-16 file, by its byte order mark or its first `<`, is read as UTF-8.

## Alternatives

- Filters on the GPU through twod's effects: faster, but svg would import ember and could not rasterise on a server or in a test without a device.
- Filtering in the filtered element's own space and transforming the result: exact under rotation, at the cost of a second resampling for every picture; icons are rarely rotated.
- Eight-bit buffers between primitives, as Skia keeps: less memory, but each step in linear light would lose the darks.

## Consequences

The filter samples (blurs, shadows, glows, colour matrices, transfers, composites, blends, units and subregions) draw within half a level of Chrome on average at 256 pixels. A filter under rotation blurs along the pixels' axes; turbulence, lighting, displacement, convolution and `feImage` are not drawn; a filter is as slow as its region is large.
