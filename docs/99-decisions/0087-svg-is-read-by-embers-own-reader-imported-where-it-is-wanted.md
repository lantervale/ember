# 0087. SVG is read by ember's own reader, imported where it is wanted

- Status: Accepted
- Date: 2026-11-17
- Area: assets
- Owner: engine

## Context

ember read no SVG, so a page whose pictures are SVG showed none of them: the four icons of csszengarden.com/221 in the web explorer. We measured three ways to read it against headless Chrome, on 22 sample files:

- Depend on tdewolff/canvas: 25 modules, 2.5 MB more gzipped wasm, and it misread 3 of the 4 target icons.
- Copy oksvg and rasterx into the engine: both unmaintained, and they failed 12 of the 22 files.
- Write our own reader: on the samples it is at the floor Chrome's own anti-aliasing leaves.

## Decision

- ember has its own reader, the pure package `svg`: the standard library and x/image only.
- It parses a file into a model of paths and paints (`svg.Picture`), apart from the CPU rasteriser that draws it, so a GPU renderer can draw the same model later.
- It is opt-in by import, as sound formats are: `import _ ".../ui/svg"` registers it with ui (`ui.RegisterVectorFormat`). ui never imports `svg`, and a game that shows no SVG links none of it.
- A picture is rasterised once at the size it is drawn at, in device pixels, and again when that size changes; a Document keeps a bounded number of sizes.
- What the reader does not draw (text, clip-path, masks, filters, patterns…) is named by `Picture.Unsupported()`, and ui reports it as a warning naming the file.

## Alternatives

- tdewolff/canvas: the most complete, but too large for a browser build, and wrong on the pictures we needed first.
- oksvg and rasterx: small, but nobody fixes them, and half our samples failed.
- Rasterise at a fixed size and let the GPU scale: one texture a picture, but blurred icons on a dense screen and at large sizes.
- Tessellate on the GPU now: sharp at every size with no raster, but a second renderer to write before anything shows. The model is kept apart so this stays open.
- Always link the reader into ui: no import to remember, but every game would carry it.

## Consequences

ember owns the correctness of its SVG: each fix is ours to make, and the samples and their Chrome thresholds (`svg/testdata`) are the measure. A file using what is not drawn shows without it, with a warning that says why.
