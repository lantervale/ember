# 0219. A transform holds fixed boxes, and a point hits through it as its pixel square

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Decisions 0214 and 0216 left a transformed element short of Chrome's in four ways: its fixed and unpositioned absolute descendants placed in the viewport, its overflow counted untransformed in vertical scrollers, lines and the root, an animation between transform lists of different lengths moved as matrices, and a point a pixel before a turned edge missed it.

## Decision

- A `transform`, `translate`, `rotate` or `scale` other than none makes its element the containing block of its fixed and absolute descendants, as `filter` does (`style.Computed.HoldsFixed`), for layout, scrolling and clips alike.
- A transformed box's scrollable overflow is turned on the page: its rect is taken out of the frame its parent lays it out in, turned, and put back; atomic inlines add theirs to their block's; the root's turns the viewport's. The bars on a vertical frame's starting sides are no overflow, and its text starts past them.
- Transform lists interpolate as CSS Transforms 2 says: the shorter padded with the identities of the longer's last functions, the common start function by function, the rest as matrices. A missing keyframe under `add` then moves as Chrome's.
- Under a transform other than a move by whole pixels, a point hits a box when the pixel square from it (right and down), taken back into the box's space, overlaps the box, as Blink carries a point through a transform as that square; culling tests ink the same way.

## Alternatives

- Growing every turned box by a pixel and a half for hit testing: close at its sides, wrong at its corners and under scales, where Chrome's reach follows the square.
- Flooring the point before making its square: Chrome's hits at half pixels show it isn't floored.

## Consequences

Edges of transformed boxes hit as Chrome's, within about 0.4 pixel at a turned corner, which Chrome reaches a little further for reasons not found. Fixed boxes under a transformed element scroll with it, as in Chrome.
