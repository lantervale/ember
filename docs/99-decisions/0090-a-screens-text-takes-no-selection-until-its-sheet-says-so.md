# 0090. A screen's text takes no selection until its sheet says so

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: engine

## Context

Only text fields could select. A page shown by ui (the web explorer's) must let its text be selected and copied as a browser does, but a game screen whose labels turn blue under a drag, or whose Cmd+A selects the HUD, is broken.

## Decision

- `user-select` is read with Chrome's values and Chrome's model: `auto`, `text`, `none`, `all`, inherited, `auto` set on an element being `text`.
- ember's user-agent sheet has `html { user-select: none }`; under `ui.BrowserSheets()` the root is `auto`, as [0085](0085-browser-sheets-are-an-option-of-a-document.md) does for `box-sizing`. A game opts in with `user-select: text` on the element to copy from. kit sets nothing, so its components behave as before.
- A Document has one selection outside its fields: two places, each a text node and a character offset in its text as shown, so it outlives layout and goes with its nodes. A field keeps its own; one taking the focus clears the page's.
- The pointer and the keys act as Chrome on macOS: drag, double and triple click, Shift and a click; a press on a link, a button, a control or `user-select: none` text neither starts nor clears a selection; Escape leaves it; copy and select all are taken only while no field has the keyboard and there is something to copy or select.
- What is copied is Chrome's `Selection.toString()`, as plain text. The highlight is the line box's height per fragment, `::selection` setting its `background-color` and the text's `color`, inherited from the parent's `::selection`.
- The API is `doc.Selection()`, `Select`, `SelectAll`, `SelectContents`, `ClearSelection`, `SelectionRects`, the event `SelectionChanged` with `OnSelectionChange` on `body`, and automation's existing `select-all` act on a selectable label.

## Alternatives

- Selectable by default, as a browser: every existing game screen would change, kit's included, and each would need `user-select: none`.
- A Document option (`ui.SelectableText()`): a second switch beside the CSS property that already says it, and no way to open one panel of a screen.
- Offsets in the collapsed text or in fragments: cheaper to map, but they move when a restyle changes the wrapping or the white space.
- The spec's `auto` (the parent's used value, `contain`): Chrome implements neither, and pages are written for Chrome.

## Consequences

A screen without `user-select: text` or `ui.BrowserSheets()` has no selectable run and no selection, and pays one nil check a frame. Selecting is tested against Chrome for what is copied, where it is drawn and how it looks; the pointer and keys, which headless Chrome cannot be driven with, are tested in ember alone, and what is left out is in the guide's [missing page](../05-ui/10-missing.md).
