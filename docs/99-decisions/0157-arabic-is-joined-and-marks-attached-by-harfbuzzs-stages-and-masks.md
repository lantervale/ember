# 0157. Arabic is joined, and marks attached, by HarfBuzz's stages and masks

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0145](0145-text-is-shaped-by-a-pure-go-opentype-layout-reader.md) shaped Latin, Greek and Cyrillic: Arabic was drawn letter by letter in its isolated forms, a combining mark left out with its cluster, and a right-to-left fragment drawn as its characters reversed, which no shaper can join.

## Decision

- `internal/shape` plans as HarfBuzz's map builder: features asked for by the script's shaper and the text, merged by tag (the last value, the first stage), each global or with a mask of its own; lookups apply stage after stage, each to the glyphs of its mask. Scripts whose letters join (Arabic, Syriac, N'Ko, Mandaic, Adlam…) take the Arabic shaper's stages: `ccmp`/`locl`, then `isol`, `fina`, `fin2`, `fin3`, `medi`, `med2`, `init` each alone, `rlig`, `calt`, then the rest; each letter gets the mask of the form HarfBuzz's joining machine gives it, from the joining types of Unicode's `ArabicShaping.txt` generated into a table (`go generate`), marks and format characters transparent.
- `Glyph.Char` carries each glyph's character. Marks are sorted by HarfBuzz's modified combining classes (shadda first, Hebrew's points in fonts' order); ligatures number the marks between their components and merge clusters; GPOS's cursive (3), mark-to-base (4), mark-to-ligature (5) and mark-to-mark (6) attach glyphs, marks lose their advances by GDEF, and attachments add their parents' offsets at the end, direction from the script, as HarfBuzz.
- A run right to left is shaped in its order and drawn from its end: `draw.Style.RightToLeft`, `twod.RightToLeft()`, and ui paints a fragment shown right to left from its logical text so. draw shapes every character of a cluster, a mark with its base.
- `font-variant-position` asks for `sups` and `subs` after `font-feature-settings`, as Chrome does; a face without them draws the text as it is, Chrome synthesising nothing.

## Alternatives

- go-text/typesetting's HarfBuzz port: complete, but go.mod is frozen. The Arabic presentation forms of the cmap as fallback for a face without `init`/`medi`/`fina`: left out, as are Indic and USE reordering.
- A plan per direction: the direction follows the script, which is the bidi level Chrome shapes an Arabic run at.

## Consequences

Arabic, Syriac, Hebrew with points, N'Ko, Adlam and Thaana shape as `hb-shape` does in the Mac's faces, and ui's Arabic lays out and draws as Chrome's. A run still stops at an element's edge, a face change or a character the face lacks, so joining stops there too.
