# 0202. A select with multiple or a size is a listbox of rows in the page

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A `select` with `multiple` or a `size` above 1 is a listbox in a browser: its options are rows of the page, several may be selected, and the pointer and the keys select them. ui showed it as a dropdown that chose one option.

## Decision

- A listbox's options and optgroups are laid out as blocks of the page, as Chrome's are: the user agent sheet gives them their look under `:-ember-listbox`, ui's own pseudo-class for Chrome's `:-internal-list-box` (padding 0, square corners, `overflow: hidden scroll`, `vertical-align: text-bottom`, selected rows by `option:checked`, other colours with the focus). Export marks the element `data-ember-listbox` for the same rules.
- Its height is its `size` (4 for `multiple` alone) in rows as tall as its tallest option or label; a group's label is a row of its own before its options, a box layout makes from the group's `label` with an option's padding and least height, standing for Chrome's shadow element, never a `::before` a page could style.
- `control.Select` keeps the selection: every option selected, the option the keys move from and the anchor of a run, and follows Chrome's list box rules: a run passes over disabled options, the page keys go a page less a row counting labels, a press's change fires once it lets go. Command toggles on macOS, Control elsewhere.
- `doc.SelectedOptions`, `SetSelectedOptions` and `Values` read and set every selected option and what a form submits; `Selected` stays the first. Automation's `select` takes a list of options and selects them alone, and records a multiple listbox's value as the list.

## Alternatives

- Drawing the rows from the control, as the dropdown's open list is: page CSS on options (colours, padding, heights) would not hold, nor hit testing, scrolling and `:checked`.
- An optgroup label as a `::before` with `content: attr(label)`: Chrome has none, so a page's own `::before` would collide, and the label would not translate.

## Consequences

Geometry, selection under scripted pointer and keys, colours in both schemes, form values and validity are compared with Chrome; a static page compares pixels. Typing finds one letter at a time, and an empty listbox's rows follow ui's font metrics, a point off Chrome's.
