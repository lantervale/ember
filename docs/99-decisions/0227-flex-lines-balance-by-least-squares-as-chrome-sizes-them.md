# 0227. Flex lines balance by least squares, as Chrome sizes them

- Status: Accepted
- Date: 2026-11-25
- Area: rendering
- Owner: engine

## Context

`flex-wrap: balance` and `flex-line-count` (CSS Flexbox 2) split a container's items into lines as long as each other. The spec gives the partition but says nothing of a balanced container's intrinsic size, and has `flex-line-count` share the cross size among its lines even where they don't balance, which Chrome doesn't do.

## Decision

- Lines are found by dynamic programming over where each line ends: the least sum of the squared room each line leaves, the most items on the earliest lines on a tie, an item alone where it overflows, a zero-sized item kept at the end of the line before it where it fits.
- Their count is the greedy count, at least `flex-line-count` (no more than the items).
- With room for any length (max-content), a line count above one splits the items into that many lines with the longest as short as it can be: that line is the container's max-content main size.
- `flex-line-count` takes the gaps off the cross size and shares the rest among its lines, as each item's room across, only in a balanced container, as Chrome.

## Alternatives

- SMAWK, near linear, as the spec suggests: the quadratic program is plain and fast enough for the dozens of items a container holds.
- The spec's room across for every multi-line container: items would be narrower than Chrome draws them.

## Consequences

Balanced containers lay out as Chrome's on WPT's balance tests; a container of thousands of items in a few lines costs a quadratic split each layout.
