# 0162. A copy puts the selection's markup beside its text

- Status: Superseded by [0224](0224-a-rich-copy-writes-chromes-markup.md) for its markup; its clipboard API stays
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A copy from a page put plain text alone on the clipboard, so text pasted into a mail or a document lost its bold, its links and its lists, where a browser's copy keeps them by writing HTML beside the text.

## Decision

- A copy puts `doc.SelectionHTML()` beside the plain text: the selected nodes, an element held in part cut to the selection, as the DOM's `Range.cloneContents` serialises them, inside the inline elements round the selection up to its block. Each element keeps its own attributes; no computed style is written.
- `platform.Window` takes both through `SetClipboardHTML(html, text)` and reads the HTML back with `ClipboardHTML()`; a backend whose system clipboard takes HTML says so by `platform.HTMLClipboard`. macOS writes `public.html` and the string as one pasteboard item, Windows `HTML Format` (CF_HTML, `internal/clipboard`) and `CF_UNICODETEXT` in one opening of the clipboard.
- Elsewhere (a Linux desktop, whose targets GLFW offers as text alone, the web) and with no backend that takes it, the clipboard gets the text alone.

## Alternatives

- Write Chrome's copy markup, each element's computed style inline: it pastes the look whatever the target's sheets, but the markup grows with every property and ties the copy to the cascade's output, where the elements and their attributes are what the program built.
- Hand the copy to the program by an event and let it fill the clipboard: every program would rebuild what the Document already knows.

## Consequences

Pasted text keeps its structure, bold, links and lists, in apps that read HTML; its colours and sizes are the target's. The markup is checked against Chrome's `cloneContents`; Chrome's own copy markup can't be read headless.
