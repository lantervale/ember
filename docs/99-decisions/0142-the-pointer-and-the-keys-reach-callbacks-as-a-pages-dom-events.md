# 0142. The pointer and the keys reach callbacks as a page's DOM events

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Building the light browser on ember left a page's tools out of reach: an address field kept the focus after Enter, a console could not hear Up in its field nor show a line of history there, a context menu could not open where it was asked, an edge dragged by reading `win.Input()` each frame, `title` showed nothing, and a header padded itself in Go from `win.TitleBarArea()`.

## Decision

- `doc.Blur()` takes the focus as the DOM's `blur()`: the field commits, the keys go back to the game.
- In a focused text field `OnKey` hears every key pressed or repeated before the field acts, bubbling as a keydown, and `e.PreventDefault()` keeps the field from it. Those keys are the field's: no `KeyPressed` is emitted, so a game's hotkeys don't fire while a player types.
- `doc.SetValue` called from a callback of the field's own events sets it at once, the caret at the end. Every other `SetValue` still waits while the player types.
- `ContextClicked` and `Event.At` carry the pointer's place; on macOS Control with the left button is a context click that neither focuses nor clicks.
- `OnPointerDown`, `OnPointerMove`, `OnPointerUp` (`PointerPressed`, `PointerMoved`, `PointerReleased`) bubble as their DOM events; `doc.SetPointerCapture(n)` sends the moves and the release to `n` until no button is down. The page's `html` and `body` stay the game's.
- `title` shows as Chrome's tooltip on Windows and Linux: after half a second, 10 and 15 points from the pointer, as `.ember-title-tip`, the html's last child while it shows, Chrome's look in the user-agent sheet and a kit tip's in kit's.
- `env()` substitutes as `var()`: `titlebar-area-*` from the window's title bar area, undefined without one as in Chrome without the overlay; the safe area and keyboard insets `0px`.
- `app-region` stays inherited, as Chrome computes it, and `none` set computes `no-drag`.

## Alternatives

- A `KeyPressed` World event for the field's keys too: every game reading keys would hear the player's typing.
- A separate `SetValueNow`: a second setter for the one case where the reaction is the field's own.
- A pointer capture held by an `Event` method: capture outlives the event, so it belongs to the Document, as its focus does.
- The title tip painted by `paint`, as a select's list: its look could not come from a sheet, so kit could not theme it.

## Consequences

A page's tools are written with callbacks beside their elements. An element appears under `html` while a title tip shows; selectors over `html`'s children may see it. Pens, fingers, `keyup` and `preventDefault` on a press stay missing.
