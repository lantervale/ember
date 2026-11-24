# 0222. A press inside the selection drags its text, and a long press selects a word

- Status: Accepted
- Date: 2026-11-24
- Area: api
- Owner: engine

## Context

A press inside selected text started a new selection, where Chrome drags the text into a field, and a finger on a touchscreen selected nothing. ui has no system drag and drop to hand a drag to, and no touchscreen here to compare a finger with Chrome's.

## Decision

- A left press inside the selection (the page's, or the focused field's but a password's; a caret place from its start to its end counts, as Chrome's) leaves it as it is. Once the pointer is 3 points from the press along either axis it drags the text; let go over a text field that takes text, the text goes in at the place under the pointer, selected there and the field focused. From the page it is copied; from a field it moves, and is copied with Option on macOS or Control elsewhere; on macOS a move takes a space with it as Chrome's smart delete does. Let go elsewhere, or back on its own selection, nothing changes; Escape cancels it. A press let go where it fell clears the selection (a field's caret there); one that moved less leaves it.
- On macOS the press drags only once held 0.15 s, as Chrome's text drag delay: before that a first move past the 3 points has a caret follow the pointer, as Chrome does, and a shorter one selects from the press, as Chrome did when its moves came a frame apart (as they come further apart its caret follows the pointer then too).
- The drag shows a ghost of the text, `.ember-drag-image`, fixed under the pointer as the press held it in the selection, in the source's font and colour at 0.75 opacity: what ui can draw for the image Chrome hands the system. The drop has no caret of its own in the field before it falls, and only drags within one Document.
- On a touchscreen (`input.Touchscreen`), a finger held 0.5 s within 15 points selects the word under it, in the page or in a field it focuses; the page's selection then shows two handles, `.ember-selection-handle`, whose drag takes that end along, the handle's point kept where the finger holds it. A tap clears it; the pointer or the keys changing the selection hide them. It follows Chrome's gesture timings and is driven by simulated touches alone: no touchscreen here, and headless Chrome turns DevTools' touches into a click however long they are held, so it is not compared.

## Alternatives

- Handing the drag to the system's drag and drop: GLFW has none to start, and a game's text would leave the window.
- Leaving the press to select, as before: a page made for a browser would not move text between its fields.

## Consequences

Selected text moves and copies between a Document's fields as in Chrome on its desktop, compared through DevTools' intercepted drags. Drops between Documents, drops from or to other programs, a field's handles and the touch selection's look are not Chrome's.
