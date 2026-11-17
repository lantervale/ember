# 0091. A window keeps the system's frame under its own title bar

- Status: Accepted
- Date: 2026-11-17
- Area: api
- Owner: user

## Context

A program such as the web explorer draws its own header where the title bar is, as a browser draws its tabs. `platform.Undecorated()` gives it the room but takes the system's window away: no buttons, no resize edges, no shadow, no rounded corners.

## Decision

- `platform.OwnTitleBar()` makes the window's content fill its title bar, the title hidden. The window stays the system's. It is set when the window is made; `win.OwnTitleBar()` reads it back.
- `win.TitleBarArea()` says what the system still draws over the content, in points: the bar's `Height` and the widths its buttons take at the `Left` and the `Right`. It is zero fullscreen, so a program reads it each frame; there is no event.
- `win.StartDrag()` hands the left press being handled to the system, which moves the window until the release. `win.DoubleClickTitleBar()` does what the system does for a double click on a title bar (on macOS zoom, minimise or nothing, by the user's setting).
- ui reads `app-region: drag | no-drag | none` (and `-webkit-app-region` in browser sheets). A left press on an element in a drag region calls `StartDrag`, a second within 400 ms at the same place `DoubleClickTitleBar`. What clicks (a control, a link, a `tabindex`) and what is inside it is never part of the region, so a header's buttons need no `no-drag`. The property is inherited, where Chrome cuts `no-drag` boxes out of `drag` boxes: the same regions for nested boxes.
- macOS does all of it: a transparent title bar over a full-size content view, with an empty compact toolbar so the bar is a header's height (40 points on macOS 26) and the system centres its buttons in it. The engine moves no button itself. Windows and Linux keep the system's title bar above the content: the area is zero and the two calls do nothing, since the system's bar drags and zooms. A page has no title bar: zero, nothing. A hidden run applies nothing and reports zero unless the program calls `win.SyncTitleBar`, which is also the tests' seam.

## Alternatives

- An undecorated window with buttons, resize edges and a drag drawn by the program: each program redoes the system's work, and gets its shadow, corners, snapping and accessibility wrong.
- A title bar drawn by the engine: a second look to keep right on every system, and a header is the program's design, not the engine's.
- An option per platform (`MacTransparentTitleBar`, `WindowsCaptionHeight`…): programs would branch on the system; one option with an area that is zero where nothing overlaps needs no branch.
- Moving the window's buttons to suit a header of any height, as Electron does: it reaches into the title bar's private views and must be redone at each resize; the header takes the system's height instead.
- Windows and Linux as undecorated windows: GLFW 3.3 gives no resize edges or buttons there, so the window would be worse than with its title bar.
- Drag regions declared to the backend as rectangles, as Electron does: the backend would hit test without knowing what is drawn over a region; ui already knows the element under the pointer.

## Consequences

A header pads itself by `TitleBarArea()` and marks its empty part `app-region: drag`, and the same code runs under a system title bar elsewhere. Windows and Linux do not get a merged header until the backend can do it honestly (GLFW 3.4's hit test, or native code).
