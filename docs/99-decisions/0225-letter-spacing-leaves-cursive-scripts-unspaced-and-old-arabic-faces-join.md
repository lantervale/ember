# 0225. Letter-spacing leaves cursive scripts unspaced, and old Arabic faces join

- Status: Accepted
- Date: 2026-11-24
- Area: rendering
- Owner: engine

## Context

`letter-spacing` parted Arabic letters that Chrome keeps joined, and Arabic in a face with no `init`, `medi` or `fina` (only the presentation forms of its cmap, as faces made before OpenType) was drawn letter by letter where Chrome's HarfBuzz joins it.

## Decision

- Letter-spacing follows each grapheme cluster (no longer each character), but none of a run of a cursive script, CSS Text's list as Chrome has it: Arabic, Syriac, Mongolian, N'Ko, Mandaic, Phags-pa and Hanifi Rohingya. Their spaces and no-break spaces stay spaced. A cluster is of its first character's script, else of the run before it, the text before a word included (`shape.EachSpaced`); layout, carets and drawing count alike.
- Optional ligatures and `calt` stay off under letter-spacing for every script, as Chrome turns them off whatever the script.
- Arabic in a face with none of the joined forms' features takes HarfBuzz's fallback: single and ligature lookups built from the presentation forms its cmap maps, with HarfBuzz's tables (`gen-arabic-table.py`, ported into `internal/shape/gen`), applied after `rlig`.

## Alternatives

- Spacing by Unicode script of each character: Chrome spaces by the shaping run's script, so a full stop after an Arabic letter is unspaced and one after a Latin letter spaced.
- Leaving old faces unjoined: what such a face shows in Chrome is joined, and the fallback is a few hundred lines over the lookups ember already applies.

## Consequences

A cursive run's text measures and draws as Chrome's. Characters of no script whose script extensions are Arabic (the Arabic comma, tatweel) in a Latin run stay spaced where Chrome starts an Arabic run for them, and Windows-1256 faces (HarfBuzz's other fallback) are not joined.
