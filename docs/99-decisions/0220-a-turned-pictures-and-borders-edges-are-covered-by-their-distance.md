# 0220. A turned picture's and border's edges are covered by their distance

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

Under a rotation or a skew, the triangles of a picture, a border or a border-image were drawn without coverage, their edges stepped from pixel to pixel where Chrome anti-aliases them. Decision 0216 smoothed a turned box's own edges only where its clip cuts a background reaching past them.

## Decision

- `draw.List.SmoothImage` (`twod.Canvas.SmoothImage`) draws an image whose quad, turned or skewed, reaches a point past its rect, its UV carrying on; the GPU covers each pixel by its distance in pixels to the edges of its UV's bounds, through the UV's screen derivatives, and samples it clamped inside them. An edge left level or upright on the screen, as a skew leaves two, covers whole pixels as before. Drawn square, it is `Image`: nothing drawn axis-aligned changes.
- ui draws an `img`, a picture alone in its background and a border-image stretched whole with it. A border-image cut by its slice reaches a point past its box at its outer edges, one texel's colour, and the box's turned clip (decision 0216) smooths them; tiles of a background reach past its view alike.
- A square border's sides, turned or skewed, are mitred pieces (decision 0132) whose edges along their side are feathered over a device pixel: drawn from half a pixel inside, a strip a pixel wide fading out across each. Their mitres stay hard where they meet the next side, so no seam shows.
- twod's sprites keep `Image`: a turned sprite's edges stay as its pixels fall, as pixel art wants; a game asks for `SmoothImage`.

## Alternatives

- A one-pixel feather of geometry for pictures too: the UV of a fading strip needs per-vertex alpha on a textured mesh the Canvas doesn't have, and each piece's own feather would seam the pieces of a border-image or the tiles of a background where they meet.
- A clip per picture in its own space (decision 0216): only the innermost rounded clip cuts, so a picture inside a rounded scroller would show past its edges.
- Coverage for every turned quad in the shader: sprites, glyphs and the pieces that meet edge to edge would all seam.

## Consequences

Turned and skewed pictures, borders and border-images are smoothed as Chrome's. Tiles still meet without the blend Chrome's repeated picture has across their seams, and a gradient tiled by `background-size` keeps its hard stops and tile edges stepped, where Chrome draws a tile picture and filters it.
