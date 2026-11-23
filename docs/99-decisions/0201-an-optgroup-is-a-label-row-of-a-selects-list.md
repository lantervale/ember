# 0201. An optgroup is a label row of a select's list

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

Pages group a select's options under `optgroup` labels, and ui had no such element: light built it as a div, and the select lost the options inside it.

## Decision

- `ui.Optgroup(label, args…)` is an element whose `option` children belong to the select round it, as the DOM's `options`: one run of indices across groups for `doc.Selected`, `SetSelected`, `Value` and automation's `select`.
- The open list shows each label as a row of its own, bold and never chosen: the arrows, typing and the pointer pass over it, and a click on it keeps the list open. A group's options are drawn behind four spaces, which Chrome also counts in the select's width (`TextIndentedToRespectGroupLabel`).
- `disabled` on the group disables its options; the group and they match `:disabled`, an enabled group `:enabled`. A required select's empty first option is a placeholder only outside any group.
- A select with `multiple` or a `size` stays a dropdown: the listbox, where Chrome draws groups as rows indented 20px, is its own feature.

## Alternatives

- Flattening groups into options at build time: the labels would be lost from the list, and `:disabled` on a group and its options would not hold.
- Drawing the labels as disabled options: they would count among the indices and automation would offer them.

## Consequences

The closed select, its width, its choice and its states are compared with Chrome; the open list is not, Chrome on macOS showing it as a native menu.
