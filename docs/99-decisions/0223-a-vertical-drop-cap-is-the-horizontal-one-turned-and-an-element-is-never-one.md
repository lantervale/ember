# 0223. A vertical drop cap is the horizontal one turned, and an element is never one

- Status: Accepted
- Date: 2026-11-24
- Area: rendering
- Owner: engine

## Context

Decision [0165](0165-a-filter-element-runs-on-the-gpu-over-an-elements-picture-and-a-drop-cap-is-a-trimmed-float.md) made a `::first-letter` with `initial-letter` a trimmed float, in horizontal text alone. CSS Inline 3 also gives `initial-letter` to the first inline child of a block and adds `initial-letter-align`; vertical text is laid out turned into its own frame (decision [0176](0176-vertical-writing-modes-are-laid-out-turned-into-their-own-frame.md)).

## Decision

- In a vertical block the cascade writes the drop cap's float in the frame the layout turns it into: its capitals' height is its width, its margin, padding and border on the side its blocks start from (right in `vertical-rl` and `sideways-rl`, left in `vertical-lr` and `sideways-lr`) place its baseline, and its ink sizes it along the line.
- The ink along the line follows `text-orientation`: a turned letter's is its glyphs' horizontal bounds; one standing upright (an ideograph under `mixed`, every letter under `upright`, none in the `sideways-*` modes) is its glyphs' vertical bounds down the room its font gives it, as Chrome sizes it.
- A drop cap reads its fonts' capitals, so a font landing styles it again.
- `initial-letter` on an element is read and computed and lays nothing out, and `initial-letter-align` is refused with a warning: Chrome 158 does both, whatever CSS Inline 3 says.

## Alternatives

- A drop cap laid out by the layout in each frame: a second path where turning the style keeps the one float path Chrome is matched on.
- `initial-letter` on a span or an image as the spec has it: it would lay pages out where Chrome doesn't.

## Consequences

Vertical drop caps land where Chrome's do, in every writing mode, with no layout change. An inline element holding a drop cap still has a box without it, where Chrome's `getBoundingClientRect` takes it in (10-missing).
