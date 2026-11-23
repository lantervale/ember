# 0128. Right to left is a mirror of the layout, and bidi orders each line

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

`direction: rtl`, the `dir` attribute and `unicode-bidi` reach every layout ui has (lines, blocks, flex, grid, tables) and the cascade (logical properties), and mixed Hebrew, Latin and numbers need the Unicode Bidirectional Algorithm, which golang.org/x/text implements without giving out the embedding levels.

## Decision

- css resolves `direction` before the other longhands; a logical declaration of an inline side is kept as its left one and marked, and the cascade puts it on the right one for an element of direction rtl, so it competes with the physical one in cascade order.
- Flex, grid and table containers of direction rtl lay out their mirror image with the Taffy-ported algorithms untouched (their own left and right padding and border swapped, their children's margins and insets), then place each child back as in a mirror; block layout places an over-constrained child at its end and an absolute box's static place on the right itself.
- Each inline formatting context reads its runs as one text, atoms as U+FFFC and each inline element's `unicode-bidi` as the bidi controls round it, and resolves its levels with ui's own UAX #9 (`ui/internal/bidi`, P2 to I2 with bracket pairs, L1 and L2, x/text's character classes); words are cut where levels change, each line is reordered by L2, and a fragment shown right to left keeps its text in order, `Fragment.Shown` drawing it reversed by grapheme, its brackets mirrored. An inline element's edges stand on the sides its own direction gives them, as Chrome's.

## Alternatives

- Taffy's own direction support: it has none yet. Rewriting each algorithm with start and end sides: every line of 6,000 ported ones touched, and the Taffy fixtures no longer a check.
- x/text's `Paragraph.Order`: it gives runs by direction alone, not levels, so numbers in right-to-left text and nested embeddings land wrong; vendoring its core would carry 1,400 lines of foreign style.
- Fragments holding their text reversed: selection, carets and copying read fragments in text order.

## Consequences

Right-to-left pages lay out as in Chrome for text, blocks, flex, grid, tables, lists and positioned boxes, checked box by box; a mirrored container costs a copy of its children's styles while it lays out. Text is not shaped, so Arabic is drawn unjoined, and selecting inside a right-to-left run measures it from its left.
