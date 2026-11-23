# 0186. A variable font's layout is shaped at its instance

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0148](0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md) drew a variable font's outlines and advances at its instance, but `internal/shape` kept the default instance's layout: GPOS's kerning and anchors without GDEF's deltas, and GSUB's and GPOS's feature variations (`rvrn`, a heavier `$`, a wider kern) never chosen. Chrome's HarfBuzz shapes at the instance, so text kerned or substituted by axis was measured and drawn apart from Chrome's.

## Decision

- A plan is made per instance: `shape.Font.PlanAt(script, features, vf, coords)` takes the varfont font and its normalized coordinates, cached by their 2.14 values; `Plan` is the default instance.
- The plan's features are those of the first feature variation whose conditions hold at the instance (axis ranges in 2.14, as HarfBuzz compares them), for GSUB and GPOS each, evaluated at the default instance too.
- Value records' and anchors' variation-index devices add GDEF's ItemVariationStore delta (varfont's store, now exported), rounded to font units as HarfBuzz rounds at a size of one unit a unit; hinting devices add nothing, as Chrome sets no ppem.
- Shaping takes the instance's advances itself (HVAR or phantom points), so marks are placed against them; ui and draw no longer correct a shaped run's advances afterwards.

## Alternatives

- Keep shaping at the default instance and correct only advances afterwards, as before: cheap, but kerning, anchors and swapped glyphs stay the default's.
- Instance the font's layout tables once per instance (as fontTools' instancer does) and shape that: a copy of GSUB, GPOS and GDEF per weight a page names, rebuilt for every change of `font-variation-settings`.

## Consequences

A variable font's kerning, marks and substitutions follow its weight and width as in Chrome, at the cost of a plan per instance a page uses. Shaped advances are whole font units where they were exact, under a 2,000th of an em a glyph. Only conditions of format 1 (an axis range) are read: a feature variation with another kind never holds.
