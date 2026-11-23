# 0145. Text is shaped by a pure-Go OpenType layout reader

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Chrome shapes text with HarfBuzz: a face's GSUB gives ligatures, real small capitals and other figures, its GPOS kerns by classes of glyphs and with spaces. ember drew each character's own glyph and kerned through x/image's `Kern`, which reads a GPOS pair only when it moves the first glyph's advance, and measures words apart: `font-variant-*` and `font-feature-settings` did nothing, and a page font's lines came out wider than Chrome's (Arial's "Hello AVATAR" by a point and more, every space it kerns).

## Decision

- `internal/shape` reads GDEF, GSUB and GPOS itself, every offset checked, and applies them as HarfBuzz's default shaper does: the lookups of the features on, in their list's order, for the run's script (its first letter's, else Latin; `DFLT` where the face has not that script), skipping the glyphs a lookup's flag passes over. GSUB: single, multiple, alternate, ligature, context, chained context, extension and reverse chaining; GPOS: single and pair adjustment, context and chained context; the old `kern` table where GPOS has no `kern` feature.
- Features are HarfBuzz's strings ("smcp,-liga,salt=2"). CSS computes them in Chrome's order: `font-variant`'s longhands, `font-kerning`, the ligatures `letter-spacing` turns off, then an `@font-face`'s own (where a `|` marks), then `font-feature-settings`. Capitals come from `smcp`, `c2sc`, `pcap`, `c2pc`, `unic`, `titl` where the face has them, as Chrome's `OpenTypeCapsSupport` decides, else are synthesised as before.
- Layout and drawing shape the same runs: each run of one face, cached per font by plan and glyphs; a plan that can touch nothing (Go's faces) costs a bitset test a glyph and allocates nothing. A word takes its kerning with the space after it and a space its kerning with the next word where the face kerns spaces.
- `twod.Features(...)` gives games the same shaping.

## Alternatives

- go-text/typesetting or HarfBuzz through cgo: complete (Arabic, marks, AAT), but go.mod is frozen and the pure packages stay free of cgo.
- x/image's `Kern` alone: no substitutions, and its GPOS misses pairs of other value formats and every lookup after the first.

## Consequences

Latin, Greek and Cyrillic text in a page's or the machine's fonts shapes as in Chrome. Marks placed on bases (GPOS 4–6), cursive attachment, AAT's `morx`/`kerx`, Arabic joining and Indic reordering stay undone, and ligatures and kerning stop at an element's edge.
