# 0115. An img chooses its picture as Chrome does

- Status: Accepted
- Date: 2026-11-21
- Area: api
- Owner: user

## Context

Pages offer a picture through `srcset`, `sizes` and `picture` with its `source`s, and many give their `img` a tiny placeholder `src`, or none. ui read only `src`, so such pages showed placeholders or nothing, and the light browser chose a candidate itself from a rough reading of the attributes.

## Decision

- An `img` with a `srcset`, or in a `picture`, chooses its picture in ui, as HTML's source selection does in Chrome: the first `source` before it whose `srcset` offers one, whose `media` holds and whose `type` ui reads, else its own `srcset` with its `src`. `ui.Picture` and `ui.Source` are elements.
- `srcset` is read by HTML's parser; `sizes` and `media` by the css package, answered for the Document's viewport as `@media` is, with the initial font's units; `auto` is the width the lazy `img` was laid out at, and such an `img` is 300 × 150 points as Chrome's sheet sizes it.
- Chrome's rule, measured against it: the least density at or above the screen's pixels a point wins, else the greatest; a denser candidate already held (decoded by the Document, or a `data:` URL) wins over it, as Chrome's memory cache does.
- The picture is shown at its pixels over its density. It is chosen again when the viewport, the pixels a point, the attributes or the sources change; the picture shown stays until the one chosen in its place lands.
- A `type` is read when a decoder is registered for it with the image package, or a format of shapes with ui: no list of formats is kept apart from the imports.

## Alternatives

- Keep choosing in the program, as light did: every program reading pages would write its own, without the viewport, the screen's density or layout's widths, and the sizes would be wrong whatever it picked.
- A step up once the screen passes the geometric mean of two densities, a rule browsers have used: not the picture Chrome shows today (a 1.5x and a 4x on a 2x screen).
- Choose at style time in css: the img's laid-out width, which `sizes="auto"` needs, is not known there.

## Consequences

A page's pictures show at the size and sharpness Chrome gives them, the Document fetching only the candidate it chose. A denser picture once shown is kept when the window shrinks, as in Chrome. AVIF sources are passed over, and lazy loading is not done: listed in [What is missing](../05-ui/10-missing.md).
