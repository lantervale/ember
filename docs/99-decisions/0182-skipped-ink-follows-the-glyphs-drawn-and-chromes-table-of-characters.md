# 0182. Skipped ink follows the glyphs drawn and Chrome's table of characters

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[Decision 0135](0135-small-capitals-and-skipped-ink-are-synthesised-from-the-faces-outlines.md) cut an underline round each character's own glyph and read `text-decoration-skip-ink: all` as `auto`. Since text is shaped ([0145](0145-text-is-shaped-by-a-pure-go-opentype-layout-reader.md)), the glyphs drawn are not always the characters' (a face's small capitals, ligatures, kerning), and Chrome's `auto` leaves some characters' ink whole, which `all` cuts.

## Decision

- Intercepts are taken from the runs as they are drawn: capitals turned into the face's features or synthesised, each run shaped by its plan, glyphs at their shaped advances and offsets.
- `auto` leaves whole the ink of the slashes, the low line, Hangul, Linear B and what Chrome's `IsCJKIdeographOrSymbol` names; `all` cuts round every glyph but the low line's. The table is Chrome's on macOS, measured by drawing every character of the CJK, symbol and pictograph blocks under both values and keeping those that differ, the CJK blocks whole.
- Lines through are never cut, under either value, as in Chrome.
- Small capitals are drawn as runs wherever text is drawn outside a line box: a select's list and a broken picture's alt text, set on the baseline of the text round them.

## Alternatives

- Generating the table from Unicode's East Asian Width or script properties: shorter, but it differs from Chrome's on symbols (arrows, daggers, circled numbers) a page can hold.
- Keeping one glyph per character for intercepts: no shaping on the decoration path, but gaps where the glyph drawn is another (a small capital, a ligature).

## Consequences

Underlines skip ink as Chrome's do on every face ui shapes; the table is a snapshot of Chrome on macOS, and a character outside the blocks measured is skipped as a Latin letter.
