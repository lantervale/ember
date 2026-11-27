# 0259. A browser's default monospace is 13px where no absolute size stands above it

- Status: Superseded by [0263](0263-the-reftests-chrome-sizes-its-default-monospace-and-draws-it-in-go-mono.md)
- Date: 2026-11-27
- Area: rendering
- Owner: engine

## Context

Browsers draw text whose family is `monospace` alone at 13px, not 16px, unless an absolute size stands above it (Chrome's FontBuilder, the "default fixed font size"). Pages built for browsers depend on it: a `pre` under `body { font-size: 12px }` is 12px, one under the default 13px, and WPT's CSS 2.1 tests mix both. ui drew `code`, `kbd`, `samp` and `pre` in Go Mono at 0.8125em, which is 13px only under 16px.

## Decision

- Under `ui.BrowserSheets()` a font's size remembers the keyword it comes from and whether an absolute length stands on its way down (`style.Font.Keyword`, `Absolute`), as Chrome's font description does; a keyword in a `monospace`-only family takes Chrome's monospace table (9, 10, 12, 13, 16, 20, 26, 39px); a size neither absolute nor a keyword scales by 13/16 going into such a family and by 16/13 coming out.
- The browser's sheet sets the root's size to `medium` and `code, kbd, samp, pre` to `monospace` at the inherited size, as Chrome's sheet does.
- A game's sheets keep 16px everywhere and code at 0.8125em of Go Mono.
- The reftest runner still names `monospace` Go Mono for Chrome, which then draws it at 16px: Chrome gives no way to draw its default monospace in a face of the page's, and its own face has other heights than Go Mono.

## Alternatives

- Keeping 0.8125em of Go Mono: wrong wherever the parent's size is absolute or code nests.
- Applying it to a game's sheets too: a game naming monospace means the size it asked for.
- Leaving a lone `monospace` to the runner's Chrome, so it takes 13px there too: Chrome then draws its own monospace face, and four times as many reftests stop matching it.

## Consequences

Pages in a browser's sheets size monospace text as Chrome does; the reftests whose author monospace takes its size from a keyword no longer match the runner's Chrome, which draws them at 16px (`ui/browsersheets_chrome_test.go` holds the real Chrome to the quirk). A quoted `"monospace"` is a family name in Chrome but ui reads it as the generic, so it takes 13px too.
