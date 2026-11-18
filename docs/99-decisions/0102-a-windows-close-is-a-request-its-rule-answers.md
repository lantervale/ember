# 0102. A window's close is a request its rule answers

- Status: Accepted
- Date: 2026-11-18
- Area: api
- Owner: user

## Context

The desktop loop ended when the first window closed, whatever else was open, and closing another window only removed it. That is right for a game, and wrong for a browser, an editor or a tool: a program with several equal windows lost them all with the first, could not ask "really close?", and could not stay running without a window as macOS programs do. What a close does is the program's choice, not the engine's.

## Decision

- A close is a request. The close button, the system's shortcut for it and `win.RequestClose()` all ask; as the next frame begins the World emits `platform.WindowCloseRequested` and follows the window's rule. A test asks the same way, with no desktop.
- The rule is the window's `platform.OnClose(rule)`, read by `win.OnClose()` and changed by `win.SetOnClose(rule)`:
  - `CloseQuitsIfPrimary`, the default: the primary window's close ends the program, another's closes it. Programs written before this decision behave as they did.
  - `CloseQuits`: ends the program.
  - `CloseQuitsIfLast`: closes the window, and ends the program when none is left.
  - `CloseOnly`: closes the window; the program runs on, without a window when it was the last.
  - `CloseAsks`: nothing. The program reads the event and decides, removing the window or quitting when it wants to.
- A rule that closes a window removes its entity and emits `platform.WindowClosed`. One that ends the program quits the World and leaves the window in it.
- The primary window is the oldest in the World: the first added, then the first added of those left. When it goes, cameras, Documents and 2D entities under no window show in the next, which takes the pads as they are, with no `GamepadConnected` again, the capture scripts, Shift+F12 and the F3 overlay. With no window left there is no primary: `wm.Input` is idle, nothing is presented, and the next window added is the primary.
- The desktop loop ends when the World quits, never by itself. Without a window it waits out the fixed step between frames, so the program ticks at that rate and leaves the processor alone. The program opens a window again by adding one.
- What the system asks of the program itself is not a window's close. macOS's Quit ends the program whatever the rules. A click on the Dock icon of a program showing no window emits `platform.ProgramReopened`, for the program to open a window; Windows and Linux have no such gesture, and a program there that closes its last window must keep its own way back (a tray, a shortcut, automation) or quit.
- Whatever an engine package keeps by window leaves with the window: `wm.OnGone(w, fn)` tells it. The renderer releases the window's pass textures, light pictures, 3D targets, timers and readbacks, twod its canvas and lists.
- On the web the page's canvas is the one window and the tab is the browser's: nothing asks a window to close. `RequestClose` still follows the rule in the World, and a World that quits stops drawing.

## Alternatives

- The engine quits with the last window: one more fixed policy. It breaks the games whose first window is the game, and the macOS convention.
- The engine quits with the primary only, as before: the problem.
- One policy on the World (an enum given to `ember.New`): windows are entities with options, and a program's windows differ. A document window asks, a palette only closes, and the rule changes while a window holds unsaved work.
- A veto callback on the window (`OnClose(func() bool)`): it must answer at once, where "really close?" is a dialog answered frames later, and it runs from the platform's event, outside the frame.
- Events only, the engine doing nothing: every program writes a system for the common cases, and a program written before would stop quitting.
- A hidden window kept open so the program outlives its last one: it shows in the system's window lists, and Quit still could not be told from a close.

## Consequences

A program picks a rule per window in one option, and `CloseAsks` with the event covers what the rules don't. Closing the primary is no longer special in the engine, so the primary can change under code that kept it: engine code asks `wm.Primary` each time. macOS's Quit cannot be refused yet, and a window's GPU and system memory is given back when it closes, which the loop never did for the first window.
