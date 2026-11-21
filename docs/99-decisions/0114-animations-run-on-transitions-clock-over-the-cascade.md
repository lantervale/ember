# 0114. Animations run on transitions' clock, over the cascade

- Status: Accepted
- Date: 2026-11-21
- Area: rendering
- Owner: engine

## Context

`@keyframes` and `animation` were refused, so a page that starts its content at `opacity: 0` and fades it in by an animation filled forwards never showed it in light, and spinners and carousels stood still. ui already ran transitions on the World's real time, painting their values through `Shown`.

## Decision

- `css.Transitions` runs animations beside transitions: the same `Compute` starts, updates and cancels them as an element's `animation-name` list changes, the same `Step(dt)` advances them on the Document's real-time clock, and transitions' interpolation gives their values. No second clock, no new type.
- A keyframe is computed as a cascade over its element's own (`overKeyframe`, under the element's `!important` declarations), once when the animation starts and again whenever the element's style changes: `var()`, `em`, `currentColor` and the implicit 0% and 100% come out as Chrome's, with no keyframe-specific value code.
- Values go where the rest of ui reads them: colours, opacity, transforms and shadows into `Shown`, as transitions, so they only repaint; box lengths and every value that can't interpolate onto `Style` itself, flagging the element `DirtyStyle` only when one changes, so layout and inheritance follow.
- Timing is Web Animations': phases, fill, directed iteration progress, per-keyframe easing. Their events are World events and element callbacks, as the Document's others; `prefers-reduced-motion` is the Document's to answer (`ui.ReducedMotion()`).

## Alternatives

- Rename `Transitions` to `Motion` with a layer type of its own, as the plan parked in ui/css said: the same code under a churn of names through ui; left for when the type grows again.
- Interpolate every value as a declaration fed back into the cascade each frame: one road for all properties, but a restyle of the element and its subtree every frame for an opacity fade, which is the commonest animation.
- Let animations run on game time: a pause menu would freeze them, and transitions already chose real time.

## Consequences

A fade, a slide or a spinner costs a repaint a frame and no restyle; an animated width costs a restyle and a layout a frame, a discrete flip one restyle. Properties outside the two interpolated sets flip halfway where Chrome interpolates (`docs/05-ui/10-missing.md`).
