# 0146. A right-to-left scroller scrolls from zero to below it

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A box of direction rtl lays its content out from its right, so what overflows it lies left of its padding box, where ui's scroll offsets, from zero to what the content leaves past the scrollport, could never reach. Chrome starts such a box at its right end and gives `scrollLeft` from zero down to minus that room. The rest of right to left that decision 0128 left (columns, collapsed borders, selections and carets, ellipses, mirrored characters, logical properties in animations, controls) follows the mirror it set, with no choice of its own.

## Decision

- A box's scrollable overflow keeps what lies before its padding box's start on an axis that scrolls from its end (direction rtl; a reversed flex box already did), and ends at the padding box's end there. The box carries `ScrollStart`, the least offset, below zero by what overflows its start, and every offset is kept between it and it plus the room past the scrollport: `doc.Scroll` and `doc.SetScroll` read and take Chrome's `scrollLeft`, the bars, the wheel, the keys and `ScrollIntoView` go through the one clamp.
- An over-constrained block in a container of direction rtl and a line too long for its box overflow the left, as CSS 2.1 and CSS Text have it.

## Alternatives

- Offsets from the left whatever the direction, the start at the far end: the program would read a box's offset differently from Chrome's and from `scrollLeft` in a page it exports, and every offset would move as the content grows.
- Laying the overflow out from the left and mirroring only the drawing: hit testing, selections and every box rectangle would need the mirror too.

## Consequences

A scroller of direction rtl starts at its right end and scrolls left by negative offsets, compared with Chrome box by box at several. Code that clamps an offset itself goes through `box.ClampScroll` or `paint.Bars.Clamp`; an offset below zero is no longer a mistake to correct.
