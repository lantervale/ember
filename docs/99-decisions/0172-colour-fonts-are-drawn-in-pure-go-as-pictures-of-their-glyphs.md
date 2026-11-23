# 0172. Colour fonts are drawn in pure Go, as pictures of their glyphs

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

A font's colour glyphs (`COLR` layers and paint graphs in `CPAL` palettes, `sbix` and `CBDT` bitmaps, `SVG` documents) drew as their outlines, often empty, and `font-palette`, `@font-palette-values` and `tech(color-…)` sources were refused; a collection (`.ttc`) named by an `@font-face` failed. x/image's sfnt reads none of these tables, go.mod is frozen and servers stay pure Go.

## Decision

- `internal/colourfont` reads the colour tables beside sfnt and draws one glyph into an RGBA picture at a size: `COLR` version 0 and 1 (every paint: layers, solid colours, linear, two-circle radial and sweep gradients with pad, repeat and reflect, every transform, Porter–Duff and blend-mode composites, clip boxes), `sbix` and `CBDT`/`CBLC` PNGs scaled from the nearest strike, and `SVG` documents through ember's `svg` package, which importing `ui/svg` registers (`colourfont.RegisterSVG`), as it registers SVG pictures (decision 0087). Chrome is the reference where the format leaves room: a sweep's angles start at 180° (`(value + 1) × 180°`), an `sbix` picture shows within its outline's bounds, as CoreText draws it.
- `draw` draws a colour glyph as its picture, cached per face, glyph and text colour, at the glyph's advance and offset from shaping, so layout and `ZWJ` sequences are the font's own; a stroke (`-webkit-text-stroke`) draws nothing of it, as Chrome. Its `Look` carries the palette and whether `SVG` glyphs are drawn.
- ui draws `font-palette` (`normal`, `light`, `dark`, a `--name` of an `@font-palette-values` for the family: `base-palette`, `override-colors`) and reads `tech(color-COLRv0|COLRv1|sbix|CBDT)` sources and `@supports font-tech()`; it leaves `SVG` glyphs to their outlines and refuses `tech(color-SVG)`, as Chrome, while twod draws them as Safari and Firefox do in a program that imports `ui/svg`. A collection draws its first font, its fragment ignored, as Chrome.

## Alternatives

- FreeType or Skia through cgo: breaks pure-Go servers and the wasm build.
- Rasterizing colour glyphs on the GPU, a shader per paint: a paint graph nests composites and clips without bound, and each glyph is drawn once per size and colour, so the CPU picture, cached like the alpha atlas, costs nothing per frame.
- Drawing `SVG` glyphs in ui too: ember's `svg` draws them, but Chrome on every platform shows their outlines, and ui's reference is Chrome.

## Consequences

Web colour fonts and the machine's emoji font (Apple Color Emoji through `ui.SystemFonts()`) draw in colour, a text's colour standing for palette entry `0xFFFF` and `currentColor`. A colour glyph's first drawing per size and colour rasterizes a picture; a text shadow casts the picture rather than its silhouette, `CFF2` outlines and `COLR` variations are not read, and the machine's emoji font is not a fallback for text in other families. A program that does not import `ui/svg` links no SVG reader: there an `SVG` glyph is drawn from the font's `COLR`, `sbix` or `CBDT` table when one has it, which always come first, else as its outline, as Chrome draws a table it does not use.
