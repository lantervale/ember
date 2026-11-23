# 0148. Variable fonts are read in pure Go, and font-display is timed on the Document's clock

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

x/image's sfnt reads no `fvar` or `gvar`, so a variable font drew its default instance at every weight, and a fetched font always swapped in when it landed, where Chrome hides a face's text for its block period and drops a font landing after its swap period ([0143](0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md) left both). go.mod is frozen, and servers stay pure Go.

## Decision

- `internal/varfont` reads `fvar`, `avar`, `gvar` (shared and private points, inferred points, intermediate regions, composites, phantom points) and `HVAR` beside sfnt: a glyph's outline and advance at any place on the axes. Layout measures a web face's advances there, and twod's glyphs take the instance as part of their `draw.Look`, rasterized from varfont's outline as made-up faces are.
- A face's instance is Chrome's: the text's `font-weight` and `font-stretch`, clamped to the face's descriptors (the font's own axes when it gives none, or `auto`), on `wght` and `wdth`, then `font-variation-settings` on any axis, the last of a tag winning.
- `font-display` is timed from when text first needs the file, on the Document's clock (the World's real time): 3 s block for `auto` and `block`, 100 ms for `fallback` and `optional`, none for `swap`; a file landing after 3 s (`fallback`) or 100 ms (`optional`) is a failed one. Text in its block period is laid out in the next family and drawn invisible. A Document never drawn has no clock and takes its fonts whenever they land.

## Alternatives

- Instancing a static font file per instance (as fontTools' instancer does) and parsing it with sfnt: a font file per weight a page names, rebuilt on every change of `font-variation-settings`.
- A shaping library with variations (HarfBuzz through cgo, or a Go port as a new module): cgo breaks pure-Go servers, and go.mod is frozen.
- Timing `font-display` on the wall clock: tests and captures run frames faster or slower than real time, where the World's clock is the one they drive.

## Consequences

Pages on a variable font (most of Google Fonts) draw every weight and width they ask for as Chrome does, and a slow font hides or keeps its fallback as Chrome would. A variable glyph is rasterized through the made-up-face path, so its first drawing per size costs a little more; `MVAR` metrics, `opsz`, `ital`/`slnt` and `CFF2` variations are not read.
