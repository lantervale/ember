# 0131. Discrete values and composition move as Chrome's

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

After decision 0122 a transition ran only between two values that interpolate, `display` was refused in a keyframe, `@starting-style`, `transition-behavior` and `animation-composition` were not read, an element out of `display: none` transitioned from the style it had while hidden, a child's own transition of an inherited value chased its parent's moving one, font weights moved in whole numbers and moving widths snapped to whole points, where Chrome snaps every border width to device pixels.

## Decision

- `display` is a discrete longhand whose flip keeps the box: from or to `none`, the other value shows through the whole interval. A discrete longhand transitions under `transition-behavior: allow-discrete` (a word of `transition` too), flipping halfway, on Style as interpolated ones do; so does a pair of values that doesn't interpolate.
- An element whose old style is missing or `display: none` has no before-change style: its runs are dropped and none starts, unless an `@starting-style` rule matches it. Those rules are parsed into the sheet's rules with a flag, left out of the cascade (`rulesOff`), and read by a probe of the element with them on, the style its transitions start from.
- `animation-composition` is a field of each `style.Animation` (a keyframe may set its own). How two values add or accumulate is an `add` of the longhand's interpolation in the table, beside `lerp`; an override or painted value composes each keyframe with the value under it (the element's, or an earlier animation's, which a non-replacing one stacks over), and an implicit keyframe is that value itself. A longhand without `add` replaces.
- While an element runs transitions on Style, the style they run to is kept, and a child that transitions is worked out a second time under it: its transitions start from that style, its shown value from the moving one.
- Border, outline and column-rule widths, `outline-offset` and `border-spacing` are snapped where the style is finished, to the Media's `Resolution` (whole points below one pixel a point); interpolations no longer snap. `font-weight` is a float, moved in quarters towards zero.

## Alternatives

- Compose keyframes with the element's style once, when they are resolved: cheaper, but wrong for a second animation adding to a first, and for an underlying value that changes.
- Keep the after-change style for the whole subtree under a transition: right for grandchildren too, but every descendant would be styled twice a frame.
- Snap only moving widths: what Chrome shows for a static `0.5px` border would still differ.

## Consequences

`TestDiscreteMovesMatchChrome`, `TestStartingStyleMatchesChrome`, `TestAnimationCompositionMatchesChrome`, `TestOwnTransitionOfAnInheritedValueMatchesChrome`, `TestWidthsSnapToDevicePixelsAsChromes` (one and two pixels a point) and `TestFontWeightKeepsQuartersAsChromes` compare them with Chrome frame by frame. A fractional border now lays out at its snapped width. `content-visibility`, `overlay`, cross-faded pictures, nested `@starting-style` and composition of positions, sizes and tracks stay in `docs/05-ui/10-missing.md`.
