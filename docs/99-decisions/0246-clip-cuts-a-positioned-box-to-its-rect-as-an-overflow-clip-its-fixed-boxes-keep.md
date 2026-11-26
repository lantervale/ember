# 0246. clip cuts a positioned box to its rect, as an overflow clip its fixed boxes keep

- Status: Accepted
- Date: 2026-11-26
- Area: rendering
- Owner: engine

## Context

ui refused CSS 2.1's `clip` and pointed to `overflow: clip`; WPT's CSS 2.1 suite tests it in 44 pages, and pages from the web still hide text from sight with `clip: rect(0 0 0 0)`.

## Decision

- `clip` is `auto` or `rect()` of four edges, each a length or `auto`, split all by commas or all by whitespace (a mix is invalid, as Chrome reads it); it computes to the lengths and the `auto`s, which a `clip: inherit` keeps, so an inherited `auto` is the inheriting box's own edge.
- It applies to absolutely and fixed positioned boxes alone. The edges are measured from the border box's top-left, `auto` the border box's own edge, in the box's own space under its transform; a right edge left of the left one, or a bottom above the top, shows nothing.
- It is one more rectangle in the painter's clip chain, cutting the box's own background, border and outline and all it holds, and hit testing with it. Unlike an overflow clip it also cuts the fixed boxes inside, as Chrome keeps a CSS clip on them; it starts no stacking context.
- It transitions edge by edge when both ends are `rect()`s with their `auto`s on the same sides, and flips otherwise, as Chrome does.
- This supersedes the refusal of `clip` in the guide's table of what ui refuses.

## Alternatives

- Computing it as `clip-path: inset()`: that is a mask and a stacking context, and a clip-path never cuts what lies beyond the border box, where a `clip` rect may reach past it to show what overflows.
- Keeping the refusal: CSS Masking deprecates `clip`, but the owner wants all of CSS 2.1, and the cost is one clip in a chain the painter already keeps.

## Consequences

`clip` costs a rectangle push like `overflow: clip`; the only special case is that the fixed boxes inside keep it, which the painter carries across their containing block.
