# 0173. Anchors are read from the last layout and laid out again until they settle

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

CSS Anchor Positioning places an absolutely positioned box by where other elements, its anchors, ended up, scrolled, in other containing blocks, possibly anchored themselves; ui's layout (a Taffy port) lays a containing block's positioned boxes out before it knows where anything stands on the page.

## Decision

- An `anchor()` or `anchor-size()` is a step of a length's `Calc`, the anchor it names kept in it; `@position-try` rules and flips are worked out by the cascade into one computed style per fallback.
- Layout places a box that reads anchors with where the last layout put them (border boxes in its containing block's space, scrolling applied), then works them out again from the boxes it made; where they moved, the boxes reading them are laid out again, retained layout redoing only them, up to four times for chains.
- A scroll that moves anchors lays their boxes out again, which tries their fallbacks again.

## Alternatives

- Resolving anchors inside one layout pass: the anchor's place on the page isn't known while its containing block lays out a positioned box, and chains and scrolled anchors would need positions the Taffy port has nowhere.
- Moving anchored boxes by the scroll without a layout, as Chrome's scroll adjustment does: their fallbacks and `position-visibility` would go stale.

## Consequences

A page with anchors lays out twice the first time its anchors move, once otherwise, and lays out its anchored boxes on each scroll that moves their anchors; anchors chained more than four deep settle over frames.
