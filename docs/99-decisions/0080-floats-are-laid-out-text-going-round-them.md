# 0080. Floats are laid out, text going round them

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

ui refused `float` and `clear` (decision 0049's subset: lay out with flex or grid). Pages written for browsers, which the web explorer shows as Documents, and plain prose round a picture both need them.

## Decision

- `float` (left, right), `clear` and `display: flow-root` are part of the CSS subset, with `text-indent`.
- Block layout ports Taffy's float placement (`compute/float.rs`), checked by its Chrome fixtures; ember's own line layout breaks each line in the room the floats leave at its height.
- A float written among inline content is a block-level child of its block container, placed before the text it was written in.
- Floats paint as CSS orders them: over the flow's backgrounds, under its text.
- Flex and grid stay the way to lay a screen out; the guide says so.

## Alternatives

- Keep refusing floats and rewrite them away in the explorer: every float-based page stacks its columns, and prose can't go round a picture.
- Place a float at the line it is written in, as browsers do when the line has room: the line breaker would place boxes while breaking, and a layout among floats could no longer be told from its inputs alone. Wrong only for a float after text in its own paragraph.

## Consequences

Layouts among floats are not cached, since they depend on the floats placed before them, so floats cost more than flex or grid for the same screen. A float in the middle of a paragraph stands a line higher than in Chrome.
