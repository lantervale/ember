# 0133. A masked element is a picture cut by the picture of its mask

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

Decision 0093 drew a `mask-image` as the shape of the background colour alone, enough for icons. Pages also fade lists and cards out with gradient masks, cut holes with several layers and `mask-composite`, draw gradient borders with Chrome's `-webkit-mask-composite: xor`, and mask whole elements, text and children included. CSS masks the element and all it holds as one image, after its filter and its clip-path, and a gradient's partial alpha makes the difference between masking that image and masking each shape apart wherever two overlap.

## Decision

- Every mask longhand takes a list of layers, read as the background's are: `mask-image` (`none`, `url()`, gradients), `mask-position`, `mask-size`, `mask-repeat`, `mask-origin` and `mask-clip` (the border box by default, an SVG shape's boxes as CSS maps them, `no-clip`), `mask-mode` and `mask-composite`; the `mask` shorthand sets them. Under `ui.BrowserSheets()`, `-webkit-mask` and `-webkit-mask-position` are read as the plain ones, and `-webkit-mask-composite`'s `source-over`, `source-in`, `source-out` and `xor` as `add`, `intersect`, `subtract` and `exclude`.
- A masked element and its content are drawn into one twod picture, as a filter's (0118), its clip-path inside it. Its mask's layers are drawn into a second picture by the code that draws background layers, in their own colours; a `luminance` layer goes through a picture of its own and an effect that makes its luminance (Chrome's coefficients, on sRGB) its alpha. The first picture is shown cut by the second's alpha: `twod.Canvas.PushClipPicture` cuts by a picture's pass as `PushClipMask` (0119) cuts by a texture.
- Layers join from the last up. `add` draws over the layers under it; `intersect` and `subtract` draw cut by those layers' picture, `subtract` inverted (`twod.Inverted()`, where all outside the picture shows); `exclude` draws each cut out by the other and sums them (`SumPicture`, a new `Plus` blend). The last layer's word has nothing to join, and a layer of no picture is left out, as in Chrome. A picture always holds the pictures it shows, so the passes keep their nested order.
- An element whose mask is one picture's alpha and which paints its background colour alone keeps 0093's silhouette: the picture tinted, no pass. A translucent masked element is faded as one picture.
- Masking starts a stacking context. As in Chrome, a masked element's content is hit in its border box alone, square, whatever the mask shows.

## Alternatives

- Draw the whole mask on the CPU into a texture: one cut and no new blend, but gradients, tiling, `space` and `round` drawn a second time, apart from the backgrounds Chrome is matched on, and a new picture each frame for an animated mask.
- Multiply each shape by the mask in place, with no picture of the element: no pass, but overlapping shapes under a partial alpha come out darker than Chrome, the text over a faded card's background first of all.
- Porter–Duff blends on the GPU for each word: fewer passes, but a blend cannot clear what a layer's shapes leave uncovered, which `intersect` must.

## Consequences

A masked element costs two passes, more for each layer joined by another word than `add` or drawn by its luminance, and its Document repaints each frame as a filter's does; icons keep costing none. A mask inside another element's clip-path replaces that clip-path's cut, which then clips by its bounds, as a nested clip-path does (0119). Supersedes [0093](0093-a-mask-image-is-the-shape-of-the-background-colour.md).
