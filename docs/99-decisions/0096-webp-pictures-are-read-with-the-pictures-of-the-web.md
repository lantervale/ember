# 0096. WebP pictures are read with the pictures of the web

- Status: Accepted
- Date: 2026-11-18
- Area: assets
- Owner: user

## Context

A Document fetching its pictures over HTTP read PNG, JPEG and GIF. News sites serve their photographs as WebP whatever the request accepts, so a page such as bbc.com/news showed none of them, each a warning of an unknown format.

## Decision

- Importing `ui/web` adds WebP to the formats ui reads, beside JPEG and GIF: lossy and lossless, with alpha. `web.Images` asks for `image/webp`.
- The decoder is `golang.org/x/image/webp`, a module ember already requires: pure Go, registered with the standard `image` package, so a picture is known by its first bytes, never by its name or its declared type.
- An animated WebP is no picture: a warning, as bytes that are none. AVIF is not read and not asked for.

## Alternatives

- Read WebP in ui itself, as PNG: every game would link a decoder for a format games do not ship.
- A package to import of its own, as `ui/svg`: one more import to remember for what any page of the web needs, where JPEG and GIF already come with `ui/web`.
- Bind libwebp: animation and speed, but cgo in a package servers and the browser build must keep pure.
- Our own reader of the container, to show an animation's first frame over x/image's `vp8` and `vp8l`: a copy of its alpha code to keep, for pictures we have not met yet.

## Consequences

A page's WebP pictures show wherever its JPEG ones do, at the cost of x/image's decoder in every program that imports `ui/web`; it is slower than libwebp. An animated WebP stays a gap, listed in [What is missing](../05-ui/10-missing.md).
