# 0200. Positioned boxes are lanes beside the flow, and forced breaks in lanes start Chrome's runs

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0185](0185-boxes-side-by-side-are-lanes-and-a-framed-wrapper-is-laid-out-per-part.md) left a positioned box beside the lines of its containing block one unit with them, cut only where their lines lined up, and a forced break inside one of boxes side by side its own, balanced as if it were not there. Chrome cuts the positioned box by its own content, beside the flow, and balances forced breaks by content runs that hold the boxes beside them whole.

## Decision

- An absolutely positioned box beside the flow is walked apart, a lane of its own, filled into the columns from the column its top shows in, where its own content allows. Balancing counts it as a lane beside the flow, its lines' breaks never merged with the flow's; columns of a set height it doesn't fit get more columns past the container, as Chrome's overflow columns. Inside boxes side by side it stays one unit with their lines.
- Boxes side by side one of which forces a break are always cut as lanes, and balancing starts from Chrome's content runs: the flow between forced breaks, each run as deep as all it holds goes, a lane without a break of its own wholly in the run it starts in, the container's bottom edges below each; the columns past the runs go to the tallest, then the shortage search goes on as before.
- The room below a flex or grid item's content stretched to another's starts no column; such an item shows in every column the row does, as Chrome stretches a row's parts, and one whose own height ends before a break beside it fills its column less its bottom edges, which end its last part. A table cell is not cut past its content: it fills the row's part of its column.

## Alternatives

- Keep the positioned box one unit with the lines beside it: it stays whole where Chrome cuts it.
- Keep balancing from the flow's height shared by the columns: a forced break in one item leaves the others cut where Chrome keeps them whole.

## Consequences

Positioned boxes beside lines, forced breaks in flex, grid and table rows, and stretched items that end early break as in Chrome. A positioned box inside a flex or grid item or a cell still goes with the lines beside it.
