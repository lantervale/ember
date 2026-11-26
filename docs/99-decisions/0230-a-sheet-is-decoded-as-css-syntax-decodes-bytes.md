# 0230. A sheet is decoded as CSS Syntax decodes bytes

- Status: Accepted
- Date: 2026-11-26
- Area: api
- Owner: engine

## Context

ui read every sheet's bytes as UTF-8. A sheet in Shift_JIS or Latin-1, with a byte order mark (which then broke its first rule) or served with a charset, lost its rules, and 43 of WPT's CSS 2.1 syntax tests with it.

## Decision

- `css.Decode(data, environment)` decodes a sheet as CSS Syntax does: its byte order mark, then its `@charset "…";` read byte for byte (UTF-16 as UTF-8), then the environment encoding, then UTF-8, by the Encoding Standard's labels (`x/text`'s `htmlindex`). `css.Load`, a `Stylesheet` and every imported sheet go through it; `@charset` is then dropped without a warning.
- An imported sheet's environment is the encoding of the sheet that imports it. A sheet given as text says its own with the `SheetOption` `ui.Encoding(label)`, UTF-8 without it.
- The charset a sheet was served with, which CSS puts before `@charset`, is the fetcher's to apply: it decodes the sheet and gives it back as UTF-8 behind a byte order mark. A byte order mark of the sheet's own still wins.
- The reftest runner decodes pages as a browser does (byte order mark, the `.headers` file's charset, the XML declaration or the meta element, else UTF-8) and serves those charsets to Chrome as WPT's server does.

## Alternatives

- A charset returned by `SheetFetcher` beside the bytes: changes a public type every fetcher implements, for what a byte order mark already says.
- One encoding for the whole Document (a `DocumentOption`): an imported sheet falls back to its importer's encoding, not the page's, so it needs one per sheet anyway.

## Consequences

Sheets in any encoding a browser reads are read alike. `web.Sheets` does not yet apply the charset of a response's Content-Type (in [what is missing](../05-ui/10-missing.md)).
