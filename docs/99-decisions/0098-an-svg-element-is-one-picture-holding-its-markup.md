# 0098. An svg element is one picture holding its markup

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: user

## Context

ui drew SVG only from a file, in an `img` or a `url()`. Pages write their logos and icons as `svg` elements in the page and size them with rules naming `svg`: bbc.com/news showed neither its logo nor its menu, and every such rule was dropped.

## Decision

- `ui.Svg(markup, args…)` is an `svg` element: a replaced element as `img`, whose picture is the SVG it holds as text, read by the format `ui/svg` registers and drawn at the size the element shows at.
- The element is in the tree and the cascade: rules match `svg`, its classes and its attributes; `width` and `height` are hints in any CSS unit, `viewBox` gives the ratio, and with none of them it is 300 by 150.
- `currentColor` in the markup is the element's computed `color`: the picture is the markup under a root carrying the element's attributes and that colour, and is made again when the node or its colour changes. `n.SetMarkup` changes what it holds.
- The shapes inside are not nodes: no selector reaches them and they take no events.

## Alternatives

- Every shape a node of the tree, styled by the cascade: what a browser does, and a second layout and paint model (paths, transforms, paint servers) in ui for what pages mostly use as icons.
- No element, the browser turning each `svg` into an `img` with a data URL: no change in ui, but `svg { height: 30px }` then matches nothing, which is how pages size them.
- A colour argument on `ui.VectorImage`'s `Rasterise`: currentColor without rewriting the root, for a format-wide interface change that only this element needs.

## Consequences

A page's inline icons show, sized and coloured by its sheets, with what the svg reader draws. A rule aimed at a shape inside (`svg path { fill: … }`) does nothing, and a script animating an svg's markup makes one picture per state; both are listed in [What is missing](../05-ui/10-missing.md).
