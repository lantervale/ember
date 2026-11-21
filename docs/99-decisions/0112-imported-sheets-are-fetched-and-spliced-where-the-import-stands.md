# 0112. Imported sheets are fetched, and spliced where the import stands

- Status: Accepted
- Date: 2026-11-21
- Area: api
- Owner: user

## Context

A sheet's `@import` rules were refused with a warning, so a page that pulls its styles in through them (a theme, a reset, Google Fonts' sheet, a framework's parts) showed without them. ember fetches nothing itself ([0096](0096-webp-pictures-are-read-with-the-pictures-of-the-web.md), [0099](0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md)).

## Decision

- `ui.FetchSheets(fetch)` gets the sheets `@import` names, off the frame, as `ui.FetchFonts` gets fonts; `web.Sheets(base)` is the fetcher over HTTP, which takes a sheet served as `text/css` or as no type, as Chrome does. A `Stylesheet`'s imports are read from its file system first, at once.
- `css` reads the imports a browser follows (`Sheet.Imports`): before any rule but `@charset` and `@layer` statements, never in a block, whose `supports()` holds; `layer()` and `supports()` only under `ui.BrowserSheets()`. `Sheet.Imported(sheets)` splices each imported sheet in where its import stands: its rules under the import's media query list and in its layer, its layers named there.
- A sheet is fetched once per Document by its resolved address, however many imports name it, and each import stands in its own place. An import of a sheet it is imported through is stopped with a warning.
- Until a sheet lands the Document shows without it; then its sheets are taken again, once a frame for all that landed. Without a fetcher an `@import` is a warning and the sheet shows as before.

## Alternatives

- Fetch the imports before the Document shows, as a browser blocks rendering on a linked sheet: a game's frame would wait on the network; the program can wait on `doc.SheetsLoading()` itself.
- Have the program resolve imports and hand ui the joined text: every program would rewrite the cascade's rules of order, layers and media, and the warnings would point at text nobody wrote.
- Insert the imported rules into the importing sheet's `ui.Sheet` (its `Rules()` and `Text()`): a script or a tool editing the sheet would see rules that are another sheet's, which CSSOM keeps apart.

## Consequences

Pages show with the sheets they import, as Chrome cascades them (`TestImportsCascadeAsChromes` follows the same imports from a local server in both). A page exported for Chrome carries each imported sheet written in its import's layer and media, so the comparison suites need no server; its `url()`s are then resolved from the importing sheet.
