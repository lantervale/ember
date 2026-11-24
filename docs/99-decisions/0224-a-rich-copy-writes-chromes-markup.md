# 0224. A rich copy writes Chrome's markup

- Status: Accepted
- Date: 2026-11-24
- Area: api
- Owner: engine

## Context

A copy's HTML held the selected elements and their attributes only (decision 0162), so a paste took its colours, fonts and sizes from the target, where a copy from Chrome pastes looking as the page did.

## Decision

- `doc.SelectionHTML()` writes what Chrome's copy writes (its `StyledMarkupSerializer`), checked against Chrome's own markup, which a drag of the selection carries headless: the nodes the selection holds, cut to it; round them the element a paste needs (the list of a whole item, the table of a row, a pre, list or heading the selection stands in, the highest bold, italic, underline or strike-through, a link); `<br class="Apple-interchange-newline">` for a paragraph break at either end; a no-break space in a span of its own at an end.
- Each element's style attribute holds what its rules and its own style declare, ember's own sheet among them since a paste knows nothing of it, written as Chrome writes declarations; the elements and text beside the highest one written also carry the text styles they show in (colour, font, spacing, alignment, white-space, the background and decorations in effect), but those Chrome's own sheet gives the element.
- macOS's pasteboard gets the markup after `<meta charset='utf-8'>`, as Chrome's; Windows keeps CF_HTML's `StartFragment` markers; the web writes `text/html` and `text/plain` as one `ClipboardItem`. A Linux desktop gets the text alone: GLFW owns the X11 selection and the Wayland data source and offers text only.

## Alternatives

- Keep the bare elements (0162): small, but a paste loses the page's look.
- Write each element's whole computed style: a paste keeps every property, but the markup grows tenfold and ties every target to ember's layout properties, which Chrome leaves out.

## Consequences

Supersedes [0162](0162-a-copy-puts-the-selections-markup-beside-its-text.md)'s markup; its clipboard API stays. Pasted text looks as it showed, at the cost of markup Chrome's size; links keep their address as written, where Chrome resolves them against the page's.
