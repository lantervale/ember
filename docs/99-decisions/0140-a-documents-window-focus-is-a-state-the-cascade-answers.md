# 0140. A Document's window focus is a state the cascade answers

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Chrome greys a selection when its window loses the focus, and `::selection:window-inactive` styles it then. ui had no notion of the window's focus: a test window was never focused, and the cascade had no way to match a rule on it.

## Decision

- A `platform.Window` is focused until its system says otherwise; `win.SetFocused(on)` stands for the system in tests and captures, as `SetKey` does for the keys.
- Each frame the Document reads its window's focus (an iframe's, its host's) into what it answers the cascade with (`css.Media.Inactive`). `::selection:window-inactive` matches only then, with a pseudo-class's specificity; the tree is styled again when the focus flips and some such rule was read, as for `prefers-color-scheme`.
- Paint is told too: a selection with no `::selection` colours is drawn in Chrome's inactive colour on macOS; an author's colours stay.
- `:window-inactive` anywhere else matches nothing, as in Chrome.

## Alternatives

- Two cascades of `::selection`, active and inactive, kept on every style: no restyle on a focus change, but twice the selection's fields on every element for a rule pages rarely write.
- A Document option or setter for the focus: a second source of truth beside the window's own, which every program would have to keep in step.

## Consequences

A window losing the focus repaints its Documents and restyles those whose sheets read `:window-inactive`. Headless Chrome's page is neither reliably focused nor not, so the inactive look is tested in ember alone.
