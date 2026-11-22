# 0118. A filtered element is a picture drawn through WGSL effects

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

CSS `filter` blurs, recolours and shadows an element and everything in it as one image, which ui could not draw: paint emits shapes onto a `twod.Canvas`, and its `opacity` only multiplies colours. The pure packages must not reach the GPU.

## Decision

- `twod.Canvas` draws a part apart: `BeginPicture` sends what follows into a pass of its own, `EndPicture(margin)` crops it to what was drawn and the margin effects may reach, and `DrawPicture(p, effects…)` composites it through `render.Effect`s, in the clip in place; shown again, the same list is drawn into another pass, so a drop shadow shows under the picture without the GPU knowing more than passes. A nested picture's pass is drawn before the one it is in. `Canvas.Effect(shader)` keeps effects from frame to frame so their shaders stay compiled.
- paint opens one picture for a filtered element and its content, and one round it for each drop shadow other filters follow; the colour functions are one Filter Effects matrix each (unpremultiplied, clamped), `blur()` a Gaussian across then down, `drop-shadow()` the alpha moved, coloured and blurred to half its radius. Sizes go through the element's transform and the Document's pixel ratio.
- The picture is drawn without the clips round the element and shown in them. A filter starts a stacking context, holds absolute and fixed descendants as Chrome does, and grows the element's ink and the view culled inside it by what it reaches.
- `filter` lists of the same functions (or one starting with the other's, or `none`) interpolate function by function; others switch halfway. `url()` filters are refused with a warning.

## Alternatives

- Per-vertex colour matrices in `internal/draw`: no blur or shadow, and overlapping children would not be one picture.
- Effects that read the picture before the chain, in `gpu`: a drop shadow in one pass, but a new binding in every effect and a GPU change for what passes already give.
- A whole-window pass per filtered element, as layers do: simpler, but blurring a window for a small card.

## Consequences

A filtered element costs a pass and a texture of its ink, and a draw of its list for each drop shadow; a Document holding one repaints each frame, as a recording with a pass picture is never replayed. Another `Surface` than `twod.Canvas` draws filtered elements unfiltered.
