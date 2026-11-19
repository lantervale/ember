# 0110. A Document scales what the wheel scrolls

- Status: Accepted
- Date: 2026-11-19
- Area: api
- Owner: user

## Context

The web explorer wants two settings a browser's reader expects: the wheel reversed, and a scroll speed. The wheel goes from the window's input straight into the Document's scrolling, a wheel over an iframe is offered to the Document it shows first, and each asks whether it can still scroll that way: a program has nowhere to put a factor.

## Decision

- `ui.WheelScroll(f)` is a Document option, with its getter and `doc.SetWheelScroll(f)`: the Document scrolls f times what the wheel says. 1 is the default and changes nothing, a negative f reverses the wheel, 0 stops it.
- The factor of the Document in the window holds for every Document shown in its iframes, whatever their own: it is what the person asked of their wheel, not a trait of a page.
- Only scrolling is scaled. `win.Input().Wheel()` stays the wheel as it comes, so a game's zoom, a page's `wheel` event a program makes of it and a number field the wheel steps are untouched.

## Alternatives

- Scale the wheel in the window's input: every reader of the wheel would be turned with it, a game's zoom and what a page's script is told included.
- Each Document its own factor, one in an iframe too, as `ui.RightDragSelects()`: a program showing pages would set it on every Document those pages frame and again at each change, and an iframe built with another factor would scroll against the page round it.
- Give the program the pieces (the box under the pointer, a scroll by an offset): it would write again which box can still scroll and the hand-over from an iframe to its page, and still have the Document scroll by the same wheel.

## Consequences

A Document without the option scrolls as before. The distance and the direction of the wheel stay the backend's; the option is a preference laid over them, and an iframe cannot have a wheel of its own.
