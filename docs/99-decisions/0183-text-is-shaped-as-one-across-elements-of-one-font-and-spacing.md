# 0183. Text is shaped as one across elements of one font and spacing

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Chrome shapes the text of an inline formatting context in runs that go on across the edges of inline elements whose font, `letter-spacing` and `word-spacing` match, unless an edge takes room (a non-zero margin, border or padding on its side) or the element stands off the baseline (CSS Text's boundary shaping). A ligature or a kerned pair across such an edge (`of<span>fi</span>ce`, `A<span>V</span>`) is one glyph run, split by character: a glyph belongs to the element its cluster starts in, so the `fi` span is empty and the ligature takes the first element's colour; a caret inside the span stands at the ligature's end. Spaces are shaped with their words, so a ligature or a contextual alternate reaches across them. ui measured and drew each element's words apart, giving another element's text only as HarfBuzz context.

## Decision

- Layout numbers the pieces of text Chrome shapes as one (left to right, horizontal), and measures a word or space beside another element's text in that run, or any of them in a face whose layout reads a space, as the advance of its own glyphs in its text shaped together with up to five characters on either side (`box.WholeFonts.AdvanceAmid`). The text round those is HarfBuzz context, as before.
- A fragment carries the text it is shaped as one with on its line (`Fragment.Lead`, `Trail`); twod draws it through the new `twod.Amid(lead, trail)` option, which shapes the three as one and draws and measures only the glyphs whose clusters start in the text. Selection edges are measured the same way.
- `shape.Plan.Mentions` also counts the glyphs a context lookup reads round the one it starts at, so a contextual alternate across a space ties its words.

## Alternatives

- Shaping each paragraph once and cutting the result into fragments, as Chrome does: the closest match, but layout, paint and selection measure text by strings through `box.Fonts` and twod, and a paragraph's shaping would have to travel with the boxes.
- Merging the fragments of elements shaped as one into one draw: their colours, decorations and backgrounds differ, which Chrome keeps per element.

## Consequences

Pieces away from an edge, in faces that don't read spaces, are measured as before. Text right to left or vertical, and line ends that cut a ligature, stay as they were ([missing](../05-ui/10-missing.md)).
