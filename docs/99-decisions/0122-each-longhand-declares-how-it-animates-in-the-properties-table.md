# 0122. Each longhand declares how it animates, in the properties' table

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

Transitions ran on colours, `opacity`, `transform` and `box-shadow` alone, and animations added the box's lengths: every other property flipped halfway, or was refused in `transition-property`, where Chrome interpolates `font-size`, `border-radius`, `background-position`, grid tracks and most of the rest. Each list of what moves lived apart from the properties, so a new property flipped without a word.

## Decision

- How a longhand animates is a field of its entry in ui/css's longhands' table (`prop.anim`): still (never animates), discrete (flips halfway), interpolated on Style, or painted into Shown. The helpers that build an entry take it: `field(get, set, how)` with an interpolation of the value's type (`lengths`, `numbers`, `integers`, `colors`, `positions`, `repeatable` lists, `shadows`, `trackLists`, `clipPaths`, `filters`, `visibility`…), `keyword` is discrete. A test fails for a longhand that declares none.
- Each interpolation follows Chrome's animation type for the property: length-percentages by computed value (px and % apart, a `calc()` of `min()` or `max()` as a sum), integers rounded halves up, widths snapped to whole points, `aspect-ratio` by its logarithm, repeatable lists to their least common multiple, an offset from the end edge as `calc(100% - x)`, keywords only with the same keyword.
- Transitions and animations read the same table: a keyframe drops a still longhand, `transition-property` takes any property ui knows, and a transition runs only between two values that interpolate. One that moves Style is applied by the restyler on the element's cascade, after its animations', its longhands applied again with it so `em` and `currentColor` follow; colours, `opacity`, `transform` and `box-shadow` stay painted through Shown.

## Alternatives

- Keep a list of what moves beside the table, as before: two places to change for a property, and a forgotten one flips silently.
- Interpolate every property on Style, painted ones too: one road, but an opacity fade would restyle its element every frame (decision 0114).
- Interpolate the declarations' text and cascade it each frame: Chrome's values come from computed values, which ui already holds typed.

## Consequences

Every property ui reads moves as in Chrome, checked by `TestInterpolationMatchesChrome` at four times for keyframes and transitions both; a transition of a layout property costs a restyle and a layout a frame, as an animated width already did. Where ui keeps less than Chrome (font weights in whole numbers, widths snapped at a point not a device pixel, pictures not cross-faded), `docs/05-ui/10-missing.md` says so.
