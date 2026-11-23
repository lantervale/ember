# 0137. A box placed past a scroller is moved back as it scrolls

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: user

## Context

The box tree follows the document, so an absolutely positioned box whose containing block is above an element that scrolls, and a fixed box inside one, are children of that element's box, and every walk of the tree (paint, hit testing, `Box.Rect`, culling) subtracts its scroll offset from them. Chrome places them by their containing block alone: they stay put while that element scrolls.

## Decision

- Such a box takes the scroll of the elements between it and its containing block into its position, the way a sticky box takes its offset and a fixed box a scrolled viewport's: `Box.Sticky` holds how far, `Pos` includes it.
- The box is live (`Tree.Live`), so `layout.Refresh` moves it again each time those elements scroll, with no layout.

## Alternatives

- Skip those scroll offsets in each walk: paint, hit testing, the selection, culling and `Box.Rect` would each climb to the containing block for every positioned box, and one forgetting would put it elsewhere.
- Lay the tree out again on each scroll: a layout per wheel step, where a scroll costs a Refresh now.

## Consequences

Every walk sees the box where Chrome draws it without knowing why; a page with many such boxes in a scroller has as many live boxes for Refresh to visit.
