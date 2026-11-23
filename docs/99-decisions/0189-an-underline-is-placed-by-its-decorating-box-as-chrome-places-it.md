# 0189. An underline is placed by its decorating box, as Chrome places it

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

`text-underline-offset` and `text-underline-position` were not read: every underline lay half its thickness below the baseline, and `text-decoration-thickness: from-font` was `auto`. The spec leaves open whose values place a line an ancestor draws, and how `under` and the vertical sides round; Chrome's answers differ from a plain reading.

## Decision

- The offset, the position and the thickness are the decorating box's, carried with its decoration as Chrome's applied decorations carry them, whatever a descendant sets; a percentage offset is of the decorating font's size.
- On horizontal and sideways lines: `auto` lies half the thickness below the baseline, a given offset in its place (0 is on the baseline); `from-font` lies at the post table's underline position plus the offset, rounded; `under` a pixel past the em box's bottom, the em split by OS/2's typographic ascender and descender. `from-font` thickness is the post table's. All in device pixels, rounded as Chrome's layout units round, measured against Chrome on macOS.
- Vertical lines are underlined off their central baseline, half the em and a pixel away: on the left, on the right for `right` and for Japanese and Korean (by `lang`) unless `left`. Their decorations go along the whole line, upright characters included.
- A variable font's ink is skipped at the instance drawn.

## Alternatives

- Reading the offset and position from the text's own style, as the spec's prose suggests: Chrome keeps the decorating box's, and pages tuned on Chrome would move.
- Placing `under` below the lowest descender of the decorating box: Chrome uses the em box of each run.

## Consequences

Underlines match Chrome's to the pixel on the pages measured. The em box uses the default instance's OS/2 and post values, not a variable font's MVAR deltas; `under` and the right side ignore the lowest descender of the decorating box, as Chrome does.
