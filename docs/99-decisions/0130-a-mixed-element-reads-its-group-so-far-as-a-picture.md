# 0130. A mixed element reads its group so far as a picture

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

`mix-blend-mode`, `background-blend-mode` and `backdrop-filter` read what lies under an element: its backdrop, as far as its group (an isolated stacking context, a backdrop root or the page). The GPU's fixed blend states give only `multiply`, `screen` and `lighten`, and none of them over a translucent backdrop as CSS mixes; a pass cannot read its own target while it draws. Filters already draw an element apart as a picture (decision 0118).

## Decision

- paint draws a group as a picture: an element some descendant mixes with or filters the backdrop of (`box.Box.Group`, worked out with the ink), the page itself when that reaches it (`box.Tree.Grouped`), a filtered element's own content picture standing for it.
- Where an element mixes or filters its backdrop, paint ends the group's picture there: that picture is the backdrop. A new picture goes on with the group, the backdrop shown first. A backdrop filter shows the backdrop again through its filters in the element's border box, rounded (blurs mirrored at its edges, as Chrome's), then the element paints. A mixed element is drawn as a picture of its own, shown by `twod.Canvas.DrawPictureOver(pic, backdrop, mix, effects…)`: the 2D shader reads the backdrop as the batch's second image and mixes, in encoded sRGB as Chrome does, `(1 − αb)·Cs + αb·B(Cb, Cs)`, the 16 modes of Compositing and Blending.
- `background-blend-mode` does the same inside the element's background, a picture of its own, so its layers mix only with the colour and the layers under them.
- An element less than opaque that is drawn as a picture (filtered, mixed, a group, filtering its backdrop) has its content drawn opaque and the picture shown through an `opacity()` matrix: faded once, as Chrome does.
- Picture passes are drawn in the order their pictures end, so a picture comes after those it shows, the nested ones and the backdrop it reads.

## Alternatives

- Copy the target mid-pass in `gpu` (a backdrop texture per mixed element): no change to paint, but a split render pass per element in the presenter, and nothing for isolation or backdrop roots.
- Fixed blend states only: three modes, wrong over translucency.
- Framebuffer fetch: not in WebGPU.

## Consequences

A mixed element costs a pass for itself and one for its group's continuation; a backdrop filter one more. A group's backdrop is what its picture holds: in an isolated group that is no backdrop root, a backdrop filter reads that group only, and at the page, what the game drew under the Document is not read. `plus-lighter` and `plus-darker` are refused.
