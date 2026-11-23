# 0161. SVG filters run in the element's own space, with every primitive Chrome draws

- Status: Accepted
- Date: 2026-11-23
- Area: assets
- Owner: engine

## Context

Decision [0150](0150-svg-filters-run-on-the-cpu-over-the-pixels-they-show-at.md) drew most filter primitives but left noise, lighting, displacement, convolution and `feImage` as clear results, filtered a turned element along the pixels' axes, and drew no `mask` of a CSS gradient. Textures, bevels and ripples in icons are made of exactly those, and Chrome is the reference every picture is checked against.

## Decision

- A filtered element whose space is turned or skewed is drawn in its own space, scaled to the picture's pixels as Skia splits a transform into a scale and the rest; the filters run there, axis-aligned, and the result is drawn turned, between pixels, as an image.
- The model's `Primitive` grows `Turbulence`, `ConvolveMatrix`, `DisplacementMap`, `DiffuseLighting`, `SpecularLighting` and `FeImage`, with `Noise`, `Convolve`, `Lighting` and `Light` values and `Items` for what an `feImage` draws; all run on the CPU over the run's float pixels.
- Each follows Chrome where Chrome parts from the letter of Filter Effects, measured against it: noise is the reference code's, its lattice wrapped before it is masked (the reference code masks first and never stitches), read at each pixel's centre and half a unit further, a stitched tile the region's size in whole pixels from the origin; lighting raises the surface by `surfaceScale` times the pixel scale; a displacement moves by whole pixels, rounded; an element in an `feImage` is drawn moved to its subregion's corner, and an SVG picture at a pixel a unit, then scaled.
- `BackgroundImage`, `BackgroundAlpha`, `FillPaint` and `StrokePaint` are unknown names, the result before; `edgeMode` of a blur is ignored; CSS functions after a `url()` work in the colour space of the primitive before them, a shadow's colour taken as it is there.
- `mask` and `mask-image` of a linear or radial gradient, repeating or not, are a `Mask` of one shape painting the gradient, repeated through a `Pattern`: placed on the fill box, cut to the stroke box, as Chrome treats an SVG element's border box. `mask-mode`, `mask-repeat`, `mask-origin` and `mask-clip` are read.
- A basic shape's `round` takes a radius per corner, scaled down together as `border-radius`; a `textPath`'s `startOffset` is of its path's `pathLength`.

## Alternatives

- Keep filtering in the picture's pixels under a rotation: one resampling fewer, but a turned element's blur, shadow and morphology go the wrong way.
- New model types for CSS masks: a gradient and a pattern already say it, and other renderers read the model.
- The noise exactly as the reference code: its stitching never takes effect, and no browser draws it so.

## Consequences

The new samples draw within half a level of Chrome on average at 256 pixels; small pictures of lighting and convolution differ more, as both work on whole pixels. A turned filter costs one more layer and resampling.
