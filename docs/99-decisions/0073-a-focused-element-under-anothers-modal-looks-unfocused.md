# 0073. A focused element under another's modal looks unfocused

- Status: Accepted
- Date: 2026-11-15
- Area: core
- Owner: architect

## Context

A window can hold several Documents, each with its own focus. When one
opens a modal, it owns the window's input ([0053](0053-a-modal-owns-its-windows-input.md)):
the others get no keys and no pointer, so their `:hover` goes, but
their focused element still matched `:focus`. A button clicked to open
a dialog in another Document stayed painted as focused under it.

## Decision

- While a Document does not take its window's input because another
  owns it, its focused element matches neither `:focus` nor
  `:focus-visible`.
- It keeps the focus: `doc.Focused(n)` stays true, and no event is
  emitted. Both pseudo-classes come back the frame the Document takes
  the input again, with the ring it had.
- The owner's own focused element is untouched, and so is a Document
  drawn under the owner (an element hosting it).

## Alternatives

- Each screen blurs before it opens another Document's modal and
  focuses again after. Every game screen has to remember it, and the
  focus's place is lost if it forgets.
- Move the focus to the owner's Document. Focus is per Document; one
  window-wide focus would change what `Focused` and Tab mean for every
  layer.

## Consequences

A screen under a dialog of another Document looks idle, as an element
focused in a browser frame that is not the focused one does. A sheet
that styles `:focus` shows nothing there until the dialog closes; a
game that wants a mark to stay uses a class of its own.
