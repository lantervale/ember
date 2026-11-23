# 0184. A control's generated content is laid out round a box of its parts

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Chrome lays a progress's, a meter's and a range's `::before` and `::after` out among the elements of their shadow tree: a progress's inner element is a block between the lines of the two, a range is a flex row of its `::before`, its slider container and its `::after`. ui gave these controls no generated boxes, drew their parts in the content box, and set a range of `appearance: none` as tall as its track.

## Decision

- Layout builds a box of kind `ControlPart` for a progress, a meter or a range that has a `::before` or an `::after`, between them: a progress's or a meter's a block as tall as its content box (`height: 100%`), a range's a flex item growing from 0 in a row the range is laid out as, its inline generated content in anonymous flex items. A control without them is laid out as before.
- Paint draws the parts of a control of `appearance: none` in that box (a range's track along it, down the content box), under its generated content; one of ui's own look keeps its content box.
- A range of `appearance: none` whose height is auto is at least as tall as its track through its `min-height`, so its generated content can make it taller; its baseline stays at its bottom.
- An inline-block of blocks stands on its last line's baseline (a scroll container among them giving its bottom), as Chrome's, where ui took its first.

## Alternatives

- Lay out a shadow tree of real nodes for every such control, as Chrome: every progress, meter and range would carry three to four boxes more, and their parts' styles would have to be re-cascaded as elements.
- Place the generated boxes in paint round the parts: hit testing, selection and `Node.Box` of the pseudo-elements would not see them.

## Consequences

The generated boxes and the parts land where Chrome puts them; a part's box exists only while the control has generated content, so `::-webkit-slider-container` and the parts' own `:hover` and `:active` still match nothing.
