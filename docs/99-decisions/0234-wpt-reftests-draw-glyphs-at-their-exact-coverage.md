# 0234. WPT reftests draw glyphs at their exact coverage

- Status: Accepted
- Date: 2026-11-26
- Area: tools
- Owner: engine

## Context

ui darkens glyphs' stems by default, as Chrome's text on macOS reads heavier than exact coverage (font smoothing). WPT's references draw boxes where their tests draw Ahem's squares, so every square of text has a darkened fringe its reference lacks: half of the CSS 2.1 failures were text drawn at the right place, and Chrome on macOS misses those references alike.

## Decision

- The sheet the reftest runner puts under every page ([0229](0229-wpt-reftests-run-on-ember-against-ember-and-against-chrome.md)) also says `:root { -webkit-font-smoothing: antialiased }`: glyphs at their exact coverage, a square of Ahem on whole pixels the box its reference draws.
- Chrome is served the same sheet, so `chrome` marks still compare like with like; a page's own `-webkit-font-smoothing` still wins, as it comes after.
- Text elsewhere in ui keeps the darkened default.

## Alternatives

- Darkening only inside a glyph's own pixels: Ahem's squares would match their boxes, but ember's text would read unlike Chrome's everywhere.
- Leaving the default: the reftests would test the platform's font smoothing rather than layout, and fail where Chrome on macOS fails too.

## Consequences

The reftests check where glyphs stand, not how heavy macOS draws them; how ui darkens text stays checked against Chrome by the font-smoothing pixel page and the text pages.
