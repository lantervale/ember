# 0214. translate, rotate and scale are their own values before transform

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

CSS Transforms 2's `translate`, `rotate` and `scale` move a box as transform functions do, but compute, serialise and animate as values of their own, and grow a scroller's overflow as `transform` does, which ui never counted.

## Decision

- Each is a field of its own in the computed style (lengths, an angle in degrees, two factors, none unless set), and paint builds one matrix from them: translate, rotate, scale, then `transform`'s functions, about `transform-origin`, shared with hit testing and selection.
- They interpolate as their types, `none` as the identity; added they sum, and a scale multiplies, accumulated too, as Chrome does.
- Only what stays in the page is read: a z other than 0, a scale's z other than 1, or an axis other than z is refused with a warning, as `transform`'s 3D functions are; `0 0 -1` turns the angle the other way.
- Layout reads all four from the element's own style: a transformed child adds the rect round its turned box, and its overflow, to its scroller's scrollable overflow. A change of them lays out again; their animations, which only Shown holds, repaint.

## Alternatives

- Folding them into `transform`'s list at compute: one matrix path, but their values, interpolation and composition would be `transform`'s, not Chrome's.
- Leaving overflow untransformed, as before: no relayout on a transform change, but scrollers end short of what Chrome scrolls to.

## Consequences

A class toggling a transform now lays its element out again, as in Chrome; an animated transform still doesn't, so a scroller's overflow follows its keyframes only once they end.
