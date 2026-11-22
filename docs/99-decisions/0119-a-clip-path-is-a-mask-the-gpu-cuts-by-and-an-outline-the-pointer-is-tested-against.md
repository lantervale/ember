# 0119. A clip-path is a mask the GPU cuts by, and an outline the pointer is tested against

- Status: Accepted
- Date: 2026-11-22
- Area: rendering
- Owner: engine

## Context

Pages cut avatars to circles, cards to slanted polygons and reveals to growing insets with `clip-path`. ui refused it and drew the whole box. The draw list could only clip to rectangles and one rounded rectangle (decision 0058), and nothing cut by a polygon, an ellipse or a path, nor kept the pointer out of what was cut.

## Decision

- `clip-path` takes `none`, `inset()`, `xywh()` and `rect()` (computed as `inset()`), `circle()`, `ellipse()`, `polygon()` and `path()` with their fill rules, in a reference box or the box alone. `url()` is refused with a warning.
- Paint turns the shape into an outline in the element's space, curves flattened (`ui/internal/outline`, pure Go, which keeps the svg reader out of ui as decision 0087 has it), and draws it once into a white mask at the device scale by its fill rule, sixteen scanlines a row, cached by outline and size. `twod.Canvas.PushClipMask` cuts what is drawn under it by that mask's alpha: the draw list keeps the mask and the transform from its space to the mask's UV on each batch, and the UI shader multiplies by the mask sampled at the pixel, under any transform. Only the innermost mask cuts; a rounded clip still rounds within it.
- Hit testing tests the point against the outline itself, with its fill rule, as Chrome does: a rectangle's right and bottom edges are out, any other outline's edges in.
- The element starts a stacking context, and its ink (decision 0101) is cut to the outline's bounds. Shapes of a kind interpolate in transitions and animations; other pairs flip, as in Chrome.

## Alternatives

- Draw the element into an offscreen pass and mask the pass, as `twod.MaskedBy` does a layer: exact for nested clips, but a pass per clipped element and its opacity drawn apart.
- Cut each triangle on the CPU against the shape: no GPU change, but hard edges, and text and images cut into fragments every frame.
- An analytic shape in the shader's uniforms: sharp at any scale, but a few kinds only, and no `path()` or polygon of many points.

## Consequences

Every basic shape draws as in Chrome and takes the pointer as Chrome does, at the cost of one mask texture per shape and size, and one more texture read for a cut batch. A clip-path inside another cuts only by its own shape and the outer one's bounds. Chrome hit-tests `inset()` and `xywh()` in a box the size of the content box at the border box's corner while painting them in the border box; ember hits what it paints.
