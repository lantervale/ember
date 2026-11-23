# 0206. A select's type-ahead and a listbox's autoscroll follow Chrome's

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A listbox ([decision 0202](0202-a-select-with-multiple-or-a-size-is-a-listbox-of-rows-in-the-page.md)) found an option one letter at a time, a dropdown took no typing, and a drag past a listbox's edge reached only the options in view. Each of these is a choice of rules and pace, measured against Chrome.

## Decision

- Typing is Chrome's type-ahead (`control.Select.Type`, on the Document's clock): letters within a second of each other are a prefix the first selected option or the next enabled one after it starts with, any case, leading spaces passed over; one letter typed again goes on to the next option starting with it. A dropdown chooses (input and change for a new choice), a listbox selects alone; a listbox no press or key has moved yet fires a change alone for every option found, as Chrome's `ListBoxOnChange` with no saved selection, and after one input and change for a new selection.
- Once a held press moves over a listbox, a pointer within 20 points of its top or bottom edge, or past it, scrolls it every frame (at 60 a second) by a point more than it lies into that belt at the top, two at the bottom, the pointer's place rounded; an option a still pointer comes over waits for it to move.
- A listbox's keys scroll it at once, as Chrome's listbox does where its other scrollers glide.
- Under forced colours, selected rows are `Highlight` under `HighlightText`, focused or not, a disabled one's `GrayText` under `Canvas`: the user agent sheet's keywords, ui forcing no palette.

## Alternatives

- The text selection's drag scroll (a speed per point past the edge, from the edge outwards): Chrome's listbox scrolls inside the belt too, at its own pace a frame.
- A listbox firing input and change only for a new selection, as its keys do: Chrome's change for an unchanged option, before any press or key, would go missing.

## Consequences

Typing, its events, the autoscroll's pace and the forced rows' keywords are compared with Chrome. The pace is a frame's, scaled by the frame's time, so a slower frame scrolls as far a second.
