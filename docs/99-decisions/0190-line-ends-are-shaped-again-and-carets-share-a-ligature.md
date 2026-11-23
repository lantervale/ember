# 0190. Line ends are shaped again and carets share a ligature

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Text shaped as one ([decision 0183](0183-text-is-shaped-as-one-across-elements-of-one-font-and-spacing.md)) kept its widths wherever a line broke, broke inside ligatures, and placed a caret inside a ligature where the characters before it measured alone. Chrome breaks no line inside a ligature that opens it, shapes the word that starts a line again without the text before it and the end of a line broken inside a word without the rest of it, and shares a ligature's advance evenly among its grapheme clusters. Its font description also holds the language, `text-rendering` and `-webkit-font-smoothing`, so elements differing in them are shaped apart.

## Decision

- `fonts.Set.Carets` (`box.CaretFonts`) measures each grapheme cluster of a text by the glyphs whose clusters start in it, shaped whole: a pair's kerning is its first's, a ligature's advance is shared evenly among the clusters it draws, and it reports where no glyph starts. Selection edges and `word-break`/`overflow-wrap` pieces use it.
- No line breaks between the graphemes of a ligature, nor at a space a ligature draws on the line whose first word starts that ligature; elsewhere a line may break inside one, which keeps its glyph at the line's end.
- A word piece carries what its width gains where a line starts at it (`cutStart`) or breaks inside its word after it (`cutEnd`); lines are broken with the shaped-whole widths and then take these, in their widths and their pieces' placing. A line ending at a space keeps its shaping, as Chrome's.
- Runs are shaped as one only within one `lang`, `text-rendering` and smoothing; `text-rendering` is read, and `optimizeSpeed` turns off the common ligatures and contextual alternates `font-variant-ligatures` doesn't name.

## Alternatives

- Reshaping each line's ends while breaking and fitting the line again, as Chrome: exact where a reshaped end changes what fits, but the breaker measures by pieces and would need the shaping at every candidate break; the difference is a kerning or a ligature's share.
- Splitting a ligature's advance by its characters' own advances: simpler to read off the shaping, but not where Chrome puts the caret.

## Consequences

A reshaped line end that would no longer fit, or would now fit more, breaks where the unreshaped widths broke it. Text right to left or set vertically is still shaped apart ([missing](../05-ui/10-missing.md)).
