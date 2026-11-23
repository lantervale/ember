# 0143. A face's look, and the faces a family has, follow its sheets

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

An `@font-face` read its family, weight, style and `unicode-range` and dropped the rest ([0099](0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)): a page's `size-adjust` and metric overrides, its condensed faces, a bold made up from a regular face, a rule inserted alone or a sheet taken away all showed other than in Chrome. Each needs a place in the font set and in twod's glyphs that keeps one face file drawn many ways.

## Decision

- Every `@font-face`, read from a file system or fetched, is a face of the set's web faces with its descriptors: widths (`font-stretch`), and a look: `size-adjust`, `ascent-override`, `descent-override`, `line-gap-override`, and kerning turned off by `font-feature-settings: "kern" 0`. A face is chosen by width first, then style, then weight, as Chrome's font matching.
- The faces text is drawn with are those of the sheets a Document has now: taking its sheets again (added, removed, disabled, replaced, a rule inserted or deleted) sets them in cascade order, keeping every face read so its file is never read or fetched twice.
- A bold or italic the family lacks is made up from the face it has, as Chrome (`font-synthesis` may forbid it): drawn, never measured, as Chrome keeps the advances. twod's glyphs take a `draw.Look` per face: scaled, slanted a quarter, grown round their outline by a quarter of Skia's fake-bold stroke on each side (fitted to Chrome on macOS), unkerned.

## Alternatives

- Separate font files or twod faces per look: the same bytes parsed and kept again for each, and a look twod's public API would have to name.
- Thickening a made-up bold by coverage, as stem darkening does: cheaper, but far from Chrome at large sizes, where an outline grows evenly.
- Dropping a face from the set when its sheet goes: its file read or fetched again when the sheet comes back.

## Consequences

Pages that size-adjust a fallback, override metrics or lean on made-up bolds lay out and draw as in Chrome. A made-up bold draws its glyph seventeen times once per size; variable fonts' instances, `font-display` periods and features other than `kern` are still not read.
