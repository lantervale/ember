# 0209. Last baselines and safe alignment share groups as Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

ui read the alignment properties as one keyword: `last baseline`, `first baseline` and the overflow positions `safe` and `unsafe` made the declaration invalid. Its grid aligned rows on first baselines through a shim measured at the item's min-content width, its flex rows only on first baselines, and nothing aligned on a baseline across a column.

## Decision

- A computed `style.Align` carries its overflow position as bits or-ed into the keyword (`AlignSafe`, `AlignUnsafe`, read by `Position()`, `Safe()`, `Unsafe()`); `last baseline` is a keyword of its own, `first baseline` computes to `baseline`, and `anchor-center` drops its overflow position. The grammars are Chrome's: `align-content` takes no `last baseline`, `justify-content` no baseline, `align-items` no `left`; the `place-*` shorthands give their first longhand the longest start of the value it reads.
- Baseline-aligned items form two sharing groups a flex line or grid track: those packed at its start (first baseline, or last in `wrap-reverse`) and those at its end (last baseline, or first in `wrap-reverse`). The end group lines up the distance from each item's last baseline to its margin box's end, the group pushed to the end. A grid item joins the first-baseline group of the row it starts in and the last-baseline group of the row it ends in; its shim is the more margin on that side, measured once its columns are sized, as wide as they make it.
- Across a column (`justify-self: baseline`, a flex column) an item's baseline is synthesized from its line-left border edge; an orthogonal flow's own baseline is reported as `across` by its turned layout. In a vertical container whose text stands, a box without a baseline makes one up across its middle; in `vertical-lr` a line's baseline is measured from the frame's top, its over side mirrored.
- Flex and grid containers report a last baseline: that of their last line's (row's) last-baseline group, else of their endmost item. An inline-flex or inline-grid in a line still stands on its first.
- `safe` puts an item that overflows at the start, for content distribution, self-alignment and positioned boxes; a block container's `align-content` is safe unless `unsafe`, as Chrome's.

## Alternatives

- A struct of keyword and overflow position: every switch and comparison on `Align` in layout changes, where the bits leave the values they already compared equal.
- Keeping Taffy's min-content shim: an item laid out narrower than its column reports a baseline of more lines than it has.

## Consequences

`TestLastBaselineFlexItemsMatchChrome`, `TestLastBaselineGridItemsMatchChrome`, `TestSafeAlignmentMatchesChrome`, `TestBaselineAlignmentInVerticalModesMatchesChrome` and `TestAlignmentValuesComputeAsInChrome` hold. Code reading an alignment from a style takes its `Position()` before comparing it with a keyword that may carry an overflow position.
