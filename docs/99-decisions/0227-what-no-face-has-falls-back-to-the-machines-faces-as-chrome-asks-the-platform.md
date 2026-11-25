# 0227. What no face has falls back to the machine's faces, as Chrome asks the platform

- Status: Accepted
- Date: 2026-11-24
- Area: rendering
- Owner: engine

## Context

Under `ui.SystemFonts()` (decision 0081) a character that no face of the font list had, nor Go Mono, drew as a missing glyph: Japanese, Chinese, Korean and Arabic on real pages in light. Chrome then asks the platform for an installed font that has it (CoreText, fontconfig, DirectWrite), by script, language and the kind of font.

## Decision

- Under `ui.SystemFonts()` only, such a character is measured and drawn with an installed face: the platform's families for its script as Chrome gets them there (a table per platform, measured against Chrome 158 on macOS), for Han, kana and the punctuation the three languages share by the language of the text (`lang`, else the Document's: Japanese, Korean, Simplified, Traditional), serif ones under a list standing on `serif`, the bold face under a bold font; a character of no script first tries the faces already chosen for the text; then the first installed family, by name, whose cmap has it.
- `style.Font` holds the language (`Lang`), inherited in the cascade.
- Cmaps are read once, cached across Documents; every installed font's is read off the frame once the scan lands, and text that waited for it is laid out again. A character's face is chosen once per context; nothing is chosen per frame.
- Layout measures a run of one face as one shaped run (Arabic joins, `halt` trims CJK punctuation); paint draws the face after the font's own and Go Mono's, only the characters layout gave it (`Only`), rebuilt when a new character is chosen, so measuring, drawing, hit testing and selection agree.
- Fonts ember cannot draw are passed over silently: PingFang (Apple's `hvgl` outlines), colour bitmaps, faces whose cmap `sfnt` refuses, LastResort.

## Alternatives

- Asking the platform itself (CoreText, fontconfig, DirectWrite): exact, but cgo on three platforms in a package that must stay pure Go, and PingFang still undrawable.
- Every installed face covering a character, in scan order, without tables: no language, no serif, Hiragino Sans for Chinese.
- A whole run's face by its first character, as Chrome shapes a segment: needs the draw side to cut runs as layout does; per character with script tables gives the same faces for the pages compared.

## Consequences

Pages in Japanese, Korean and Arabic lay out as Chrome's on macOS to the pixel row; Chinese without serif draws in Hiragino Sans GB where Chrome draws PingFang. Without the option nothing changes, so a Document still looks the same everywhere.
