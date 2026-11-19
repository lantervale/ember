# 0100. The GPU holds a picture while the picture lives

- Status: Accepted
- Date: 2026-11-18
- Area: rendering
- Owner: engine

## Context

The 2D renderer kept the texture of every `render.Texture` and the atlas of every sized font it had drawn, in maps keyed by their pointers, for the life of the program. A game loads its pictures once, so nothing showed; a Document draws an SVG again at each size, and a browser shows page after page: resizing one fluid page kept 800 MB of pictures nothing drew any more, on the heap and on the GPU.

## Decision

- The renderer's pictures and atlases are keyed weakly (`gpu.kept`): once the program holds a `render.Texture` or a sized font no longer, it is collected, and its GPU texture is released at the start of the next frame.
- What is made of a picture and cached beside it follows the same rule: a mask's silhouette and a pixelated copy in `ui/paint` go when their picture does.
- Whoever makes pictures bounds what it keeps (a Document's vector pictures, decision 0087); the renderer adds no budget of its own.

## Alternatives

- Evict what was not drawn for N frames, as the 3D renderer does for meshes and material textures: also frees what is held and off screen, but a game's sprite sheet not drawn for ten seconds is uploaded again in the middle of a frame, and a picture dropped stays up to N frames.
- A byte budget with a least-recently-drawn order: a second policy over the owners' own, and a frame drawing more than the budget uploads every frame.
- Have owners release explicitly (`tex.Release()`): every Document, tool and game would have to, and one that forgets leaks as before.

## Consequences

Memory follows what the program holds: a page closed or a size no longer kept gives its pictures back at the next collection, with no call. A picture is released a collection late, and a picture the program holds without drawing stays on the GPU, as before.
