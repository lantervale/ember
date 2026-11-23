# 0135. Small capitals and skipped ink are synthesised from the face's outlines

- Status: Superseded by [0182](0182-skipped-ink-follows-the-glyphs-drawn-and-chromes-table-of-characters.md)
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

`font-variant: small-caps` and `text-decoration-skip-ink` both need more of a face than its advances: small capitals the face's own `smcp` glyphs, which neither the Go faces nor most game fonts have, and skipped ink where each glyph's outline crosses a line. ui shapes no OpenType feature, and draws a decoration as rectangles.

## Decision

- Small capitals are synthesised as Chrome synthesises them for a face without `smcp`: the text cut into runs, those it makes small uppercased (ß as SS) at 70% of the font size rounded to a whole one (`SimpleFontData::CreateScaledFontData`), the others as they are. `style.Font` carries `Caps`, so `fonts.Set.Advance` measures the runs and paint draws them, and layout, selection and carets follow without knowing.
- Skipped ink is Skia's `getTextIntercepts`: each glyph's outline, flattened, crossed with the line's band gives the stretch of x it covers; the line is drawn clipped out round each, dilated by its thickness up to 13 points, as Blink's `DecorationsStripeIntercepts`.
- A decoration casts the text's shadows, each line drawn in the shadow's colour and taps before itself, as Chrome's shadow looper draws it.
- The other `font-variant` longhands are kept and written back, and draw nothing: ui applies no OpenType feature.

## Alternatives

- Shaping with OpenType features (HarfBuzz or a Go shaper): the face's real small capitals, tabular figures and ligatures, but a shaper in a pure package and every measurement through it; synthesis is what Chrome draws for the faces games ship.
- Gaps from glyph bounding boxes: no outline reading, but an underline would break under every glyph whose box reaches it, not where its ink does.
- A shader cutting the line by the glyphs' coverage: exact, but a decoration would need the text's atlas and its own GPU kind.

## Consequences

Small capitals and underlines match Chrome's on faces without `smcp`; a face that has its own small capitals is still drawn synthesised, where Chrome uses them, and `font-variant-numeric` and the rest change nothing until ui shapes features.
