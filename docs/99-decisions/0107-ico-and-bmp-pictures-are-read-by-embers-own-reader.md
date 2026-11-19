# 0107. ICO and BMP pictures are read by ember's own reader

- Status: Accepted
- Date: 2026-11-19
- Area: assets
- Owner: user

## Context

A site's favicon is an ICO file, and a page showing `<img src="/favicon.ico">` showed no picture, as one showing a BMP file: [0096](0096-webp-pictures-are-read-with-the-pictures-of-the-web.md) left both out. An ICO holds PNG files or bitmaps as a BMP file's, without their file header and with a mask after them, so reading one is reading the other.

## Decision

- Importing `ui/web` adds ICO and BMP to the formats ui reads, beside JPEG, GIF and WebP, known by their first bytes. `web.Images` asks for `image/x-icon`, `image/vnd.microsoft.icon` and `image/bmp`.
- The reader is ember's own, `ui/internal/ico`, registered with the standard `image` package: bitmaps of 1, 4, 8, 16, 24 and 32 bits a pixel, from the bottom or the top, with bit masks; a run-length encoded bitmap is an error.
- Alpha is Chrome's, checked by the pixel page `ico-bmp`: a BMP file has alpha only where its header names the alpha bits; an icon of 32 bits has it unless it is zero everywhere, and is then cut by its mask, as an icon of fewer bits is.
- An ICO shows the largest of its pictures, then the one of the most bits a pixel: a decoder of the `image` package is not told the size a picture shows at.
- The files come from the network: nothing over 16384 pixels a side or 2^25 pixels is built (the limits of the svg reader), every offset is checked, and a file saying anything else is a warning, never a panic. Fuzz tests hold it to that.

## Alternatives

- `golang.org/x/image/bmp`, already required: it reads 8, 24 and 32 bits only, no 16 bits and no masks of one's own, caps no size, and reads a file header an icon's bitmap has not; the ICO reader would have needed its own bitmap reader anyway.
- A module that reads ICO: one more dependency for a few hundred lines, and its limits to audit.
- Choosing the entry nearest the size shown: the `image` package's decoders take bytes alone, so it would need a route of its own from layout to the decoder, for a favicon most often shown small beside one drawn larger than it needs.
- Reading run-length encoded bitmaps: pages do not serve them; they stay a gap, listed in [What is missing](../05-ui/10-missing.md).

## Consequences

A favicon and a BMP picture show wherever a JPEG does, in every program importing `ui/web`, and the largest picture of an ICO is scaled down where a smaller one of the file would have been sharper. AVIF stays unread and unasked for.
