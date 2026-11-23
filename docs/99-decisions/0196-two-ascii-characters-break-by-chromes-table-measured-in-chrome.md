# 0196. Two ASCII characters break by Chrome's table, measured in Chrome

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Between two printable ASCII characters Chrome never asks ICU where a line may break: Blink reads a table of its own, kept for compatibility with old browsers, with no break after `!`, `}` or `|` before a letter, none inside `$$` or `%%`, one before `(`, `[`, `{` or `<` after most punctuation and one after `-` and `?`. ui broke such pairs as UAX #14 has them, so code, URLs and punctuated Latin text broke where Chrome keeps them whole and the other way round.

## Decision

- The table is measured, not copied: `go generate` in `ui/layout` runs a program that lays every pair out in headless Chrome in a box of no width, under `word-break: normal` and `break-all`, and writes the two 94×94 bit tables with the Chrome version they came from.
- `softBreaks` leaves UAX #14's answer out wherever both characters are printable ASCII and takes the table's instead, across elements too; a hyphen before a digit breaks only after a letter or a digit, as Blink keeps a minus sign with its number.
- `word-break: break-all` takes the break-all table between two such characters, and its own rules elsewhere.
- The rule that kept `a/b` whole is gone: the table keeps it, and Chrome does break after a slash before a letter beyond ASCII.

## Alternatives

- Copying Blink's table by hand from its source: as exact, but nothing checks it against the Chrome the tests run.
- Reclassing ASCII characters before uniseg, as decision 0195 does for strictness: no class gives the table's rows, which are per character.

## Consequences

Text in ASCII breaks exactly where Chrome breaks it. A newer Chrome that changes the table needs the generator run again; the Chrome suite on every pair says when.
