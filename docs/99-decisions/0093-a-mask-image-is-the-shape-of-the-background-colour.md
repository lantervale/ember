# 0093. A mask image is the shape of the background colour

- Status: Accepted
- Date: 2026-11-18
- Area: rendering
- Owner: engine

## Context

Pages draw their icons as an empty element with a `background-color` and a `mask-image`: the picture gives the shape, the colour follows the theme. ui refused `mask-image`, kept the colour, and every icon of such a page (Wikipedia's header, tabs and search field) was a filled square.

## Decision

- `mask-image` takes `none` or one `url()`; `mask-size`, `mask-repeat` and `mask-position` are read as their `background-` twins. The mask is placed in the border box.
- An element with a mask paints its background colour as the picture's silhouette (white, the picture's alpha, made once on the CPU and cached by texture), tinted with the colour and tiled like a background image. No colour is painted until the picture is there.
- Only the background colour is masked. A background image, the border and the content draw whole.
- The `mask` shorthand, gradients, several layers, `mask-origin`, `mask-clip`, `mask-mode` and `mask-composite` stay refused, each with its warning.

## Alternatives

- Mask the whole element through an offscreen pass, as `twod.MaskedBy` does for a layer: what CSS says, but a pass per icon, dozens on one page, for a case pages do not use.
- Keep refusing and tell authors to use a background image: fine for a game's screens, but a page of the web cannot be rewritten, and the square is worse than nothing.
- Drop the background colour of a masked element: no square, and no icon either.

## Consequences

Icons made of a colour and a mask draw as in Chrome, at the cost of one more texture per mask picture. An element that masks its content or its border shows them whole, listed in [What is missing](../05-ui/10-missing.md).
