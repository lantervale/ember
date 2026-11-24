# 0218. A br clears from the end of its line, and blocks follow the floats where they were laid out

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A `br` with `clear` (or `<br clear>`) moved the next line below the floats only in the middle of its block's lines: where it ended them the block kept its height, so the blocks after it rose beside the floats, and ember ended its line before the elements the break closes, putting their end edges on the next line. Pages clear rows of floats this way (WPT's flex and grid tests). Blocks after a first block whose margins collapse out through their parent's top were also laid out in the context a margin off the floats before them, so they cleared them, or went round them, too low.

## Decision

- A forced break's line takes the end edges of the elements right after it, as Chrome's: a span holding a `br` ends on the br's line, with its padding and border.
- The br clears only where it is its line's last thing, as Chrome's: a br before the end of an element clears nothing. The line after it starts below the floats its `clear` names; with no line after it, the block's content reaches their bottom, so the block holds them and what follows goes below.
- Once a block in the flow is laid out, its parent's place in the formatting context is set from where the block was laid out and where it settled, when its first block's margins collapsed out through its top or its own place moved: the floats placed inside it and the blocks after it then agree.

## Alternatives

- Clearing at a br inside an element too, as CSS 2 reads: Chrome does not, and its pages are what ember is checked against.
- Laying a block out again once its margins settle, as Chrome resolves a block's place in the context lazily: a second layout of every such block, where moving the context's origin once costs nothing.

## Consequences

`<br clear=all>` between paragraphs and at the end of a block of floats lays out as in Chrome, and both WPT tests pass without a stand-in for the br. A line's pieces may now hold its forced break, which the passes over a line's pieces skip.
