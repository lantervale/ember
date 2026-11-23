# 0153. Text is rasterized at its own size, and dashes are laid out in device pixels

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: user

## Context

A face kept one font a whole size, so text at 13.44px or 13.33px was drawn from the 13px glyphs scaled, softer and of other shapes than Chrome's. Square dashed and dotted sides were spread over whole points with hard edges, where Chrome counts and places dashes in device pixels and smooths their ends.

## Decision

- A face keeps fonts at sixteenths of a point: text is rasterized at the nearest, its glyphs scaled by at most a thirty-second of a point to its size. Layout's measures are unchanged.
- A space character a face lacks (U+2000 to U+200A, U+202F, U+205F) takes the room HarfBuzz gives it in that face, an em or a fraction of one, a digit's, a full stop's or half a space, as Chrome draws it rather than falling back to another face.
- Dashes follow Chrome's stroke in device pixels: its dash and gap ratios, its choice of how many, two shortened dashes or a solid side when short, edges at fractions of a pixel smoothed, a dash down a side at quarter pixels as Skia samples it. Dots up to three pixels wide are squares at both ends with the rest laid between; wider ones round.

## Alternatives

- Rasterizing at the exact size, one font each: sharpest, but an animated `font-size` would make a font and an atlas a frame without bound.
- Fonts at quarter points: fewer atlases, but glyphs scaled by up to 1% blur across texels.
- Dashes in points with hard edges, as before: simpler, but off Chrome by a pixel and by its smoothing at every fraction.

## Consequences

Text of any size is as sharp as a whole size, and the dashes page drops from 0.0021 to 0.0005 against Chrome, at the cost of up to sixteen fonts a point a face for animated sizes.
