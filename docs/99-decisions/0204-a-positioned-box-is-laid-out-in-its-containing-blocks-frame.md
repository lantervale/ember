# 0204. A positioned box is laid out in its containing block's frame

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Vertical writing modes are laid out turned ([0176](0176-vertical-writing-modes-are-laid-out-turned-into-their-own-frame.md)): each node's style is turned into the frame of its parent's content. An absolutely positioned box whose containing block is outside its parent, in another frame (a box of a vertical grid or block whose containing block is the page's horizontal one), was placed by that block with its style still in its parent's frame, sideways, and written back onto the page through its parent's frame. Its static place, turned up to that block, kept the edge it aligns to in its parent's frame, though the axis runs the other way on the page (the block start of `vertical-rl` is the right).

## Decision

- The containing block lays such a box out with its style turned into the block's own frame, the box's own content in its own as any orthogonal flow, and the box keeps that block (`placer`) to be written: its place in the block turned onto the page, less the places of its ancestors up to the block, written before it.
- A static position turned into another frame flips the edge it aligns to on each axis that runs the other way, as scroll overflow is turned.
- A table whose rows have no cell on a baseline (the plain layout, cells `vertical-align: middle` by default) has its first row's lowest cell's content-box bottom for its baseline, as Chrome's and CSS 2's: an inline table no longer falls back to the baseline synthesized across its margin box, central on a vertical line.

## Alternatives

- Lay positioned boxes out in their parent's frame and turn their insets and margins at the containing block: every inset, margin, size and alignment would be turned by hand, where turning the style once reuses the turned layout of an orthogonal flow.
- Keep the box's place relative to its parent in the containing block's frame: a parent in another frame has no place in it, the page is the one frame both share.

## Consequences

A positioned box escaping a vertical box, by its static place, its insets or its alignment in a grid, lands where Chrome 157 puts it in every writing mode, and an inline table stands on its first row as Chrome's on vertical and horizontal lines. A replaced box escaping that way still takes its natural size turned into its parent's frame.
