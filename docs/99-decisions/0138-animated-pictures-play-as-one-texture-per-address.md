# 0138. Animated pictures play as one texture per address

- Status: Accepted
- Date: 2026-11-23
- Area: assets
- Owner: user

## Context

An animated GIF showed its first frame and an animated WebP showed nothing; a run-length encoded bitmap and a cursor file were errors ([0107](0107-ico-and-bmp-pictures-are-read-by-embers-own-reader.md)). The `image` package reads one frame, and a texture is one picture.

## Decision

- `ui/internal/animated` reads an animation into its frames whole, as they show: GIF through `image/gif`'s frames, WebP by its own reading of the container, each frame's bitstream decoded by `golang.org/x/image/webp`. The readers register there from `ui/web`, so ui plays the formats a program imported.
- Frames are drawn as Chrome's are, checked frame by frame against Chrome's `ImageDecoder`: on a clear canvas whatever background the file names, disposed of before the next, a WebP frame blended unless it says not to. A GIF without a loop count plays once and one of n plays n+1 times; a WebP's n plays n times; 0 is for ever. A frame of 10 ms or less shows for 100 ms, as Chrome's player has it. All frames together hold at most 2^25 pixels; those past it are dropped.
- A Document keeps one texture an address: every `img`, background and border image of it shows the same frame, as Chrome plays one picture of an address. Its clock, the World's real time, swaps the texture's image and repaints; the animation begins on the first frame drawn after it lands and stays on its last frame once played. Nothing pauses it off-screen.
- The ICO reader reads run-length encoded bitmaps of 4 and 8 bits, in an icon too, where Chrome leaves the pixels its runs skip opaque black and stops at a run leaving the picture, and cursor files. An ICO keeps showing its largest picture: Chrome's does, whatever the size it shows at.

## Alternatives

- One texture a frame: no upload on a change of frame, but every frame held on the GPU for as long as the picture is.
- A texture an element: each `img` would begin its own animation, where Chrome's of one address move together.
- Decoding frames as they show, as Chrome does: less memory, but a decoder that keeps its state between frames, which neither `image/gif` nor the WebP decoder offers.
- AVIF: no reader in pure Go, and go.mod takes no new module; an AV1 decoder is far larger than this change. It stays unread.
- `loading="lazy"` waiting for its picture to near the viewport: headless Chrome, ui's reference, fetches every lazy picture before the page's load event, a hidden one too, so no margin of Chrome's can be checked; ui keeps asking for a picture when it is laid out.

## Consequences

Animated GIFs and WebPs move in every program importing `ui/web`, at the cost of their frames decoded whole in memory and one upload a change of frame. A frame shown repaints the Document.
