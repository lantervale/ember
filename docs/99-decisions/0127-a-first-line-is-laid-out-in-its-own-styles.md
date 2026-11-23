# 0127. A first line is laid out in its own styles

- Status: Accepted
- Date: 2026-11-22
- Area: api
- Owner: user

## Context

`::first-line` and `::first-letter` were refused (decision 0089), so drop caps and run-in leads in pages written for browsers lost their rules. Neither is an element: what a first line holds is known only once lines are broken, and a first letter is part of a text node, wherever in the block's inline elements, `::before` or first child block that text is.

## Decision

- Both pseudo-elements take Chrome's property lists (`ui/css/pseudo.go`), a test asking Chrome for their computed values.
- After the cascade and the counters, one pass over the tree, run while some sheet has such a rule or had one, keeps per block its first line's style (`Pipeline.FirstLine`, inheriting from the block), the style there of each inline element and `::before` on it (cascaded again over the first line's), the same for a first child block whose first line it is, and a node for the box behind the line where it has a background.
- Layout cuts a block's inline content twice, once in its own styles and once in the first line's, the same pieces of other widths. The first line is broken, sized and placed with the second, and its fragments carry their style (`Fragment.Style`), which paint and selection read.
- A first letter is a generated node of its block, as `::before` is, holding a text node of the letter and the punctuation round it; the text node it came from notes where it ends, and layout puts the node's box (a span or a float) before the rest of that text.

## Alternatives

- Break the first line in the run's own styles and restyle what lands on it: wrong wherever the first line's font is wider or narrower, the common case.
- Split each run on the first line into a text box of its own: selection, hit testing and the box tree assume one box a text node.
- Keep the first letter inside its text node's box: a float or a border round part of a fragment has no box to lay out or draw.

## Consequences

A Document with no such rule pays nothing; one with a `::first-line` cascades the inline elements of each first line twice. The first letter's text is the generated node's, so it is not selected with its text node.
