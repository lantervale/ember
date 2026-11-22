# 0120. Container queries restyle after layout until their answers hold

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: engine

## Context

`@container` and the `cq*` units make a style depend on a box layout gives: the cascade ran once a frame before layout, and nothing it read came back from it. A component that changes shape with the room its panel leaves had to be written in Go.

## Decision

- `container-type` (`normal`, `size`, `inline-size`), `container-name` and the `container` shorthand, `@container` with an optional name, a list of conditions, the size features (`width`, `height`, `inline-size`, `block-size`, `aspect-ratio`, `orientation`, `min-`/`max-`, ranges, `and`/`or`/`not`) and `style()` on custom properties are part of the subset, always on, as `@media` is (decision 0086).
- A condition asks the nearest ancestor (the element itself for its pseudo-elements) that has its name and whose type answers every size feature it reads; a size it doesn't know is unknown, false unless a later layout knows it. `cqw`, `cqh`, `cqi`, `cqb`, `cqmin` and `cqmax` measure the nearest container on their axis, the viewport with none.
- Layout honours containment as Chrome does: a size container's size, an inline-size container's width, are what they would be with no content, its content laid out in them; every container is an independent formatting context. Inline boxes, tables and their parts are no size containers.
- The loop: style, lay out, then `Transitions.Requery` reads each container's content box. Only a container whose box changed is asked, and it restyles what it holds only when one condition of the sheets answers otherwise for its new box than its old, or a length under it read a container unit. Then style and layout run again, in the same `Document.Layout`, at most eight times: once a level of nesting.
- `scroll-state()`, `style()` on standard properties and a `scroll-state` container type are refused with a warning; a page exported for Chrome carries its `@container` rules.

## Alternatives

- Interleave style and layout per container, as Chrome does: the cascade and the Taffy port each run on a whole tree, and splitting them would rewrite both.
- Restyle the whole tree whenever any container resizes: a window dragged by its edge would restyle every frame where most resizes cross no boundary.
- Answer queries from the last frame's boxes and never loop: every crossing would show one wrong frame, and the first frame of every screen would be wrong.
- Keep refusing them: components keep reflowing from Go code that reads boxes and swaps classes.

## Consequences

A frame that crosses a container's boundary styles and lays out twice or more, the restyle limited to that container's subtree; one that crosses none costs a comparison per resized container. A chain of containers nested deeper than eight levels, each flipping the next, is left at the eighth answer.
