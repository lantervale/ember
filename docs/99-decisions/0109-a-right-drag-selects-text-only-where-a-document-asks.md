# 0109. A right drag selects text only where a Document asks

- Status: Accepted
- Date: 2026-11-19
- Area: api
- Owner: user

## Context

The web explorer shows a card of what to do with a passage, and wants it for text selected with the right button, the left one's selection staying a plain one. ui follows Chrome ([0090](0090-a-screens-text-takes-no-selection-until-its-sheet-says-so.md)), where only the left button selects, and a program cannot select from outside: the places under the pointer, the drag's scrolling and the iframes' gating are the Document's.

## Decision

- `ui.RightDragSelects()` is a Document option, off by default, with its getter. Each Document takes it for itself, one shown in an iframe too.
- With it, a right press where a left press would begin a selection is armed; once the pointer has been held 4 points from it, the drag `dragStart` already asks of a draggable element, it selects by characters from the press to the pointer and scrolls as a left drag does.
- Let go with text selected, the drag is a `ui.RightDragSelected{Node}` (`ui.OnRightDragSelect` on `body`) and no `ContextClicked`: not the Document's, and not those of the Documents whose iframes it is shown through, whether they have the option or not.
- A right press let go where it fell, or after a drag that left nothing selected, is the right click it always was and leaves the selection alone.

## Alternatives

- Always select with the right button: every screen with an `OnContextMenu` on selectable text would lose its menu to a hand that moved, and no browser does it.
- Give the program the pieces (a place from a point, a public drag): each program would write the threshold, the scrolling and the iframe's hold of the pointer again, and still hear the right click of the Document around.
- A field of `Selection` saying which button made it: it would outlive the gesture, and say nothing of when the button is let go, which is when a program acts.
- Keep the `ContextClicked` and let the program tell the two apart: the menu and the selection's own answer would both open unless every listener knew of the drag.

## Consequences

Without the option nothing changes. With it the right button has two meanings told apart by a 4 point move, so a menu wanted on a selection is asked by a click, not at the end of a drag; the gesture is ember's own and is tested in ember alone, Chrome having none to compare with.
