# 0099. A page's fonts are fetched, and WOFF2 is read in pure Go

- Status: Accepted
- Date: 2026-11-18
- Area: assets
- Owner: user

## Context

A Document read `@font-face` files only from a `Stylesheet`'s file system, as TrueType. A page of the web names its fonts by address and serves them as WOFF2, so its text showed in the next family of the list, wrapped elsewhere, and its icon fonts showed as letters or boxes.

## Decision

- `ui.FetchFonts(fetch)` gets the fonts of `@font-face` rules no file system holds, off the frame, as `ui.FetchImages` gets pictures; `web.Fonts(base)` is the fetcher over HTTP. A font is fetched once per Document, and only when text needs a character its face draws (`unicode-range`).
- Until a font lands the text shows in the next family, then the Document lays out again, once a frame for all that landed: `font-display: swap`, whatever the rule says. A character only a face on its way would draw (an icon) shows as nothing meanwhile.
- TrueType and OpenType (CFF) files are read as they are; WOFF and WOFF2 are unpacked by `ui/internal/woff`, our own reader: strict on sizes and offsets, bounded at 32 MB, fuzzed.
- WOFF2's Brotli is `github.com/andybalholm/brotli`, the one new dependency: pure Go, no cgo, MIT.

## Alternatives

- No WOFF2, only WOFF and TrueType: most pages name a WOFF2 file alone, so their fonts would still not show.
- Bind Google's `woff2` and `brotli` libraries: the reference, but cgo in `ui`, a package servers and the browser build keep pure.
- Ask servers for WOFF or TrueType by a User-Agent that old: works for Google Fonts, not for a page's own files, and a browser's program lies about itself for it.
- Our own Brotli decoder: a thousand lines to keep right for what a maintained module does.
- Honour `font-display: block` and `optional`: timers per face for a difference of a few frames; recorded as a gap instead.

## Consequences

Pages show in their own fonts and icons once they land, at the cost of a layout when they do and of one module in every program that imports `ui`. Variable fonts draw their default instance, colour fonts and collections are not read, and the descriptors that reshape a face (`size-adjust`, `ascent-override`…) do nothing: all in [What is missing](../05-ui/10-missing.md).
