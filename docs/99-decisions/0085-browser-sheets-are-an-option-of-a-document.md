# 0085. Browser sheets are an option of a Document

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: engine

## Context

ui reads a CSS subset and warns of the rest (decision 0049), which is right for a game's sheet, where a prefix or an `@supports` is a mistake. A sheet written for browsers is full of them, and its widths are the content's. The web explorer rewrote such sheets as text before ui had them: it unwrapped `@supports` and `@layer` unread, renamed `-webkit-` properties and cut the other engines' with regular expressions, and prepended a sheet for `box-sizing`. The text rewrites knew nothing of the grammar (a prefixed value, selector or at-rule kept its warning), and no other tool could use them.

## Decision

- `ui.BrowserSheets()` is one Document option, off by default, with `doc.BrowserSheets()`: every sheet and style attribute of the Document is read as Chrome reads it, wherever the option stands among the others. Without it a Document reads and warns exactly as before.
- `@supports` is evaluated against ui itself: a declaration is supported when the same parser keeps it without a warning, `selector()` when it reads the selector. A block whose condition fails goes without a warning.
- `@layer` is real cascade layers: statements, blocks, nested and anonymous layers, ranked across the Document's sheets; unlayered rules win, and `!important` reverses the order.
- `box-sizing` starts as `content-box`, by a second user-agent sheet (`css.BrowserDefault`) holding the controls Chrome sizes by their border.
- Vendor prefixes are one table per vendor in `ui/css/vendors.go` (`-webkit-`, `-moz-`, `-ms-`, `-o-`, `-khtml-`), each saying what happens to its properties, values, selectors, at-rules and media features: read as the plain name, dropped without a warning, or left with its warning. The rule is Chrome's: what Chrome reads as a plain name ui reads as that name, in its place in the cascade; what Chrome drops ui drops in silence; what Chrome reads and ui can't keeps its warning. The guide prints the tables, and a test keeps the two equal.
- The `-webkit-` lists are checked against the Chrome the suite runs (`TestChromeReadsWebkitPropertiesAsTheTable` and its neighbours), so a new Chrome that changes an alias fails a test.
- The explorer passes the option and no longer rewrites a sheet.

## Alternatives

- Keep rewriting text in the tool: each tool that shows a page repeats it, regular expressions misread strings, comments and nesting, and `@supports` and `@layer` can only be assumed, not answered.
- Always lenient: a game's sheet would lose the warnings that catch `-webkit-` copied from the web, a typo in an at-rule, and a width that silently means another box. The subset's warnings are how a screen is kept equal to Chrome.
- One option per feature (`Supports()`, `Layers()`, `ContentBox()`, `Prefixes()`): four names for one intent, a page needing all four; none is useful alone on a game's sheet.
- A per-sheet option: the cascade layers and the user-agent `box-sizing` are the Document's, and style attributes belong to no sheet.
- Assume every `@supports` true: a sheet's fallback for a missing feature (`@supports not (display: grid)`) would apply beside the rule it replaces.

## Consequences

A Document with the option is silent about names no browser but another reads, so it is the wrong way to check a sheet. Rules ui still can't read warn as before (`columns`, `animation`, Chrome's own `-webkit-` properties and values, `@import`). A declaration ui refuses whose value holds another engine's word goes whole in silence, as Chrome drops it, even when a plain part of it was the fault. A page exported for Chrome carries the sheet as it was written, so Chrome reads the original.
